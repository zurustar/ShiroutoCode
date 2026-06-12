package config

import (
	"io/fs"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// helper: Options with injected, hermetic file system and a valid workspace.
func baseOpts() Options {
	return Options{
		WorkingDir: "/ws",
		HomeDir:    "/home/u",
		Env:        map[string]string{},
		Args:       nil,
		ReadFile:   func(string) ([]byte, error) { return nil, fs.ErrNotExist },
		DirExists:  func(string) bool { return true },
	}
}

func TestDefaults(t *testing.T) {
	o := baseOpts()
	o.Env = map[string]string{"SHIROUTO_MODEL": "m1"} // model is required
	cfg, err := Load(o)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Endpoint != "http://localhost:1234/v1" {
		t.Errorf("endpoint default = %q", cfg.Endpoint)
	}
	if cfg.MaxSteps != 25 {
		t.Errorf("maxSteps default = %d", cfg.MaxSteps)
	}
	if cfg.LogLevel != "info" || cfg.LogFormat != "text" {
		t.Errorf("log defaults = %q/%q", cfg.LogLevel, cfg.LogFormat)
	}
	if cfg.Guardrail.ConfirmMode != "prompt" || cfg.Guardrail.NonInteractivePolicy != "stop" {
		t.Errorf("guardrail defaults = %+v", cfg.Guardrail)
	}
}

func TestToolMode(t *testing.T) {
	o := baseOpts()
	o.Env = map[string]string{"SHIROUTO_MODEL": "m"}
	cfg, err := Load(o)
	if err != nil || cfg.ToolMode != "auto" {
		t.Fatalf("default toolMode = %q err=%v", cfg.ToolMode, err)
	}
	// override via env
	o.Env["SHIROUTO_TOOL_MODE"] = "json"
	cfg, err = Load(o)
	if err != nil || cfg.ToolMode != "json" {
		t.Errorf("toolMode = %q err=%v", cfg.ToolMode, err)
	}
	// invalid rejected
	o.Env["SHIROUTO_TOOL_MODE"] = "bogus"
	if _, err := Load(o); err == nil {
		t.Error("invalid toolMode should fail validation")
	}
}

func TestEnvMapping(t *testing.T) {
	o := baseOpts()
	o.Env = map[string]string{
		"SHIROUTO_MODEL":     "envmodel",
		"SHIROUTO_MAX_STEPS": "7",
		"SHIROUTO_ENDPOINT":  "http://localhost:9999/v1",
		"SHIROUTO_LOG_LEVEL": "debug",
	}
	cfg, err := Load(o)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Model != "envmodel" || cfg.MaxSteps != 7 || cfg.Endpoint != "http://localhost:9999/v1" || cfg.LogLevel != "debug" {
		t.Errorf("env not applied: %+v", cfg)
	}
}

func TestYAMLProjectOverridesHome(t *testing.T) {
	o := baseOpts()
	o.ReadFile = func(path string) ([]byte, error) {
		switch {
		case strings.HasSuffix(path, ".shiroutocode.yaml"): // project
			return []byte("model: projmodel\nmaxSteps: 3\n"), nil
		case strings.HasSuffix(path, "config.yaml"): // home
			return []byte("model: homemodel\nmaxSteps: 99\nendpoint: http://home:1/v1\n"), nil
		}
		return nil, fs.ErrNotExist
	}
	cfg, err := Load(o)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Model != "projmodel" || cfg.MaxSteps != 3 {
		t.Errorf("project should override home: %+v", cfg)
	}
	if cfg.Endpoint != "http://home:1/v1" {
		t.Errorf("home-only value should survive: %+v", cfg)
	}
}

func TestInvalidYAMLIsError(t *testing.T) {
	o := baseOpts()
	o.ReadFile = func(path string) ([]byte, error) {
		if strings.HasSuffix(path, ".shiroutocode.yaml") {
			return []byte("model: [unterminated"), nil
		}
		return nil, fs.ErrNotExist
	}
	if _, err := Load(o); err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

// R4 / P2: multiple validation failures reported together.
func TestValidationAggregatesErrors(t *testing.T) {
	o := baseOpts()
	o.Env = map[string]string{
		"SHIROUTO_ENDPOINT":  "not a url",
		"SHIROUTO_MAX_STEPS": "0",
	}
	o.DirExists = func(string) bool { return false } // workspace invalid
	_, err := Load(o)
	if err == nil {
		t.Fatal("expected aggregated error")
	}
	msg := err.Error()
	// model is intentionally NOT required here — it is resolved at the CLI layer.
	for _, want := range []string{"endpoint", "maxSteps", "workspace"} {
		if !strings.Contains(strings.ToLower(msg), strings.ToLower(want)) {
			t.Errorf("aggregated error missing %q: %s", want, msg)
		}
	}
}

// R4: zero vs unset — explicitly setting maxSteps:0 must fail (not silently
// fall back to the default 25).
func TestZeroValueDistinctFromUnset(t *testing.T) {
	o := baseOpts()
	o.ReadFile = func(path string) ([]byte, error) {
		if strings.HasSuffix(path, ".shiroutocode.yaml") {
			return []byte("model: m\nmaxSteps: 0\n"), nil
		}
		return nil, fs.ErrNotExist
	}
	if _, err := Load(o); err == nil {
		t.Fatal("maxSteps:0 must be rejected")
	}
}

// extraDenyPatterns must load from a config file into the guardrail policy.
// Project file overrides home file (consistent with other keys).
func TestExtraDenyPatternsFromYAML(t *testing.T) {
	o := baseOpts()
	o.Env = map[string]string{"SHIROUTO_MODEL": "m"}
	o.ReadFile = func(path string) ([]byte, error) {
		switch {
		case strings.HasSuffix(path, ".shiroutocode.yaml"): // project
			return []byte("extraDenyPatterns:\n  - \"rm -rf /\"\n  - \":(){:|:&};:\"\n"), nil
		case strings.HasSuffix(path, "config.yaml"): // home
			return []byte("extraDenyPatterns:\n  - \"home-only\"\n"), nil
		}
		return nil, fs.ErrNotExist
	}
	cfg, err := Load(o)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := cfg.Guardrail.ExtraDenyPatterns
	want := []string{"rm -rf /", ":(){:|:&};:"}
	if len(got) != len(want) {
		t.Fatalf("extraDenyPatterns = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("extraDenyPatterns[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

// Absent extraDenyPatterns leaves the guardrail policy with none.
func TestExtraDenyPatternsDefaultEmpty(t *testing.T) {
	o := baseOpts()
	o.Env = map[string]string{"SHIROUTO_MODEL": "m"}
	cfg, err := Load(o)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Guardrail.ExtraDenyPatterns) != 0 {
		t.Errorf("expected no deny patterns by default, got %#v", cfg.Guardrail.ExtraDenyPatterns)
	}
}

// R5: defaults carry no secrets.
func TestDefaultsHaveNoSecrets(t *testing.T) {
	d := defaults("/ws")
	blob := strings.ToLower(d.Endpoint + " " + d.Model + " " + d.LogFile)
	for _, bad := range []string{"token", "password", "secret", "apikey", "api_key"} {
		if strings.Contains(blob, bad) {
			t.Errorf("default contains secret-like content: %q", blob)
		}
	}
}

// R1 (PBT): effective value = highest-priority present source
// (flag > env > project > home > default).
func TestPrecedencePBT(t *testing.T) {
	const def = "http://localhost:1234/v1"
	rapid.Check(t, func(rt *rapid.T) {
		// presence + distinct value per source
		homeOn := rapid.Bool().Draw(rt, "home")
		projOn := rapid.Bool().Draw(rt, "proj")
		envOn := rapid.Bool().Draw(rt, "env")
		flagOn := rapid.Bool().Draw(rt, "flag")

		o := baseOpts()
		o.Env = map[string]string{"SHIROUTO_MODEL": "m"} // keep Load valid
		o.ReadFile = func(path string) ([]byte, error) {
			if strings.HasSuffix(path, ".shiroutocode.yaml") && projOn {
				return []byte("endpoint: http://proj/v1\n"), nil
			}
			if strings.HasSuffix(path, "config.yaml") && homeOn {
				return []byte("endpoint: http://home/v1\n"), nil
			}
			return nil, fs.ErrNotExist
		}
		if envOn {
			o.Env["SHIROUTO_ENDPOINT"] = "http://env/v1"
		}
		if flagOn {
			o.Args = []string{"-endpoint", "http://flag/v1"}
		}

		want := def
		if homeOn {
			want = "http://home/v1"
		}
		if projOn {
			want = "http://proj/v1"
		}
		if envOn {
			want = "http://env/v1"
		}
		if flagOn {
			want = "http://flag/v1"
		}

		cfg, err := Load(o)
		if err != nil {
			rt.Fatalf("load error: %v", err)
		}
		if cfg.Endpoint != want {
			rt.Fatalf("home=%v proj=%v env=%v flag=%v: got %q want %q",
				homeOn, projOn, envOn, flagOn, cfg.Endpoint, want)
		}
	})
}

// R4 (PBT): URL validation accepts http/https with a host, rejects others.
func TestEndpointURLValidationPBT(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		scheme := rapid.SampledFrom([]string{"http", "https"}).Draw(rt, "scheme")
		host := rapid.StringMatching(`[a-z]{1,8}(:[0-9]{2,5})?`).Draw(rt, "host")
		good := scheme + "://" + host + "/v1"
		if err := validateEndpoint(good); err != nil {
			rt.Fatalf("valid URL rejected %q: %v", good, err)
		}
	})
	bad := []string{"", "not a url", "ftp://x/y", "://nohost", "http://", "justtext"}
	for _, b := range bad {
		if err := validateEndpoint(b); err == nil {
			t.Errorf("invalid URL accepted: %q", b)
		}
	}
}
