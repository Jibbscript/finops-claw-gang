package observability

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Metrics holds OTel metric instruments for the FinOps system.
type Metrics struct {
	AnomalyCount    metric.Int64Counter
	ApprovalLatency metric.Float64Histogram
	SavingsRealized metric.Float64Counter
	ActivityCalls   metric.Int64Counter
}

// NewMetrics creates the FinOps metric instruments.
func NewMetrics() (*Metrics, error) {
	meter := otel.Meter("finops")

	anomalyCount, err := meter.Int64Counter("finops.anomaly.count",
		metric.WithDescription("Number of anomalies processed"),
	)
	if err != nil {
		return nil, err
	}

	approvalLatency, err := meter.Float64Histogram("finops.approval.latency_seconds",
		metric.WithDescription("Time from pending to approval decision"),
	)
	if err != nil {
		return nil, err
	}

	savingsRealized, err := meter.Float64Counter("finops.savings.realized_dollars",
		metric.WithDescription("Realized cost savings in dollars"),
	)
	if err != nil {
		return nil, err
	}

	activityCalls, err := meter.Int64Counter("finops.activity.calls",
		metric.WithDescription("Number of activity invocations"),
	)
	if err != nil {
		return nil, err
	}

	return &Metrics{
		AnomalyCount:    anomalyCount,
		ApprovalLatency: approvalLatency,
		SavingsRealized: savingsRealized,
		ActivityCalls:   activityCalls,
	}, nil
}

// RecordAnomalyProcessed records a processed anomaly.
func (m *Metrics) RecordAnomalyProcessed(ctx context.Context, category, severity string) {
	m.AnomalyCount.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("category", category),
			attribute.String("severity", severity),
		),
	)
}

// RecordApprovalLatency records the time from pending to decision.
func (m *Metrics) RecordApprovalLatency(ctx context.Context, d time.Duration) {
	m.ApprovalLatency.Record(ctx, d.Seconds())
}

// RecordActivity records an activity invocation.
func (m *Metrics) RecordActivity(ctx context.Context, name string) {
	m.ActivityCalls.Add(ctx, 1,
		metric.WithAttributes(attribute.String("activity", name)),
	)
}
