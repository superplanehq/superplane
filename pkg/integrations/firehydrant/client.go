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

// Service represents a FireHydrant service
// FireHydrant uses "functionalities" and "services" - we'll use services here
type Service struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
}

type ServiceResponse struct {
	Data ServiceData `json:"data"`
}

type ServiceData struct {
	ID         string            `json:"id"`
	Type       string            `json:"type"`
	Attributes ServiceAttributes `json:"attributes"`
}

type ServiceAttributes struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
}

// ServicesResponse represents the response for listing services
type ServicesResponse struct {
	Data []ServiceData `json:"data"`
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

	services := make([]Service, 0, len(response.Data))
	for _, data := range response.Data {
		services = append(services, Service{
			ID:          data.ID,
			Name:        data.Attributes.Name,
			Slug:        data.Attributes.Slug,
			Description: data.Attributes.Description,
		})
	}

	return services, nil
}

func (c *Client) GetService(id string) (*Service, error) {
	url := fmt.Sprintf("%s/services/%s", c.BaseURL, id)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var response ServiceResponse
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &Service{
		ID:          response.Data.ID,
		Name:        response.Data.Attributes.Name,
		Slug:        response.Data.Attributes.Slug,
		Description: response.Data.Attributes.Description,
	}, nil
}

// Severity represents a FireHydrant severity level
type Severity struct {
	ID          string `json:"id"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
}

type SeverityData struct {
	ID         string             `json:"id"`
	Type       string             `json:"type"`
	Attributes SeverityAttributes `json:"attributes"`
}

type SeverityAttributes struct {
	Slug        string `json:"slug"`
	Description string `json:"description"`
}

type SeveritiesResponse struct {
	Data []SeverityData `json:"data"`
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

	severities := make([]Severity, 0, len(response.Data))
	for _, data := range response.Data {
		severities = append(severities, Severity{
			ID:          data.ID,
			Slug:        data.Attributes.Slug,
			Description: data.Attributes.Description,
		})
	}

	return severities, nil
}

// Incident represents a FireHydrant incident
type Incident struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	StartedAt   string `json:"started_at"`
	ResolvedAt  string `json:"resolved_at"`
	ArchivedAt  string `json:"archived_at"`
	URL         string `json:"html_url"`
}

type IncidentData struct {
	ID         string             `json:"id"`
	Type       string             `json:"type"`
	Attributes IncidentAttributes `json:"attributes"`
}

type IncidentAttributes struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	StartedAt   string `json:"started_at"`
	ResolvedAt  string `json:"resolved_at"`
	ArchivedAt  string `json:"archived_at"`
	HTMLURL     string `json:"html_url"`
}

type IncidentResponse struct {
	Data IncidentData `json:"data"`
}

type IncidentsResponse struct {
	Data []IncidentData `json:"data"`
}

// CreateIncidentRequest represents the request to create an incident
type CreateIncidentRequest struct {
	Data CreateIncidentData `json:"data"`
}

type CreateIncidentData struct {
	Type       string                   `json:"type"`
	Attributes CreateIncidentAttributes `json:"attributes"`
}

type CreateIncidentAttributes struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Severity    string   `json:"severity,omitempty"`
	Services    []string `json:"services,omitempty"`
}

func (c *Client) CreateIncident(name, description, severity string, services []string) (*Incident, error) {
	request := CreateIncidentRequest{
		Data: CreateIncidentData{
			Type: "incident",
			Attributes: CreateIncidentAttributes{
				Name:        name,
				Description: description,
				Severity:    severity,
				Services:    services,
			},
		},
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

	var response IncidentResponse
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &Incident{
		ID:          response.Data.ID,
		Name:        response.Data.Attributes.Name,
		Description: response.Data.Attributes.Description,
		Severity:    response.Data.Attributes.Severity,
		Status:      response.Data.Attributes.Status,
		CreatedAt:   response.Data.Attributes.CreatedAt,
		StartedAt:   response.Data.Attributes.StartedAt,
		ResolvedAt:  response.Data.Attributes.ResolvedAt,
		ArchivedAt:  response.Data.Attributes.ArchivedAt,
		URL:         response.Data.Attributes.HTMLURL,
	}, nil
}

func (c *Client) GetIncident(id string) (*Incident, error) {
	url := fmt.Sprintf("%s/incidents/%s", c.BaseURL, id)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var response IncidentResponse
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &Incident{
		ID:          response.Data.ID,
		Name:        response.Data.Attributes.Name,
		Description: response.Data.Attributes.Description,
		Severity:    response.Data.Attributes.Severity,
		Status:      response.Data.Attributes.Status,
		CreatedAt:   response.Data.Attributes.CreatedAt,
		StartedAt:   response.Data.Attributes.StartedAt,
		ResolvedAt:  response.Data.Attributes.ResolvedAt,
		ArchivedAt:  response.Data.Attributes.ArchivedAt,
		URL:         response.Data.Attributes.HTMLURL,
	}, nil
}

func (c *Client) ListIncidents() ([]Incident, error) {
	url := fmt.Sprintf("%s/incidents", c.BaseURL)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var response IncidentsResponse
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	incidents := make([]Incident, 0, len(response.Data))
	for _, data := range response.Data {
		incidents = append(incidents, Incident{
			ID:          data.ID,
			Name:        data.Attributes.Name,
			Description: data.Attributes.Description,
			Severity:    data.Attributes.Severity,
			Status:      data.Attributes.Status,
			CreatedAt:   data.Attributes.CreatedAt,
			StartedAt:   data.Attributes.StartedAt,
			ResolvedAt:  data.Attributes.ResolvedAt,
			ArchivedAt:  data.Attributes.ArchivedAt,
			URL:         data.Attributes.HTMLURL,
		})
	}

	return incidents, nil
}

// WebhookEndpoint represents a FireHydrant webhook configuration
// FireHydrant uses outbound webhooks configured in the UI or via API
type WebhookEndpoint struct {
	ID     string `json:"id"`
	URL    string `json:"url"`
	Secret string `json:"secret"`
}

type WebhookEndpointData struct {
	ID         string                    `json:"id"`
	Type       string                    `json:"type"`
	Attributes WebhookEndpointAttributes `json:"attributes"`
}

type WebhookEndpointAttributes struct {
	URL    string `json:"url"`
	Secret string `json:"secret"`
	Active bool   `json:"active"`
}

type WebhookEndpointResponse struct {
	Data WebhookEndpointData `json:"data"`
}

type CreateWebhookEndpointRequest struct {
	Data CreateWebhookEndpointData `json:"data"`
}

type CreateWebhookEndpointData struct {
	Type       string                          `json:"type"`
	Attributes CreateWebhookEndpointAttributes `json:"attributes"`
}

type CreateWebhookEndpointAttributes struct {
	URL    string `json:"url"`
	Active bool   `json:"active"`
}

func (c *Client) CreateWebhookEndpoint(url string) (*WebhookEndpoint, error) {
	request := CreateWebhookEndpointRequest{
		Data: CreateWebhookEndpointData{
			Type: "webhook_endpoint",
			Attributes: CreateWebhookEndpointAttributes{
				URL:    url,
				Active: true,
			},
		},
	}

	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	apiURL := fmt.Sprintf("%s/webhooks/endpoints", c.BaseURL)
	responseBody, err := c.execRequest(http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response WebhookEndpointResponse
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &WebhookEndpoint{
		ID:     response.Data.ID,
		URL:    response.Data.Attributes.URL,
		Secret: response.Data.Attributes.Secret,
	}, nil
}

func (c *Client) DeleteWebhookEndpoint(id string) error {
	url := fmt.Sprintf("%s/webhooks/endpoints/%s", c.BaseURL, id)
	_, err := c.execRequest(http.MethodDelete, url, nil)
	return err
}
