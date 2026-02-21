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
	ID           string `json:"id"`
	SequentialID int    `json:"sequential_id"`
	Title        string `json:"title"`
	Slug         string `json:"slug"`
	Summary      string `json:"summary"`
	Status       string `json:"status"`
	Severity     string `json:"severity"`
	StartedAt    string `json:"started_at"`
	ResolvedAt   string `json:"resolved_at"`
	MitigatedAt  string `json:"mitigated_at"`
	UpdatedAt    string `json:"updated_at"`
	URL          string `json:"url"`
}

type IncidentData struct {
	ID         string             `json:"id"`
	Type       string             `json:"type"`
	Attributes IncidentAttributes `json:"attributes"`
}

type IncidentAttributes struct {
	Title        string `json:"title"`
	SequentialID int    `json:"sequential_id"`
	Slug         string `json:"slug"`
	Summary      string `json:"summary"`
	Status       string `json:"status"`
	Severity     any    `json:"severity"`
	StartedAt    string `json:"started_at"`
	ResolvedAt   string `json:"resolved_at"`
	MitigatedAt  string `json:"mitigated_at"`
	UpdatedAt    string `json:"updated_at"`
	URL          string `json:"url"`
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

// severityString extracts the severity slug from the API response.
// Rootly returns severity as a string (slug) or an object with slug/name fields.
func severityString(v any) string {
	switch s := v.(type) {
	case string:
		return s
	case map[string]any:
		if slug, ok := s["slug"].(string); ok {
			return slug
		}
		if name, ok := s["name"].(string); ok {
			return name
		}
	}

	return ""
}

// incidentFromData converts a JSON:API IncidentData to a flat Incident struct.
func incidentFromData(data IncidentData) *Incident {
	return &Incident{
		ID:           data.ID,
		SequentialID: data.Attributes.SequentialID,
		Title:        data.Attributes.Title,
		Slug:         data.Attributes.Slug,
		Summary:      data.Attributes.Summary,
		Status:       data.Attributes.Status,
		Severity:     severityString(data.Attributes.Severity),
		StartedAt:    data.Attributes.StartedAt,
		ResolvedAt:   data.Attributes.ResolvedAt,
		MitigatedAt:  data.Attributes.MitigatedAt,
		UpdatedAt:    data.Attributes.UpdatedAt,
		URL:          data.Attributes.URL,
	}
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

	return incidentFromData(response.Data), nil
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

	return incidentFromData(response.Data), nil
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

	// Normalize severity to string for consistency with other components
	if sev, ok := result["severity"]; ok {
		result["severity"] = severityString(sev)
	}

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

	// Only resolve the relationships we explicitly requested via the include
	// parameter. The API response contains many more relationships (e.g.
	// severity, user, started_by) that may have "data": null linkage. If we
	// iterated over all of them, a null data value would overwrite a field
	// already populated from attributes.
	requestedRelationships := map[string]bool{
		"services":     true,
		"groups":       true,
		"events":       true,
		"action_items": true,
	}

	relationships, _ := data["relationships"].(map[string]any)
	for relName, relValue := range relationships {
		if !requestedRelationships[relName] {
			continue
		}

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
		incidents = append(incidents, *incidentFromData(data))
	}

	return incidents, nil
}

// Team represents a Rootly team (group)
type Team struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
}

type TeamData struct {
	ID         string         `json:"id"`
	Type       string         `json:"type"`
	Attributes TeamAttributes `json:"attributes"`
}

type TeamAttributes struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
}

type TeamsResponse struct {
	Data []TeamData `json:"data"`
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

	teams := make([]Team, 0, len(response.Data))
	for _, data := range response.Data {
		teams = append(teams, Team{
			ID:          data.ID,
			Name:        data.Attributes.Name,
			Slug:        data.Attributes.Slug,
			Description: data.Attributes.Description,
		})
	}

	return teams, nil
}

// SubStatus represents a Rootly sub-status (custom status)
type SubStatus struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	ParentStatus string `json:"parent_status"`
}

type SubStatusData struct {
	ID         string              `json:"id"`
	Type       string              `json:"type"`
	Attributes SubStatusAttributes `json:"attributes"`
}

type SubStatusAttributes struct {
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	ParentStatus string `json:"parent_status"`
}

type SubStatusesResponse struct {
	Data []SubStatusData `json:"data"`
}

func (c *Client) ListSubStatuses() ([]SubStatus, error) {
	url := fmt.Sprintf("%s/sub_statuses", c.BaseURL)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var response SubStatusesResponse
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	subStatuses := make([]SubStatus, 0, len(response.Data))
	for _, data := range response.Data {
		subStatuses = append(subStatuses, SubStatus{
			ID:           data.ID,
			Name:         data.Attributes.Name,
			Slug:         data.Attributes.Slug,
			ParentStatus: data.Attributes.ParentStatus,
		})
	}

	return subStatuses, nil
}

// UpdateIncidentRequest represents the JSON:API request to update an incident
type UpdateIncidentRequest struct {
	Data UpdateIncidentData `json:"data"`
}

type UpdateIncidentData struct {
	ID         string                   `json:"id"`
	Type       string                   `json:"type"`
	Attributes UpdateIncidentAttributes `json:"attributes"`
}

type UpdateIncidentAttributes struct {
	Title       string            `json:"title,omitempty"`
	Summary     string            `json:"summary,omitempty"`
	Status      string            `json:"status,omitempty"`
	SubStatusID string            `json:"sub_status_id,omitempty"`
	SeverityID  string            `json:"severity_id,omitempty"`
	ServiceIDs  []string          `json:"service_ids,omitempty"`
	GroupIDs    []string          `json:"group_ids,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
}

func (c *Client) UpdateIncident(id string, attrs UpdateIncidentAttributes) (*Incident, error) {
	request := UpdateIncidentRequest{
		Data: UpdateIncidentData{
			ID:         id,
			Type:       "incidents",
			Attributes: attrs,
		},
	}

	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	url := fmt.Sprintf("%s/incidents/%s", c.BaseURL, id)
	responseBody, err := c.execRequest(http.MethodPatch, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response IncidentResponse
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return incidentFromData(response.Data), nil
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
