package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListModelsParsesAndSorts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"list","data":[
			{"id":"qwen2.5-coder","object":"model"},
			{"id":"google/gemma-4-12b","object":"model"},
			{"id":"llama-3.2-3b","object":"model"}
		]}`))
	}))
	defer srv.Close()

	c := New(srv.URL+"/v1", "")
	got, err := c.ListModels(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"google/gemma-4-12b", "llama-3.2-3b", "qwen2.5-coder"} // sorted
	if len(got) != len(want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestListModelsEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"object":"list","data":[]}`))
	}))
	defer srv.Close()

	c := New(srv.URL+"/v1", "")
	got, err := c.ListModels(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty, got %#v", got)
	}
}

func TestListModelsConnectionErrorIsClassified(t *testing.T) {
	// Point at a closed port to force a transport error.
	c := New("http://127.0.0.1:1/v1", "")
	_, err := c.ListModels(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	var le *LLMError
	if !errorsAs(err, &le) {
		t.Fatalf("expected *LLMError, got %T", err)
	}
	if le.Kind != ErrUnreachable {
		t.Errorf("kind = %d, want ErrUnreachable", le.Kind)
	}
}

func TestSetModelAndModel(t *testing.T) {
	c := New("http://localhost:1234/v1", "initial")
	if c.Model() != "initial" {
		t.Errorf("Model() = %q, want initial", c.Model())
	}
	c.SetModel("switched")
	if c.Model() != "switched" {
		t.Errorf("after SetModel, Model() = %q, want switched", c.Model())
	}
}

// small local helper to avoid importing errors in the test file twice
func errorsAs(err error, target **LLMError) bool {
	for err != nil {
		if le, ok := err.(*LLMError); ok {
			*target = le
			return true
		}
		type unwrap interface{ Unwrap() error }
		if u, ok := err.(unwrap); ok {
			err = u.Unwrap()
		} else {
			break
		}
	}
	return false
}
