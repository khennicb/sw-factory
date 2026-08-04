// Package activity holds the workflow-engine's own Temporal activities.
//
// This first pair exists purely to prove the orchestration backbone works
// end to end: a native Go activity (Ping) and an agent-shim activity
// (AgentPing) that calls out to the Python process in agents/service over
// HTTP, per the runtime boundary described in agents/README.md. Real
// activities (GitHub integration, agent shims for planning/implementation/
// review/etc.) land in their own services/<name> and agents/<name>
// packages in later steps — this package is not where they belong.
package activity

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	baseactivity "github.com/khennicb/sw-factory/pkg/activity"
)

// PingInput/PingOutput are intentionally trivial — this activity has no
// real-world side effect. It exists so the hello-world workflow has at
// least one "local" activity in its sequence alongside the RPC-backed one.
type PingInput struct {
	Message string `json:"message"`
}

type PingOutput struct {
	Message string `json:"message"`
}

// Ping is a native Go activity: no RPC, no external process, just proof
// that the Temporal worker can execute an activity and return a typed
// result to the workflow.
func Ping(ctx context.Context, in PingInput) (PingOutput, error) {
	return PingOutput{Message: fmt.Sprintf("pong: %s", in.Message)}, nil
}

// AgentPingInput/AgentPingOutput mirror the JSON contract the agents/service
// FastAPI stub exposes at POST /rpc/ping. Keeping the wire shape as plain
// structs (not tied to baseactivity.Input/Result) is deliberate: those two
// stay reserved for the fields every activity shares, while each shim
// defines its own request/response payload shape.
type AgentPingInput struct {
	TaskID  baseactivity.TaskID `json:"taskId"`
	Message string              `json:"message"`
}

type AgentPingOutput struct {
	Message string               `json:"message"`
	Verdict baseactivity.Verdict `json:"verdict"`
}

// AgentPingActivity is a thin shim: it does no work itself, it only speaks
// HTTP to the out-of-process agent and translates the reply back into a
// typed Go result. This is the pattern every real agents/<name> activity
// (planning, implementation, review, ...) will follow starting in Step 6 —
// this file is the reference implementation of that shape, kept minimal.
type AgentPingActivity struct {
	Config baseactivity.AgentRPCConfig
	Client *http.Client
}

func NewAgentPingActivity(cfg baseactivity.AgentRPCConfig) *AgentPingActivity {
	client := &http.Client{Timeout: cfg.Timeout}
	if cfg.Timeout == 0 {
		client.Timeout = 10 * time.Second
	}
	return &AgentPingActivity{Config: cfg, Client: client}
}

func (a *AgentPingActivity) AgentPing(ctx context.Context, in AgentPingInput) (AgentPingOutput, error) {
	body, err := json.Marshal(in)
	if err != nil {
		return AgentPingOutput{}, fmt.Errorf("marshal agent ping request: %w", err)
	}

	url := a.Config.Endpoint + "/rpc/ping"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return AgentPingOutput{}, fmt.Errorf("build agent ping request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.Client.Do(req)
	if err != nil {
		// Returned as an error (not a Failed verdict) so Temporal's
		// activity retry policy handles transient agent-service
		// unavailability, matching Step 1's "retries and history"
		// requirement.
		return AgentPingOutput{}, fmt.Errorf("call agent service at %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return AgentPingOutput{}, fmt.Errorf("agent service returned status %d", resp.StatusCode)
	}

	var out AgentPingOutput
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return AgentPingOutput{}, fmt.Errorf("decode agent ping response: %w", err)
	}
	return out, nil
}
