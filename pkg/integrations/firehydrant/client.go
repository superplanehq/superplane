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

// Severity represents a FireHydrant severity level
type Severity struct {
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Type        string `json:"type"`
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

// Priority represents a FireHydrant priority level
type Priority struct {
	Slug        string `json:"slug"`
	Description string `json:"description"`
}

type PrioritiesResponse struct {
	Data []Priority `json:"data"`
}

func (c *Client) ListPriorities() ([]Priority, error) {
	url := fmt.Sprintf("%s/priorities", c.BaseURL)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var response PrioritiesResponse
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return response.Data, nil
}

// Service represents a FireHydrant service
type Service struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
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

// Team represents a FireHydrant team
type Team struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Slug        string `json:"slug"`
}

type TeamsResponse struct {
	Data []Team `json:"data"`
}

func (c *Client) ListTeams() ([]Team, error) {
	url := fmt.Sprintf("%s/teams", c.BaseURL)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var response TeamsResponse
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return response.Data, nil
}

// Incident represents a FireHydrant incident
type Incident struct {
	ID                    string      `json:"id"`
	Name                  string      `json:"name"`
	Number                int         `json:"number"`
	Description           string      `json:"description"`
	Summary               string      `json:"summary"`
	CustomerImpactSummary string      `json:"customer_impact_summary"`
	CurrentMilestone      string      `json:"current_milestone"`
	CreatedAt             string      `json:"created_at"`
	UpdatedAt             string      `json:"updated_at"`
	StartedAt             string      `json:"started_at"`
	Severity              *Severity   `json:"severity"`
	Priority              *Priority   `json:"priority"`
	Tags                  []string    `json:"tags"`
	Impacts               []any       `json:"impacts"`
	Milestones            []Milestone `json:"milestones"`
}

// Milestone represents a FireHydrant incident milestone
type Milestone struct {
	Type       string `json:"type"`
	OccurredAt string `json:"occurred_at"`
}

// CreateIncidentRequest represents the request body for creating an incident
type CreateIncidentRequest struct {
	Name                  string   `json:"name"`
	Summary               string   `json:"summary,omitempty"`
	Description           string   `json:"description,omitempty"`
	CustomerImpactSummary string   `json:"customer_impact_summary,omitempty"`
	Severity              string   `json:"severity,omitempty"`
	Priority              string   `json:"priority,omitempty"`
	TagList               []string `json:"tag_list,omitempty"`
	TeamIDs               []string `json:"team_ids,omitempty"`
}

func (c *Client) CreateIncident(request CreateIncidentRequest) (*Incident, error) {
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

// Webhook represents a FireHydrant webhook configuration
type Webhook struct {
	ID     string `json:"id"`
	URL    string `json:"url"`
	State  string `json:"state"`
	Secret string `json:"secret"`
}

type WebhooksResponse struct {
	Data []Webhook `json:"data"`
}

type CreateWebhookRequest struct {
	URL    string `json:"url"`
	State  string `json:"state,omitempty"`
	Secret string `json:"secret,omitempty"`
}

func (c *Client) CreateWebhook(webhookURL, secret string) (*Webhook, error) {
	request := CreateWebhookRequest{
		URL:    webhookURL,
		State:  "active",
		Secret: secret,
	}

	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	url := fmt.Sprintf("%s/webhooks", c.BaseURL)
	responseBody, err := c.execRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var webhook Webhook
	err = json.Unmarshal(responseBody, &webhook)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &webhook, nil
}

func (c *Client) DeleteWebhook(id string) error {
	url := fmt.Sprintf("%s/webhooks/%s", c.BaseURL, id)
	_, err := c.execRequest(http.MethodDelete, url, nil)
	return err
}

func (c *Client) ListWebhooks() ([]Webhook, error) {
	url := fmt.Sprintf("%s/webhooks", c.BaseURL)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var response WebhooksResponse
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return response.Data, nil
}
