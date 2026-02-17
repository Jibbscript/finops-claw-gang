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
		json.NewEncoder(w).Encode(map[string]string{
			"issuer":   issuerURL,
			"jwks_uri": issuerURL + "/jwks",
		})
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jwks)
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
		json.NewEncoder(w).Encode(map[string]string{
			"tenant_id": TenantFromContext(r.Context()),
			"user_id":   UserFromContext(r.Context()),
		})
	})
}

func TestOIDCAuth_ValidToken(t *testing.T) {
	env := setupAuthMiddleware(t)
	handler := env.middleware(echoHandler())

	claims := map[string]any{
		"iss":       env.issuerURL,
		"aud":       "test-audience",
		"sub":       "user-123",
		"tenant_id": "tenant-abc",
		"exp":       time.Now().Add(time.Hour).Unix(),
		"iat":       time.Now().Unix(),
	}
	token := signJWT(t, env.key, claims)

	req := httptest.NewRequest("GET", "/api/v1/workflows", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "tenant-abc", body["tenant_id"])
	assert.Equal(t, "user-123", body["user_id"])
}

func TestOIDCAuth_MissingHeader(t *testing.T) {
	env := setupAuthMiddleware(t)
	handler := env.middleware(echoHandler())

	req := httptest.NewRequest("GET", "/api/v1/workflows", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestOIDCAuth_ExpiredToken(t *testing.T) {
	env := setupAuthMiddleware(t)
	handler := env.middleware(echoHandler())

	claims := map[string]any{
		"iss": env.issuerURL,
		"aud": "test-audience",
		"sub": "user-123",
		"exp": time.Now().Add(-time.Hour).Unix(),
		"iat": time.Now().Add(-2 * time.Hour).Unix(),
	}
	token := signJWT(t, env.key, claims)

	req := httptest.NewRequest("GET", "/api/v1/workflows", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestOIDCAuth_WrongAudience(t *testing.T) {
	env := setupAuthMiddleware(t)
	handler := env.middleware(echoHandler())

	claims := map[string]any{
		"iss": env.issuerURL,
		"aud": "wrong-audience",
		"sub": "user-123",
		"exp": time.Now().Add(time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}
	token := signJWT(t, env.key, claims)

	req := httptest.NewRequest("GET", "/api/v1/workflows", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestOIDCAuth_HealthBypassesAuth(t *testing.T) {
	env := setupAuthMiddleware(t)
	handler := env.middleware(echoHandler())

	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestOIDCAuth_InvalidFormat(t *testing.T) {
	env := setupAuthMiddleware(t)
	handler := env.middleware(echoHandler())

	req := httptest.NewRequest("GET", "/api/v1/workflows", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
