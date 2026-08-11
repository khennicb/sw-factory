# docs/

Human-facing project documentation (architecture decisions, how the pieces
fit together, runbooks) — as opposed to `.ai/`, which is agent-facing
knowledge indexed by `services/repository-intelligence` for LLM consumption.

`instructions/implem_1.txt` is the source of truth for the build order;
each step gets a write-up here once implemented, covering decisions made,
how to reproduce, and verification evidence.

| Doc | Covers |
|---|---|
| [step-1-workflow-engine.md](./step-1-workflow-engine.md) | Temporal + monorepo skeleton, base Activity contracts, hello-world workflow, agents/service runtime boundary. |
| [step-2-state-machine.md](./step-2-state-machine.md) | Task state graph (`pkg/statemachine`), transition validation, budget tracking, transition log — spec gaps resolved and why. |

`temporal/*.md` (outside this directory) documents the already-deployed
Temporal Docker Compose stack this project runs against.
