package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sort"
)

// modelsResponse mirrors the OpenAI-compatible GET /v1/models payload that
// LM Studio serves.
type modelsResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

// ListModels returns the IDs of the models the server currently exposes,
// sorted for stable display. Transport/HTTP failures are classified into
// LLMError so callers can surface the same friendly guidance as completions.
func (c *Client) ListModels(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/models", nil)
	if err != nil {
		return nil, classifyError(err, 0, nil)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, classifyError(err, 0, nil)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, classifyError(nil, resp.StatusCode, b)
	}

	var mr modelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&mr); err != nil {
		return nil, newDecodeError(err)
	}
	ids := make([]string, 0, len(mr.Data))
	for _, m := range mr.Data {
		if m.ID != "" {
			ids = append(ids, m.ID)
		}
	}
	sort.Strings(ids)
	return ids, nil
}

// Model returns the model id the client currently targets.
func (c *Client) Model() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.model
}

// SetModel switches the target model (used by the interactive picker and the
// REPL /model command). Safe for concurrent use.
func (c *Client) SetModel(model string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.model = model
}
