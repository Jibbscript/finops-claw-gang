// Package kubecost provides an HTTP client for the KubeCost allocation API,
// satisfying triage.KubeCostQuerier.
package kubecost

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// Client queries the KubeCost allocation API.
type Client struct {
	endpoint   string
	httpClient *http.Client
}

// New creates a KubeCost client with the given endpoint URL.
func New(endpoint string) *Client {
	return &Client{
		endpoint: endpoint,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// NewWithHTTPClient creates a KubeCost client with a custom HTTP client (for testing).
func NewWithHTTPClient(endpoint string, httpClient *http.Client) *Client {
	return &Client{
		endpoint:   endpoint,
		httpClient: httpClient,
	}
}

// Allocation queries the KubeCost /model/allocation endpoint.
func (c *Client) Allocation(window, aggregate string) (map[string]any, error) {
	u, err := url.Parse(c.endpoint)
	if err != nil {
		return nil, fmt.Errorf("kubecost: invalid endpoint: %w", err)
	}
	u.Path = "/model/allocation"
	q := u.Query()
	q.Set("window", window)
	q.Set("aggregate", aggregate)
	u.RawQuery = q.Encode()

	resp, err := c.httpClient.Get(u.String())
	if err != nil {
		return nil, fmt.Errorf("kubecost: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("kubecost: unexpected status %d", resp.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("kubecost: decode response: %w", err)
	}

	return result, nil
}
