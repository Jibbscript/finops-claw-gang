package api

import (
	"net/http"

	"github.com/finops-claw-gang/finops-go/internal/agui"
	"github.com/finops-claw-gang/finops-go/internal/temporal/querier"
)

// Server is the HTTP API server for the FinOps Generative UI.
type Server struct {
	querier querier.WorkflowQuerier
	mux     *http.ServeMux
	handler http.Handler
}

// New creates a Server with the given querier and CORS origins.
func New(q querier.WorkflowQuerier, corsOrigins []string) *Server {
	s := &Server{querier: q, mux: http.NewServeMux()}
	s.routes()
	s.handler = requestID(logging(cors(corsOrigins, s.mux)))
	return s
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.handler.ServeHTTP(w, r)
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /api/v1/health", s.handleHealth)
	s.mux.HandleFunc("GET /api/v1/workflows", s.handleListWorkflows)
	s.mux.HandleFunc("GET /api/v1/workflows/{id}", s.handleGetWorkflow)
	s.mux.HandleFunc("GET /api/v1/workflows/{id}/ui", s.handleGetWorkflowUI)
	s.mux.HandleFunc("POST /api/v1/workflows/{id}/approve", s.handleApprove)
	s.mux.HandleFunc("POST /api/v1/workflows/{id}/deny", s.handleDeny)
	s.mux.HandleFunc("GET /api/v1/workflows/{id}/stream", agui.StreamHandler(s.querier, agui.DefaultConfig()))
}
