// Package activity defines the base contracts every Temporal activity in
// this project — whether it's a "real" deterministic integration (GitHub,
// CI, deployment) or a thin shim over an out-of-process agent — is expected
// to satisfy.
//
// Two runtime shapes exist side by side:
//
//  1. Native Go activities: business logic lives directly in the Go
//     function registered with the Temporal worker (e.g. services/
//     github-integration's CreateIssue).
//  2. Agent shims: the Go activity function does almost nothing itself. It
//     marshals ActivityInput, calls out to the corresponding agents/<name>
//     process over gRPC/HTTP, and unmarshals the reply into ActivityOutput.
//     All actual judgment (what to implement, whether a diff is reviewable,
//     how to fix a failing test) happens on the other side of that call,
//     in whatever language/SDK fits the agent best. See agents/README.md
//     for the runtime-boundary rationale.
//
// Neither shape is special-cased by the workflow engine: both register as
// ordinary Temporal activities and both return a Result, which is what lets
// Step 4's Transition Router stay agent-agnostic — it only ever consumes a
// Verdict, never a free-text agent output.
package activity

import "time"

// TaskID identifies the task (GitHub issue-backed unit of work) an activity
// invocation is acting on. It is threaded through every activity so logs,
// idempotency keys, and the transition log (see the statemachine package,
// Step 2) can all be correlated back to a single task.
type TaskID string

// Verdict is the closed set of outcomes an activity may report back to the
// Transition Router. It deliberately excludes free text: whatever an agent
// or integration "thinks" stays on its side of the RPC boundary, and only
// a Verdict crosses back into the deterministic workflow engine.
//
// Not every activity uses every value — e.g. GitHub integration activities
// only ever emit VerdictSucceeded/VerdictFailed, while the (future) Review
// Agent shim also emits VerdictChangesRequested/VerdictEscalate. The full
// per-state legal set is defined in Step 4's Transition Router, not here.
type Verdict string

const (
	VerdictSucceeded        Verdict = "SUCCEEDED"
	VerdictFailed           Verdict = "FAILED"
	VerdictApproved         Verdict = "APPROVED"
	VerdictChangesRequested Verdict = "CHANGES_REQUESTED"
	VerdictFixable          Verdict = "FIXABLE"
	VerdictUnfixable        Verdict = "UNFIXABLE"
	VerdictEscalate         Verdict = "ESCALATE"
)

// Input is the envelope every activity receives. Payload carries the
// activity-specific request, kept as a typed field on the concrete input
// struct in each service package — Input itself only standardizes the
// fields that are always present.
type Input struct {
	TaskID TaskID `json:"taskId"`
	// IdempotencyKey lets check-then-create activities (Step 3's
	// CreateIssue/CreatePullRequest) recognize a Temporal retry of an
	// activity that already succeeded against the external system.
	IdempotencyKey string `json:"idempotencyKey"`
}

// Result is the envelope every activity returns. Reason is a short,
// human-readable explanation (logged to the transition log) — it is not
// meant to carry agent reasoning back into the router; that's what Verdict
// is for.
type Result struct {
	Verdict Verdict `json:"verdict"`
	Reason  string  `json:"reason,omitempty"`
}

// AgentRPCConfig configures the Go-side shim's call to an agents/<name>
// process. Each shim activity in services/<name> embeds one of these to
// know where to dial and how long to wait before letting Temporal's own
// activity-level retry policy take over.
type AgentRPCConfig struct {
	// Endpoint is the base URL (HTTP) or target (gRPC) of the agent
	// process, e.g. "http://localhost:9101".
	Endpoint string
	Timeout  time.Duration
}
