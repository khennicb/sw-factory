# Step 1: Bootstrap the Workflow Engine Infrastructure

Implements Step 1 of [`instructions/implem_1.txt`](../instructions/implem_1.txt).
Completed 2026-08-05.

## Starting state

Before this work began:

- **Temporal server was already deployed** (from a prior session) via Docker
  Compose in [`temporal/`](../temporal/) — server 1.29.1, UI at
  `192.168.1.135:8080`, gRPC at `192.168.1.135:7233`/`localhost:7233`. This
  step built on top of it rather than re-provisioning.
- No Go toolchain installed on the host (`rasp-pi-power`, Debian 13 aarch64).
- No Node/TS runtime installed. Python 3.13.5 was present.
- Repo was otherwise empty except for `temporal/` and `instructions/`.

## Decisions made

Three things were ambiguous enough in the spec to ask the user rather than
assume. Answers below, with the reasoning for each.

### 1. How to install Go

**Chosen: official tarball from go.dev, not the apt package.**

- apt's Go package on Debian trixie lags upstream releases; the tarball
  gets the current version (1.26.5) and matches how most Go projects
  recommend installing.
- Installed to `~/go-sdk` (`GOROOT`), with `GOPATH=~/go`, both added to
  `PATH` via `~/.bashrc` and `~/.profile`. Downloaded
  `go1.26.5.linux-arm64.tar.gz` (host arch is `aarch64`).
- Consequence: **`go` is not on `PATH` inside non-interactive/non-login
  shells** (e.g. this agent's Bash tool doesn't source `.bashrc`). Every
  command in this doc and in CI-style scripts needs:
  ```bash
  export GOROOT="$HOME/go-sdk"; export GOPATH="$HOME/go"
  export PATH="$GOROOT/bin:$GOPATH/bin:$PATH"
  ```
  Interactive login shells (a normal terminal) pick this up automatically
  from the profile.

### 2. Language for the `agents/*` service

**Chosen: Python (FastAPI), not TypeScript.**

- The spec's runtime-boundary reasoning (§ "Decide the runtime boundary
  now") explicitly calls out Python/TS as the strongest fit for agent SDKs;
  Python was already present on the host and needed no new toolchain
  install, unlike Node.
- See [`agents/README.md`](../agents/README.md) for the full rationale,
  including why HTTP (not gRPC) was picked as the Step 1 transport: gRPC
  would mean standing up a protobuf codegen pipeline before there's a real
  schema worth locking down. The spec says "gRPC/HTTP", not gRPC
  specifically — this can change transport later without touching the
  `pkg/activity` contract.

### 3. Telemetry scope

**Chosen: skip entirely in Step 1**, deferred to whichever later step needs
it. No OpenTelemetry SDK wiring, no Tempo/Grafana containers were added.
This keeps Step 1 scoped to the orchestration backbone itself; revisit when
there's enough activity/workflow surface area for traces to be useful.

## What was built

| Path | Status | Notes |
|---|---|---|
| `pkg/activity/activity.go` | Real | Shared `Input`/`Result`/`Verdict` envelope every activity (native or agent-shim) uses. `Verdict` is a closed enum by design — see file's doc comment. |
| `services/workflow-engine/` | Real | Temporal Go SDK worker (`cmd/worker`), starter CLI (`cmd/starter`), `HelloWorkflow`, `Ping` (native activity), `AgentPing` (RPC-shim activity). |
| `services/repository-intelligence/` | Interface + stub real | `interface.go` locks `RepositoryIntelligence.GetContext` per Step 3's explicit ask; `stub.go` is canned data. Full impl is Step 5. |
| `agents/service/` | Real | FastAPI stub, `POST /rpc/ping` + `GET /healthz`. Not a domain agent — exists purely to prove the RPC boundary. |
| `services/{transition-router,ai-router,github-integration,ci-integration,deployment-integration,browser-testing}/` | Placeholder | `README.md` only, naming the step that implements each. |
| `agents/{planning,task-decomposition,repository-context,implementation,review,test-analysis,documentation}/` | Placeholder | `README.md` only; `review` and `test-analysis` READMEs also record their eventual verdict contracts (`Approved\|ChangesRequested\|Escalate`, `Fixable\|Unfixable\|Escalate`) since those are already fixed by the spec. |
| `.ai/`, `docs/` | Placeholder | Directory + `README.md` explaining intended future contents. |
| `go.mod` | Real | `github.com/khennicb/sw-factory`, Go 1.26, `go.temporal.io/sdk` v1.47.0. |

## How to reproduce the smoke test

Requires the Temporal Docker Compose stack running (`cd temporal && sudo
docker compose up -d`) and the Go/PATH exports from Decision 1 above.

```bash
# terminal 1 — agent-service stub
cd agents/service
python3 -m venv .venv && source .venv/bin/activate
pip install -r requirements.txt
uvicorn main:app --host 0.0.0.0 --port 9101

# terminal 2 — workflow-engine worker
cd services/workflow-engine
go run ./cmd/worker

# terminal 3 — trigger one execution and wait for its result
cd services/workflow-engine
go run ./cmd/starter
```

## Verification evidence

Ran on 2026-08-05 against the live Temporal server (not mocked):

```
$ go run ./cmd/starter
2026/08/05 01:23:59 starting workflow hello-world-1785885839 on task queue hello-world (temporal=localhost:7233)
workflow completed: runID=019fcf17-4fe3-76ba-8d8e-0b57e97626c5
  ping activity   -> "pong: hello from the starter CLI"
  agent ping      -> "agent-service received: pong: hello from the starter CLI (task=step1-smoke-test)" (verdict=SUCCEEDED)
```

Cross-checked against Temporal's own history via the admin CLI
(`temporal workflow show --workflow-id hello-world-1785885839`), confirming
durable execution rather than just a successful client call:

```
Progress:
  ID  Time                     Type
    1  2026-08-04T23:23:59Z  WorkflowExecutionStarted
    5  2026-08-04T23:23:59Z  ActivityTaskScheduled     (Ping)
    7  2026-08-04T23:23:59Z  ActivityTaskCompleted     (Ping)
   11  2026-08-04T23:23:59Z  ActivityTaskScheduled     (AgentPing)
   13  2026-08-04T23:23:59Z  ActivityTaskCompleted     (AgentPing)
   17  2026-08-04T23:23:59Z  WorkflowExecutionCompleted

Status          COMPLETED
Result          {"AgentPingMessage":"agent-service received: pong: hello from the starter CLI (task=step1-smoke-test)","AgentVerdict":"SUCCEEDED","PingMessage":"pong: hello from the starter CLI"}
```

Also verified independently:
- `go build ./...` and `go vet ./...` — clean.
- `curl -X POST http://localhost:9101/rpc/ping ...` — agent-service stub
  responds correctly in isolation, before wiring it into the workflow.

This satisfies Step 1's stated output: *"A running Temporal dev server + a
workflow that can dispatch activities with retries and history."* The
retry policy (`InitialInterval: 1s`, `BackoffCoefficient: 2.0`,
`MaximumAttempts: 5`) is configured in `HelloWorkflow` but wasn't exercised
by this run since nothing failed — it will get exercised naturally once
Step 3's GitHub integration activities (which can hit real transient
failures) land.

## Known deferrals (intentional, not gaps)

- Telemetry (OTel → Tempo) — see Decision 3.
- gRPC transport for `agents/*` — HTTP for now, see Decision 2.
- No Go workspace (`go.work`) split — single root `go.mod` covers all of
  `services/*` and `pkg/*`. Revisit if a service ever needs a conflicting
  dependency version.
- `services/repository-intelligence` is interface + canned stub only; real
  `.ai/` indexing is Step 5.

## Next step

Step 2: implement the task state machine as code
(`instructions/implem_1.txt`) — pure logic, no Temporal/AI dependency,
100% unit-test coverage target.
