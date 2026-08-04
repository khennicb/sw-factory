// Package workflow holds the workflow-engine's Temporal workflow
// definitions. HelloWorkflow is the "hello world" workflow called for in
// Step 1: it runs a simple sequence of activities — one native Go activity,
// one RPC shim to the agents/service stub — to prove the orchestration
// backbone (retries, history, activity dispatch) is stable before anything
// else is built on top of it.
package workflow

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	baseactivity "github.com/khennicb/sw-factory/pkg/activity"
	act "github.com/khennicb/sw-factory/services/workflow-engine/internal/activity"
)

// HelloWorkflowInput is what a caller (services/workflow-engine/cmd/starter,
// or later the real Task Workflow) passes in.
type HelloWorkflowInput struct {
	TaskID  baseactivity.TaskID
	Message string
}

// HelloWorkflowResult is the sequence's combined output, useful for the
// starter CLI to print and for tests to assert against.
type HelloWorkflowResult struct {
	PingMessage      string
	AgentPingMessage string
	AgentVerdict     baseactivity.Verdict
}

// HelloWorkflow executes two activities in sequence:
//  1. Ping — a native Go activity, no external dependency.
//  2. AgentPing — an RPC shim activity that calls the agents/service Python
//     stub, proving the cross-language activity boundary end to end.
//
// Retry policy is deliberately explicit here (rather than relying on SDK
// defaults) since this workflow is the reference example for every later
// activity in the project.
func HelloWorkflow(ctx workflow.Context, in HelloWorkflowInput) (HelloWorkflowResult, error) {
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    30 * time.Second,
			MaximumAttempts:    5,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	logger := workflow.GetLogger(ctx)
	logger.Info("HelloWorkflow started", "taskId", in.TaskID)

	var pingOut act.PingOutput
	if err := workflow.ExecuteActivity(ctx, act.Ping, act.PingInput{Message: in.Message}).Get(ctx, &pingOut); err != nil {
		return HelloWorkflowResult{}, err
	}

	var agentOut act.AgentPingOutput
	agentIn := act.AgentPingInput{TaskID: in.TaskID, Message: pingOut.Message}
	// AgentPing is registered as an activity by name (see cmd/worker) since
	// it's a method on a struct carrying the agent-service endpoint config,
	// not a bare function like Ping.
	if err := workflow.ExecuteActivity(ctx, "AgentPing", agentIn).Get(ctx, &agentOut); err != nil {
		return HelloWorkflowResult{}, err
	}

	logger.Info("HelloWorkflow completed", "taskId", in.TaskID, "verdict", agentOut.Verdict)

	return HelloWorkflowResult{
		PingMessage:      pingOut.Message,
		AgentPingMessage: agentOut.Message,
		AgentVerdict:     agentOut.Verdict,
	}, nil
}
