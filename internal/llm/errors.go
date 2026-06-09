package llm

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
)

// ErrorKind classifies an LLM connectivity failure (Functional R6).
type ErrorKind int

const (
	ErrUnreachable   ErrorKind = iota // connection refused / DNS
	ErrTimeout                        // deadline / idle timeout
	ErrHTTPStatus                     // 4xx / 5xx
	ErrModelNotFound                  // model missing
	ErrBadStream                      // malformed / truncated SSE
	ErrDecode                         // undecodable response (JSON fallback)
)

// LLMError is a classified, user-safe error. UserMessage never contains
// internal details (SECURITY-09); the wrapped cause is for logs only.
type LLMError struct {
	Kind        ErrorKind
	StatusCode  int
	UserMessage string
	Retryable   bool
	wrapped     error
}

func (e *LLMError) Error() string { return e.UserMessage }
func (e *LLMError) Unwrap() error { return e.wrapped }

func newBadStream(err error) *LLMError {
	return &LLMError{Kind: ErrBadStream, UserMessage: "応答ストリームが壊れています。再試行してください。", Retryable: true, wrapped: err}
}

func newDecodeError(err error) *LLMError {
	return &LLMError{Kind: ErrDecode, UserMessage: "応答を解釈できませんでした。", Retryable: false, wrapped: err}
}

// classifyError maps an HTTP/transport failure to an LLMError. Exactly one of
// (err) or (status>=400) is expected to be set.
func classifyError(err error, status int, body []byte) *LLMError {
	if status >= 400 {
		lb := strings.ToLower(string(body))
		if status == 404 || (strings.Contains(lb, "model") && (strings.Contains(lb, "not found") || strings.Contains(lb, "no such") || strings.Contains(lb, "does not exist"))) {
			return &LLMError{Kind: ErrModelNotFound, StatusCode: status,
				UserMessage: "モデルが見つかりません。モデル名を確認してください。", Retryable: false, wrapped: errFromBody(status, body)}
		}
		if status >= 500 {
			return &LLMError{Kind: ErrHTTPStatus, StatusCode: status,
				UserMessage: fmt.Sprintf("LLMサーバがエラーを返しました (HTTP %d)。時間をおいて再試行してください。", status), Retryable: true, wrapped: errFromBody(status, body)}
		}
		return &LLMError{Kind: ErrHTTPStatus, StatusCode: status,
			UserMessage: fmt.Sprintf("リクエストが拒否されました (HTTP %d)。設定（モデル名/エンドポイント）を確認してください。", status), Retryable: false, wrapped: errFromBody(status, body)}
	}
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || isTimeout(err) {
			return &LLMError{Kind: ErrTimeout,
				UserMessage: "応答がタイムアウトしました。モデルのロード状況やネットワークを確認してください。", Retryable: true, wrapped: err}
		}
		return &LLMError{Kind: ErrUnreachable,
			UserMessage: "LM Studio に接続できません。起動状態と Endpoint(URL/ポート) を確認してください。", Retryable: true, wrapped: err}
	}
	return nil
}

// classifyCtx maps a context error after cancellation/deadline.
func classifyCtx(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return &LLMError{Kind: ErrTimeout, UserMessage: "応答がタイムアウトしました。", Retryable: false, wrapped: err}
	}
	return err // context.Canceled: surface as-is (user/abort initiated)
}

func isTimeout(err error) bool {
	var ne net.Error
	if errors.As(err, &ne) {
		return ne.Timeout()
	}
	return false
}

// errFromBody keeps a sanitized internal cause without leaking the raw body to
// callers (it lives only in wrapped, used for logs).
func errFromBody(status int, body []byte) error {
	return fmt.Errorf("http %d: %s", status, string(body))
}
