package pagerduty

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/superplanehq/superplane/pkg/core"
)

type Client struct {
	AccessToken string
	BaseURL     string
}

func NewClient(ctx core.AppInstallationContext) (*Client, error) {
	// Get access token from secrets
	secrets, err := ctx.GetSecrets()
	if err != nil {
		return nil, fmt.Errorf("failed to get secrets: %v", err)
	}

	var accessToken string
	for _, secret := range secrets {
		if secret.Name == "accessToken" {
			accessToken = string(secret.Value)
			break
		}
	}

	if accessToken == "" {
		return nil, fmt.Errorf("access token not found")
	}

	return &Client{
		AccessToken: accessToken,
		BaseURL:     "https://api.pagerduty.com",
	}, nil
}

func (c *Client) execRequest(method, url string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("error building request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.pagerduty+json;version=2")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.AccessToken))

	res, err := http.DefaultClient.Do(req)
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

// User represents a PagerDuty user
type User struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (c *Client) GetCurrentUser() (*User, error) {
	url := fmt.Sprintf("%s/users/me", c.BaseURL)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		User User `json:"user"`
	}
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &response.User, nil
}

// CreateIncidentRequest represents the request to create an incident
type CreateIncidentRequest struct {
	Incident IncidentPayload `json:"incident"`
}

// IncidentPayload represents the incident data
type IncidentPayload struct {
	Type        string           `json:"type"` // "incident"
	Title       string           `json:"title"`
	Service     ServiceReference `json:"service"`
	Urgency     string           `json:"urgency,omitempty"` // "high" or "low"
	Body        *IncidentBody    `json:"body,omitempty"`
	Assignments []Assignment     `json:"assignments,omitempty"`
}

// ServiceReference represents a reference to a service
type ServiceReference struct {
	ID   string `json:"id"`
	Type string `json:"type"` // "service_reference"
}

// IncidentBody represents the incident description
type IncidentBody struct {
	Type    string `json:"type"` // "incident_body"
	Details string `json:"details"`
}

// Assignment represents an assignment of an incident to a user
type Assignment struct {
	Assignee Assignee `json:"assignee"`
}

// Assignee represents a user assigned to an incident
type Assignee struct {
	ID   string `json:"id"`
	Type string `json:"type"` // "user_reference"
}

// Incident represents a PagerDuty incident
type Incident struct {
	ID             string `json:"id"`
	Type           string `json:"type"`
	IncidentNumber int    `json:"incident_number"`
	Title          string `json:"title"`
	Status         string `json:"status"`
	Urgency        string `json:"urgency"`
	HTMLURL        string `json:"html_url"`
	CreatedAt      string `json:"created_at"`
}

func (c *Client) CreateIncident(params *CreateIncidentRequest) (*Incident, error) {
	url := fmt.Sprintf("%s/incidents", c.BaseURL)
	body, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response struct {
		Incident Incident `json:"incident"`
	}
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &response.Incident, nil
}

// WebhookSubscription represents a PagerDuty webhook subscription
type WebhookSubscription struct {
	ID     string   `json:"id"`
	Type   string   `json:"type"` // "webhook_subscription"
	Events []string `json:"events"`
}

// CreateWebhookSubscription creates a new webhook subscription
// filterType can be "account_reference", "service_reference", or "team_reference"
// filterID is the service or team ID (empty for account_reference)
func (c *Client) CreateWebhookSubscription(url string, events []string, filterType, filterID string) (*WebhookSubscription, error) {
	apiURL := fmt.Sprintf("%s/webhook_subscriptions", c.BaseURL)

	// Build filter based on type
	var filter map[string]any
	if filterType == "service_reference" {
		filter = map[string]any{
			"type": "service_reference",
			"id":   filterID,
		}
	} else if filterType == "team_reference" {
		filter = map[string]any{
			"type": "team_reference",
			"id":   filterID,
		}
	} else {
		// Default to account_reference
		filter = map[string]any{
			"type": "account_reference",
		}
	}

	subscription := map[string]any{
		"webhook_subscription": map[string]any{
			"type": "webhook_subscription",
			"delivery_method": map[string]any{
				"type": "http_delivery_method",
				"url":  url,
			},
			"events": events,
			"filter": filter,
		},
	}

	body, err := json.Marshal(subscription)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response struct {
		WebhookSubscription WebhookSubscription `json:"webhook_subscription"`
	}
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &response.WebhookSubscription, nil
}

func (c *Client) DeleteWebhookSubscription(id string) error {
	url := fmt.Sprintf("%s/webhook_subscriptions/%s", c.BaseURL, id)
	_, err := c.execRequest(http.MethodDelete, url, nil)
	return err
}
