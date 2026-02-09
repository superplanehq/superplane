package rootly

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/superplanehq/superplane/pkg/core"
)

const BaseURL = "https://api.rootly.com/v1"

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

	req.Header.Set("Content-Type", "application/vnd.api+json")
	req.Header.Set("Accept", "application/vnd.api+json")
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

// Service represents a Rootly service
type Service struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
}

// ServiceResponse represents the JSON:API response for a service
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

// ServicesResponse represents the JSON:API response for listing services
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

// Severity represents a Rootly severity level
type Severity struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
}

type SeverityData struct {
	ID         string             `json:"id"`
	Type       string             `json:"type"`
	Attributes SeverityAttributes `json:"attributes"`
}

type SeverityAttributes struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
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
			Name:        data.Attributes.Name,
			Slug:        data.Attributes.Slug,
			Description: data.Attributes.Description,
			Severity:    data.Attributes.Severity,
		})
	}

	return severities, nil
}

// Incident represents a Rootly incident
type Incident struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Summary     string `json:"summary"`
	Status      string `json:"status"`
	Severity    string `json:"severity"`
	StartedAt   string `json:"started_at"`
	ResolvedAt  string `json:"resolved_at"`
	MitigatedAt string `json:"mitigated_at"`
	URL         string `json:"url"`
}

type IncidentData struct {
	ID         string             `json:"id"`
	Type       string             `json:"type"`
	Attributes IncidentAttributes `json:"attributes"`
}

type IncidentAttributes struct {
	Title       string `json:"title"`
	Summary     string `json:"summary"`
	Status      string `json:"status"`
	Severity    string `json:"severity"`
	StartedAt   string `json:"started_at"`
	ResolvedAt  string `json:"resolved_at"`
	MitigatedAt string `json:"mitigated_at"`
	URL         string `json:"url"`
}

type IncidentResponse struct {
	Data IncidentData `json:"data"`
}

type IncidentsResponse struct {
	Data []IncidentData `json:"data"`
}

// IncidentEvent represents a Rootly incident event (timeline note).
type IncidentEvent struct {
	ID         string `json:"id"`
	Event      string `json:"event"`
	Visibility string `json:"visibility"`
	OccurredAt string `json:"occurred_at"`
	CreatedAt  string `json:"created_at"`
}

type IncidentEventData struct {
	ID         string                  `json:"id"`
	Type       string                  `json:"type"`
	Attributes IncidentEventAttributes `json:"attributes"`
}

type IncidentEventAttributes struct {
	Event      string `json:"event"`
	Visibility string `json:"visibility"`
	OccurredAt string `json:"occurred_at"`
	CreatedAt  string `json:"created_at"`
}

type IncidentEventResponse struct {
	Data IncidentEventData `json:"data"`
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
	Title    string `json:"title"`
	Summary  string `json:"summary,omitempty"`
	Severity string `json:"severity,omitempty"`
}

func (c *Client) CreateIncident(title, summary, severity string) (*Incident, error) {
	request := CreateIncidentRequest{
		Data: CreateIncidentData{
			Type: "incidents",
			Attributes: CreateIncidentAttributes{
				Title:    title,
				Summary:  summary,
				Severity: severity,
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
		Title:       response.Data.Attributes.Title,
		Summary:     response.Data.Attributes.Summary,
		Status:      response.Data.Attributes.Status,
		Severity:    response.Data.Attributes.Severity,
		StartedAt:   response.Data.Attributes.StartedAt,
		ResolvedAt:  response.Data.Attributes.ResolvedAt,
		MitigatedAt: response.Data.Attributes.MitigatedAt,
		URL:         response.Data.Attributes.URL,
	}, nil
}

// CreateIncidentEventRequest represents the request to create an incident event.
type CreateIncidentEventRequest struct {
	Data CreateIncidentEventData `json:"data"`
}

type CreateIncidentEventData struct {
	Type       string                        `json:"type"`
	Attributes CreateIncidentEventAttributes `json:"attributes"`
}

type CreateIncidentEventAttributes struct {
	Event      string `json:"event"`
	Visibility string `json:"visibility,omitempty"`
}

func (c *Client) CreateIncidentEvent(incidentID, event, visibility string) (*IncidentEvent, error) {
	request := CreateIncidentEventRequest{
		Data: CreateIncidentEventData{
			Type: "incident_events",
			Attributes: CreateIncidentEventAttributes{
				Event:      event,
				Visibility: visibility,
			},
		},
	}

	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	url := fmt.Sprintf("%s/incidents/%s/events", c.BaseURL, incidentID)
	responseBody, err := c.execRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response IncidentEventResponse
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &IncidentEvent{
		ID:         response.Data.ID,
		Event:      response.Data.Attributes.Event,
		Visibility: response.Data.Attributes.Visibility,
		OccurredAt: response.Data.Attributes.OccurredAt,
		CreatedAt:  response.Data.Attributes.CreatedAt,
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
		Title:       response.Data.Attributes.Title,
		Summary:     response.Data.Attributes.Summary,
		Status:      response.Data.Attributes.Status,
		Severity:    response.Data.Attributes.Severity,
		StartedAt:   response.Data.Attributes.StartedAt,
		ResolvedAt:  response.Data.Attributes.ResolvedAt,
		MitigatedAt: response.Data.Attributes.MitigatedAt,
		URL:         response.Data.Attributes.URL,
	}, nil
}

// GetIncidentDetailed fetches an incident with related resources and returns
// a flattened map containing attributes and resolved relationships.
func (c *Client) GetIncidentDetailed(id string) (map[string]any, error) {
	url := fmt.Sprintf("%s/incidents/%s?include=services,groups,events,action_items", c.BaseURL, id)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var raw map[string]any
	err = json.Unmarshal(responseBody, &raw)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	data, ok := raw["data"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected response format: missing data")
	}

	// Start with attributes as the base result
	attributes, _ := data["attributes"].(map[string]any)
	if attributes == nil {
		attributes = make(map[string]any)
	}

	result := make(map[string]any, len(attributes)+1)
	for k, v := range attributes {
		result[k] = v
	}

	// Set the top-level ID from data
	result["id"] = data["id"]

	// Build an index from the "included" array: "type:id" -> resolved object
	includedIndex := map[string]map[string]any{}
	if included, ok := raw["included"].([]any); ok {
		for _, item := range included {
			inc, ok := item.(map[string]any)
			if !ok {
				continue
			}
			incType, _ := inc["type"].(string)
			incID, _ := inc["id"].(string)
			if incType == "" || incID == "" {
				continue
			}
			resolved := map[string]any{"id": incID}
			if attrs, ok := inc["attributes"].(map[string]any); ok {
				for k, v := range attrs {
					resolved[k] = v
				}
			}
			includedIndex[incType+":"+incID] = resolved
		}
	}

	// Resolve relationships using the included index
	relationships, _ := data["relationships"].(map[string]any)
	for relName, relValue := range relationships {
		rel, ok := relValue.(map[string]any)
		if !ok {
			continue
		}

		relData := rel["data"]
		if relData == nil {
			result[relName] = nil
			continue
		}

		// Array relationship
		if arr, ok := relData.([]any); ok {
			resolved := make([]map[string]any, 0, len(arr))
			for _, ref := range arr {
				refMap, ok := ref.(map[string]any)
				if !ok {
					continue
				}
				refType, _ := refMap["type"].(string)
				refID, _ := refMap["id"].(string)
				key := refType + ":" + refID
				if obj, found := includedIndex[key]; found {
					resolved = append(resolved, obj)
				}
			}
			result[relName] = resolved
			continue
		}

		// Single relationship
		if refMap, ok := relData.(map[string]any); ok {
			refType, _ := refMap["type"].(string)
			refID, _ := refMap["id"].(string)
			key := refType + ":" + refID
			if obj, found := includedIndex[key]; found {
				result[relName] = obj
			}
		}
	}

	return result, nil
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
			Title:       data.Attributes.Title,
			Summary:     data.Attributes.Summary,
			Status:      data.Attributes.Status,
			Severity:    data.Attributes.Severity,
			StartedAt:   data.Attributes.StartedAt,
			ResolvedAt:  data.Attributes.ResolvedAt,
			MitigatedAt: data.Attributes.MitigatedAt,
			URL:         data.Attributes.URL,
		})
	}

	return incidents, nil
}

// WebhookEndpoint represents a Rootly webhook endpoint
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
	URL            string   `json:"url"`
	Secret         string   `json:"secret"`
	EventTypes     []string `json:"event_types"`
	Enabled        bool     `json:"enabled"`
	SigningEnabled bool     `json:"signing_enabled"`
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
	Name           string   `json:"name"`
	URL            string   `json:"url"`
	EventTypes     []string `json:"event_types"`
	Enabled        bool     `json:"enabled"`
	SigningEnabled bool     `json:"signing_enabled"`
}

func (c *Client) CreateWebhookEndpoint(url string, events []string) (*WebhookEndpoint, error) {
	request := CreateWebhookEndpointRequest{
		Data: CreateWebhookEndpointData{
			Type: "webhooks_endpoints",
			Attributes: CreateWebhookEndpointAttributes{
				Name:           "SuperPlane",
				URL:            url,
				EventTypes:     events,
				Enabled:        true,
				SigningEnabled: true,
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
