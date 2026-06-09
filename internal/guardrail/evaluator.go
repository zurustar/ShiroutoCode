package guardrail

import (
	"regexp"
	"strings"
)

// Evaluator decides Allow/Confirm/Deny for an action (R3-R10).
type Evaluator struct {
	policy    Policy
	rules     []Rule
	extraDeny []*regexp.Regexp
}

// NewEvaluator builds an evaluator from policy, merging any extra deny patterns.
func NewEvaluator(policy Policy) *Evaluator {
	e := &Evaluator{policy: policy, rules: defaultRules()}
	for _, p := range policy.ExtraDenyPatterns {
		if re, err := regexp.Compile(p); err == nil {
			e.extraDeny = append(e.extraDeny, re)
		}
	}
	return e
}

// Evaluate returns the verdict for an action.
func (e *Evaluator) Evaluate(a Action) Decision {
	// 1. Workspace scope for file operations (R3).
	switch a.Kind {
	case FileWrite, FileDelete:
		for _, p := range a.Paths {
			_, within, err := resolveWithin(e.policy.WorkspaceRoot, p)
			if err != nil || !within {
				return Decision{Deny, "ワークスペース外への書き込み/削除はできません"}
			}
		}
	case FileRead:
		for _, p := range a.Paths {
			_, within, err := resolveWithin(e.policy.WorkspaceRoot, p)
			if err != nil {
				return Decision{Confirm, "パスを解決できません（確認が必要です）"}
			}
			if !within {
				return Decision{Confirm, "ワークスペース外の読み取りには確認が必要です"}
			}
		}
	}

	// 2. Extra deny patterns from config (command line).
	if a.Kind == Command || a.Kind == GitOp {
		nc := normalize(a.CommandLine)
		for _, re := range e.extraDeny {
			if re.MatchString(nc) {
				return Decision{Deny, "設定された禁止パターンに一致します"}
			}
		}
	}

	// 3. Built-in rule table (first match wins; Deny rules precede Confirm).
	for _, r := range e.rules {
		if r.Kind == a.Kind && r.Match(a) {
			return Decision{r.Decision, r.Reason}
		}
	}

	// 4. Fail-closed defaults (R9).
	switch a.Kind {
	case Unknown:
		return Decision{Confirm, "未知の操作種別のため確認が必要です"}
	case Command:
		if strings.TrimSpace(a.CommandLine) == "" {
			return Decision{Confirm, "コマンドを解釈できません"}
		}
	}

	// 5. Otherwise allow (normal in-workspace operation, R10).
	return Decision{Allow, ""}
}
