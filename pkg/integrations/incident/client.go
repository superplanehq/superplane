package incident

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/superplanehq/superplane/pkg/core"
)

const BaseURL = "https://api.incident.io"

// Severity represents an incident.io severity level.
type Severity struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Rank        int    `json:"rank"`
}

// SeverityListResponse is the API response for listing severities.
type SeverityListResponse struct {
	Severities []Severity `json:"severities"`
}

// IncidentV2 represents an incident from incident.io API v2.
type IncidentV2 struct {
	ID             string             `json:"id"`
	Name           string             `json:"name"`
	Summary        string             `json:"summary"`
	Reference      string             `json:"reference"`
	Permalink      string             `json:"permalink"`
	Visibility     string             `json:"visibility"`
	CreatedAt      string             `json:"created_at"`
	UpdatedAt      string             `json:"updated_at"`
	Severity       *SeverityRef       `json:"severity,omitempty"`
	IncidentStatus *IncidentStatusRef `json:"incident_status,omitempty"`
}

// SeverityRef is a severity reference in incident payloads.
type SeverityRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// IncidentStatusRef is an incident status reference.
type IncidentStatusRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Client struct {
	Token   string
	BaseURL string
	http    core.HTTPContext
}

func NewClient(http core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	apiKey, err := ctx.GetConfig("apiKey")
	if err != nil {
		return nil, fmt.Errorf("error getting API key: %w", err)
	}

	return &Client{
		Token:   string(apiKey),
		BaseURL: BaseURL,
		http:    http,
	}, nil
}

func (c *Client) execRequest(method, path string, body io.Reader) ([]byte, error) {
	url := c.BaseURL + path
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("error building request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Token)

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %w", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading body: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("request got %d: %s", res.StatusCode, string(responseBody))
	}

	return responseBody, nil
}

// ListSeverities returns severities for the organization (used for Create Incident severity picker).
func (c *Client) ListSeverities() ([]Severity, error) {
	// incident.io severities are under v1
	responseBody, err := c.execRequest(http.MethodGet, "/v1/severities", nil)
	if err != nil {
		return nil, err
	}

	var response SeverityListResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing severities response: %w", err)
	}

	return response.Severities, nil
}

// CreateIncidentRequest is the request body for POST /v2/incidents.
type CreateIncidentRequest struct {
	Name           string `json:"name"`
	IdempotencyKey string `json:"idempotency_key"`
	SeverityID     string `json:"severity_id,omitempty"`
	Visibility     string `json:"visibility,omitempty"` // "public" or "private"
	Summary        string `json:"summary,omitempty"`
}

// CreateIncidentResponse is the response from POST /v2/incidents.
type CreateIncidentResponse struct {
	Incident IncidentV2 `json:"incident"`
}

// CreateIncident creates a new incident via POST /v2/incidents.
func (c *Client) CreateIncident(name, idempotencyKey, severityID, visibility, summary string) (*IncidentV2, error) {
	if visibility == "" {
		visibility = "public"
	}

	req := CreateIncidentRequest{
		Name:           name,
		IdempotencyKey: idempotencyKey,
		SeverityID:     severityID,
		Visibility:     visibility,
		Summary:        summary,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %w", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, "/v2/incidents", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response CreateIncidentResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing create incident response: %w", err)
	}

	return &response.Incident, nil
}
