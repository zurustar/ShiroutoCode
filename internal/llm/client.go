package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"math"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/zurustar/shiroutocode/internal/log"
)

// Client is the LM Studio (OpenAI-compatible) client.
type Client struct {
	baseURL     string
	model       string
	httpClient  *http.Client
	logger      log.Logger
	maxRetries  int
	backoffBase time.Duration
	idle        time.Duration

	mu   sync.Mutex
	caps Caps
}

// Option configures a Client.
type Option func(*Client)

func WithHTTPClient(h *http.Client) Option   { return func(c *Client) { c.httpClient = h } }
func WithLogger(l log.Logger) Option         { return func(c *Client) { c.logger = l } }
func WithMaxRetries(n int) Option            { return func(c *Client) { c.maxRetries = n } }
func WithBackoffBase(d time.Duration) Option { return func(c *Client) { c.backoffBase = d } }
func WithIdleTimeout(d time.Duration) Option { return func(c *Client) { c.idle = d } }

// New builds a Client for the given base URL (e.g. http://localhost:1234/v1)
// and model.
func New(baseURL, model string, opts ...Option) *Client {
	c := &Client{
		baseURL:     strings.TrimRight(baseURL, "/"),
		model:       model,
		httpClient:  &http.Client{},
		logger:      log.New("error", log.FormatText, io.Discard),
		maxRetries:  2,
		backoffBase: 200 * time.Millisecond,
		idle:        60 * time.Second,
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// Complete sends a completion request and returns a streaming response. In auto
// mode it tries native function calling first and falls back to the JSON
// protocol once if the server rejects tools (Functional R2 / NFR design P5).
func (c *Client) Complete(ctx context.Context, req Request) (Stream, error) {
	mode := c.resolveMode(req)
	stream, err := c.doStreaming(ctx, req, mode)
	if err != nil && mode == ToolModeFunction && req.ToolMode == ToolModeAuto && len(req.Tools) > 0 {
		if le, ok := err.(*LLMError); ok && le.Kind == ErrHTTPStatus && le.StatusCode == http.StatusBadRequest {
			c.setCaps(false)
			c.logger.Warn("llm: function calling rejected, falling back to JSON tool protocol")
			return c.doStreaming(ctx, req, ToolModeJSON)
		}
	}
	return stream, err
}

func (c *Client) resolveMode(req Request) ToolMode {
	switch req.ToolMode {
	case ToolModeFunction:
		return ToolModeFunction
	case ToolModeJSON:
		return ToolModeJSON
	default: // auto
		c.mu.Lock()
		defer c.mu.Unlock()
		if c.caps.Known && !c.caps.SupportsFunctionCalling {
			return ToolModeJSON
		}
		return ToolModeFunction
	}
}

func (c *Client) setCaps(supportsFC bool) {
	c.mu.Lock()
	c.caps = Caps{Known: true, SupportsFunctionCalling: supportsFC}
	c.mu.Unlock()
}

// doStreaming builds the payload for the mode, sends with retries until headers
// are established, and returns a Stream.
func (c *Client) doStreaming(ctx context.Context, req Request, mode ToolMode) (Stream, error) {
	body, err := buildBody(c.model, req, mode)
	if err != nil {
		return nil, newDecodeError(err)
	}

	attempt := 0
	for {
		resp, sendErr := c.send(ctx, body)
		if sendErr == nil && resp.StatusCode < 400 {
			if mode == ToolModeFunction {
				c.setCaps(true)
			}
			return newStream(ctx, resp.Body, mode, c.idle), nil
		}

		var le *LLMError
		if sendErr != nil {
			le = classifyError(sendErr, 0, nil)
		} else {
			b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			_ = resp.Body.Close()
			le = classifyError(nil, resp.StatusCode, b)
		}

		if !le.Retryable || attempt >= c.maxRetries {
			return nil, le
		}
		c.logger.Warn("llm: retrying", "attempt", attempt+1, "kind", int(le.Kind))
		select {
		case <-time.After(c.backoff(attempt)):
		case <-ctx.Done():
			return nil, classifyCtx(ctx.Err())
		}
		attempt++
	}
}

func (c *Client) send(ctx context.Context, body []byte) (*http.Response, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	return c.httpClient.Do(httpReq)
}

func (c *Client) backoff(attempt int) time.Duration {
	d := time.Duration(float64(c.backoffBase) * math.Pow(2, float64(attempt)))
	jitter := time.Duration(rand.Int63n(int64(c.backoffBase) + 1))
	return d + jitter
}

// --- payload assembly (LC2 requestBuilder) ---

type chatRequestJSON struct {
	Model       string         `json:"model"`
	Messages    []chatMsgJSON  `json:"messages"`
	Stream      bool           `json:"stream"`
	Tools       []chatToolJSON `json:"tools,omitempty"`
	Temperature *float64       `json:"temperature,omitempty"`
	MaxTokens   *int           `json:"max_tokens,omitempty"`
}

type chatMsgJSON struct {
	Role       string             `json:"role"`
	Content    string             `json:"content"`
	ToolCallID string             `json:"tool_call_id,omitempty"`
	ToolCalls  []chatToolCallJSON `json:"tool_calls,omitempty"`
}

type chatToolCallJSON struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type chatToolJSON struct {
	Type     string `json:"type"`
	Function struct {
		Name        string         `json:"name"`
		Description string         `json:"description,omitempty"`
		Parameters  map[string]any `json:"parameters,omitempty"`
	} `json:"function"`
}

func buildBody(model string, req Request, mode ToolMode) ([]byte, error) {
	msgs := make([]chatMsgJSON, 0, len(req.Messages)+1)

	if mode == ToolModeJSON && len(req.Tools) > 0 {
		// Inject the JSON-protocol instruction as a leading system message.
		msgs = append(msgs, chatMsgJSON{Role: RoleSystem, Content: fallbackSystemPrompt(req.Tools)})
	}
	for _, m := range req.Messages {
		cm := chatMsgJSON{Role: m.Role, Content: m.Content, ToolCallID: m.ToolCallID}
		for _, tc := range m.ToolCalls {
			j := chatToolCallJSON{ID: tc.ID, Type: "function"}
			j.Function.Name = tc.Name
			if tc.Args != nil {
				if b, err := json.Marshal(tc.Args); err == nil {
					j.Function.Arguments = string(b)
				}
			}
			cm.ToolCalls = append(cm.ToolCalls, j)
		}
		msgs = append(msgs, cm)
	}

	payload := chatRequestJSON{
		Model:       model,
		Messages:    msgs,
		Stream:      true,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}
	// Tools are sent only in function mode (Functional R1).
	if mode == ToolModeFunction {
		for _, t := range req.Tools {
			j := chatToolJSON{Type: "function"}
			j.Function.Name = t.Name
			j.Function.Description = t.Description
			j.Function.Parameters = t.Parameters
			payload.Tools = append(payload.Tools, j)
		}
	}
	return json.Marshal(payload)
}
