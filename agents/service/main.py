"""agents/service — Step 1 runtime-boundary stub.

This is not one of the seven domain agents (planning, implementation,
review, ...). It's the minimal process needed to prove out the pattern
every real agent will follow starting in Step 6: the Go workflow-engine
calls this over HTTP via a thin activity shim
(services/workflow-engine/internal/activity/hello.go's AgentPingActivity),
and gets back a typed JSON response carrying a closed-enum verdict — never
free text — for the Transition Router (Step 4) to consume.

Endpoints:
    POST /rpc/ping   - the only endpoint today; echoes the caller's message
                       back with a SUCCEEDED verdict.
    GET  /healthz    - liveness check for local dev / future container
                       orchestration.
"""

from __future__ import annotations

from fastapi import FastAPI
from pydantic import BaseModel

app = FastAPI(title="sw-factory agents/service", version="0.1.0")


class PingRequest(BaseModel):
    taskId: str
    message: str


class PingResponse(BaseModel):
    message: str
    verdict: str = "SUCCEEDED"


@app.get("/healthz")
def healthz() -> dict[str, str]:
    return {"status": "ok"}


@app.post("/rpc/ping", response_model=PingResponse)
def rpc_ping(req: PingRequest) -> PingResponse:
    return PingResponse(message=f"agent-service received: {req.message} (task={req.taskId})")
