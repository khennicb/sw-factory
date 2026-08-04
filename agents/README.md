# agents/

This directory is deliberately **not Go code**. Per the spec (§ runtime
boundary) and the Step 1 plan, the orchestration layer (`services/
workflow-engine`, Temporal) and the agent layer are two separate processes
talking over RPC, not one binary:

- **`services/workflow-engine`** (Go, Temporal SDK) owns *when* work
  happens: retries, timeouts, state transitions, history. It has zero LLM
  dependency.
- **`agents/*`** (this directory) owns *what a task should think*: planning,
  decomposition, implementation, review, etc. Each subdirectory here will
  become an independent process, free to pick whatever language/LLM SDK
  suits it best — most agent SDKs (Claude Agent SDK, LangChain, ...) are
  strongest in Python or TypeScript, and coupling the workflow engine to
  that choice would be a mistake.

The two layers meet at a narrow contract: a Go **activity shim** in
`services/<name>` makes an RPC call to the matching `agents/<name>` process
and translates the reply into a `pkg/activity.Verdict` — see
`pkg/activity/activity.go` for why that boundary is a closed enum, not free
text. `services/workflow-engine/internal/activity/hello.go`'s `AgentPing`
is the reference implementation of that shape.

## Transport (Step 1)

`agents/service` is a minimal **FastAPI/HTTP** stub exposing that contract
today (`POST /rpc/ping`). HTTP was chosen over gRPC for Step 1 to avoid
pulling in a protobuf toolchain before there's a real schema to lock down —
the spec says "gRPC/HTTP", not gRPC specifically. If a later step needs
gRPC (streaming, strict schema enforcement across languages), the contract
in `pkg/activity` doesn't change, only the transport in `agents/service` and
its Go-side shim.

## Layout

```
agents/
  service/                # Step 1: shared runtime (FastAPI app, /rpc/ping)
  planning/                # Step 6+: Planning Agent
  task-decomposition/      # Step 6+: Task Decomposition Agent
  repository-context/      # Step 5+: Repository Context Agent
  implementation/          # Step 6: Implementation Agent (first real AI component)
  review/                  # Step 7: Review Agent -> Approved | ChangesRequested | Escalate
  test-analysis/           # Step 8: Test Analysis Agent -> Fixable | Unfixable | Escalate
  documentation/           # Step 10: Documentation Agent
```

Each of the seven domain subdirectories is currently an empty placeholder
(see its own `README.md`) — no agent logic exists yet. Only `service/` runs
today, and only to prove the RPC boundary works before any real agent is
built on top of it.

## Running the Step 1 stub

```bash
cd agents/service
python3 -m venv .venv && source .venv/bin/activate
pip install -r requirements.txt
uvicorn main:app --host 0.0.0.0 --port 9101
```

The workflow-engine worker calls it at `AGENT_SERVICE_URL` (default
`http://localhost:9101`).
