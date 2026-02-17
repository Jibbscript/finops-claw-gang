package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/finops-claw-gang/finops-go/internal/api"
	"github.com/finops-claw-gang/finops-go/internal/domain"
	"github.com/finops-claw-gang/finops-go/internal/temporal/activities"
	"github.com/finops-claw-gang/finops-go/internal/temporal/querier"
	"github.com/finops-claw-gang/finops-go/internal/temporal/workflows"
)

type stubQuerier struct {
	workflows []querier.WorkflowSummary
	state     *workflows.WorkflowResult
	desc      *querier.WorkflowDescription
	approval  string
	err       error
}

func (s *stubQuerier) ListWorkflows(_ context.Context, _ querier.ListOptions) ([]querier.WorkflowSummary, error) {
	return s.workflows, s.err
}

func (s *stubQuerier) GetWorkflowState(_ context.Context, _ string) (*workflows.WorkflowResult, error) {
	return s.state, s.err
}

func (s *stubQuerier) DescribeWorkflow(_ context.Context, _ string) (*querier.WorkflowDescription, error) {
	return s.desc, s.err
}

func (s *stubQuerier) SubmitApproval(_ context.Context, _ string, _ activities.ApprovalResponse) (string, error) {
	return s.approval, s.err
}

func newTestServer(t *testing.T, q querier.WorkflowQuerier) *httptest.Server {
	t.Helper()
	srv, err := api.New(q, []string{"*"}, api.OIDCConfig{})
	require.NoError(t, err)
	return httptest.NewServer(srv)
}

func TestHealth(t *testing.T) {
	ts := newTestServer(t, &stubQuerier{})
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/health")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "ok", body["status"])
}

func TestListWorkflows(t *testing.T) {
	q := &stubQuerier{
		workflows: []querier.WorkflowSummary{
			{WorkflowID: "wf-1", Status: "Running"},
			{WorkflowID: "wf-2", Status: "Completed"},
		},
	}
	ts := newTestServer(t, q)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/workflows")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var wfs []querier.WorkflowSummary
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&wfs))
	assert.Len(t, wfs, 2)
}

func TestGetWorkflow(t *testing.T) {
	state := domain.NewFinOpsState(domain.NewTenantContext("t1"))
	state.CurrentPhase = "triage"
	q := &stubQuerier{
		state: &workflows.WorkflowResult{State: state, Reason: workflows.ReasonCompleted},
	}
	ts := newTestServer(t, q)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/workflows/wf-1")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result workflows.WorkflowResult
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "triage", result.State.CurrentPhase)
}

func TestGetWorkflowUI(t *testing.T) {
	state := domain.NewFinOpsState(domain.NewTenantContext("t1"))
	state.CurrentPhase = "watcher"
	state.Anomaly = &domain.CostAnomaly{Service: "EC2", DeltaDollars: 500}
	q := &stubQuerier{
		state: &workflows.WorkflowResult{State: state},
	}
	ts := newTestServer(t, q)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/workflows/wf-1/ui")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var schema map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&schema))
	assert.Equal(t, "v1", schema["ui_schema_version"])
	assert.Equal(t, "watcher", schema["phase"])
}

func TestApprove(t *testing.T) {
	q := &stubQuerier{approval: "approved"}
	ts := newTestServer(t, q)
	defer ts.Close()

	body := `{"by": "ops-user"}`
	resp, err := http.Post(ts.URL+"/api/v1/workflows/wf-1/approve", "application/json", strings.NewReader(body))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "approved", result["result"])
}

func TestDeny(t *testing.T) {
	q := &stubQuerier{approval: "denied"}
	ts := newTestServer(t, q)
	defer ts.Close()

	body := `{"by": "ops-lead", "reason": "too risky"}`
	resp, err := http.Post(ts.URL+"/api/v1/workflows/wf-1/deny", "application/json", strings.NewReader(body))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestApprove_MissingBy(t *testing.T) {
	ts := newTestServer(t, &stubQuerier{})
	defer ts.Close()

	body := `{}`
	resp, err := http.Post(ts.URL+"/api/v1/workflows/wf-1/approve", "application/json", strings.NewReader(body))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestListWorkflows_Error(t *testing.T) {
	q := &stubQuerier{err: fmt.Errorf("temporal unavailable")}
	ts := newTestServer(t, q)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/workflows")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestRequestIDHeader(t *testing.T) {
	ts := newTestServer(t, &stubQuerier{})
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/health")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.NotEmpty(t, resp.Header.Get("X-Request-ID"))
}

func TestCORSHeaders(t *testing.T) {
	ts := newTestServer(t, &stubQuerier{})
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/health")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
}
