package tools

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// allowLoopback relaxes the SSRF guard so tests can reach httptest servers,
// which always listen on 127.0.0.1.
func allowLoopback(wt *WebTool) *WebTool {
	wt.blockIP = func(net.IP) bool { return false }
	return wt
}

func TestWebFetchGET(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s", r.Method)
		}
		w.Write([]byte("page body"))
	}))
	defer srv.Close()

	wt := allowLoopback(NewWebTool(5 * time.Second))
	res, err := wt.Execute(context.Background(), map[string]any{"url": srv.URL})
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if !strings.Contains(res.Output, "page body") || res.ExitCode != 200 {
		t.Errorf("result: code=%d out=%q", res.ExitCode, res.Output)
	}
}

func TestWebFetchRejectsNonHTTP(t *testing.T) {
	wt := NewWebTool(5 * time.Second)
	for _, u := range []string{"file:///etc/passwd", "ftp://host/x", "not-a-url"} {
		if _, err := wt.Execute(context.Background(), map[string]any{"url": u}); err == nil {
			t.Errorf("expected rejection for %q", u)
		}
	}
}

func TestWebFetchSizeCap(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(make([]byte, 3<<20)) // 3 MiB
	}))
	defer srv.Close()
	wt := allowLoopback(NewWebTool(5 * time.Second))
	res, err := wt.Execute(context.Background(), map[string]any{"url": srv.URL})
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if !res.Truncated || len(res.Output) > defaultMaxOutput {
		t.Errorf("expected truncation, got len=%d truncated=%v", len(res.Output), res.Truncated)
	}
}

// F-03: isBlockedIP classifies SSRF-relevant addresses.
func TestIsBlockedIP(t *testing.T) {
	blocked := []string{
		"127.0.0.1", "::1", // loopback
		"169.254.169.254",                       // cloud metadata (link-local)
		"10.0.0.5", "192.168.1.1", "172.16.0.1", // private
		"0.0.0.0",   // unspecified
		"224.0.0.1", // multicast
		"fc00::1",   // unique-local
	}
	for _, s := range blocked {
		if !isBlockedIP(net.ParseIP(s)) {
			t.Errorf("%s should be blocked", s)
		}
	}
	if isBlockedIP(nil) != true {
		t.Errorf("unparseable IP should be blocked (fail closed)")
	}
	for _, s := range []string{"8.8.8.8", "1.1.1.1", "93.184.216.34"} {
		if isBlockedIP(net.ParseIP(s)) {
			t.Errorf("%s (public) should be allowed", s)
		}
	}
}

// F-03: the dialer refuses to connect to a loopback target with the real guard.
func TestWebFetchBlocksLoopback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("secret"))
	}))
	defer srv.Close()
	wt := NewWebTool(5 * time.Second) // real guard, not relaxed
	if _, err := wt.Execute(context.Background(), map[string]any{"url": srv.URL}); err == nil {
		t.Errorf("expected loopback fetch to be blocked")
	}
}
