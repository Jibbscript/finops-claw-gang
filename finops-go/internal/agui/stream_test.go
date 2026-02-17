package agui_test

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/finops-claw-gang/finops-go/internal/agui"
	"github.com/finops-claw-gang/finops-go/internal/domain"
	"github.com/finops-claw-gang/finops-go/internal/temporal/activities"
	"github.com/finops-claw-gang/finops-go/internal/temporal/querier"
	"github.com/finops-claw-gang/finops-go/internal/temporal/workflows"
)

type stubQuerier struct {
	state     *workflows.WorkflowResult
	callCount int
	err       error
}

func (s *stubQuerier) ListWorkflows(_ context.Context, _ querier.ListOptions) ([]querier.WorkflowSummary, error) {
	return nil, nil
}

func (s *stubQuerier) GetWorkflowState(_ context.Context, _ string) (*workflows.WorkflowResult, error) {
	s.callCount++
	return s.state, s.err
}

func (s *stubQuerier) DescribeWorkflow(_ context.Context, _ string) (*querier.WorkflowDescription, error) {
	return nil, nil
}

func (s *stubQuerier) SubmitApproval(_ context.Context, _ string, _ activities.ApprovalResponse) (string, error) {
	return "", nil
}

func TestStreamHandler_CompletedWorkflow(t *testing.T) {
	state := domain.NewFinOpsState(domain.NewTenantContext("t1"))
	state.CurrentPhase = "completed"
	state.ShouldTerminate = true
	state.Anomaly = &domain.CostAnomaly{Service: "EC2", DeltaDollars: 500}

	q := &stubQuerier{
		state: &workflows.WorkflowResult{State: state, Reason: workflows.ReasonCompleted},
	}

	cfg := agui.StreamConfig{PollInterval: 50 * time.Millisecond, MaxDuration: 5 * time.Second}
	handler := agui.StreamHandler(q, cfg)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/workflows/{id}/stream", handler)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/workflows/wf-1/stream")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"))

	// Read events.
	events := parseSSE(t, resp)
	require.True(t, len(events) >= 3, "expected at least 3 events (RUN_STARTED, STATE_SNAPSHOT, RUN_FINISHED), got %d", len(events))
	assert.Equal(t, "RUN_STARTED", events[0].Type)
	assert.Equal(t, "STATE_SNAPSHOT", events[1].Type)
	assert.Equal(t, "RUN_FINISHED", events[2].Type)
}

func TestStreamHandler_ErrorQuerying(t *testing.T) {
	q := &stubQuerier{err: assert.AnError}

	cfg := agui.StreamConfig{PollInterval: 50 * time.Millisecond, MaxDuration: 5 * time.Second}
	handler := agui.StreamHandler(q, cfg)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/workflows/{id}/stream", handler)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/workflows/wf-1/stream")
	require.NoError(t, err)
	defer resp.Body.Close()

	events := parseSSE(t, resp)
	require.True(t, len(events) >= 2)
	assert.Equal(t, "RUN_STARTED", events[0].Type)
	assert.Equal(t, "RUN_ERROR", events[1].Type)
}

type sseEvent struct {
	Type string
	Data string
}

func parseSSE(t *testing.T, resp *http.Response) []sseEvent {
	t.Helper()
	var events []sseEvent
	scanner := bufio.NewScanner(resp.Body)
	var current sseEvent
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event: ") {
			current.Type = strings.TrimPrefix(line, "event: ")
		} else if strings.HasPrefix(line, "data: ") {
			current.Data = strings.TrimPrefix(line, "data: ")
		} else if line == "" && current.Type != "" {
			events = append(events, current)
			current = sseEvent{}
		}
	}
	return events
}

func TestEventSerialization(t *testing.T) {
	event := agui.Event{
		Type:       agui.EventRunStarted,
		Timestamp:  time.Date(2026, 2, 17, 0, 0, 0, 0, time.UTC),
		WorkflowID: "wf-test",
	}

	data, err := json.Marshal(event)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, "RUN_STARTED", decoded["type"])
	assert.Equal(t, "wf-test", decoded["workflow_id"])
}
