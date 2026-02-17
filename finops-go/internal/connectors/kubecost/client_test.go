package kubecost

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllocation(t *testing.T) {
	fixture := map[string]any{
		"allocations": map[string]any{
			"default": map[string]any{
				"totalCost": 120.5,
				"delta":     15.3,
			},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/model/allocation", r.URL.Path)
		assert.Equal(t, "7d", r.URL.Query().Get("window"))
		assert.Equal(t, "namespace", r.URL.Query().Get("aggregate"))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(fixture)
	}))
	defer srv.Close()

	client := NewWithHTTPClient(srv.URL, srv.Client())
	result, err := client.Allocation("7d", "namespace")
	require.NoError(t, err)

	allocs, ok := result["allocations"].(map[string]any)
	require.True(t, ok)
	ns, ok := allocs["default"].(map[string]any)
	require.True(t, ok)
	assert.InDelta(t, 120.5, ns["totalCost"].(float64), 0.01)
}

func TestAllocation_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := NewWithHTTPClient(srv.URL, srv.Client())
	_, err := client.Allocation("7d", "namespace")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status 500")
}
