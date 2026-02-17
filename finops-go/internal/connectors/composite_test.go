package connectors

import (
	"testing"

	"github.com/finops-claw-gang/finops-go/internal/temporal/activities"
)

// Compile-time interface satisfaction checks.
var (
	_ activities.CostDeps  = (*AWSCostClient)(nil)
	_ activities.InfraDeps = (*AWSInfraClient)(nil)
)

func TestInterfaceSatisfaction(t *testing.T) {
	// This test exists to ensure compile-time checks above are exercised.
	// If AWSCostClient or AWSInfraClient don't satisfy the interfaces,
	// this file won't compile.
	t.Log("composite adapters satisfy activities.CostDeps and activities.InfraDeps")
}
