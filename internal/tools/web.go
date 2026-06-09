package tools

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// WebTool fetches a URL with GET. Only http/https are allowed; the response is
// read up to a size cap and redirects are limited (NFR design, Functional R7).
type WebTool struct {
	client *http.Client
	maxOut int
}

func NewWebTool(timeout time.Duration) *WebTool {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &WebTool{
		client: &http.Client{
			Timeout: timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return errors.New("too many redirects")
				}
				return nil
			},
		},
		maxOut: defaultMaxOutput,
	}
}

func (t *WebTool) Name() string        { return "web_fetch" }
func (t *WebTool) Description() string { return "Fetch the contents of an http(s) URL (GET)." }

func (t *WebTool) Execute(ctx context.Context, args map[string]any) (ToolResult, error) {
	raw := argString(args, "url")
	if raw == "" {
		return ToolResult{}, fmt.Errorf("url is required")
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return ToolResult{}, fmt.Errorf("only http(s) URLs are allowed")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return ToolResult{}, err
	}
	resp, err := t.client.Do(req)
	if err != nil {
		return ToolResult{}, err
	}
	defer resp.Body.Close()

	limited := io.LimitReader(resp.Body, int64(t.maxOut)+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return ToolResult{}, err
	}
	truncated := false
	if len(body) > t.maxOut {
		body = body[:t.maxOut]
		truncated = true
	}
	return ToolResult{
		Output:    string(body),
		ExitCode:  resp.StatusCode,
		Truncated: truncated,
	}, nil
}
