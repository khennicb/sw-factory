package statemachine

import (
	"reflect"
	"sort"
	"testing"
)

// wantGraph is the full expected adjacency list, spelled out independently
// of the package's own `transitions` map so this test can't pass just by
// re-checking the implementation against itself. It's the executable
// version of the table in this package's doc comment.
var wantGraph = map[State][]State{
	StateCreated:              {StateReady, StateCancelled},
	StateReady:                {StateCollectContext},
	StateCollectContext:       {StateImplementing},
	StateImplementing:         {StateReviewing, StateTesting, StateFailed},
	StateReviewing:            {StateImplementing, StateTesting, StateHumanReview, StateDocumenting},
	StateTesting:              {StateImplementing, StateReadyToMerge, StateFailed, StateDocumenting},
	StateDocumenting:          {StateReadyToMerge},
	StateReadyToMerge:         {StateMerging},
	StateMerging:              {StateDeploying},
	StateDeploying:            {StateValidatingDeployment, StateFailed},
	StateValidatingDeployment: {StateDone, StateRollback},
	StateRollback:             {StateFailed},
	StateHumanReview: {
		StateReady, StateCollectContext, StateImplementing, StateReviewing,
		StateTesting, StateDocumenting, StateReadyToMerge, StateMerging,
		StateDeploying, StateValidatingDeployment, StateFailed, StateCancelled,
	},
	StateDone:      {},
	StateFailed:    {},
	StateCancelled: {},
}

func sorted(states []State) []State {
	out := append([]State(nil), states...)
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// TestAllStatesCovered ensures every State constant has an entry in
// wantGraph (and therefore in AllStates/transitions) — catches a state
// being added to the const block but never wired into the graph.
func TestAllStatesCovered(t *testing.T) {
	all := AllStates()
	if len(all) != len(wantGraph) {
		t.Fatalf("AllStates() has %d states, wantGraph has %d — a state is missing from one of them", len(all), len(wantGraph))
	}
	for _, s := range all {
		if _, ok := wantGraph[s]; !ok {
			t.Errorf("state %s returned by AllStates() has no entry in wantGraph", s)
		}
	}
}

// TestIsValidTransition is exhaustive: for every (from, to) pair across the
// full state space, it asserts IsValidTransition matches wantGraph exactly
// — both that every legal edge is accepted and that every other pair
// (including self-loops and reverse edges not listed) is rejected.
func TestIsValidTransition(t *testing.T) {
	all := AllStates()
	for _, from := range all {
		legal := map[State]bool{}
		for _, to := range wantGraph[from] {
			legal[to] = true
		}
		for _, to := range all {
			got := IsValidTransition(from, to)
			want := legal[to]
			if got != want {
				t.Errorf("IsValidTransition(%s, %s) = %v, want %v", from, to, got, want)
			}
		}
	}
}

// TestSpecCallouts pins the exact scenarios the instructions and spec call
// out by name, so a future refactor of the table can't silently reintroduce
// a rejected illegal jump or lose one of the deliberately-added edges.
func TestSpecCallouts(t *testing.T) {
	cases := []struct {
		name string
		from State
		to   State
		want bool
	}{
		{"implem_1.txt's canonical illegal jump", StateCreated, StateMerging, false},
		{"REVIEWING can escalate to HUMAN_REVIEW", StateReviewing, StateHumanReview, true},
		{"REVIEWING can reach DOCUMENTING (§9 example 1, modeling decision 1)", StateReviewing, StateDocumenting, true},
		{"TESTING can reach DOCUMENTING (§9 example 2, modeling decision 1)", StateTesting, StateDocumenting, true},
		{"IMPLEMENTING cannot jump straight to HUMAN_REVIEW (spec-literal budget choice)", StateImplementing, StateHumanReview, false},
		{"ROLLBACK reaches FAILED (modeling decision 3)", StateRollback, StateFailed, true},
		{"ROLLBACK is not itself terminal-and-childless", StateRollback, StateDone, false},
		{"HUMAN_REVIEW cannot resume into CREATED (modeling decision 4)", StateHumanReview, StateCreated, false},
		{"HUMAN_REVIEW can resume into IMPLEMENTING (modeling decision 4)", StateHumanReview, StateImplementing, true},
		{"HUMAN_REVIEW can abandon via CANCELLED (modeling decision 4)", StateHumanReview, StateCancelled, true},
		{"CANCELLED only reachable from CREATED, not READY", StateReady, StateCancelled, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsValidTransition(tc.from, tc.to); got != tc.want {
				t.Errorf("IsValidTransition(%s, %s) = %v, want %v", tc.from, tc.to, got, tc.want)
			}
		})
	}
}

func TestIsTerminal(t *testing.T) {
	terminal := map[State]bool{StateDone: true, StateFailed: true, StateCancelled: true}
	for _, s := range AllStates() {
		want := terminal[s]
		if got := IsTerminal(s); got != want {
			t.Errorf("IsTerminal(%s) = %v, want %v", s, got, want)
		}
	}
}

func TestIsKnownState(t *testing.T) {
	if !IsKnownState(StateReviewing) {
		t.Error("IsKnownState(StateReviewing) = false, want true")
	}
	if IsKnownState(State("NOT_A_REAL_STATE")) {
		t.Error("IsKnownState of a made-up state = true, want false")
	}
}

func TestTransitionsFromIsDefensiveCopy(t *testing.T) {
	got := TransitionsFrom(StateCreated)
	want := []State{StateReady, StateCancelled}
	if !reflect.DeepEqual(sorted(got), sorted(want)) {
		t.Fatalf("TransitionsFrom(StateCreated) = %v, want %v", got, want)
	}

	// Mutating the returned slice must not affect the package's internal
	// table — otherwise one caller's mutation could corrupt every future
	// caller's view of the graph.
	got[0] = StateFailed
	again := TransitionsFrom(StateCreated)
	if !reflect.DeepEqual(sorted(again), sorted(want)) {
		t.Fatalf("mutating a TransitionsFrom result leaked into the internal table: got %v after mutation, want unaffected %v", again, want)
	}
}

func TestTransitionsFromUnknownState(t *testing.T) {
	if got := TransitionsFrom(State("NOT_A_REAL_STATE")); got != nil {
		t.Errorf("TransitionsFrom(unknown) = %v, want nil", got)
	}
}
