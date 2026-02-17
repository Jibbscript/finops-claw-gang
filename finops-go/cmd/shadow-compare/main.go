// shadow-compare runs the Go and Python FinOps pipelines on the same golden fixtures,
// compares triage/analysis/policy outputs, and produces a JSON diff report.
// Exit code 0 = all phases match. Exit code 1 = divergence detected. Exit code 2 = error.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"

	"github.com/finops-claw-gang/finops-go/internal/shadow"
)

func main() {
	fixturesDir := flag.String("fixtures-dir", "", "path to golden fixtures directory (required)")
	pythonPath := flag.String("python-path", "python", "path to Python interpreter")
	service := flag.String("service", "EC2", "service name for the anomaly")
	delta := flag.Float64("delta", 750, "delta dollars for the anomaly")
	goOnly := flag.Bool("go-only", false, "run only the Go pipeline (skip Python comparison)")
	flag.Parse()

	if *fixturesDir == "" {
		fmt.Fprintln(os.Stderr, "error: --fixtures-dir is required")
		flag.Usage()
		os.Exit(2)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	// Run Go pipeline
	logger.Info("running Go pipeline", "fixtures", *fixturesDir, "service", *service, "delta", *delta)
	goRunner := &shadow.GoRunner{FixturesDir: *fixturesDir}
	goJSON, err := goRunner.Run(ctx, *service, *delta)
	if err != nil {
		logger.Error("Go pipeline failed", "error", err)
		os.Exit(2)
	}

	if *goOnly {
		fmt.Println(string(goJSON))
		return
	}

	// Run Python pipeline
	logger.Info("running Python pipeline", "python", *pythonPath)
	pyRunner := &shadow.PythonRunner{
		PythonPath:  *pythonPath,
		FixturesDir: *fixturesDir,
	}
	pyJSON, err := pyRunner.Run(ctx, *service, *delta)
	if err != nil {
		logger.Error("Python pipeline failed", "error", err)
		os.Exit(2)
	}

	// Compare
	result, err := shadow.Compare(goJSON, pyJSON)
	if err != nil {
		logger.Error("comparison failed", "error", err)
		os.Exit(2)
	}

	out, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		logger.Error("marshal result failed", "error", err)
		os.Exit(2)
	}
	fmt.Println(string(out))

	if !result.AllMatch {
		logger.Warn("divergence detected", "summary", result.Summary)
		os.Exit(1)
	}

	logger.Info("all phases match")
}
