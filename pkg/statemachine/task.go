package statemachine

import (
	"fmt"
	"time"

	baseactivity "github.com/khennicb/sw-factory/pkg/activity"
)

// TransitionRecord is one append-only entry in a Task's transition log, per
// spec §14 ("Every transition is visible", "Reasons for every transition").
type TransitionRecord struct {
	From   State
	To     State
	Reason string
	// Actor identifies what caused the transition, e.g. "system",
	// "review-agent", "human:<username>", or "router". Free-form by
	// design — unlike Verdict (pkg/activity), this never crosses back
	// into routing logic, it's purely for the audit trail.
	Actor     string
	Timestamp time.Time
}

// Task is one GitHub-issue-backed unit of work moving through the state
// graph (spec §7, "Task Workflow"). It is the value Step 4's Transition
// Router and the eventual Task Workflow operate on.
type Task struct {
	ID     baseactivity.TaskID
	State  State
	Budget Budget
	// Log is the append-only transition history. Never mutate in place —
	// treat Task as an immutable value and go through ApplyTransition.
	Log []TransitionRecord
}

// NewTask creates a Task in StateCreated with no transition history yet,
// and starts its budget's duration clock at startedAt.
func NewTask(id baseactivity.TaskID, budget Budget, startedAt time.Time) Task {
	budget.StartedAt = startedAt
	return Task{
		ID:     id,
		State:  StateCreated,
		Budget: budget,
	}
}

// ApplyTransition moves a task from its current state to `to`, recording
// the transition. It returns a new Task value — the input task is never
// mutated — so callers (and tests) can compare before/after freely and
// Temporal's replay determinism is never at risk of an aliased shared
// struct.
//
// Returns an error without changing anything if from → to is not a legal
// edge in the graph (see IsValidTransition) — this is the guardrail spec §3
// calls "deterministic orchestration": even a malformed or buggy caller
// (a future Transition Router bug, a corrupted verdict) cannot force the
// task outside the state graph.
func ApplyTransition(task Task, to State, reason, actor string, at time.Time) (Task, error) {
	if !IsKnownState(to) {
		return Task{}, fmt.Errorf("statemachine: unknown target state %q", to)
	}
	if !IsValidTransition(task.State, to) {
		return Task{}, fmt.Errorf("statemachine: illegal transition %s -> %s", task.State, to)
	}

	next := task
	next.State = to
	next.Log = append(append([]TransitionRecord(nil), task.Log...), TransitionRecord{
		From:      task.State,
		To:        to,
		Reason:    reason,
		Actor:     actor,
		Timestamp: at,
	})
	return next, nil
}
