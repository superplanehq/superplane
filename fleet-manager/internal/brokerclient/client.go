// Package brokerclient is a minimal HTTP client fleet-manager uses to talk to
// task-broker (fleet registration, task-counts during reconcile). Only
// fleet-manager calls broker; broker never initiates HTTP to fleet-manager.
package brokerclient

import (
	"bytes"
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
// authToken is the broker's control-plane bearer token.
func New(baseURL, authToken string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		//This prevents breaking when the base url is provided with a trailing / already
		baseURL:   strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		authToken: strings.TrimSpace(authToken),
	}
}

// CreateRunnerRegistration creates a one-time token for one runner VM.
func (c *Client) CreateRunnerRegistration(ctx context.Context, fleetID string) (api.CreateRunnerRegistrationResponse, error) {
	var out api.CreateRunnerRegistrationResponse
	if c == nil || c.baseURL == "" {
		return out, fmt.Errorf("brokerclient: client and base url required")
	}
	body, err := json.Marshal(api.CreateRunnerRegistrationRequest{FleetID: strings.TrimSpace(fleetID)})
	if err != nil {
		return out, err
	}
	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost, c.baseURL+"/v1/runners/registrations", bytes.NewReader(body),
	)
	if err != nil {
		return out, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.authToken)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return out, fmt.Errorf("brokerclient: registration http: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return out, fmt.Errorf("brokerclient: registration http %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return out, fmt.Errorf("brokerclient: decode registration: %w", err)
	}
	return out, nil
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

// DrainRunners asks task-broker to stop assigning new tasks to runner streams selected
// for EC2 termination. The response separates idle/drained runners from busy runners.
func (c *Client) DrainRunners(ctx context.Context, req api.DrainRunnersRequest) (api.DrainRunnersResponse, error) {
	var out api.DrainRunnersResponse
	if c == nil {
		return out, fmt.Errorf("brokerclient: nil client")
	}
	if c.baseURL == "" {
		return out, fmt.Errorf("brokerclient: empty base url")
	}
	req.FleetID = strings.TrimSpace(req.FleetID)
	if req.FleetID == "" {
		return out, fmt.Errorf("brokerclient: fleet_id required")
	}
	if len(req.RunnerIDs) == 0 {
		return out, fmt.Errorf("brokerclient: runner_ids required")
	}

	body, err := json.Marshal(req)
	if err != nil {
		return out, fmt.Errorf("brokerclient: encode drain request: %w", err)
	}
	endpoint := c.baseURL + "/v1/runners/drain"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return out, fmt.Errorf("brokerclient: build drain request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.authToken != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.authToken)
	}
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return out, fmt.Errorf("brokerclient: drain http: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return out, fmt.Errorf("brokerclient: drain http %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return out, fmt.Errorf("brokerclient: decode drain response: %w", err)
	}
	return out, nil
}

// RegisterFleet calls POST /v1/fleets to register or upsert a fleet. Non-2xx responses become errors.
func (c *Client) RegisterFleet(ctx context.Context, req api.RegisterFleetRequest) (api.FleetResponse, error) {
	var out api.FleetResponse
	if c == nil {
		return out, fmt.Errorf("brokerclient: nil client")
	}
	if c.baseURL == "" {
		return out, fmt.Errorf("brokerclient: empty base url")
	}
	req.ID = strings.TrimSpace(req.ID)
	if req.ID == "" {
		return out, fmt.Errorf("brokerclient: fleet id required")
	}

	body, err := json.Marshal(req)
	if err != nil {
		return out, fmt.Errorf("brokerclient: encode request: %w", err)
	}
	endpoint := c.baseURL + "/v1/fleets"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return out, fmt.Errorf("brokerclient: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.authToken != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.authToken)
	}
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return out, fmt.Errorf("brokerclient: http: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return out, fmt.Errorf("brokerclient: http %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return out, fmt.Errorf("brokerclient: decode: %w", err)
	}
	return out, nil
}
