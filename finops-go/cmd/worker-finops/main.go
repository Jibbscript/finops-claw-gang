// Command worker-finops runs the Temporal worker for FinOps workflows.
// In Phase 2 it uses stub fixtures; Phase 3 replaces with real AWS connectors.
package main

import (
	"log"
	"os"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"github.com/finops-claw-gang/finops-go/internal/executor"
	"github.com/finops-claw-gang/finops-go/internal/temporal/activities"
	"github.com/finops-claw-gang/finops-go/internal/temporal/versioning"
	"github.com/finops-claw-gang/finops-go/internal/temporal/workflows"
	"github.com/finops-claw-gang/finops-go/internal/testutil"
)

func main() {
	fixturesDir := os.Getenv("FIXTURES_DIR")
	if fixturesDir == "" {
		fixturesDir = testutil.GoldenDir()
	}

	c, err := client.Dial(client.Options{})
	if err != nil {
		log.Fatalf("unable to create Temporal client: %v", err)
	}
	defer c.Close()

	cost := &testutil.StubCost{FixturesDir: fixturesDir}
	infra := &testutil.StubInfra{FixturesDir: fixturesDir}
	kube := &testutil.StubKubeCost{FixturesDir: fixturesDir}
	exec := executor.NewExecutor(infra)

	acts := &activities.Activities{
		Cost:     cost,
		Infra:    infra,
		KubeCost: kube,
		Executor: exec,
	}

	w := worker.New(c, versioning.QueueAnomaly, worker.Options{})
	w.RegisterWorkflow(workflows.AnomalyLifecycleWorkflow)
	w.RegisterWorkflow(workflows.ScheduledDetectionWorkflow)
	w.RegisterActivity(acts)

	log.Printf("starting worker on queue %s (fixtures=%s)", versioning.QueueAnomaly, fixturesDir)
	if err := w.Run(worker.InterruptCh()); err != nil {
		log.Fatalf("worker failed: %v", err)
	}
}
