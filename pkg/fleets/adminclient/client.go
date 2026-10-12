package adminclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const maxResponseBytes = 4 << 20

type Client struct {
	baseURL    *url.URL
	token      string
	httpClient *http.Client
}

type HTTPError struct {
	StatusCode int
	Body       string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("admin API returned %d: %s", e.StatusCode, e.Body)
}

func IsStatus(err error, status int) bool {
	var httpErr *HTTPError
	return errors.As(err, &httpErr) && httpErr.StatusCode == status
}

func New(baseURL, installationAdminToken string, httpClient *http.Client) (*Client, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return nil, fmt.Errorf("parse SuperPlane URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("SuperPlane URL must use http or https")
	}
	if parsed.Host == "" {
		return nil, fmt.Errorf("SuperPlane URL must include a host")
	}
	token := strings.TrimSpace(installationAdminToken)
	if token == "" {
		return nil, fmt.Errorf("installation admin token is required")
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	return &Client{baseURL: parsed, token: token, httpClient: httpClient}, nil
}

func (c *Client) ListFleets(ctx context.Context) ([]Fleet, error) {
	var response struct {
		Fleets []Fleet `json:"fleets"`
	}
	if err := c.do(ctx, http.MethodGet, "/admin/api/installation/fleets", nil, nil, &response); err != nil {
		return nil, err
	}
	return response.Fleets, nil
}

func (c *Client) DescribeFleet(ctx context.Context, fleetID string) (Fleet, error) {
	var response struct {
		Fleet Fleet `json:"fleet"`
	}
	err := c.do(ctx, http.MethodGet, fleetPath(fleetID), nil, nil, &response)
	return response.Fleet, err
}

func (c *Client) GetFleetCapacity(
	ctx context.Context,
	fleetID, generation string,
	waitSeconds int,
) (Capacity, error) {
	query := url.Values{}
	if generation != "" {
		query.Set("generation", generation)
	}
	if waitSeconds > 0 {
		query.Set("waitSeconds", strconv.Itoa(waitSeconds))
	}
	var response Capacity
	err := c.do(ctx, http.MethodGet, fleetPath(fleetID)+"/capacity", query, nil, &response)
	return response, err
}

func (c *Client) CreateRunner(
	ctx context.Context,
	fleetID string,
	request CreateRunnerRequest,
) (CreateRunnerResponse, error) {
	var response CreateRunnerResponse
	var err error
	for attempt := 0; attempt < 2; attempt++ {
		err = c.do(ctx, http.MethodPost, fleetPath(fleetID)+"/runners", nil, request, &response)
		if err == nil || !retryable(err) || ctx.Err() != nil {
			return response, err
		}
	}
	return response, err
}

func (c *Client) ListRunners(
	ctx context.Context,
	fleetID string,
	states []string,
	limit int,
) ([]Runner, error) {
	query := listQuery(states, limit)
	var response struct {
		Runners []Runner `json:"runners"`
	}
	if err := c.do(ctx, http.MethodGet, fleetPath(fleetID)+"/runners", query, nil, &response); err != nil {
		return nil, err
	}
	return response.Runners, nil
}

func (c *Client) DescribeRunner(ctx context.Context, fleetID, runnerID string) (Runner, error) {
	var response struct {
		Runner Runner `json:"runner"`
	}
	path := fleetPath(fleetID) + "/runners/" + url.PathEscape(strings.TrimSpace(runnerID))
	err := c.do(ctx, http.MethodGet, path, nil, nil, &response)
	return response.Runner, err
}

func (c *Client) DeleteRunner(ctx context.Context, fleetID, runnerID string) (Runner, error) {
	var response struct {
		Runner Runner `json:"runner"`
	}
	path := fleetPath(fleetID) + "/runners/" + url.PathEscape(strings.TrimSpace(runnerID))
	err := c.do(ctx, http.MethodDelete, path, nil, nil, &response)
	return response.Runner, err
}

func (c *Client) do(
	ctx context.Context,
	method, path string,
	query url.Values,
	requestBody any,
	responseBody any,
) error {
	var body io.Reader
	if requestBody != nil {
		encoded, err := json.Marshal(requestBody)
		if err != nil {
			return fmt.Errorf("encode admin API request: %w", err)
		}
		body = bytes.NewReader(encoded)
	}

	endpoint := *c.baseURL
	endpoint.Path += path
	endpoint.RawQuery = query.Encode()
	request, err := http.NewRequestWithContext(ctx, method, endpoint.String(), body)
	if err != nil {
		return fmt.Errorf("build admin API request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "Bearer "+c.token)
	if requestBody != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("call admin API: %w", err)
	}
	defer response.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil {
		return fmt.Errorf("read admin API response: %w", err)
	}
	if len(raw) > maxResponseBytes {
		return fmt.Errorf("admin API response exceeds %d bytes", maxResponseBytes)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return &HTTPError{
			StatusCode: response.StatusCode,
			Body:       strings.TrimSpace(string(raw)),
		}
	}
	if responseBody == nil || len(bytes.TrimSpace(raw)) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, responseBody); err != nil {
		return fmt.Errorf("decode admin API response: %w", err)
	}
	return nil
}

func fleetPath(fleetID string) string {
	return "/admin/api/installation/fleets/" + url.PathEscape(strings.TrimSpace(fleetID))
}

func listQuery(states []string, limit int) url.Values {
	query := url.Values{}
	for _, state := range states {
		if state = strings.TrimSpace(state); state != "" {
			query.Add("states", state)
		}
	}
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}
	return query
}

func retryable(err error) bool {
	var httpErr *HTTPError
	return !errors.As(err, &httpErr) || httpErr.StatusCode >= http.StatusInternalServerError
}
