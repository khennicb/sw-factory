# Step 2: Task State Machine as Code

Implements Step 2 of [`instructions/implem_1.txt`](../instructions/implem_1.txt).
Completed 2026-08-12.

## Starting state

Step 1 was complete: Temporal running, a hello-world workflow proving the
Go/Temporal ↔ Python agent-service RPC boundary, and `pkg/activity` locking
in the shared `Input`/`Result`/`Verdict` envelope. No state machine, budget
tracking, or transition log existed yet.

## Decisions made

Spec §8 (the per-state "Possible transitions" lists) and §9 (the dynamic
transition examples) don't fully agree with each other, and §11 (budgets)
doesn't say which states can be forced into `HUMAN_REVIEW`/`FAILED`. Three
of these gaps were ambiguous enough to confirm with the user rather than
assume, since Step 4's Transition Router validates every transition it
proposes against this table — get the table wrong and the router either
can't do its job or can do more than it should.

### 1. Budget-exceeded escape edges: spec-literal, not universal

**Chosen: only the edges §8 literally lists** — `REVIEWING → HUMAN_REVIEW`,
and `IMPLEMENTING|TESTING|DEPLOYING → FAILED` — rather than adding a
`HUMAN_REVIEW`/`FAILED` escape edge from every non-terminal state.

- §11 says "exceeding a budget transitions to HUMAN_REVIEW or FAILED"
  without qualification, which reads as universal. But the user chose
  spec-literal fidelity over inferring an unwritten rule.
- Consequence for Step 4: the Transition Router can only enforce a budget
  by forcing `HUMAN_REVIEW`/`FAILED` from a state that already has that
  edge in this table. From e.g. `COLLECT_CONTEXT` or `MERGING`, a budget
  breach cannot force either target — Step 4 will need its own policy for
  that case (most likely: let the current step finish and enforce at the
  next state that does have the edge). This is flagged in
  `pkg/statemachine/statemachine.go`'s package doc so it isn't rediscovered
  as a bug later.

### 2. HUMAN_REVIEW resume target: any active state

**Chosen: `HUMAN_REVIEW` can transition into any non-`CREATED` active
state** (`READY` through `VALIDATING_DEPLOYMENT`), or terminate via
`FAILED`/`CANCELLED` — not just back into `REVIEWING`, the only state §8
lists as an entry point into `HUMAN_REVIEW`.

- `implem_1.txt` describes the resume signal's verdict as being "fed into
  `route()` exactly like any other verdict" — implying the router, not this
  package, decides where a resume actually goes case by case. This package
  only needs to allow the *widest* structurally sound set so the router
  is never blocked by the graph.
- A human resolving an escalation may reasonably want to redirect the task
  anywhere sensible (e.g. send it straight back to `IMPLEMENTING` after
  fixing something out-of-band), not just retry the state that escalated.

### 3. DOCUMENTING reachability: add the two §9-implied edges

**Chosen: add `REVIEWING → DOCUMENTING` and `TESTING → DOCUMENTING`** on
top of §8's explicit lists for those two states.

- Taken completely literally, §8 leaves `DOCUMENTING` unreachable — no
  other state lists it as a target. §9's two worked examples both show a
  path through `DOCUMENTING` (`... → REVIEWING → DOCUMENTING → MERGE` and
  `... → TESTING → DOCUMENTING → MERGE`), which only makes sense if those
  edges exist.

### Decided without asking (low ambiguity)

- **`ROLLBACK → FAILED`**: §8's `ROLLBACK` prose ("Restore previous
  deployment. Open issue. Task marked failed.") states the outcome
  directly; modeled as a single outgoing edge rather than treating
  `ROLLBACK` as itself terminal.
- **`CANCELLED` as a third terminal state**: it has no `##` heading of its
  own in §8, only appearing as a transition target from `CREATED`. Modeled
  alongside `DONE`/`FAILED` as terminal, reachable only from `CREATED`.
- **Package location**: `pkg/statemachine`, mirroring `pkg/activity` —
  shared, dependency-free, importable by any future service (Step 4's
  `transition-router`, and eventually the real Task Workflow in
  `workflow-engine`) without pulling in Temporal or agent-RPC code.
- **`ApplyTransition` is a pure function returning a new `Task`**, never
  mutating its input — matches `instructions/implem_1.txt`'s
  `applyTransition(task, to) → task` signature and keeps the type safe to
  share across Temporal workflow replays, where aliased mutable state is a
  determinism hazard.
- **Budget limits are opt-in**: a zero-valued `Max*` field means
  "unlimited" for that dimension, rather than every unset field silently
  capping at zero. This matches how the spec's budget list (§11) reads as
  a menu of independent knobs, not a fixed set every task must configure.

## What was built

| Path | Status | Notes |
|---|---|---|
| `pkg/statemachine/statemachine.go` | Real | `State` enum (16 states, §8 names verbatim), the transition adjacency table, `IsValidTransition`, `IsTerminal`, `IsKnownState`, `AllStates`, `TransitionsFrom`. Package doc spells out all five modeling decisions above so the table's provenance is never a mystery. |
| `pkg/statemachine/budget.go` | Real | `Budget` (limits + consumption counters, opt-in per dimension), `Policy` (`STRICT`/`LENIENT`, matching `implem_1.txt`'s `budget.Policy` pseudocode), `Exceeded`/`ExceededReasons` (deterministic — take `now` as a parameter rather than reading the clock internally). Tracks and reports only; does not itself force a transition — that's Step 4's job, within the edges decision 1 leaves available. |
| `pkg/statemachine/task.go` | Real | `Task` (id, state, budget, append-only `Log`), `TransitionRecord` (`from`, `to`, `reason`, `actor`, `timestamp` — spec §14's "reasons for every transition"), `NewTask`, `ApplyTransition` (validates against `IsValidTransition`, returns a new `Task`, rejects unknown target states and illegal edges without mutating the input on failure). |
| `pkg/statemachine/*_test.go` | Real | Exhaustive `IsValidTransition` check across the full 16×16 state pair space against an independently-spelled-out expected graph (not a re-check of the implementation against itself); named test cases pinning every spec callout and modeling decision by name; `Task`/`Budget` behavior tests including immutability and defensive-copy checks. |

## Verification evidence

Ran on 2026-08-12:

```
$ go build ./... && go vet ./...
(clean, no output)

$ go test ./pkg/statemachine/... -v -cover
... (36 subtests, all PASS)
PASS
coverage: 100.0% of statements
ok      github.com/khennicb/sw-factory/pkg/statemachine       0.011s

$ go build ./... && go vet ./... && go test ./...
(whole repo, all packages — clean, no regressions from Step 1)
```

100% statement coverage matches Step 2's stated target in
`instructions/implem_1.txt`.

## Known deferrals (intentional, not gaps)

- **Verdict → State mapping** (e.g. "Review Agent said `ChangesRequested`,
  which specific state does that map to from `REVIEWING`?") is explicitly
  Step 4's `Transition Router`, not this package. This package only answers
  "is `from → to` a legal edge", never "which edge should be taken".
- **Budget enforcement** (actually forcing a transition when
  `Budget.Exceeded()` is true) is Step 4's job for the same reason — this
  package only tracks and reports.
- **Persistence**: `Task`/`TransitionRecord` are plain in-memory Go values.
  Durable storage is Temporal's own workflow history once this is wired
  into a real Task Workflow (Step 3+) — no separate database is introduced
  here.

## Next step

Step 3: GitHub Integration service (real + mock mode) and the webhook
receiver for CI status, per `instructions/implem_1.txt`. This is what lets
the Task Workflow actually create PRs and run the
`IMPLEMENTING → REVIEWING → TESTING → MERGING` loop end to end — still with
the transition sequence hardcoded/linear until Step 4's Transition Router
(now unblocked by this step's table) makes it dynamic.
