package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestWebFetchGET(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s", r.Method)
		}
		w.Write([]byte("page body"))
	}))
	defer srv.Close()

	wt := NewWebTool(5 * time.Second)
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
	wt := NewWebTool(5 * time.Second)
	res, err := wt.Execute(context.Background(), map[string]any{"url": srv.URL})
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if !res.Truncated || len(res.Output) > defaultMaxOutput {
		t.Errorf("expected truncation, got len=%d truncated=%v", len(res.Output), res.Truncated)
	}
}
