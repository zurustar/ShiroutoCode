// Package config loads and validates ShiroutoCode's effective configuration
// from multiple sources, merged by precedence: flag > env > project file >
// home file > built-in defaults (business rules R1-R5).
//
// Loading is fail-closed: any validation failure aborts startup with an
// aggregated, sanitized error (P2/P4, SECURITY-09/15). All external inputs
// (file readers, directory checks) are injectable so the loader is hermetically
// testable.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// GuardrailPolicy configures safety behavior (structure only; evaluation lives
// in U3 Guardrail).
type GuardrailPolicy struct {
	ConfirmMode          string   // "prompt" | "deny"
	ExtraDenyPatterns    []string // user-supplied denylist additions
	NonInteractivePolicy string   // "stop" | "deny" (non-TTY behavior)
}

// Config is the validated, immutable effective configuration.
type Config struct {
	Endpoint  string
	Model     string
	MaxSteps  int
	Workspace string
	Guardrail GuardrailPolicy
	LogLevel  string
	LogFormat string
	LogFile   string
	ToolMode  string // auto | function | json (hybrid tool-calling, FR-2/Q2)
}

// Options carries the loader inputs. The zero value is filled with OS-backed
// defaults by Load; tests inject hermetic implementations.
type Options struct {
	Args       []string
	Env        map[string]string
	WorkingDir string
	HomeDir    string
	ReadFile   func(string) ([]byte, error)
	DirExists  func(string) bool
}

// partial mirrors Config with pointer fields so an unset value (nil) is
// distinguishable from an explicit zero value (P3).
type partial struct {
	Endpoint    *string `yaml:"endpoint"`
	Model       *string `yaml:"model"`
	MaxSteps    *int    `yaml:"maxSteps"`
	Workspace   *string `yaml:"workspace"`
	LogLevel    *string `yaml:"logLevel"`
	LogFormat   *string `yaml:"logFormat"`
	LogFile     *string `yaml:"logFile"`
	ConfirmMode *string `yaml:"confirmMode"`
	ToolMode    *string `yaml:"toolMode"`
	// ExtraDenyPatterns is list-valued, so a nil slice (key absent) means
	// "unset" and a present overlay replaces the prior value (R1 semantics).
	ExtraDenyPatterns []string `yaml:"extraDenyPatterns"`
}

func defaults(workingDir string) Config {
	return Config{
		Endpoint:  "http://localhost:1234/v1",
		Model:     "",
		MaxSteps:  25,
		Workspace: workingDir,
		LogLevel:  "info",
		LogFormat: "text",
		LogFile:   "",
		ToolMode:  "auto",
		Guardrail: GuardrailPolicy{ConfirmMode: "prompt", NonInteractivePolicy: "stop"},
	}
}

// Load merges all sources and validates the result.
func Load(o Options) (Config, error) {
	o = withOSDefaults(o)

	cfg := defaults(o.WorkingDir)

	// Low -> high precedence (R1). Each overlay only sets present fields.
	homePart, err := readYAMLFile(o, filepath.Join(o.HomeDir, ".config", "shiroutocode", "config.yaml"))
	if err != nil {
		return Config{}, err
	}
	projPart, err := readYAMLFile(o, filepath.Join(o.WorkingDir, ".shiroutocode.yaml"))
	if err != nil {
		return Config{}, err
	}
	envPart, err := readEnv(o.Env)
	if err != nil {
		return Config{}, err
	}
	flagPart, err := readFlags(o.Args)
	if err != nil {
		return Config{}, err
	}

	for _, p := range []partial{homePart, projPart, envPart, flagPart} {
		overlay(&cfg, p)
	}

	// Resolve workspace to an absolute path before validation.
	if abs, aerr := filepath.Abs(cfg.Workspace); aerr == nil {
		cfg.Workspace = abs
	}

	if verr := validate(cfg, o.DirExists); verr != nil {
		return Config{}, verr
	}
	return cfg, nil
}

func withOSDefaults(o Options) Options {
	if o.ReadFile == nil {
		o.ReadFile = os.ReadFile
	}
	if o.DirExists == nil {
		o.DirExists = func(p string) bool {
			fi, err := os.Stat(p)
			return err == nil && fi.IsDir()
		}
	}
	if o.WorkingDir == "" {
		if wd, err := os.Getwd(); err == nil {
			o.WorkingDir = wd
		}
	}
	if o.HomeDir == "" {
		if hd, err := os.UserHomeDir(); err == nil {
			o.HomeDir = hd
		}
	}
	if o.Env == nil {
		o.Env = map[string]string{}
	}
	return o
}

func readYAMLFile(o Options, path string) (partial, error) {
	data, err := o.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return partial{}, nil // absent file is normal (R3)
		}
		return partial{}, fmt.Errorf("could not read config file: %w", err)
	}
	var p partial
	if err := yaml.Unmarshal(data, &p); err != nil {
		return partial{}, fmt.Errorf("invalid YAML in config file: %w", err)
	}
	return p, nil
}

func readEnv(env map[string]string) (partial, error) {
	var p partial
	if v, ok := env["SHIROUTO_ENDPOINT"]; ok {
		p.Endpoint = strptr(v)
	}
	if v, ok := env["SHIROUTO_MODEL"]; ok {
		p.Model = strptr(v)
	}
	if v, ok := env["SHIROUTO_WORKSPACE"]; ok {
		p.Workspace = strptr(v)
	}
	if v, ok := env["SHIROUTO_LOG_LEVEL"]; ok {
		p.LogLevel = strptr(v)
	}
	if v, ok := env["SHIROUTO_LOG_FORMAT"]; ok {
		p.LogFormat = strptr(v)
	}
	if v, ok := env["SHIROUTO_LOG_FILE"]; ok {
		p.LogFile = strptr(v)
	}
	if v, ok := env["SHIROUTO_TOOL_MODE"]; ok {
		p.ToolMode = strptr(v)
	}
	if v, ok := env["SHIROUTO_MAX_STEPS"]; ok {
		n, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return partial{}, fmt.Errorf("invalid SHIROUTO_MAX_STEPS: must be an integer")
		}
		p.MaxSteps = &n
	}
	return p, nil
}

func readFlags(args []string) (partial, error) {
	// Note: a real flag parse is wired in U5; here we accept the flags U1 owns
	// for configuration. Use a quiet, isolated parse over a known key set.
	var p partial
	set := map[string]*string{}
	get := func(name string) *string {
		v := new(string)
		set[name] = v
		return v
	}
	endpoint := get("-endpoint")
	model := get("-model")
	workspace := get("-workspace")
	logLevel := get("-log-level")
	logFormat := get("-log-format")
	logFile := get("-log-file")
	toolMode := get("-tool-mode")
	maxSteps := get("-max-steps")

	i := 0
	for i < len(args) {
		a := args[i]
		dst, ok := set[a]
		if !ok {
			// also support --flag form
			dst, ok = set[strings.TrimPrefix(a, "-")]
		}
		if !ok && strings.HasPrefix(a, "--") {
			dst, ok = set["-"+strings.TrimPrefix(a, "--")]
		}
		if !ok {
			i++
			continue
		}
		if i+1 >= len(args) {
			return partial{}, fmt.Errorf("flag %s requires a value", a)
		}
		*dst = args[i+1]
		i += 2
	}

	if *endpoint != "" {
		p.Endpoint = strptr(*endpoint)
	}
	if *model != "" {
		p.Model = strptr(*model)
	}
	if *workspace != "" {
		p.Workspace = strptr(*workspace)
	}
	if *logLevel != "" {
		p.LogLevel = strptr(*logLevel)
	}
	if *logFormat != "" {
		p.LogFormat = strptr(*logFormat)
	}
	if *logFile != "" {
		p.LogFile = strptr(*logFile)
	}
	if *toolMode != "" {
		p.ToolMode = strptr(*toolMode)
	}
	if *maxSteps != "" {
		n, err := strconv.Atoi(strings.TrimSpace(*maxSteps))
		if err != nil {
			return partial{}, fmt.Errorf("invalid -max-steps: must be an integer")
		}
		p.MaxSteps = &n
	}
	return p, nil
}

func overlay(cfg *Config, p partial) {
	if p.Endpoint != nil {
		cfg.Endpoint = *p.Endpoint
	}
	if p.Model != nil {
		cfg.Model = *p.Model
	}
	if p.MaxSteps != nil {
		cfg.MaxSteps = *p.MaxSteps
	}
	if p.Workspace != nil {
		cfg.Workspace = *p.Workspace
	}
	if p.LogLevel != nil {
		cfg.LogLevel = *p.LogLevel
	}
	if p.LogFormat != nil {
		cfg.LogFormat = *p.LogFormat
	}
	if p.LogFile != nil {
		cfg.LogFile = *p.LogFile
	}
	if p.ConfirmMode != nil {
		cfg.Guardrail.ConfirmMode = *p.ConfirmMode
	}
	if p.ToolMode != nil {
		cfg.ToolMode = *p.ToolMode
	}
	if p.ExtraDenyPatterns != nil {
		cfg.Guardrail.ExtraDenyPatterns = p.ExtraDenyPatterns
	}
}

func validate(cfg Config, dirExists func(string) bool) error {
	var errs []error
	if strings.TrimSpace(cfg.Model) == "" {
		errs = append(errs, errors.New("model is not set: provide --model, SHIROUTO_MODEL, or 'model' in a config file"))
	}
	if err := validateEndpoint(cfg.Endpoint); err != nil {
		errs = append(errs, err)
	}
	if cfg.MaxSteps <= 0 {
		errs = append(errs, fmt.Errorf("maxSteps must be greater than 0 (got %d)", cfg.MaxSteps))
	}
	if dirExists == nil || !dirExists(cfg.Workspace) {
		errs = append(errs, errors.New("workspace directory does not exist or is not a directory"))
	}
	switch cfg.ToolMode {
	case "auto", "function", "json":
	default:
		errs = append(errs, fmt.Errorf("toolMode must be auto|function|json (got %q)", cfg.ToolMode))
	}
	return errors.Join(errs...)
}

func validateEndpoint(raw string) error {
	if strings.TrimSpace(raw) == "" {
		return errors.New("endpoint must be set to a valid URL")
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return errors.New("endpoint must be a valid http(s) URL with a host")
	}
	return nil
}

func strptr(s string) *string { return &s }
