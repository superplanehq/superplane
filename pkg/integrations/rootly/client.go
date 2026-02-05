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

// Incident represents a Rootly incident (basic fields)
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

// IncidentUser represents a user associated with an incident
type IncidentUser struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// IncidentService represents a service associated with an incident
type IncidentService struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// IncidentGroup represents a group associated with an incident
type IncidentGroup struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// IncidentEvent represents a timeline event for an incident
type IncidentEvent struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	Summary   string `json:"summary"`
	CreatedAt string `json:"created_at"`
}

// IncidentActionItem represents an action item for an incident
type IncidentActionItem struct {
	ID          string `json:"id"`
	Summary     string `json:"summary"`
	Description string `json:"description"`
	Status      string `json:"status"`
	Priority    string `json:"priority"`
	DueAt       string `json:"due_at"`
}

// IncidentDetailed represents a Rootly incident with all related data
type IncidentDetailed struct {
	ID           string               `json:"id"`
	SequentialID int                  `json:"sequential_id"`
	Title        string               `json:"title"`
	Slug         string               `json:"slug"`
	Status       string               `json:"status"`
	Summary      string               `json:"summary"`
	Severity     string               `json:"severity"`
	URL          string               `json:"url"`
	StartedAt    string               `json:"started_at"`
	MitigatedAt  string               `json:"mitigated_at"`
	ResolvedAt   string               `json:"resolved_at"`
	User         *IncidentUser        `json:"user"`
	StartedBy    *IncidentUser        `json:"started_by"`
	Services     []IncidentService    `json:"services"`
	Groups       []IncidentGroup      `json:"groups"`
	Events       []IncidentEvent      `json:"events"`
	ActionItems  []IncidentActionItem `json:"action_items"`
}

type IncidentData struct {
	ID            string                 `json:"id"`
	Type          string                 `json:"type"`
	Attributes    IncidentAttributes     `json:"attributes"`
	Relationships *IncidentRelationships `json:"relationships,omitempty"`
}

type IncidentAttributes struct {
	Title        string `json:"title"`
	Summary      string `json:"summary"`
	Status       string `json:"status"`
	Severity     string `json:"severity"`
	StartedAt    string `json:"started_at"`
	ResolvedAt   string `json:"resolved_at"`
	MitigatedAt  string `json:"mitigated_at"`
	URL          string `json:"url"`
	SequentialID int    `json:"sequential_id"`
	Slug         string `json:"slug"`
}

type IncidentRelationships struct {
	User      *RelationshipData `json:"user,omitempty"`
	StartedBy *RelationshipData `json:"started_by,omitempty"`
	Services  *RelationshipList `json:"services,omitempty"`
	Groups    *RelationshipList `json:"groups,omitempty"`
}

type RelationshipData struct {
	Data *ResourceIdentifier `json:"data,omitempty"`
}

type RelationshipList struct {
	Data []ResourceIdentifier `json:"data,omitempty"`
}

type ResourceIdentifier struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

// IncidentDetailedResponse represents the full JSON:API response with included data
type IncidentDetailedResponse struct {
	Data     IncidentData  `json:"data"`
	Included []IncludedRef `json:"included,omitempty"`
}

// IncludedRef represents an included resource in a JSON:API response
type IncludedRef struct {
	ID         string         `json:"id"`
	Type       string         `json:"type"`
	Attributes map[string]any `json:"attributes"`
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

// GetIncidentWithDetails retrieves an incident with all related data
func (c *Client) GetIncidentWithDetails(id string) (*IncidentDetailed, error) {
	// Request incident with included relationships
	url := fmt.Sprintf("%s/incidents/%s?include=user,started_by,services,groups", c.BaseURL, id)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var response IncidentDetailedResponse
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing incident response: %v", err)
	}

	incident := &IncidentDetailed{
		ID:           response.Data.ID,
		SequentialID: response.Data.Attributes.SequentialID,
		Title:        response.Data.Attributes.Title,
		Slug:         response.Data.Attributes.Slug,
		Status:       response.Data.Attributes.Status,
		Summary:      response.Data.Attributes.Summary,
		Severity:     response.Data.Attributes.Severity,
		URL:          response.Data.Attributes.URL,
		StartedAt:    response.Data.Attributes.StartedAt,
		MitigatedAt:  response.Data.Attributes.MitigatedAt,
		ResolvedAt:   response.Data.Attributes.ResolvedAt,
		Services:     []IncidentService{},
		Groups:       []IncidentGroup{},
		Events:       []IncidentEvent{},
		ActionItems:  []IncidentActionItem{},
	}

	// Build a map of included resources for easy lookup
	includedMap := make(map[string]map[string]any)
	for _, inc := range response.Included {
		key := fmt.Sprintf("%s:%s", inc.Type, inc.ID)
		includedMap[key] = inc.Attributes
		includedMap[key]["id"] = inc.ID
	}

	// Resolve user relationship
	if response.Data.Relationships != nil {
		if response.Data.Relationships.User != nil && response.Data.Relationships.User.Data != nil {
			key := fmt.Sprintf("%s:%s", response.Data.Relationships.User.Data.Type, response.Data.Relationships.User.Data.ID)
			if attrs, ok := includedMap[key]; ok {
				incident.User = &IncidentUser{
					ID:    getString(attrs, "id"),
					Name:  getString(attrs, "name"),
					Email: getString(attrs, "email"),
				}
			}
		}

		if response.Data.Relationships.StartedBy != nil && response.Data.Relationships.StartedBy.Data != nil {
			key := fmt.Sprintf("%s:%s", response.Data.Relationships.StartedBy.Data.Type, response.Data.Relationships.StartedBy.Data.ID)
			if attrs, ok := includedMap[key]; ok {
				incident.StartedBy = &IncidentUser{
					ID:    getString(attrs, "id"),
					Name:  getString(attrs, "name"),
					Email: getString(attrs, "email"),
				}
			}
		}

		if response.Data.Relationships.Services != nil {
			for _, svc := range response.Data.Relationships.Services.Data {
				key := fmt.Sprintf("%s:%s", svc.Type, svc.ID)
				if attrs, ok := includedMap[key]; ok {
					incident.Services = append(incident.Services, IncidentService{
						ID:   getString(attrs, "id"),
						Name: getString(attrs, "name"),
						Slug: getString(attrs, "slug"),
					})
				}
			}
		}

		if response.Data.Relationships.Groups != nil {
			for _, grp := range response.Data.Relationships.Groups.Data {
				key := fmt.Sprintf("%s:%s", grp.Type, grp.ID)
				if attrs, ok := includedMap[key]; ok {
					incident.Groups = append(incident.Groups, IncidentGroup{
						ID:   getString(attrs, "id"),
						Name: getString(attrs, "name"),
						Slug: getString(attrs, "slug"),
					})
				}
			}
		}
	}

	// Fetch incident events
	events, err := c.listIncidentEvents(id)
	if err == nil {
		incident.Events = events
	}

	// Fetch action items
	actionItems, err := c.listIncidentActionItems(id)
	if err == nil {
		incident.ActionItems = actionItems
	}

	return incident, nil
}

// getString safely extracts a string from a map
func getString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// listIncidentEvents fetches timeline events for an incident
func (c *Client) listIncidentEvents(incidentID string) ([]IncidentEvent, error) {
	url := fmt.Sprintf("%s/incidents/%s/events", c.BaseURL, incidentID)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Data []struct {
			ID         string `json:"id"`
			Attributes struct {
				Kind      string `json:"kind"`
				Summary   string `json:"summary"`
				CreatedAt string `json:"created_at"`
			} `json:"attributes"`
		} `json:"data"`
	}

	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing events response: %v", err)
	}

	events := make([]IncidentEvent, 0, len(response.Data))
	for _, e := range response.Data {
		events = append(events, IncidentEvent{
			ID:        e.ID,
			Kind:      e.Attributes.Kind,
			Summary:   e.Attributes.Summary,
			CreatedAt: e.Attributes.CreatedAt,
		})
	}

	return events, nil
}

// listIncidentActionItems fetches action items for an incident
func (c *Client) listIncidentActionItems(incidentID string) ([]IncidentActionItem, error) {
	url := fmt.Sprintf("%s/incidents/%s/action_items", c.BaseURL, incidentID)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Data []struct {
			ID         string `json:"id"`
			Attributes struct {
				Summary     string `json:"summary"`
				Description string `json:"description"`
				Status      string `json:"status"`
				Priority    string `json:"priority"`
				DueAt       string `json:"due_at"`
			} `json:"attributes"`
		} `json:"data"`
	}

	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing action items response: %v", err)
	}

	items := make([]IncidentActionItem, 0, len(response.Data))
	for _, item := range response.Data {
		items = append(items, IncidentActionItem{
			ID:          item.ID,
			Summary:     item.Attributes.Summary,
			Description: item.Attributes.Description,
			Status:      item.Attributes.Status,
			Priority:    item.Attributes.Priority,
			DueAt:       item.Attributes.DueAt,
		})
	}

	return items, nil
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
