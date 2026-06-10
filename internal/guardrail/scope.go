package guardrail

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// resolveWithin resolves target (relative paths are taken relative to root),
// follows symlinks on the deepest existing ancestor, and reports whether the
// real path lies within root (NFR design P1). A resolution error means "not
// safely within" (fail-closed).
func resolveWithin(root, target string) (resolved string, within bool, err error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", false, err
	}
	rootReal, err := filepath.EvalSymlinks(rootAbs)
	if err != nil {
		return "", false, err
	}

	t := target
	if !filepath.IsAbs(t) {
		t = filepath.Join(rootReal, t)
	}
	t = filepath.Clean(t)

	real, err := evalExisting(t)
	if err != nil {
		return "", false, err
	}
	within = real == rootReal || strings.HasPrefix(real, rootReal+string(os.PathSeparator))
	return real, within, nil
}

// isGitInternalPath reports whether the canonical path resolved points inside a
// .git directory. resolved is already symlink-canonical, so a plain string
// scan for a ".git" path segment is sufficient (F-02). It does not match
// sibling names like ".gitignore" or ".github".
func isGitInternalPath(resolved string) bool {
	sep := string(os.PathSeparator)
	return strings.Contains(resolved, sep+".git"+sep) || strings.HasSuffix(resolved, sep+".git")
}

// reSensitivePath matches well-known credential/secret stores. Matching is on
// the canonical path so obfuscated relatives cannot slip through (F-04).
var reSensitivePath = regexp.MustCompile(`/\.ssh/|/\.aws/|/\.gnupg/|/\.netrc|/etc/shadow|/etc/sudoers|id_rsa|id_ed25519|id_ecdsa|id_dsa`)

// isSensitivePath reports whether a path looks like a credential/secret store.
func isSensitivePath(resolved string) bool {
	return reSensitivePath.MatchString(resolved)
}

// evalExisting resolves symlinks on the deepest existing ancestor of p, then
// re-appends the non-existent trailing components (so not-yet-created files can
// be checked).
func evalExisting(p string) (string, error) {
	cur := p
	var rest []string
	for {
		if resolved, err := filepath.EvalSymlinks(cur); err == nil {
			if len(rest) == 0 {
				return resolved, nil
			}
			parts := []string{resolved}
			for i := len(rest) - 1; i >= 0; i-- {
				parts = append(parts, rest[i])
			}
			return filepath.Join(parts...), nil
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return "", fmt.Errorf("cannot resolve path %q", p)
		}
		rest = append(rest, filepath.Base(cur))
		cur = parent
	}
}
