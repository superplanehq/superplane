package firehydrant

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/superplanehq/superplane/pkg/core"
)

const BaseURL = "https://api.firehydrant.io/v1"

type Client struct {
	Token   string
	BaseURL string
	http    core.HTTPContext
}

func NewClient(http core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	apiKey, err := ctx.GetConfig("apiKey")
	if err != nil {
		return nil, fmt.Errorf("error getting API key: %v", err)
	}

	return &Client{
		Token:   string(apiKey),
		BaseURL: BaseURL,
		http:    http,
	}, nil
}

func (c *Client) execRequest(method, url string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("error building request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token))

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %v", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading body: %v", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("request got %d code: %s", res.StatusCode, string(responseBody))
	}

	return responseBody, nil
}

// Service represents a FireHydrant service.
type Service struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
}

type ServicesResponse struct {
	Data []Service `json:"data"`
}

func (c *Client) ListServices() ([]Service, error) {
	url := fmt.Sprintf("%s/services", c.BaseURL)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var response ServicesResponse
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return response.Data, nil
}

// Severity represents a FireHydrant severity level.
type Severity struct {
	Slug        string `json:"slug"`
	Description string `json:"description"`
}

type SeveritiesResponse struct {
	Data []Severity `json:"data"`
}

func (c *Client) ListSeverities() ([]Severity, error) {
	url := fmt.Sprintf("%s/severities", c.BaseURL)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var response SeveritiesResponse
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return response.Data, nil
}

// Incident represents a FireHydrant incident.
type Incident struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Description      string `json:"description"`
	Severity         string `json:"severity"`
	Priority         string `json:"priority"`
	CurrentMilestone string `json:"current_milestone"`
	Active           bool   `json:"active"`
	CreatedAt        string `json:"created_at"`
	StartedAt        string `json:"started_at"`
	IncidentURL      string `json:"incident_url"`
	Private          bool   `json:"private"`
	Services         []any  `json:"services,omitempty"`
	Environments     []any  `json:"environments,omitempty"`
	TagList          []any  `json:"tag_list,omitempty"`
	Labels           any    `json:"labels,omitempty"`
}

// CreateIncidentRequest represents the request body for creating an incident.
type CreateIncidentRequest struct {
	Name        string `json:"name"`
	Severity    string `json:"severity,omitempty"`
	Description string `json:"description,omitempty"`
	Priority    string `json:"priority,omitempty"`
}

// CreateIncidentResponse represents the API response for creating an incident.
type CreateIncidentResponse Incident

func (c *Client) CreateIncident(name, severity, description string) (*Incident, error) {
	request := CreateIncidentRequest{
		Name:        name,
		Severity:    severity,
		Description: description,
	}

	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	url := fmt.Sprintf("%s/incidents", c.BaseURL)
	responseBody, err := c.execRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var incident Incident
	err = json.Unmarshal(responseBody, &incident)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &incident, nil
}

func (c *Client) GetIncident(id string) (*Incident, error) {
	url := fmt.Sprintf("%s/incidents/%s", c.BaseURL, id)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var incident Incident
	err = json.Unmarshal(responseBody, &incident)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &incident, nil
}
