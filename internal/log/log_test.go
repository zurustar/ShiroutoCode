package log

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// parseJSONLine decodes the last non-empty JSON log line written to buf.
func parseLastJSON(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) == 0 || lines[0] == "" {
		t.Fatalf("no log output")
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &m); err != nil {
		t.Fatalf("invalid JSON log line %q: %v", lines[len(lines)-1], err)
	}
	return m
}

// R6: secret-keyed values must never appear raw; masked to "***".
func TestMaskingSecretKeys(t *testing.T) {
	secretKeys := []string{"authorization", "Authorization", "token", "API_KEY", "api_key", "secret", "password"}
	for _, k := range secretKeys {
		var buf bytes.Buffer
		lg := New("info", FormatJSON, &buf)
		lg.Info("hello", k, "super-secret-value")
		m := parseLastJSON(t, &buf)
		if got, ok := m[k]; ok {
			if got != "***" {
				t.Errorf("key %q: expected masked ***, got %v", k, got)
			}
		} else {
			t.Errorf("key %q missing in output %v", k, m)
		}
		if strings.Contains(buf.String(), "super-secret-value") {
			t.Errorf("key %q: raw secret leaked: %s", k, buf.String())
		}
	}
}

// R6 (PBT): masking is idempotent at the attribute level.
func TestMaskingIdempotent(t *testing.T) {
	rules := DefaultMaskRules()
	keys := []string{"authorization", "token", "secret", "password", "api_key", "prompt", "messages", "normalkey", "count"}
	rapid.Check(t, func(rt *rapid.T) {
		key := rapid.SampledFrom(keys).Draw(rt, "key")
		val := rapid.String().Draw(rt, "val")
		once := maskValue(key, val, false, rules)
		twice := maskValue(key, once, false, rules)
		if once != twice {
			rt.Fatalf("not idempotent for key=%q: once=%q twice=%q", key, once, twice)
		}
	})
}

// R6: prompt bodies summarized unless debug; full text at debug.
func TestPromptSummarizedUnlessDebug(t *testing.T) {
	long := strings.Repeat("x", 500)

	var infoBuf bytes.Buffer
	New("info", FormatJSON, &infoBuf).Info("call", "prompt", long)
	if strings.Contains(infoBuf.String(), long) {
		t.Errorf("prompt body leaked at info level: %s", infoBuf.String())
	}
	m := parseLastJSON(t, &infoBuf)
	if s, _ := m["prompt"].(string); !strings.Contains(s, "chars") {
		t.Errorf("expected summarized prompt at info, got %q", s)
	}

	var dbgBuf bytes.Buffer
	New("debug", FormatJSON, &dbgBuf).Info("call", "prompt", long)
	if !strings.Contains(dbgBuf.String(), long) {
		t.Errorf("expected full prompt at debug level")
	}
}

// R8 (PBT): records below configured level are suppressed; at/above emitted.
func TestLevelFilter(t *testing.T) {
	levels := []string{"debug", "info", "warn", "error"}
	order := map[string]int{"debug": 0, "info": 1, "warn": 2, "error": 3}
	rapid.Check(t, func(rt *rapid.T) {
		cfgLvl := rapid.SampledFrom(levels).Draw(rt, "cfg")
		msgLvl := rapid.SampledFrom(levels).Draw(rt, "msg")
		var buf bytes.Buffer
		lg := New(cfgLvl, FormatJSON, &buf)
		switch msgLvl {
		case "debug":
			// Logger interface has no Debug; emulate via Info gate is not valid.
			// Skip debug emission through interface; covered by info/warn/error.
			return
		case "info":
			lg.Info("m")
		case "warn":
			lg.Warn("m")
		case "error":
			lg.Error("m")
		}
		emitted := strings.TrimSpace(buf.String()) != ""
		want := order[msgLvl] >= order[cfgLvl]
		if emitted != want {
			rt.Fatalf("cfg=%s msg=%s emitted=%v want=%v", cfgLvl, msgLvl, emitted, want)
		}
	})
}

// R7: every record carries time and level; With(correlationID) propagates.
func TestTimestampLevelAndCorrelation(t *testing.T) {
	var buf bytes.Buffer
	lg := New("info", FormatJSON, &buf).With("correlation_id", "abc123")
	lg.Info("first")
	lg.Warn("second")
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %q", len(lines), buf.String())
	}
	for _, ln := range lines {
		var m map[string]any
		if err := json.Unmarshal([]byte(ln), &m); err != nil {
			t.Fatalf("bad json: %v", err)
		}
		if _, ok := m["time"]; !ok {
			t.Errorf("missing time in %q", ln)
		}
		if _, ok := m["level"]; !ok {
			t.Errorf("missing level in %q", ln)
		}
		if m["correlation_id"] != "abc123" {
			t.Errorf("correlation_id not propagated in %q", ln)
		}
	}
}

// Sanity: text format also works and masks.
func TestTextFormatMasks(t *testing.T) {
	var buf bytes.Buffer
	New("info", FormatText, &buf).Info("m", "token", "leaky")
	if strings.Contains(buf.String(), "leaky") {
		t.Errorf("text format leaked secret: %s", buf.String())
	}
}
