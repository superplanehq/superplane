// Package brokerclient is a minimal HTTP client fleet-manager uses to pull
// task-counts from task-broker during reconcile. Only fleet-manager calls broker;
// broker never initiates HTTP to fleet-manager.
package brokerclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/superplane/runner/shared/api"
)

// Client is safe for concurrent use.
type Client struct {
	httpClient *http.Client
	baseURL    string
	authToken  string
}

// New returns a Client. baseURL is the task-broker URL (e.g. http://broker:8081),
// authToken is the broker's bearer token (same value as runner AUTH_TOKEN).
func New(baseURL, authToken string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		//This prevents breaking when the base url is provided with a trailing / already
		baseURL:   strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		authToken: strings.TrimSpace(authToken),
	}
}

// FleetTaskCounts calls GET /v1/fleets/{id}/task-counts. Non-2xx responses become errors.
func (c *Client) FleetTaskCounts(ctx context.Context, fleetID string) (api.FleetTaskCountsResponse, error) {
	var out api.FleetTaskCountsResponse
	if c == nil {
		return out, fmt.Errorf("brokerclient: nil client")
	}
	if c.baseURL == "" {
		return out, fmt.Errorf("brokerclient: empty base url")
	}
	fleetID = strings.TrimSpace(fleetID)
	if fleetID == "" {
		return out, fmt.Errorf("brokerclient: fleet_id required")
	}

	endpoint := c.baseURL + "/v1/fleets/" + url.PathEscape(fleetID) + "/task-counts"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return out, fmt.Errorf("brokerclient: build request: %w", err)
	}
	if c.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.authToken)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return out, fmt.Errorf("brokerclient: http: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return out, fmt.Errorf("brokerclient: http %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return out, fmt.Errorf("brokerclient: decode: %w", err)
	}
	return out, nil
}
