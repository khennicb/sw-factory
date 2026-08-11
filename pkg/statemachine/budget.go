package statemachine

import "time"

// Policy decides what happens when a Budget is exceeded, mirroring the
// pseudocode in instructions/implem_1.txt Step 4 (`budget.Policy ==
// PolicyStrict`). It travels with the Budget rather than living in the
// (not-yet-built) Transition Router because it's a per-task configuration
// choice, not routing logic.
type Policy string

const (
	// PolicyStrict means an exceeded budget routes the task to FAILED.
	PolicyStrict Policy = "STRICT"
	// PolicyLenient means an exceeded budget routes the task to
	// HUMAN_REVIEW instead, giving a person a chance to raise the limit
	// or take over.
	PolicyLenient Policy = "LENIENT"
)

// Budget tracks the limits from spec §11 ("Every task has budgets") and
// the consumption counted against them so far. A zero value for any Max*
// field means "unlimited" for that dimension — callers opt into enforcing
// a given limit by setting it above zero, rather than every unset field
// silently capping at zero.
//
// This package only tracks and reports budget state; per the modeling
// decision in statemachine.go, it does NOT reach into the transition graph
// to force HUMAN_REVIEW/FAILED itself. That enforcement — deciding to
// override whatever the current step's Verdict said — is Step 4's job, and
// only for the states where such an edge actually exists in the graph
// (REVIEWING→HUMAN_REVIEW; IMPLEMENTING/TESTING/DEPLOYING→FAILED).
type Budget struct {
	Policy Policy

	MaxTokens            int
	MaxLocalIterations   int
	MaxRemoteEscalations int
	MaxReviewCycles      int
	MaxTestRetries       int
	MaxDuration          time.Duration

	TokensUsed        int
	LocalIterations   int
	RemoteEscalations int
	ReviewCycles      int
	TestRetries       int
	StartedAt         time.Time
}

// Exceeded reports whether any tracked limit has been breached as of now.
// now is taken as a parameter (rather than read internally via
// time.Now()) so callers — and this package's own tests — get
// deterministic, reproducible results.
func (b Budget) Exceeded(now time.Time) bool {
	return len(b.exceededReasons(now)) > 0
}

// ExceededReasons returns a human-readable reason string per breached
// limit (empty if none are breached), suitable for the transition log's
// Reason field (spec §14: "Reasons for every transition"). Order is fixed
// so callers get stable, testable output.
func (b Budget) ExceededReasons(now time.Time) []string {
	return b.exceededReasons(now)
}

func (b Budget) exceededReasons(now time.Time) []string {
	var reasons []string
	if b.MaxTokens > 0 && b.TokensUsed >= b.MaxTokens {
		reasons = append(reasons, "token budget exceeded")
	}
	if b.MaxLocalIterations > 0 && b.LocalIterations >= b.MaxLocalIterations {
		reasons = append(reasons, "local iteration budget exceeded")
	}
	if b.MaxRemoteEscalations > 0 && b.RemoteEscalations >= b.MaxRemoteEscalations {
		reasons = append(reasons, "remote escalation budget exceeded")
	}
	if b.MaxReviewCycles > 0 && b.ReviewCycles >= b.MaxReviewCycles {
		reasons = append(reasons, "review cycle budget exceeded")
	}
	if b.MaxTestRetries > 0 && b.TestRetries >= b.MaxTestRetries {
		reasons = append(reasons, "test retry budget exceeded")
	}
	if b.MaxDuration > 0 && !b.StartedAt.IsZero() && now.Sub(b.StartedAt) >= b.MaxDuration {
		reasons = append(reasons, "workflow duration budget exceeded")
	}
	return reasons
}
