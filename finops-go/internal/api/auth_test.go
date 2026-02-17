package api

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testOIDCServer creates a fake OIDC issuer serving JWKS.
func testOIDCServer(t *testing.T, key *rsa.PrivateKey) *httptest.Server {
	t.Helper()
	jwk := jose.JSONWebKey{Key: &key.PublicKey, KeyID: "test-kid", Algorithm: "RS256", Use: "sig"}
	jwks := jose.JSONWebKeySet{Keys: []jose.JSONWebKey{jwk}}

	mux := http.NewServeMux()
	var issuerURL string

	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{
			"issuer":   issuerURL,
			"jwks_uri": issuerURL + "/jwks",
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(jwks); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	ts := httptest.NewServer(mux)
	issuerURL = ts.URL
	return ts
}

// signJWT creates a signed JWT with the given claims.
func signJWT(t *testing.T, key *rsa.PrivateKey, claims map[string]any) string {
	t.Helper()
	sig, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.RS256, Key: key},
		(&jose.SignerOptions{}).WithHeader("kid", "test-kid"),
	)
	require.NoError(t, err)

	raw, err := jwt.Signed(sig).Claims(claims).Serialize()
	require.NoError(t, err)
	return raw
}

type authTestEnv struct {
	key        *rsa.PrivateKey
	middleware func(http.Handler) http.Handler
	issuerURL  string
}

func setupAuthMiddleware(t *testing.T) authTestEnv {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	issuerServer := testOIDCServer(t, key)
	t.Cleanup(issuerServer.Close)

	oidcCtx := oidc.InsecureIssuerURLContext(t.Context(), issuerServer.URL)
	provider, err := oidc.NewProvider(oidcCtx, issuerServer.URL)
	require.NoError(t, err)

	middleware := oidcAuth(provider, "test-audience")
	return authTestEnv{key: key, middleware: middleware, issuerURL: issuerServer.URL}
}

// echoHandler returns a handler that writes tenant/user info from context.
func echoHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewEncoder(w).Encode(map[string]string{
			"tenant_id": TenantFromContext(r.Context()),
			"user_id":   UserFromContext(r.Context()),
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}

func TestOIDCAuth(t *testing.T) {
	env := setupAuthMiddleware(t)
	handler := env.middleware(echoHandler())

	now := time.Now()
	validToken := signJWT(t, env.key, map[string]any{
		"iss": env.issuerURL, "aud": "test-audience",
		"sub": "user-123", "tenant_id": "tenant-abc",
		"exp": now.Add(time.Hour).Unix(), "iat": now.Unix(),
	})
	expiredToken := signJWT(t, env.key, map[string]any{
		"iss": env.issuerURL, "aud": "test-audience", "sub": "user-123",
		"exp": now.Add(-time.Hour).Unix(), "iat": now.Add(-2 * time.Hour).Unix(),
	})
	wrongAudienceToken := signJWT(t, env.key, map[string]any{
		"iss": env.issuerURL, "aud": "wrong-audience", "sub": "user-123",
		"exp": now.Add(time.Hour).Unix(), "iat": now.Unix(),
	})

	tests := []struct {
		name       string
		path       string
		authHeader string
		wantStatus int
		wantBody   map[string]string // nil = don't check body
	}{
		{
			name:       "valid token",
			path:       "/api/v1/workflows",
			authHeader: "Bearer " + validToken,
			wantStatus: http.StatusOK,
			wantBody:   map[string]string{"tenant_id": "tenant-abc", "user_id": "user-123"},
		},
		{
			name:       "missing header",
			path:       "/api/v1/workflows",
			authHeader: "",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "expired token",
			path:       "/api/v1/workflows",
			authHeader: "Bearer " + expiredToken,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "wrong audience",
			path:       "/api/v1/workflows",
			authHeader: "Bearer " + wrongAudienceToken,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "health bypasses auth",
			path:       "/api/v1/health",
			authHeader: "",
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid format (Basic auth)",
			path:       "/api/v1/workflows",
			authHeader: "Basic dXNlcjpwYXNz",
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)

			if tt.wantBody != nil {
				var body map[string]string
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
				for k, v := range tt.wantBody {
					assert.Equal(t, v, body[k], "body[%s]", k)
				}
			}
		})
	}
}
