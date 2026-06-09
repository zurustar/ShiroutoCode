// Package guardrail is the safety core of ShiroutoCode (SECURITY-11). Every
// tool execution flows through the ToolDispatcher, which evaluates the action
// (Allow / Confirm / Deny), requests human confirmation when needed, and fails
// closed when it cannot decide safely.
package guardrail

import "context"

// ActionKind categorizes a tool action for evaluation.
type ActionKind int

const (
	FileRead ActionKind = iota
	FileWrite
	FileDelete
	Command
	GitOp
	WebFetch
	Unknown
)

// Action is the normalized input to the evaluator.
type Action struct {
	Kind        ActionKind
	Tool        string
	Paths       []string
	CommandLine string
	URL         string
}

// DecisionKind is the guardrail verdict.
type DecisionKind int

const (
	Allow DecisionKind = iota
	Confirm
	Deny
)

// Decision is a verdict with a human-readable reason.
type Decision struct {
	Kind   DecisionKind
	Reason string
}

// Policy configures evaluation (sourced from U1 Config).
type Policy struct {
	WorkspaceRoot     string
	ConfirmMode       string // "prompt" | "deny"
	ExtraDenyPatterns []string
}

// Confirmer asks a human to approve a Confirm action. Implemented by the
// frontend (U5). A nil Confirmer means non-interactive: Confirm/Deny do not
// execute (fail-closed, R8).
type Confirmer interface {
	Confirm(ctx context.Context, a Action, reason string) (bool, error)
}

// BlockedError indicates an action was not executed by the guardrail.
type BlockedError struct {
	Reason string
}

func (e *BlockedError) Error() string { return e.Reason }
