// Package statemachine implements the Task state machine from spec §8
// (instructions/specs_1.txt) as pure, deterministic Go: the closed set of
// states a task can be in, the legal edges between them, and a validator
// that rejects anything outside that graph.
//
// This package has zero Temporal and zero AI dependency by design (Step 2
// of instructions/implem_1.txt) — it exists so the graph itself can be
// unit-tested exhaustively before anything is wired to a live workflow.
// Step 4's Transition Router is a pure function layered on top: given a
// Verdict (pkg/activity) and a current State, it picks one target from
// TransitionsFrom(state) and calls IsValidTransition before applying it.
// The router never needs to consult the graph structure directly.
//
// # Modeling decisions (spec gaps filled in, not spec-literal)
//
// §8's per-state transition lists and §9's dynamic-transition examples
// disagree in three places. Resolved as follows (confirmed with the repo
// owner rather than assumed — see docs/step-2-state-machine.md):
//
//  1. DOCUMENTING is unreachable under a literal reading of §8: no other
//     state lists it as a target. §9's examples show both
//     REVIEWING → DOCUMENTING and TESTING → DOCUMENTING as valid dynamic
//     paths, so both edges are added here on top of §8's explicit lists.
//  2. §11's budget enforcement ("exceeding a budget transitions to
//     HUMAN_REVIEW or FAILED") is deliberately NOT modeled as a universal
//     escape edge from every state. The graph stays spec-literal: only
//     REVIEWING can reach HUMAN_REVIEW, and only IMPLEMENTING, TESTING, and
//     DEPLOYING can reach FAILED (directly; ROLLBACK also reaches FAILED,
//     see below). Step 4's router must budget-check within these edges —
//     it cannot force HUMAN_REVIEW/FAILED from a state that has no such
//     edge here. If a later step needs budget enforcement from every
//     state, that is a deliberate follow-up change to this table, not an
//     oversight.
//  3. ROLLBACK has no §8 "Possible transitions" list, but its prose says
//     "Task marked failed" — modeled as a single ROLLBACK → FAILED edge.
//  4. HUMAN_REVIEW's resume target is unspecified beyond "Can later
//     resume" (§8) and implem_1.txt's note that the resume signal's
//     verdict is "fed into route() exactly like any other verdict".
//     Modeled permissively: HUMAN_REVIEW can resume into any non-CREATED
//     active state (READY through VALIDATING_DEPLOYMENT), or terminate via
//     FAILED/CANCELLED — a human resolving an escalation may redirect the
//     task anywhere reasonable, not just back to the state that escalated.
//  5. CANCELLED has no §8 heading of its own (only referenced as a target
//     from CREATED). Modeled as a third terminal state alongside DONE and
//     FAILED, reachable only from CREATED, per the literal text.
package statemachine

// State is the closed set of states a Task can occupy, per spec §8.
type State string

const (
	StateCreated              State = "CREATED"
	StateReady                State = "READY"
	StateCollectContext       State = "COLLECT_CONTEXT"
	StateImplementing         State = "IMPLEMENTING"
	StateReviewing            State = "REVIEWING"
	StateTesting              State = "TESTING"
	StateDocumenting          State = "DOCUMENTING"
	StateReadyToMerge         State = "READY_TO_MERGE"
	StateMerging              State = "MERGING"
	StateDeploying            State = "DEPLOYING"
	StateValidatingDeployment State = "VALIDATING_DEPLOYMENT"
	StateRollback             State = "ROLLBACK"
	StateHumanReview          State = "HUMAN_REVIEW"
	StateDone                 State = "DONE"
	StateFailed               State = "FAILED"
	StateCancelled            State = "CANCELLED"
)

// transitions is the authoritative adjacency list for the state graph. Keys
// missing from this map (there are none — every State constant has an
// entry, even terminal ones with an empty slice) would mean "unknown
// state", not "no transitions".
var transitions = map[State][]State{
	StateCreated: {StateReady, StateCancelled},
	StateReady:   {StateCollectContext},

	StateCollectContext: {StateImplementing},

	StateImplementing: {StateReviewing, StateTesting, StateFailed},

	// DOCUMENTING added per modeling decision 1 (see package doc).
	StateReviewing: {StateImplementing, StateTesting, StateHumanReview, StateDocumenting},

	// DOCUMENTING added per modeling decision 1 (see package doc).
	StateTesting: {StateImplementing, StateReadyToMerge, StateFailed, StateDocumenting},

	StateDocumenting: {StateReadyToMerge},

	StateReadyToMerge: {StateMerging},

	StateMerging: {StateDeploying},

	StateDeploying: {StateValidatingDeployment, StateFailed},

	StateValidatingDeployment: {StateDone, StateRollback},

	// Per modeling decision 3.
	StateRollback: {StateFailed},

	// Per modeling decision 4: any non-CREATED active state, or a
	// terminal exit.
	StateHumanReview: {
		StateReady, StateCollectContext, StateImplementing, StateReviewing,
		StateTesting, StateDocumenting, StateReadyToMerge, StateMerging,
		StateDeploying, StateValidatingDeployment, StateFailed, StateCancelled,
	},

	// Terminal states: no outgoing transitions.
	StateDone:      {},
	StateFailed:    {},
	StateCancelled: {},
}

// AllStates returns every state in the graph, in declaration order. Useful
// for exhaustive test iteration and for building UIs/CLIs that need to
// enumerate the graph.
func AllStates() []State {
	// Declared explicitly (not derived from the transitions map, whose
	// key order is unspecified) so callers get a stable, spec-ordered
	// list matching §8's document order.
	return []State{
		StateCreated, StateReady, StateCollectContext, StateImplementing,
		StateReviewing, StateTesting, StateDocumenting, StateReadyToMerge,
		StateMerging, StateDeploying, StateValidatingDeployment, StateRollback,
		StateHumanReview, StateDone, StateFailed, StateCancelled,
	}
}

// IsKnownState reports whether s is one of the closed set of States this
// package defines.
func IsKnownState(s State) bool {
	_, ok := transitions[s]
	return ok
}

// IsTerminal reports whether s has no legal outgoing transitions, i.e. a
// task in this state is done — successfully or not — and the workflow
// should stop making transition decisions.
func IsTerminal(s State) bool {
	targets, ok := transitions[s]
	return ok && len(targets) == 0
}

// TransitionsFrom returns the states directly reachable from s. The
// returned slice is a defensive copy — callers may not mutate the package's
// internal table through it. Returns nil for an unknown state.
func TransitionsFrom(s State) []State {
	targets, ok := transitions[s]
	if !ok {
		return nil
	}
	out := make([]State, len(targets))
	copy(out, targets)
	return out
}

// IsValidTransition reports whether from → to is a legal edge in the state
// graph. This is the single source of truth every caller — Step 4's
// Transition Router, ApplyTransition below, and anything else that moves a
// task between states — must consult before committing a transition.
func IsValidTransition(from, to State) bool {
	for _, candidate := range transitions[from] {
		if candidate == to {
			return true
		}
	}
	return false
}
