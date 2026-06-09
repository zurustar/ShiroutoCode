package llm

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

type fakeTimeout struct{}

func (fakeTimeout) Error() string   { return "i/o timeout" }
func (fakeTimeout) Timeout() bool   { return true }
func (fakeTimeout) Temporary() bool { return true }

func TestClassifyHTTPStatuses(t *testing.T) {
	cases := []struct {
		status    int
		body      string
		wantKind  ErrorKind
		retryable bool
	}{
		{404, `{"error":"model not found"}`, ErrModelNotFound, false},
		{400, `{"error":"the model 'x' does not exist"}`, ErrModelNotFound, false},
		{400, `bad request`, ErrHTTPStatus, false},
		{401, `unauthorized`, ErrHTTPStatus, false},
		{500, `boom`, ErrHTTPStatus, true},
		{503, `unavailable`, ErrHTTPStatus, true},
	}
	for _, c := range cases {
		le := classifyError(nil, c.status, []byte(c.body))
		if le.Kind != c.wantKind || le.Retryable != c.retryable {
			t.Errorf("status %d body %q: got kind=%d retryable=%v, want kind=%d retryable=%v",
				c.status, c.body, le.Kind, le.Retryable, c.wantKind, c.retryable)
		}
	}
}

func TestClassifyNetErrors(t *testing.T) {
	le := classifyError(fakeTimeout{}, 0, nil)
	if le.Kind != ErrTimeout || !le.Retryable {
		t.Errorf("timeout: got kind=%d retryable=%v", le.Kind, le.Retryable)
	}
	le = classifyError(context.DeadlineExceeded, 0, nil)
	if le.Kind != ErrTimeout {
		t.Errorf("deadline: got kind=%d", le.Kind)
	}
	le = classifyError(errors.New("dial tcp 127.0.0.1:1234: connect: connection refused"), 0, nil)
	if le.Kind != ErrUnreachable || !le.Retryable {
		t.Errorf("refused: got kind=%d retryable=%v", le.Kind, le.Retryable)
	}
}

// R6 (PBT): UserMessage never leaks raw body content (internal info).
func TestUserMessageNoLeakPBT(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		marker := "SEKRET_" + rapid.StringMatching(`[A-Za-z0-9]{6,16}`).Draw(rt, "marker")
		status := rapid.SampledFrom([]int{400, 401, 404, 500, 503}).Draw(rt, "status")
		body := fmt.Sprintf(`{"error":"%s token=%s"}`, marker, marker)
		le := classifyError(nil, status, []byte(body))
		if strings.Contains(le.UserMessage, marker) {
			rt.Fatalf("UserMessage leaked body marker: %q", le.UserMessage)
		}
	})
}
