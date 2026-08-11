package statemachine

import (
	"testing"
	"time"

	baseactivity "github.com/khennicb/sw-factory/pkg/activity"
)

func TestNewTask(t *testing.T) {
	start := time.Date(2026, 8, 12, 9, 0, 0, 0, time.UTC)
	task := NewTask(baseactivity.TaskID("task-1"), Budget{MaxTokens: 1000}, start)

	if task.State != StateCreated {
		t.Errorf("NewTask state = %s, want %s", task.State, StateCreated)
	}
	if len(task.Log) != 0 {
		t.Errorf("NewTask log = %v, want empty", task.Log)
	}
	if !task.Budget.StartedAt.Equal(start) {
		t.Errorf("NewTask budget.StartedAt = %v, want %v", task.Budget.StartedAt, start)
	}
	if task.Budget.MaxTokens != 1000 {
		t.Errorf("NewTask budget.MaxTokens = %d, want 1000 (caller's budget fields must survive)", task.Budget.MaxTokens)
	}
}

func TestApplyTransitionHappyPath(t *testing.T) {
	start := time.Date(2026, 8, 12, 9, 0, 0, 0, time.UTC)
	task := NewTask(baseactivity.TaskID("task-1"), Budget{}, start)

	at1 := start.Add(time.Minute)
	next, err := ApplyTransition(task, StateReady, "created and immediately readied", "system", at1)
	if err != nil {
		t.Fatalf("ApplyTransition(CREATED->READY) unexpected error: %v", err)
	}
	if next.State != StateReady {
		t.Errorf("next.State = %s, want %s", next.State, StateReady)
	}
	if len(next.Log) != 1 {
		t.Fatalf("next.Log has %d entries, want 1", len(next.Log))
	}
	rec := next.Log[0]
	if rec.From != StateCreated || rec.To != StateReady || rec.Reason != "created and immediately readied" || rec.Actor != "system" || !rec.Timestamp.Equal(at1) {
		t.Errorf("unexpected log record: %+v", rec)
	}

	// Original task must be untouched — ApplyTransition is pure.
	if task.State != StateCreated {
		t.Errorf("original task.State mutated to %s, want it to remain %s", task.State, StateCreated)
	}
	if len(task.Log) != 0 {
		t.Errorf("original task.Log mutated to %v, want it to remain empty", task.Log)
	}

	at2 := start.Add(2 * time.Minute)
	next2, err := ApplyTransition(next, StateCollectContext, "picked up by worker", "router", at2)
	if err != nil {
		t.Fatalf("ApplyTransition(READY->COLLECT_CONTEXT) unexpected error: %v", err)
	}
	if len(next2.Log) != 2 {
		t.Fatalf("next2.Log has %d entries, want 2 (history should accumulate)", len(next2.Log))
	}
	if len(next.Log) != 1 {
		t.Errorf("appending to next2's log leaked back into next.Log: %v", next.Log)
	}
}

func TestApplyTransitionRejectsIllegalEdge(t *testing.T) {
	task := NewTask(baseactivity.TaskID("task-1"), Budget{}, time.Now())

	_, err := ApplyTransition(task, StateMerging, "skip ahead", "buggy-caller", time.Now())
	if err == nil {
		t.Fatal("ApplyTransition(CREATED->MERGING) returned nil error, want an error for an illegal transition")
	}

	// Task must be unaffected by the rejected attempt.
	if task.State != StateCreated {
		t.Errorf("task.State = %s after a rejected transition, want unchanged %s", task.State, StateCreated)
	}
	if len(task.Log) != 0 {
		t.Errorf("task.Log = %v after a rejected transition, want unchanged empty", task.Log)
	}
}

func TestApplyTransitionRejectsUnknownState(t *testing.T) {
	task := NewTask(baseactivity.TaskID("task-1"), Budget{}, time.Now())

	_, err := ApplyTransition(task, State("NOT_A_REAL_STATE"), "typo", "system", time.Now())
	if err == nil {
		t.Fatal("ApplyTransition to an unknown state returned nil error, want an error")
	}
}

func TestApplyTransitionFromTerminalStateFails(t *testing.T) {
	task := Task{ID: baseactivity.TaskID("task-1"), State: StateDone}

	_, err := ApplyTransition(task, StateReady, "trying to resurrect a done task", "system", time.Now())
	if err == nil {
		t.Fatal("ApplyTransition from a terminal state returned nil error, want an error")
	}
}
