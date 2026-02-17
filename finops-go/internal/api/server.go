package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"

	"github.com/finops-claw-gang/finops-go/internal/agui"
	"github.com/finops-claw-gang/finops-go/internal/temporal/querier"
)

// Server is the HTTP API server for the FinOps Generative UI.
type Server struct {
	querier querier.WorkflowQuerier
	mux     *http.ServeMux
	handler http.Handler
}

// New creates a Server with the given querier, CORS origins, and optional OIDC config.
// Returns an error if OIDC is enabled but the provider cannot be reached.
func New(q querier.WorkflowQuerier, corsOrigins []string, oidcCfg OIDCConfig) (*Server, error) {
	s := &Server{querier: q, mux: http.NewServeMux()}
	s.routes()

	var handler http.Handler = s.mux
	handler = cors(corsOrigins, handler)
	handler = logging(handler)
	handler = requestID(handler)

	if oidcCfg.Enabled {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		provider, err := oidc.NewProvider(ctx, oidcCfg.IssuerURL)
		if err != nil {
			return nil, fmt.Errorf("oidc provider %s: %w", oidcCfg.IssuerURL, err)
		}
		handler = oidcAuth(provider, oidcCfg.Audience)(handler)
		slog.Info("OIDC authentication enabled", "issuer", oidcCfg.IssuerURL)
	}

	s.handler = handler
	return s, nil
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
