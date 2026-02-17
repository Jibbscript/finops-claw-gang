package shadow

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/finops-claw-gang/finops-go/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGoRunner_GoldenFixtures(t *testing.T) {
	goldenDir := testutil.GoldenDir()

	runner := &GoRunner{FixturesDir: goldenDir}
	out, err := runner.Run(context.Background(), "EC2", 750)
	require.NoError(t, err)

	var result map[string]any
	err = json.Unmarshal(out, &result)
	require.NoError(t, err)

	// Verify all three phases are present
	assert.Contains(t, result, "triage")
	assert.Contains(t, result, "analysis")
	assert.Contains(t, result, "approval")

	// Verify triage result has expected fields
	triageMap, ok := result["triage"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, triageMap, "category")
	assert.Contains(t, triageMap, "severity")
	assert.Contains(t, triageMap, "confidence")

	// Verify analysis result has expected fields
	analysisMap, ok := result["analysis"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, analysisMap, "root_cause_narrative")
	assert.Contains(t, analysisMap, "recommended_actions")

	// Verify approval result has expected fields
	approvalMap, ok := result["approval"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, approvalMap, "status")
	assert.Contains(t, approvalMap, "details")
}

func TestGoRunner_Deterministic(t *testing.T) {
	goldenDir := testutil.GoldenDir()
	runner := &GoRunner{FixturesDir: goldenDir}

	out1, err := runner.Run(context.Background(), "EC2", 750)
	require.NoError(t, err)
	out2, err := runner.Run(context.Background(), "EC2", 750)
	require.NoError(t, err)

	// The triage and policy outputs should be deterministic.
	// Analysis generates random action IDs, so we compare structurally.
	var r1, r2 map[string]any
	require.NoError(t, json.Unmarshal(out1, &r1))
	require.NoError(t, json.Unmarshal(out2, &r2))

	// Triage should be byte-identical (no random elements)
	t1, _ := json.Marshal(r1["triage"])
	t2, _ := json.Marshal(r2["triage"])
	assert.Equal(t, string(t1), string(t2), "triage should be deterministic")

	// Approval should be identical
	a1, _ := json.Marshal(r1["approval"])
	a2, _ := json.Marshal(r2["approval"])
	assert.Equal(t, string(a1), string(a2), "approval should be deterministic")
}
