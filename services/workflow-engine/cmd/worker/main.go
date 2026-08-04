// Command worker runs the workflow-engine's Temporal worker: it polls
// TaskQueue, executes HelloWorkflow and its activities. Point it at the
// already-deployed Temporal server (see temporal/README.md) via
// TEMPORAL_ADDRESS; it defaults to the local port-forward at localhost:7233.
package main

import (
	"log"
	"os"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	baseactivity "github.com/khennicb/sw-factory/pkg/activity"
	act "github.com/khennicb/sw-factory/services/workflow-engine/internal/activity"
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
	agentServiceURL := getenv("AGENT_SERVICE_URL", "http://localhost:9101")

	c, err := client.Dial(client.Options{HostPort: temporalAddress})
	if err != nil {
		log.Fatalf("unable to create Temporal client (address %s): %v", temporalAddress, err)
	}
	defer c.Close()

	w := worker.New(c, TaskQueue, worker.Options{})

	w.RegisterWorkflow(wf.HelloWorkflow)
	w.RegisterActivity(act.Ping)

	agentPing := act.NewAgentPingActivity(baseactivity.AgentRPCConfig{Endpoint: agentServiceURL})
	// Registered under the explicit name "AgentPing" to match the string
	// name used in workflow.ExecuteActivity in hello.go, since this is a
	// struct method, not a bare function.
	w.RegisterActivityWithOptions(agentPing.AgentPing, activity.RegisterOptions{Name: "AgentPing"})

	log.Printf("workflow-engine worker starting: temporal=%s taskQueue=%s agentService=%s", temporalAddress, TaskQueue, agentServiceURL)

	if err := w.Run(worker.InterruptCh()); err != nil {
		log.Fatalf("worker stopped with error: %v", err)
	}
}
