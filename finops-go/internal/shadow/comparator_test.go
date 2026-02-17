package shadow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompare_AllMatch(t *testing.T) {
	t.Parallel()
	data := []byte(`{"triage":{"category":"deploy_related"},"analysis":{"confidence":0.4},"approval":{"status":"auto_approved"}}`)

	result, err := Compare(data, data)
	require.NoError(t, err)
	assert.True(t, result.AllMatch)
	assert.Equal(t, "all phases match", result.Summary)
	assert.Len(t, result.Phases, 3)
	for _, p := range result.Phases {
		assert.True(t, p.Match, "phase %s should match", p.Phase)
		assert.Empty(t, p.DiffLines)
	}
}

func TestCompare_TriageDivergence(t *testing.T) {
	t.Parallel()
	goJSON := []byte(`{"triage":{"category":"deploy_related"},"analysis":{"confidence":0.4},"approval":{"status":"auto_approved"}}`)
	pyJSON := []byte(`{"triage":{"category":"expected_growth"},"analysis":{"confidence":0.4},"approval":{"status":"auto_approved"}}`)

	result, err := Compare(goJSON, pyJSON)
	require.NoError(t, err)
	assert.False(t, result.AllMatch)
	assert.Contains(t, result.Summary, "triage")
	assert.NotContains(t, result.Summary, "analysis")

	for _, p := range result.Phases {
		if p.Phase == "triage" {
			assert.False(t, p.Match)
			assert.NotEmpty(t, p.DiffLines)
		} else {
			assert.True(t, p.Match)
		}
	}
}

func TestCompare_MultiplePhasesDivergent(t *testing.T) {
	t.Parallel()
	goJSON := []byte(`{"triage":{"category":"a"},"analysis":{"confidence":0.4},"approval":{"status":"approved"}}`)
	pyJSON := []byte(`{"triage":{"category":"b"},"analysis":{"confidence":0.9},"approval":{"status":"denied"}}`)

	result, err := Compare(goJSON, pyJSON)
	require.NoError(t, err)
	assert.False(t, result.AllMatch)
	assert.Contains(t, result.Summary, "triage")
	assert.Contains(t, result.Summary, "analysis")
	assert.Contains(t, result.Summary, "approval")
}

func TestCompare_MissingPhase(t *testing.T) {
	t.Parallel()
	goJSON := []byte(`{"triage":{"category":"a"}}`)
	pyJSON := []byte(`{"triage":{"category":"a"},"analysis":{"confidence":0.5}}`)

	result, err := Compare(goJSON, pyJSON)
	require.NoError(t, err)
	assert.False(t, result.AllMatch)
	// analysis is present in py but null in go
	assert.Contains(t, result.Summary, "analysis")
}

func TestCompare_InvalidJSON(t *testing.T) {
	t.Parallel()
	_, err := Compare([]byte("not json"), []byte(`{}`))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parse Go output")

	_, err = Compare([]byte(`{}`), []byte("not json"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parse Python output")
}

func TestSimpleDiff(t *testing.T) {
	t.Parallel()
	a := "line1\nline2\nline3"
	b := "line1\nchanged\nline3"

	diff := simpleDiff(a, b)
	assert.Contains(t, diff, "line 2:")
	assert.Contains(t, diff, "go: line2")
	assert.Contains(t, diff, "py: changed")
	assert.NotContains(t, diff, "line 1:")
	assert.NotContains(t, diff, "line 3:")
}

func TestSimpleDiff_DifferentLengths(t *testing.T) {
	t.Parallel()
	a := "line1\nline2"
	b := "line1\nline2\nline3"

	diff := simpleDiff(a, b)
	assert.Contains(t, diff, "line 3:")
	assert.Contains(t, diff, "go: ")
	assert.Contains(t, diff, "py: line3")
}
