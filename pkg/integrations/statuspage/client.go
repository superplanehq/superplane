package statuspage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/superplanehq/superplane/pkg/core"
)

type Client struct {
	Token   string
	PageID  string
	BaseURL string
	http    core.HTTPContext
}

func NewClient(httpCtx core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	apiKey, err := ctx.GetConfig("apiKey")
	if err != nil {
		return nil, fmt.Errorf("error finding apiKey: %v", err)
	}

	pageID, err := ctx.GetConfig("pageId")
	if err != nil {
		return nil, fmt.Errorf("error finding pageId: %v", err)
	}

	return &Client{
		Token:   string(apiKey),
		PageID:  string(pageID),
		BaseURL: "https://api.statuspage.io/v1",
		http:    httpCtx,
	}, nil
}

func (c *Client) execRequest(method, url string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("error building request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("OAuth %s", c.Token))

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

// Page represents a Statuspage page
type Page struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// GetPage retrieves the current page to verify credentials
func (c *Client) GetPage() (*Page, error) {
	url := fmt.Sprintf("%s/pages/%s", c.BaseURL, c.PageID)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var page Page
	err = json.Unmarshal(responseBody, &page)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &page, nil
}

// Component represents a Statuspage component
type Component struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ListComponents retrieves all components for the page
func (c *Client) ListComponents() ([]Component, error) {
	url := fmt.Sprintf("%s/pages/%s/components", c.BaseURL, c.PageID)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var components []Component
	err = json.Unmarshal(responseBody, &components)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return components, nil
}

// CreateIncidentRequest represents the request to create an incident
type CreateIncidentRequest struct {
	Incident IncidentPayload `json:"incident"`
}

// IncidentPayload holds fields for creating/updating an incident
type IncidentPayload struct {
	Name               string   `json:"name"`
	Status             string   `json:"status,omitempty"`
	ImpactOverride     string   `json:"impact_override,omitempty"`
	Body               string   `json:"body,omitempty"`
	ComponentIDs       []string `json:"component_ids,omitempty"`
	ComponentStatus    string   `json:"component_status,omitempty"`
	DeliverNotifications *bool  `json:"deliver_notifications,omitempty"`
}

// CreateIncident creates a new incident on the page
func (c *Client) CreateIncident(payload IncidentPayload) (any, error) {
	request := CreateIncidentRequest{
		Incident: payload,
	}

	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	url := fmt.Sprintf("%s/pages/%s/incidents", c.BaseURL, c.PageID)
	responseBody, err := c.execRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response map[string]any
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return response, nil
}

// UpdateIncidentRequest represents the request to update an incident
type UpdateIncidentRequest struct {
	Incident UpdateIncidentPayload `json:"incident"`
}

// UpdateIncidentPayload holds fields for updating an incident
type UpdateIncidentPayload struct {
	Name               string   `json:"name,omitempty"`
	Status             string   `json:"status,omitempty"`
	ImpactOverride     string   `json:"impact_override,omitempty"`
	Body               string   `json:"body,omitempty"`
	ComponentIDs       []string `json:"component_ids,omitempty"`
	ComponentStatus    string   `json:"component_status,omitempty"`
	DeliverNotifications *bool  `json:"deliver_notifications,omitempty"`
}

// UpdateIncident updates an existing incident on the page
func (c *Client) UpdateIncident(incidentID string, payload UpdateIncidentPayload) (any, error) {
	request := UpdateIncidentRequest{
		Incident: payload,
	}

	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	url := fmt.Sprintf("%s/pages/%s/incidents/%s", c.BaseURL, c.PageID, incidentID)
	responseBody, err := c.execRequest(http.MethodPatch, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response map[string]any
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return response, nil
}

// GetIncident retrieves a specific incident by ID
func (c *Client) GetIncident(incidentID string) (any, error) {
	url := fmt.Sprintf("%s/pages/%s/incidents/%s", c.BaseURL, c.PageID, incidentID)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var response map[string]any
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return response, nil
}
