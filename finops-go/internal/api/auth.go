package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
)

// OIDCConfig holds OIDC authentication settings.
type OIDCConfig struct {
	IssuerURL string
	Audience  string
	Enabled   bool
}

type contextKey string

const (
	ctxTenantID contextKey = "tenant_id"
	ctxUserID   contextKey = "user_id"
)

// TenantFromContext extracts the tenant ID from the request context.
func TenantFromContext(ctx context.Context) string {
	v, _ := ctx.Value(ctxTenantID).(string)
	return v
}

// UserFromContext extracts the user ID from the request context.
func UserFromContext(ctx context.Context) string {
	v, _ := ctx.Value(ctxUserID).(string)
	return v
}

// oidcAuth returns middleware that verifies JWT Bearer tokens using OIDC discovery.
// The /health endpoint bypasses authentication.
func oidcAuth(provider *oidc.Provider, audience string) func(http.Handler) http.Handler {
	verifier := provider.Verifier(&oidc.Config{ClientID: audience})

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Health check bypasses auth.
			if r.URL.Path == "/api/v1/health" {
				next.ServeHTTP(w, r)
				return
			}

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeError(w, http.StatusUnauthorized, "missing Authorization header")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				writeError(w, http.StatusUnauthorized, "invalid Authorization header format")
				return
			}

			token, err := verifier.Verify(r.Context(), parts[1])
			if err != nil {
				writeError(w, http.StatusUnauthorized, "invalid token: "+err.Error())
				return
			}

			// Extract claims for tenant and user context.
			var claims struct {
				TenantID string `json:"tenant_id"`
				Sub      string `json:"sub"`
				Email    string `json:"email"`
			}
			if err := token.Claims(&claims); err != nil {
				writeError(w, http.StatusUnauthorized, "invalid token claims")
				return
			}

			ctx := r.Context()
			if claims.TenantID != "" {
				ctx = context.WithValue(ctx, ctxTenantID, claims.TenantID)
			}
			userID := claims.Sub
			if userID == "" {
				userID = claims.Email
			}
			if userID != "" {
				ctx = context.WithValue(ctx, ctxUserID, userID)
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
