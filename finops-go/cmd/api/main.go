// Command api runs the HTTP API server for the FinOps Generative UI.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.temporal.io/sdk/client"

	"github.com/finops-claw-gang/finops-go/internal/api"
	"github.com/finops-claw-gang/finops-go/internal/config"
	"github.com/finops-claw-gang/finops-go/internal/observability"
	"github.com/finops-claw-gang/finops-go/internal/temporal/querier"
)

func main() {
	cfg, err := config.LoadFromEnv()
	if err != nil {
		slog.Error("config error", "error", err)
		os.Exit(1)
	}

	logger := observability.InitLogger(cfg.LogLevel)
	temporalLogger := observability.NewTemporalSlogAdapter(logger)

	if cfg.OTelEnabled {
		shutdown, err := observability.InitTracer(context.Background(), "api")
		if err != nil {
			logger.Error("otel init failed", "error", err)
		} else {
			defer shutdown(context.Background())
		}
	}

	c, err := client.Dial(client.Options{
		Logger: temporalLogger,
	})
	if err != nil {
		logger.Error("unable to create Temporal client", "error", err)
		os.Exit(1)
	}
	defer c.Close()

	q := querier.New(c)

	oidcCfg := api.OIDCConfig{
		IssuerURL: cfg.OIDCIssuer,
		Audience:  cfg.OIDCAudience,
		Enabled:   cfg.OIDCEnabled(),
	}
	srv := api.New(q, cfg.CORSOrigins, oidcCfg)

	var handler http.Handler = srv
	if cfg.OTelEnabled {
		handler = otelhttp.NewHandler(handler, "finops-api")
	}

	addr := ":" + cfg.APIPort
	logger.Info("starting API server", "addr", addr, "oidc_enabled", oidcCfg.Enabled)
	if err := http.ListenAndServe(addr, handler); err != nil {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}
}
