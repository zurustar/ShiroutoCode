// Package log provides structured logging for ShiroutoCode with built-in
// masking of sensitive information (SECURITY-03 / NFR-5).
//
// It wraps the standard library log/slog. Masking is implemented as a
// slog.Handler decorator (NFR design P1) so callers never have to remember to
// mask: any attribute whose key looks like a secret is redacted, and prompt
// bodies are summarized unless the debug level is enabled.
package log

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"strings"
)

// Format selects the log encoding.
type Format string

const (
	FormatText Format = "text"
	FormatJSON Format = "json"
)

// Logger is the minimal logging surface used across ShiroutoCode.
type Logger interface {
	Debug(msg string, kv ...any)
	Info(msg string, kv ...any)
	Warn(msg string, kv ...any)
	Error(msg string, kv ...any)
	// With returns a child logger that includes the given key/value pairs on
	// every subsequent record (e.g. a correlation ID).
	With(kv ...any) Logger
}

type slogLogger struct {
	l *slog.Logger
}

// New builds a Logger writing to w in the given format, filtered at level.
// level is one of debug|info|warn|error (defaults to info when unknown).
func New(level string, format Format, w io.Writer) Logger {
	lvl := parseLevel(level)
	opts := &slog.HandlerOptions{Level: lvl}

	var base slog.Handler
	switch format {
	case FormatJSON:
		base = slog.NewJSONHandler(w, opts)
	default:
		base = slog.NewTextHandler(w, opts)
	}

	h := &maskingHandler{
		inner: base,
		rules: DefaultMaskRules(),
		debug: lvl <= slog.LevelDebug,
	}
	return &slogLogger{l: slog.New(h)}
}

func (s *slogLogger) Debug(msg string, kv ...any) { s.l.Debug(msg, kv...) }
func (s *slogLogger) Info(msg string, kv ...any)  { s.l.Info(msg, kv...) }
func (s *slogLogger) Warn(msg string, kv ...any)  { s.l.Warn(msg, kv...) }
func (s *slogLogger) Error(msg string, kv ...any) { s.l.Error(msg, kv...) }
func (s *slogLogger) With(kv ...any) Logger       { return &slogLogger{l: s.l.With(kv...)} }

func parseLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// MaskRuleSet defines which attribute keys are treated as secrets (fully
// redacted) and which are prompt bodies (summarized unless debug).
type MaskRuleSet struct {
	secretSubstrings []string
	promptKeys       []string
}

// DefaultMaskRules returns the built-in masking policy (R6). Matching is by
// case-insensitive substring on the key, which fails safe (over-masking is
// preferred over leaking).
func DefaultMaskRules() MaskRuleSet {
	return MaskRuleSet{
		secretSubstrings: []string{"authorization", "token", "api_key", "apikey", "secret", "password"},
		promptKeys:       []string{"prompt", "messages", "completion"},
	}
}

func (r MaskRuleSet) isSecret(lowerKey string) bool {
	for _, s := range r.secretSubstrings {
		if strings.Contains(lowerKey, s) {
			return true
		}
	}
	return false
}

func (r MaskRuleSet) isPrompt(lowerKey string) bool {
	for _, k := range r.promptKeys {
		if strings.Contains(lowerKey, k) {
			return true
		}
	}
	return false
}

// promptMarker matches an already-summarized prompt value so masking stays
// idempotent (R6).
var promptMarker = regexp.MustCompile(`^<[^:>]+:\d+ chars>$`)

const secretMask = "***"

// maskValue applies the masking policy to a single string value given its key.
// It is pure and idempotent, which makes it directly property-testable.
func maskValue(key, value string, debug bool, rules MaskRuleSet) string {
	lk := strings.ToLower(key)
	if rules.isSecret(lk) {
		return secretMask
	}
	if rules.isPrompt(lk) && !debug {
		if promptMarker.MatchString(value) {
			return value // already summarized
		}
		return fmt.Sprintf("<%s:%d chars>", key, len(value))
	}
	return value
}

// maskAttr applies masking to a slog.Attr, recursing into groups.
func maskAttr(a slog.Attr, debug bool, rules MaskRuleSet) slog.Attr {
	if a.Value.Kind() == slog.KindGroup {
		grp := a.Value.Group()
		out := make([]slog.Attr, len(grp))
		for i, g := range grp {
			out[i] = maskAttr(g, debug, rules)
		}
		return slog.Attr{Key: a.Key, Value: slog.GroupValue(out...)}
	}
	lk := strings.ToLower(a.Key)
	if rules.isSecret(lk) {
		return slog.String(a.Key, secretMask)
	}
	if rules.isPrompt(lk) && !debug {
		return slog.String(a.Key, maskValue(a.Key, a.Value.String(), debug, rules))
	}
	return a
}

// maskingHandler decorates an slog.Handler, masking attributes before they are
// emitted (NFR design P1).
type maskingHandler struct {
	inner slog.Handler
	rules MaskRuleSet
	debug bool
}

func (h *maskingHandler) Enabled(ctx context.Context, lvl slog.Level) bool {
	return h.inner.Enabled(ctx, lvl)
}

func (h *maskingHandler) Handle(ctx context.Context, r slog.Record) error {
	nr := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)
	r.Attrs(func(a slog.Attr) bool {
		nr.AddAttrs(maskAttr(a, h.debug, h.rules))
		return true
	})
	return h.inner.Handle(ctx, nr)
}

func (h *maskingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	masked := make([]slog.Attr, len(attrs))
	for i, a := range attrs {
		masked[i] = maskAttr(a, h.debug, h.rules)
	}
	return &maskingHandler{inner: h.inner.WithAttrs(masked), rules: h.rules, debug: h.debug}
}

func (h *maskingHandler) WithGroup(name string) slog.Handler {
	return &maskingHandler{inner: h.inner.WithGroup(name), rules: h.rules, debug: h.debug}
}
