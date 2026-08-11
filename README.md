# sw-factory

An autonomous software factory: a deterministic Temporal workflow engine
orchestrates GitHub-issue-backed tasks through a state machine, delegating
judgment (planning, implementation, review, ...) to separate AI agent
processes over RPC. See `instructions/implem_1.txt` for the build plan this
repo follows.

**Status: Step 2 (Task State Machine as Code) implemented.**
See [`docs/step-2-state-machine.md`](./docs/step-2-state-machine.md) for
decisions made, reproduction steps, and verification evidence (Step 1:
[`docs/step-1-workflow-engine.md`](./docs/step-1-workflow-engine.md)).

## What exists today

- **Temporal server**: already deployed via Docker Compose — see
  `temporal/README.md` for access details (UI, gRPC endpoint, CLI). Not
  re-provisioned by this step.
- **Monorepo skeleton**: `services/*`, `agents/*`, `.ai/`, `docs/` — see
  layout below. Only `services/workflow-engine`, `services/
  repository-intelligence` (interface + canned stub), and `agents/service`
  have real code; everything else is a placeholder `README.md` pointing at
  the step that implements it.
- **Hello-world Temporal workflow** (`services/workflow-engine`): a
  workflow that runs a native Go activity (`Ping`) followed by an RPC-shim
  activity (`AgentPing`) that calls the Python `agents/service` stub over
  HTTP — proving both the Go/Temporal orchestration path and the
  cross-language agent boundary end to end, with retries and history
  visible in the Temporal UI.
- **Base `Activity` contracts** (`pkg/activity`): the shared `Input`,
  `Result`, and closed-enum `Verdict` type every activity — real or agent
  shim — will use, so Step 4's Transition Router never has to parse free
  text.
- **Task state machine** (`pkg/statemachine`): the full 16-state graph from
  spec §8 (three spec gaps resolved — see
  [`docs/step-2-state-machine.md`](./docs/step-2-state-machine.md)),
  `IsValidTransition`, budget tracking (§11), and an append-only
  transition log (§14) — pure Go, zero Temporal/AI dependency, 100% test
  coverage. This is the contract Step 4's Transition Router validates every
  proposed transition against.

Telemetry (OTel → Tempo) is explicitly deferred past this step.

## Layout

```
services/
  workflow-engine/          # Go, Temporal SDK — real (Step 1)
  transition-router/        # Step 4 — will consume pkg/statemachine
  repository-intelligence/  # interface + stub real (Step 1/3), full impl Step 5
  ai-router/                # Step 6
  github-integration/       # Step 3
  ci-integration/           # Step 9
  deployment-integration/   # later
  browser-testing/          # Step 11
agents/                     # NOT Go — separate process(es) over RPC, see agents/README.md
  service/                  # Step 1 runtime-boundary stub (FastAPI, /rpc/ping)
  planning/                 # Step 6+
  task-decomposition/       # Step 6+
  repository-context/       # Step 5+
  implementation/           # Step 6
  review/                   # Step 7
  test-analysis/            # Step 8
  documentation/            # Step 10
pkg/activity/                # shared Activity Input/Result/Verdict contracts (Go)
pkg/statemachine/            # Task state graph, transition validation, budgets, transition log (Go) — real (Step 2)
.ai/                          # agent-facing repo knowledge, indexed in Step 5
docs/                         # human-facing project docs
temporal/                     # already-deployed Temporal Docker Compose stack + runbooks
instructions/                 # the build plan (implem_1.txt) this repo follows
```

## Running the Step 1 smoke test

Requires the Temporal server already deployed in `temporal/` to be running
(`cd temporal && sudo docker compose up -d`), and Go on `PATH` (installed to
`~/go-sdk` — see `~/.bashrc`).

```bash
# 1. start the agent-service stub (separate terminal)
cd agents/service
python3 -m venv .venv && source .venv/bin/activate
pip install -r requirements.txt
uvicorn main:app --host 0.0.0.0 --port 9101

# 2. start the workflow-engine worker (separate terminal)
cd services/workflow-engine
go run ./cmd/worker

# 3. trigger one HelloWorkflow execution and wait for its result
cd services/workflow-engine
go run ./cmd/starter
```

`go run ./cmd/starter` should print the workflow's result, and the run
should be visible in the Temporal UI (see `temporal/README.md` for the
URL).

## Running the Step 2 state machine tests

No external dependencies (Temporal, agents/service) needed — pure Go:

```bash
cd pkg/statemachine
go test ./... -v -cover
```

## Next steps

Per `instructions/implem_1.txt`: Step 3 (GitHub integration + webhook
receiver), then Step 4 (Transition Router, now unblocked by
`pkg/statemachine`) — both still zero-AI, fully deterministic and
unit-testable, before any real agent is wired in at Step 5+.
