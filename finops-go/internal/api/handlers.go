package api

import (
	"encoding/json"
	"net/http"

	"github.com/finops-claw-gang/finops-go/internal/temporal/activities"
	"github.com/finops-claw-gang/finops-go/internal/temporal/querier"
	"github.com/finops-claw-gang/finops-go/internal/temporal/versioning"
	"github.com/finops-claw-gang/finops-go/internal/uischema"
)

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleListWorkflows(w http.ResponseWriter, r *http.Request) {
	opts := querier.ListOptions{
		TaskQueue: versioning.QueueAnomaly,
	}
	if status := r.URL.Query().Get("status"); status != "" {
		opts.StatusFilter = status
	}

	workflows, err := s.querier.ListWorkflows(r.Context(), opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, workflows)
}

func (s *Server) handleGetWorkflow(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "workflow id required")
		return
	}

	result, err := s.querier.GetWorkflowState(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleGetWorkflowUI(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "workflow id required")
		return
	}

	result, err := s.querier.GetWorkflowState(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	schema := uischema.Build(result.State)
	writeJSON(w, http.StatusOK, schema)
}

func (s *Server) handleApprove(w http.ResponseWriter, r *http.Request) {
	s.handleApprovalAction(w, r, true)
}

func (s *Server) handleDeny(w http.ResponseWriter, r *http.Request) {
	s.handleApprovalAction(w, r, false)
}

func (s *Server) handleApprovalAction(w http.ResponseWriter, r *http.Request, approved bool) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "workflow id required")
		return
	}

	var body struct {
		By     string `json:"by"`
		Reason string `json:"reason,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.By == "" {
		writeError(w, http.StatusBadRequest, "'by' field is required")
		return
	}

	resp := activities.ApprovalResponse{
		Approved: approved,
		By:       body.By,
		Reason:   body.Reason,
	}
	result, err := s.querier.SubmitApproval(r.Context(), id, resp)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"result": result})
}
