// Command starter kicks off one HelloWorkflow execution and waits for its
// result. This is the "trivial ping" end-to-end test called for in Step 1:
// `go run ./cmd/starter` should complete successfully against a running
// worker + Temporal server and print the result.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"go.temporal.io/sdk/client"

	baseactivity "github.com/khennicb/sw-factory/pkg/activity"
	wf "github.com/khennicb/sw-factory/services/workflow-engine/internal/workflow"
)

const TaskQueue = "hello-world"

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	temporalAddress := getenv("TEMPORAL_ADDRESS", "localhost:7233")

	c, err := client.Dial(client.Options{HostPort: temporalAddress})
	if err != nil {
		log.Fatalf("unable to create Temporal client (address %s): %v", temporalAddress, err)
	}
	defer c.Close()

	workflowID := fmt.Sprintf("hello-world-%d", time.Now().Unix())
	options := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: TaskQueue,
	}

	input := wf.HelloWorkflowInput{
		TaskID:  baseactivity.TaskID("step1-smoke-test"),
		Message: "hello from the starter CLI",
	}

	log.Printf("starting workflow %s on task queue %s (temporal=%s)", workflowID, TaskQueue, temporalAddress)

	run, err := c.ExecuteWorkflow(context.Background(), options, wf.HelloWorkflow, input)
	if err != nil {
		log.Fatalf("unable to start workflow: %v", err)
	}

	var result wf.HelloWorkflowResult
	if err := run.Get(context.Background(), &result); err != nil {
		log.Fatalf("workflow execution failed: %v", err)
	}

	fmt.Printf("workflow completed: runID=%s\n", run.GetRunID())
	fmt.Printf("  ping activity   -> %q\n", result.PingMessage)
	fmt.Printf("  agent ping      -> %q (verdict=%s)\n", result.AgentPingMessage, result.AgentVerdict)
}
