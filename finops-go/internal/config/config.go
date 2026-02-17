// Package config provides application configuration loaded from environment variables.
package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

// Mode determines whether the worker uses stub fixtures or real AWS connectors.
type Mode string

const (
	ModeStub       Mode = "stub"
	ModeProduction Mode = "production"
)

// Config holds all application configuration.
type Config struct {
	Mode             Mode
	FixturesDir      string
	AWSRegion        string
	AWSProfile       string
	CrossAccountRole string
	CURDatabase      string
	CURTable         string
	CURWorkgroup     string
	CUROutputBucket  string
	KubeCostEndpoint string

	// Worker settings.
	WorkerQueues string // comma-separated queue list (env FINOPS_WORKER_QUEUES)

	// API server settings.
	APIPort     string
	CORSOrigins []string

	// OIDC settings.
	OIDCIssuer   string
	OIDCAudience string

	// Observability.
	LogLevel    string
	OTelEnabled bool

	AWSDocBinaryPath string
	SweepAccounts    string

	// Shadow-run.
	ShadowPythonPath string

	// Rate limits (requests per second). Zero means use default.
	RateLimitCE     float64
	RateLimitAthena float64
	RateLimitCW     float64
	RateLimitSTS    float64
}

// LoadFromEnv reads configuration from environment variables with sensible defaults.
func LoadFromEnv() (Config, error) {
	cfg := Config{
		Mode:             Mode(envOr("FINOPS_MODE", "stub")),
		FixturesDir:      os.Getenv("FIXTURES_DIR"),
		AWSRegion:        envOr("AWS_REGION", "us-east-1"),
		AWSProfile:       os.Getenv("AWS_PROFILE"),
		CrossAccountRole: os.Getenv("FINOPS_CROSS_ACCOUNT_ROLE"),
		CURDatabase:      os.Getenv("FINOPS_CUR_DATABASE"),
		CURTable:         os.Getenv("FINOPS_CUR_TABLE"),
		CURWorkgroup:     envOr("FINOPS_CUR_WORKGROUP", "primary"),
		CUROutputBucket:  os.Getenv("FINOPS_CUR_OUTPUT_BUCKET"),
		KubeCostEndpoint: os.Getenv("FINOPS_KUBECOST_ENDPOINT"),
		WorkerQueues:     os.Getenv("FINOPS_WORKER_QUEUES"),
		APIPort:          envOr("FINOPS_API_PORT", "8080"),
		CORSOrigins:      parseCORSOrigins(os.Getenv("FINOPS_CORS_ORIGINS")),
		OIDCIssuer:       os.Getenv("FINOPS_OIDC_ISSUER"),
		OIDCAudience:     os.Getenv("FINOPS_OIDC_AUDIENCE"),
		LogLevel:         envOr("FINOPS_LOG_LEVEL", "info"),
		OTelEnabled:      os.Getenv("FINOPS_OTEL_ENABLED") == "true",
		AWSDocBinaryPath: envOr("FINOPS_AWSDOC_BINARY", "aws-doctor"),
		SweepAccounts:    os.Getenv("FINOPS_SWEEP_ACCOUNTS"),
		ShadowPythonPath: envOr("FINOPS_SHADOW_PYTHON", "python"),
		RateLimitCE:      envFloat("FINOPS_RATELIMIT_CE", 5),
		RateLimitAthena:  envFloat("FINOPS_RATELIMIT_ATHENA", 5),
		RateLimitCW:      envFloat("FINOPS_RATELIMIT_CW", 20),
		RateLimitSTS:     envFloat("FINOPS_RATELIMIT_STS", 10),
	}

	if cfg.Mode != ModeStub && cfg.Mode != ModeProduction {
		return Config{}, fmt.Errorf("config: invalid FINOPS_MODE %q (must be stub or production)", cfg.Mode)
	}

	if cfg.Mode == ModeProduction {
		if cfg.CURDatabase == "" {
			return Config{}, fmt.Errorf("config: FINOPS_CUR_DATABASE required in production mode")
		}
		if cfg.CURTable == "" {
			return Config{}, fmt.Errorf("config: FINOPS_CUR_TABLE required in production mode")
		}
		if cfg.CUROutputBucket == "" {
			return Config{}, fmt.Errorf("config: FINOPS_CUR_OUTPUT_BUCKET required in production mode")
		}
	}

	return cfg, nil
}

// OIDCEnabled returns true when OIDC authentication is configured.
func (c Config) OIDCEnabled() bool {
	return c.OIDCIssuer != ""
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envFloat(key string, fallback float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		slog.Warn("ignoring invalid env var", "key", key, "value", v, "fallback", fallback)
		return fallback
	}
	return f
}

func parseCORSOrigins(raw string) []string {
	if raw == "" {
		return []string{"*"}
	}
	var origins []string
	for _, o := range strings.Split(raw, ",") {
		if t := strings.TrimSpace(o); t != "" {
			origins = append(origins, t)
		}
	}
	if len(origins) == 0 {
		return []string{"*"}
	}
	return origins
}
