# Deployment Guide

## Binaries

| Binary | Purpose |
|--------|---------|
| `worker-finops` | Temporal worker processing anomaly lifecycles, detection, and execution |
| `api` | HTTP API server with AG-UI SSE streaming |
| `cli` | CLI for triggering workflows and inspecting state |
| `mcp-finops` | MCP server (stdio transport) for AI assistant integration |
| `shadow-compare` | Offline Go-vs-Python comparison tool for CI |

## Environment Variables

### Core

| Variable | Default | Description |
|----------|---------|-------------|
| `FINOPS_MODE` | `stub` | `stub` (fixtures) or `production` (real AWS) |
| `FIXTURES_DIR` | _(none)_ | Path to fixture files in stub mode |
| `TEMPORAL_ADDRESS` | `localhost:7233` | Temporal server gRPC address |

### AWS

| Variable | Default | Description |
|----------|---------|-------------|
| `AWS_REGION` | `us-east-1` | Default AWS region |
| `AWS_PROFILE` | _(none)_ | AWS credential profile |
| `FINOPS_CROSS_ACCOUNT_ROLE` | _(none)_ | IAM role ARN for cross-account access |
| `FINOPS_CUR_DATABASE` | _(none)_ | Athena database for CUR (required in production) |
| `FINOPS_CUR_TABLE` | _(none)_ | Athena table for CUR (required in production) |
| `FINOPS_CUR_WORKGROUP` | `primary` | Athena workgroup |
| `FINOPS_CUR_OUTPUT_BUCKET` | _(none)_ | S3 bucket for Athena results (required in production) |

### Worker

| Variable | Default | Description |
|----------|---------|-------------|
| `FINOPS_WORKER_QUEUES` | `anomaly` | Comma-separated queue list: `anomaly`, `detect`, `exec` |

### API Server

| Variable | Default | Description |
|----------|---------|-------------|
| `FINOPS_API_PORT` | `8080` | HTTP port |
| `FINOPS_CORS_ORIGINS` | `*` | Comma-separated allowed origins |

### Authentication (OIDC)

| Variable | Default | Description |
|----------|---------|-------------|
| `FINOPS_OIDC_ISSUER` | _(none)_ | OIDC issuer URL (e.g., `https://accounts.google.com`). Auth is disabled when empty. |
| `FINOPS_OIDC_AUDIENCE` | _(none)_ | Expected JWT audience claim |

### Observability

| Variable | Default | Description |
|----------|---------|-------------|
| `FINOPS_LOG_LEVEL` | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `FINOPS_OTEL_ENABLED` | `false` | Enable OpenTelemetry tracing and metrics |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | _(none)_ | OTLP HTTP endpoint (standard OTel env var) |

### Rate Limits

| Variable | Default | Description |
|----------|---------|-------------|
| `FINOPS_RATELIMIT_CE` | `5` | Cost Explorer requests/second |
| `FINOPS_RATELIMIT_ATHENA` | `5` | Athena requests/second |
| `FINOPS_RATELIMIT_CW` | `20` | CloudWatch requests/second |
| `FINOPS_RATELIMIT_STS` | `10` | STS requests/second |

### AWS Doctor

| Variable | Default | Description |
|----------|---------|-------------|
| `FINOPS_AWSDOC_BINARY` | `aws-doctor` | Path to aws-doctor CLI |
| `FINOPS_SWEEP_ACCOUNTS` | _(none)_ | Comma-separated AWS account IDs for sweep workflow |

### Shadow-Run

| Variable | Default | Description |
|----------|---------|-------------|
| `FINOPS_SHADOW_PYTHON` | `python` | Path to Python interpreter for shadow comparison |

### KubeCost

| Variable | Default | Description |
|----------|---------|-------------|
| `FINOPS_KUBECOST_ENDPOINT` | _(none)_ | KubeCost API base URL |

## Queue Topology

The worker supports three task queues with different concurrency profiles:

| Queue | Name | Purpose | Concurrency |
|-------|------|---------|-------------|
| Anomaly | `finops-anomaly` | Stateful lifecycle workflows, sweep workflows | 10 activities, 10 workflows |
| Detect | `finops-detect` | Read-heavy scheduled detection | 20 activities, 5 workflows |
| Exec | `finops-exec` | Write operations (restricted) | 3 activities, 1 workflow |

Start workers for all queues to prevent activity hangs:

```bash
# Single binary, one queue per instance
FINOPS_WORKER_QUEUES=anomaly worker-finops &
FINOPS_WORKER_QUEUES=detect  worker-finops &
FINOPS_WORKER_QUEUES=exec    worker-finops &
```

Or run a single worker on all queues:

```bash
FINOPS_WORKER_QUEUES=anomaly,detect,exec worker-finops
```

## Docker Compose (Local Development)

```bash
cd finops-go
docker compose up
```

This starts:
- PostgreSQL (port 5432)
- Temporal server (gRPC 7233, HTTP 7234)
- Temporal UI (port 8233)
- 3 worker instances (one per queue)
- API server (port 8080)

## IAM Setup (Production)

### Worker Role

The worker needs these IAM permissions:

- `ce:GetCostAndUsage`, `ce:GetReservationCoverage`, `ce:GetReservationUtilization`, `ce:GetSavingsPlansCoverage`, `ce:GetSavingsPlansUtilization`
- `athena:StartQueryExecution`, `athena:GetQueryExecution`, `athena:GetQueryResults`
- `s3:GetObject`, `s3:PutObject` (for Athena output bucket)
- `cloudwatch:GetMetricStatistics`
- `tag:GetResources`
- `codedeploy:ListDeployments`, `codedeploy:GetDeployment`
- `sts:AssumeRole` (for per-tenant cross-account access)

### Per-Tenant Cross-Account Access

When `TenantContext.IAMRoleARN` is set, the worker assumes the tenant's IAM role for each activity invocation. The role must trust the worker's account:

```json
{
  "Effect": "Allow",
  "Principal": {"AWS": "arn:aws:iam::WORKER_ACCOUNT:role/finops-worker"},
  "Action": "sts:AssumeRole"
}
```

Session credentials are cached and auto-refreshed 5 minutes before expiry.

## OIDC Configuration

Set `FINOPS_OIDC_ISSUER` to enable JWT authentication on all API endpoints except `/health`. The middleware:

1. Extracts `Bearer` token from the `Authorization` header
2. Verifies the token via OIDC discovery (JWKS auto-fetched and cached)
3. Validates the `aud` claim against `FINOPS_OIDC_AUDIENCE`
4. Injects `tenant_id` and `user_id` claims into the request context

Example (Auth0):

```bash
FINOPS_OIDC_ISSUER=https://your-tenant.auth0.com/
FINOPS_OIDC_AUDIENCE=https://finops-api.example.com
```
