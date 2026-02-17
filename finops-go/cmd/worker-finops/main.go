// Command worker-finops runs the Temporal worker for FinOps workflows.
// Supports stub mode (fixtures) and production mode (real AWS connectors).
// Supports multi-queue operation via FINOPS_WORKER_QUEUES env var.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"golang.org/x/sync/errgroup"

	"github.com/finops-claw-gang/finops-go/internal/config"
	"github.com/finops-claw-gang/finops-go/internal/connectors"
	awsauth "github.com/finops-claw-gang/finops-go/internal/connectors/aws"
	"github.com/finops-claw-gang/finops-go/internal/connectors/awsdoctor"
	"github.com/finops-claw-gang/finops-go/internal/connectors/kubecost"
	"github.com/finops-claw-gang/finops-go/internal/domain"
	"github.com/finops-claw-gang/finops-go/internal/executor"
	"github.com/finops-claw-gang/finops-go/internal/observability"
	"github.com/finops-claw-gang/finops-go/internal/temporal/activities"
	"github.com/finops-claw-gang/finops-go/internal/temporal/queues"
	"github.com/finops-claw-gang/finops-go/internal/temporal/versioning"
	"github.com/finops-claw-gang/finops-go/internal/temporal/workflows"
	"github.com/finops-claw-gang/finops-go/internal/testutil"
	"github.com/finops-claw-gang/finops-go/internal/triage"
)

// awsdoctorAdapter wraps a Runner to implement triage.WasteQuerier (and thus AWSDocDeps).
type awsdoctorAdapter struct {
	runner awsdoctor.Runner
}

func (a *awsdoctorAdapter) Waste(ctx context.Context, accountID, region, profile string) ([]domain.WasteFinding, error) {
	report, err := a.runner.Waste(ctx, awsdoctor.RunOpts{Region: region, Profile: profile})
	if err != nil {
		return nil, err
	}
	return awsdoctor.MapWasteFindings(report, region), nil
}

func main() {
	cfg, err := config.LoadFromEnv()
	if err != nil {
		slog.Error("config error", "error", err)
		os.Exit(1)
	}

	logger := observability.InitLogger(cfg.LogLevel)
	temporalLogger := observability.NewTemporalSlogAdapter(logger)

	if cfg.OTelEnabled {
		shutdown, err := observability.InitTracer(context.Background(), "worker-finops")
		if err != nil {
			logger.Error("otel init failed", "error", err)
		} else {
			defer shutdown(context.Background())
		}
	}

	var (
		cost     activities.CostDeps
		infra    activities.InfraDeps
		kubeCost triage.KubeCostQuerier
		awsDoc   activities.AWSDocDeps
	)

	switch cfg.Mode {
	case config.ModeProduction:
		awsCfg, err := awsauth.NewAWSConfig(context.Background(), cfg.AWSRegion, cfg.AWSProfile, cfg.CrossAccountRole)
		if err != nil {
			logger.Error("aws config failed", "error", err)
			os.Exit(1)
		}

		cost = connectors.NewAWSCostClient(awsCfg, cfg.CURDatabase, cfg.CURTable, cfg.CURWorkgroup, cfg.CUROutputBucket)
		infra = connectors.NewAWSInfraClient(awsCfg)

		if cfg.KubeCostEndpoint != "" {
			kubeCost = kubecost.New(cfg.KubeCostEndpoint)
		} else {
			kubeCost = &testutil.StubKubeCost{FixturesDir: testutil.GoldenDir()}
		}

		awsDoc = &awsdoctorAdapter{
			runner: awsdoctor.NewBinaryRunner(cfg.AWSDocBinaryPath),
		}

	default: // stub mode
		fixturesDir := cfg.FixturesDir
		if fixturesDir == "" {
			fixturesDir = testutil.GoldenDir()
		}
		cost = &testutil.StubCost{FixturesDir: fixturesDir}
		infra = &testutil.StubInfra{FixturesDir: fixturesDir}
		kubeCost = &testutil.StubKubeCost{FixturesDir: fixturesDir}
		awsDoc = &testutil.StubAWSDoctor{FixturesDir: fixturesDir}
	}

	c, err := client.Dial(client.Options{
		Logger: temporalLogger,
	})
	if err != nil {
		logger.Error("unable to create Temporal client", "error", err)
		os.Exit(1)
	}
	defer c.Close()

	exec := executor.NewExecutor(infra)

	acts := &activities.Activities{
		Cost:     cost,
		Infra:    infra,
		KubeCost: kubeCost,
		AWSDoc:   awsDoc,
		Executor: exec,
	}

	queueNames, err := queues.ParseQueues(cfg.WorkerQueues)
	if err != nil {
		logger.Error("parse queues failed", "error", err)
		os.Exit(1)
	}
	queueConfigs := queues.DefaultConfigs()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	g, ctx := errgroup.WithContext(ctx)
	for _, qName := range queueNames {
		qcfg := queueConfigs[qName]
		w := worker.New(c, qName, qcfg.Options)

		switch qName {
		case versioning.QueueAnomaly:
			w.RegisterWorkflow(workflows.AnomalyLifecycleWorkflow)
			w.RegisterWorkflow(workflows.AWSDocSweepWorkflow)
			w.RegisterActivity(acts)
		case versioning.QueueDetect:
			w.RegisterWorkflow(workflows.ScheduledDetectionWorkflow)
			w.RegisterActivity(acts)
		case versioning.QueueExec:
			w.RegisterActivity(acts)
		}

		logger.Info("starting worker", "queue", qName, "mode", cfg.Mode)
		g.Go(func() error {
			return w.Run(worker.InterruptCh())
		})
	}

	if err := g.Wait(); err != nil {
		logger.Error("worker failed", "error", err)
		os.Exit(1)
	}
}
