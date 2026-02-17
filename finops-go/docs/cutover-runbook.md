# Python-to-Go Cutover Runbook

Step-by-step procedure for transitioning from the Python LangGraph pipeline to the Go Temporal pipeline.

## Prerequisites

- [ ] All Phase 6 tests pass: `go test ./... -count=1 -race`
- [ ] `go vet ./...` clean
- [ ] Docker Compose stack starts cleanly
- [ ] OIDC provider configured (if applicable)
- [ ] IAM roles provisioned for all target accounts

## Step 1: Shadow-Run Validation

Run the shadow comparison tool in CI to confirm Go and Python produce identical outputs on the golden fixture set.

```bash
shadow-compare \
  --fixtures-dir tests/golden \
  --python-path /path/to/venv/bin/python \
  --service EC2 \
  --delta 750
```

**Gate**: Exit code must be 0 (all phases match). If divergent, inspect the JSON diff output and fix before proceeding.

To run Go-only (no Python dependency):

```bash
shadow-compare --fixtures-dir tests/golden --go-only
```

## Step 2: Deploy Workers in Stub Mode

Deploy the Go workers alongside the Python stack, pointing at the same Temporal cluster but with `FINOPS_MODE=stub`.

```bash
FINOPS_MODE=stub
FINOPS_WORKER_QUEUES=anomaly,detect,exec
TEMPORAL_ADDRESS=temporal.internal:7233
```

Verify:
- [ ] Workers register on all 3 queues (check Temporal UI)
- [ ] Health endpoint returns 200: `curl http://api:8080/health`

## Step 3: Production Dry Run

Switch one worker to `FINOPS_MODE=production` with read-only IAM permissions (no execution queue). This validates real AWS data flows without executing any actions.

```bash
FINOPS_MODE=production
FINOPS_WORKER_QUEUES=anomaly,detect
FINOPS_CUR_DATABASE=your_cur_db
FINOPS_CUR_TABLE=your_cur_table
FINOPS_CUR_OUTPUT_BUCKET=s3://your-athena-results/
```

Verify:
- [ ] Triage classifies anomalies correctly (compare with Python output)
- [ ] Analysis produces sensible recommendations
- [ ] No errors in structured logs
- [ ] Rate limiting is functioning (check logs for rate limit waits)

## Step 4: Enable Execution

Add the exec queue worker with appropriate IAM permissions.

```bash
FINOPS_WORKER_QUEUES=exec
```

Verify:
- [ ] Executor safety gate blocks critical actions
- [ ] Tag-based protection works (`do-not-modify`, `manual-only`)
- [ ] Auto-approved low-risk actions execute successfully
- [ ] Pending actions wait for human approval via Update handler

## Step 5: Enable API + OIDC

Deploy the API server with OIDC authentication.

```bash
FINOPS_OIDC_ISSUER=https://your-idp.example.com/
FINOPS_OIDC_AUDIENCE=https://finops-api.example.com
FINOPS_API_PORT=8080
```

Verify:
- [ ] `/health` returns 200 without auth
- [ ] All other endpoints return 401 without valid token
- [ ] Valid JWT grants access and tenant context is correct

## Step 6: Enable Observability

```bash
FINOPS_OTEL_ENABLED=true
OTEL_EXPORTER_OTLP_ENDPOINT=https://otel-collector.internal:4318
FINOPS_LOG_LEVEL=info
```

Verify:
- [ ] Traces appear in your observability backend
- [ ] Metrics: `finops.anomaly.count`, `finops.approval.latency`, `finops.savings.realized`
- [ ] Structured JSON logs with correlation IDs

## Step 7: Decommission Python

Once the Go pipeline has been stable for a sufficient observation period:

1. Stop the Python LangGraph workers
2. Remove the Python cron triggers
3. Archive the `finops-deterministic/` directory
4. Update DNS/load balancer to point exclusively at the Go API

## Rollback Procedure

At any step, if issues arise:

1. **Stop Go workers**: Scale to 0 replicas
2. **Restart Python workers**: They use the same Temporal cluster and can pick up from the last checkpoint
3. **Investigate**: Check structured logs, traces, and the Temporal UI for workflow failures
4. **Fix and re-deploy**: Address the issue, run shadow comparison again, and restart from the failed step

Note: Temporal workflows are durable. In-flight workflows will resume when workers come back online. No data is lost during worker restarts.
