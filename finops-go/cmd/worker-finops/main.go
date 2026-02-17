// Command worker-finops runs the Temporal worker for FinOps workflows.
// Supports stub mode (fixtures) and production mode (real AWS connectors).
package main

import (
	"context"
	"log"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"github.com/finops-claw-gang/finops-go/internal/config"
	"github.com/finops-claw-gang/finops-go/internal/connectors"
	awsauth "github.com/finops-claw-gang/finops-go/internal/connectors/aws"
	"github.com/finops-claw-gang/finops-go/internal/connectors/kubecost"
	"github.com/finops-claw-gang/finops-go/internal/executor"
	"github.com/finops-claw-gang/finops-go/internal/temporal/activities"
	"github.com/finops-claw-gang/finops-go/internal/temporal/versioning"
	"github.com/finops-claw-gang/finops-go/internal/temporal/workflows"
	"github.com/finops-claw-gang/finops-go/internal/testutil"
	"github.com/finops-claw-gang/finops-go/internal/triage"
)

func main() {
	cfg, err := config.LoadFromEnv()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	var (
		cost     activities.CostDeps
		infra    activities.InfraDeps
		kubeCost triage.KubeCostQuerier
	)

	switch cfg.Mode {
	case config.ModeProduction:
		awsCfg, err := awsauth.NewAWSConfig(context.Background(), cfg.AWSRegion, cfg.AWSProfile, cfg.CrossAccountRole)
		if err != nil {
			log.Fatalf("aws config: %v", err)
		}

		cost = connectors.NewAWSCostClient(awsCfg, cfg.CURDatabase, cfg.CURTable, cfg.CURWorkgroup, cfg.CUROutputBucket)
		infra = connectors.NewAWSInfraClient(awsCfg)

		if cfg.KubeCostEndpoint != "" {
			kubeCost = kubecost.New(cfg.KubeCostEndpoint)
		} else {
			kubeCost = &testutil.StubKubeCost{FixturesDir: testutil.GoldenDir()}
		}

	default: // stub mode
		fixturesDir := cfg.FixturesDir
		if fixturesDir == "" {
			fixturesDir = testutil.GoldenDir()
		}
		cost = &testutil.StubCost{FixturesDir: fixturesDir}
		infra = &testutil.StubInfra{FixturesDir: fixturesDir}
		kubeCost = &testutil.StubKubeCost{FixturesDir: fixturesDir}
	}

	c, err := client.Dial(client.Options{})
	if err != nil {
		log.Fatalf("unable to create Temporal client: %v", err)
	}
	defer c.Close()

	exec := executor.NewExecutor(infra)

	acts := &activities.Activities{
		Cost:     cost,
		Infra:    infra,
		KubeCost: kubeCost,
		Executor: exec,
	}

	w := worker.New(c, versioning.QueueAnomaly, worker.Options{})
	w.RegisterWorkflow(workflows.AnomalyLifecycleWorkflow)
	w.RegisterWorkflow(workflows.ScheduledDetectionWorkflow)
	w.RegisterActivity(acts)

	log.Printf("starting worker on queue %s (mode=%s)", versioning.QueueAnomaly, cfg.Mode)
	if err := w.Run(worker.InterruptCh()); err != nil {
		log.Fatalf("worker failed: %v", err)
	}
}
