package tools

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"syscall"
	"time"
)

// WebTool fetches a URL with GET. Only http/https are allowed; the response is
// read up to a size cap and redirects are limited (NFR design, Functional R7).
// Connections to loopback/link-local/private/metadata addresses are blocked to
// prevent SSRF (F-03).
type WebTool struct {
	client *http.Client
	maxOut int
	// blockIP decides whether a resolved destination IP is forbidden. It is a
	// field so tests (which connect to loopback httptest servers) can relax it;
	// production uses isBlockedIP.
	blockIP func(net.IP) bool
}

func NewWebTool(timeout time.Duration) *WebTool {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	t := &WebTool{maxOut: defaultMaxOutput, blockIP: isBlockedIP}
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	// Control runs after DNS resolution with the concrete address being dialed,
	// so it also defeats DNS-rebinding and is re-applied on every redirect.
	dialer.Control = func(network, address string, c syscall.RawConn) error {
		host, _, err := net.SplitHostPort(address)
		if err != nil {
			return err
		}
		if t.blockIP(net.ParseIP(host)) {
			return fmt.Errorf("blocked connection to non-public address %s", host)
		}
		return nil
	}
	t.client = &http.Client{
		Timeout:   timeout,
		Transport: &http.Transport{DialContext: dialer.DialContext},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return errors.New("too many redirects")
			}
			return nil
		},
	}
	return t
}

// isBlockedIP reports whether ip is one the agent must not reach: loopback,
// link-local (incl. the 169.254.169.254 cloud-metadata endpoint), private
// ranges, unique-local IPv6, and unspecified/multicast (F-03). A nil IP (could
// not be parsed) is treated as blocked, failing closed.
func isBlockedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() || ip.IsUnspecified() || ip.IsPrivate() {
		return true
	}
	return false
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
