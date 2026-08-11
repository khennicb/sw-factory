package statemachine

import (
	"reflect"
	"testing"
	"time"
)

func TestBudgetNotExceededWhenUnderLimitsAndUnlimitedFieldsIgnored(t *testing.T) {
	now := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)
	b := Budget{
		MaxTokens:  1000,
		TokensUsed: 999,
		// Every other Max* field left at zero == unlimited, regardless of
		// how much has been consumed against it.
		LocalIterations: 1_000_000,
	}
	if b.Exceeded(now) {
		t.Errorf("Exceeded() = true, want false (under the one set limit, others unlimited)")
	}
	if reasons := b.ExceededReasons(now); len(reasons) != 0 {
		t.Errorf("ExceededReasons() = %v, want empty", reasons)
	}
}

func TestBudgetExceededPerDimension(t *testing.T) {
	now := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)
	started := now.Add(-time.Hour)

	cases := []struct {
		name   string
		budget Budget
		reason string
	}{
		{"tokens", Budget{MaxTokens: 100, TokensUsed: 100}, "token budget exceeded"},
		{"tokens over", Budget{MaxTokens: 100, TokensUsed: 150}, "token budget exceeded"},
		{"local iterations", Budget{MaxLocalIterations: 5, LocalIterations: 5}, "local iteration budget exceeded"},
		{"remote escalations", Budget{MaxRemoteEscalations: 2, RemoteEscalations: 2}, "remote escalation budget exceeded"},
		{"review cycles", Budget{MaxReviewCycles: 3, ReviewCycles: 3}, "review cycle budget exceeded"},
		{"test retries", Budget{MaxTestRetries: 4, TestRetries: 4}, "test retry budget exceeded"},
		{"duration", Budget{MaxDuration: 30 * time.Minute, StartedAt: started}, "workflow duration budget exceeded"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if !tc.budget.Exceeded(now) {
				t.Fatalf("Exceeded() = false, want true for %+v", tc.budget)
			}
			reasons := tc.budget.ExceededReasons(now)
			if !reflect.DeepEqual(reasons, []string{tc.reason}) {
				t.Errorf("ExceededReasons() = %v, want [%s]", reasons, tc.reason)
			}
		})
	}
}

func TestBudgetExceededReasonsAccumulate(t *testing.T) {
	now := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)
	b := Budget{
		MaxTokens:          10,
		TokensUsed:         10,
		MaxLocalIterations: 3,
		LocalIterations:    5,
	}
	reasons := b.ExceededReasons(now)
	want := []string{"token budget exceeded", "local iteration budget exceeded"}
	if !reflect.DeepEqual(reasons, want) {
		t.Errorf("ExceededReasons() = %v, want %v", reasons, want)
	}
}

func TestBudgetDurationNotExceededWithZeroStartedAt(t *testing.T) {
	now := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)
	b := Budget{MaxDuration: time.Minute} // StartedAt left zero-valued
	if b.Exceeded(now) {
		t.Error("Exceeded() = true with a zero StartedAt, want false (duration check should not fire on an unset clock)")
	}
}

func TestBudgetPolicyValues(t *testing.T) {
	if PolicyStrict == PolicyLenient {
		t.Fatal("PolicyStrict and PolicyLenient must be distinct values")
	}
}
