# services/transition-router

Not implemented yet — planned for **Step 4 (still no AI — pure function on the state machine)** of instructions/implem_1.txt.

Registers as Temporal activities against the workflow engine in
`services/workflow-engine` once built; see `pkg/activity` for the
shared Input/Result/Verdict envelope every activity in this repo uses.

The state graph this router validates every proposed transition against is
already built: `pkg/statemachine` (Step 2, see
`docs/step-2-state-machine.md`) provides `IsValidTransition(from, to)`,
`Task`/`ApplyTransition` (with the append-only transition log), and
`Budget`/`Policy` (tracking only — this router is what actually decides to
force `HUMAN_REVIEW`/`FAILED` on a breach, and only via the edges
`pkg/statemachine` already exposes for that; see that package's doc comment
for which states currently have such an edge and which don't).
