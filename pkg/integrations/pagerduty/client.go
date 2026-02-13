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
	AuthType string
	Token    string
	BaseURL  string
	http     core.HTTPContext
}

func NewClient(http core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	authType, err := ctx.GetConfig("authType")
	if err != nil {
		return nil, fmt.Errorf("error finding auth type: %v", err)
	}

	switch string(authType) {
	case AuthTypeAPIToken:
		apiToken, err := ctx.GetConfig("apiToken")
		if err != nil {
			return nil, err
		}

		return &Client{
			Token:    string(apiToken),
			AuthType: AuthTypeAPIToken,
			BaseURL:  "https://api.pagerduty.com",
			http:     http,
		}, nil

	case AuthTypeAppOAuth:
		secrets, err := ctx.GetSecrets()
		if err != nil {
			return nil, fmt.Errorf("failed to get secrets: %v", err)
		}

		var accessToken string
		for _, secret := range secrets {
			if secret.Name == AppAccessToken {
				accessToken = string(secret.Value)
				break
			}
		}

		if accessToken == "" {
			return nil, fmt.Errorf("app OAuth access token not found")
		}

		return &Client{
			Token:    accessToken,
			AuthType: AuthTypeAppOAuth,
			BaseURL:  "https://api.pagerduty.com",
			http:     http,
		}, nil
	}

	return nil, fmt.Errorf("unknown auth type %s", authType)
}

func (c *Client) execRequest(method, url string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("error building request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.pagerduty+json;version=2")

	if c.AuthType == AuthTypeAppOAuth {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token))
	} else {
		req.Header.Set("Authorization", fmt.Sprintf("Token token=%s", c.Token))
	}

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

type CreateIncidentRequest struct {
	Incident IncidentPayload `json:"incident"`
}

type IncidentPayload struct {
	Type    string           `json:"type"` // "incident"
	Title   string           `json:"title"`
	Service ServiceReference `json:"service"`
	Urgency string           `json:"urgency,omitempty"` // "high" or "low"
	Body    *IncidentBody    `json:"body,omitempty"`
}

type ServiceReference struct {
	ID   string `json:"id"`
	Type string `json:"type"` // "service_reference"
}

type IncidentBody struct {
	Type    string `json:"type"` // "incident_body"
	Details string `json:"details"`
}

func (c *Client) CreateIncident(title, service, urgency, description, fromEmail string) (any, error) {
	request := CreateIncidentRequest{
		Incident: IncidentPayload{
			Type:    "incident",
			Title:   title,
			Urgency: urgency,
			Service: ServiceReference{
				ID:   service,
				Type: "service_reference",
			},
		},
	}

	if description != "" {
		request.Incident.Body = &IncidentBody{
			Type:    "incident_body",
			Details: description,
		}
	}

	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	url := fmt.Sprintf("%s/incidents", c.BaseURL)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("error building request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.pagerduty+json;version=2")

	if fromEmail != "" {
		req.Header.Set("From", fromEmail)
	}

	if c.AuthType == AuthTypeAppOAuth {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token))
	} else {
		req.Header.Set("Authorization", fmt.Sprintf("Token token=%s", c.Token))
	}

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

	var response map[string]any
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return response, nil
}

type UpdateIncidentRequest struct {
	Incident UpdateIncidentPayload `json:"incident"`
}

type UpdateIncidentPayload struct {
	Type             string               `json:"type"` // "incident_reference"
	Status           string               `json:"status,omitempty"`
	Title            string               `json:"title,omitempty"`
	Priority         *PriorityReference   `json:"priority,omitempty"`
	EscalationPolicy *EscalationPolicyRef `json:"escalation_policy,omitempty"`
	EscalationLevel  int                  `json:"escalation_level,omitempty"`
	Assignments      []AssignmentPayload  `json:"assignments,omitempty"`
	Body             *IncidentBody        `json:"body,omitempty"`
}

type PriorityReference struct {
	ID   string `json:"id"`
	Type string `json:"type"` // "priority_reference"
}

type EscalationPolicyRef struct {
	ID   string `json:"id"`
	Type string `json:"type"` // "escalation_policy_reference"
}

type AssignmentPayload struct {
	Assignee UserReference `json:"assignee"`
}

type UserReference struct {
	ID   string `json:"id"`
	Type string `json:"type"` // "user_reference"
}

func (c *Client) UpdateIncident(
	incidentID string,
	fromEmail string,
	status string,
	priority string,
	title string,
	description string,
	escalationPolicy string,
	escalationLevel int,
	assignees []string,
) (any, error) {
	request := UpdateIncidentRequest{
		Incident: UpdateIncidentPayload{
			Type: "incident_reference",
		},
	}

	// Only include fields that are provided
	if status != "" {
		request.Incident.Status = status
	}

	if priority != "" {
		request.Incident.Priority = &PriorityReference{
			ID:   priority,
			Type: "priority_reference",
		}
	}

	if title != "" {
		request.Incident.Title = title
	}

	if description != "" {
		request.Incident.Body = &IncidentBody{
			Type:    "incident_body",
			Details: description,
		}
	}

	if escalationPolicy != "" {
		request.Incident.EscalationPolicy = &EscalationPolicyRef{
			ID:   escalationPolicy,
			Type: "escalation_policy_reference",
		}
	}

	if escalationLevel > 0 {
		request.Incident.EscalationLevel = escalationLevel
	}

	if len(assignees) > 0 {
		assignments := make([]AssignmentPayload, 0, len(assignees))
		for _, userID := range assignees {
			assignments = append(assignments, AssignmentPayload{
				Assignee: UserReference{
					ID:   userID,
					Type: "user_reference",
				},
			})
		}
		request.Incident.Assignments = assignments
	}

	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	url := fmt.Sprintf("%s/incidents/%s", c.BaseURL, incidentID)

	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("error building request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.pagerduty+json;version=2")

	if fromEmail != "" {
		req.Header.Set("From", fromEmail)
	}

	if c.AuthType == AuthTypeAppOAuth {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token))
	} else {
		req.Header.Set("Authorization", fmt.Sprintf("Token token=%s", c.Token))
	}

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

	var response map[string]any
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return response, nil
}

type AddNoteRequest struct {
	Note NotePayload `json:"note"`
}

type NotePayload struct {
	Content string `json:"content"`
}

func (c *Client) AddIncidentNote(incidentID string, fromEmail string, content string) error {
	request := AddNoteRequest{
		Note: NotePayload{
			Content: content,
		},
	}

	body, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("error marshaling request: %v", err)
	}

	url := fmt.Sprintf("%s/incidents/%s/notes", c.BaseURL, incidentID)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("error building request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.pagerduty+json;version=2")

	if fromEmail != "" {
		req.Header.Set("From", fromEmail)
	}

	if c.AuthType == AuthTypeAppOAuth {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token))
	} else {
		req.Header.Set("Authorization", fmt.Sprintf("Token token=%s", c.Token))
	}

	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("error executing request: %v", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("error reading body: %v", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("request got %d code: %s", res.StatusCode, string(responseBody))
	}

	return nil
}

func (c *Client) GetIncident(incidentID string) (any, error) {
	url := fmt.Sprintf("%s/incidents/%s", c.BaseURL, incidentID)
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

type WebhookSubscription struct {
	ID             string         `json:"id"`
	Events         []string       `json:"events"`
	DeliveryMethod DeliveryMethod `json:"delivery_method"`
}

type DeliveryMethod struct {
	Type   string `json:"type"`
	URL    string `json:"url"`
	Secret string `json:"secret"`
}

func (c *Client) CreateWebhookSubscription(url string, events []string, filter WebhookFilter) (*WebhookSubscription, error) {
	apiURL := fmt.Sprintf("%s/webhook_subscriptions", c.BaseURL)
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

type Service struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	HTMLURL string `json:"html_url"`
}

func (c *Client) ListServices() ([]Service, error) {
	apiURL := fmt.Sprintf("%s/services", c.BaseURL)
	responseBody, err := c.execRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Services []Service `json:"services"`
	}

	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return response.Services, nil
}

func (c *Client) GetService(id string) (*Service, error) {
	apiURL := fmt.Sprintf("%s/services/%s", c.BaseURL, id)
	responseBody, err := c.execRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Service Service `json:"service"`
	}

	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &response.Service, nil
}

type Priority struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (c *Client) ListPriorities() ([]Priority, error) {
	apiURL := fmt.Sprintf("%s/priorities", c.BaseURL)
	responseBody, err := c.execRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Priorities []Priority `json:"priorities"`
	}

	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return response.Priorities, nil
}

type User struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	HTMLURL string `json:"html_url"`
}

func (c *Client) ListUsers() ([]User, error) {
	apiURL := fmt.Sprintf("%s/users", c.BaseURL)
	responseBody, err := c.execRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Users []User `json:"users"`
	}

	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return response.Users, nil
}

type EscalationPolicy struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	HTMLURL string `json:"html_url"`
}

func (c *Client) ListEscalationPolicies() ([]EscalationPolicy, error) {
	apiURL := fmt.Sprintf("%s/escalation_policies", c.BaseURL)
	responseBody, err := c.execRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		EscalationPolicies []EscalationPolicy `json:"escalation_policies"`
	}

	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return response.EscalationPolicies, nil
}

// Incident represents a PagerDuty incident returned from the API
type Incident struct {
	ID               string            `json:"id"`
	IncidentNumber   int               `json:"incident_number"`
	Title            string            `json:"title"`
	Description      string            `json:"description"`
	Status           string            `json:"status"`
	Urgency          string            `json:"urgency"`
	HTMLURL          string            `json:"html_url"`
	CreatedAt        string            `json:"created_at"`
	UpdatedAt        string            `json:"updated_at"`
	Service          *ServiceRef       `json:"service"`
	EscalationPolicy *ServiceRef       `json:"escalation_policy"`
	Assignments      []Assignment      `json:"assignments"`
	Acknowledgements []Acknowledgement `json:"acknowledgements"`
	Priority         *PriorityRef      `json:"priority"`
}

// ServiceRef represents a reference to a service in incident responses
type ServiceRef struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Summary string `json:"summary"`
	HTMLURL string `json:"html_url"`
}

// Assignment represents an incident assignment
type Assignment struct {
	At       string      `json:"at"`
	Assignee *ServiceRef `json:"assignee"`
}

// Acknowledgement represents an incident acknowledgement
type Acknowledgement struct {
	At           string      `json:"at"`
	Acknowledger *ServiceRef `json:"acknowledger"`
}

// PriorityRef represents a reference to a priority
type PriorityRef struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Summary string `json:"summary"`
}

// ListIncidents retrieves incidents from PagerDuty filtered by status and optionally by service IDs.
// By default, it returns open incidents (triggered and acknowledged).
func (c *Client) ListIncidents(serviceIDs []string) ([]Incident, error) {
	apiURL := fmt.Sprintf("%s/incidents?statuses[]=triggered&statuses[]=acknowledged", c.BaseURL)

	// Add service ID filters if provided
	for _, serviceID := range serviceIDs {
		apiURL += fmt.Sprintf("&service_ids[]=%s", serviceID)
	}

	responseBody, err := c.execRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Incidents []Incident `json:"incidents"`
	}

	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return response.Incidents, nil
}

type SnoozeIncidentRequest struct {
	Duration int `json:"duration"` // duration in seconds
}

func (c *Client) SnoozeIncident(incidentID string, fromEmail string, duration int) (any, error) {
	request := SnoozeIncidentRequest{
		Duration: duration,
	}

	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	url := fmt.Sprintf("%s/incidents/%s/snooze", c.BaseURL, incidentID)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("error building request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.pagerduty+json;version=2")

	if fromEmail != "" {
		req.Header.Set("From", fromEmail)
	}

	if c.AuthType == AuthTypeAppOAuth {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token))
	} else {
		req.Header.Set("Authorization", fmt.Sprintf("Token token=%s", c.Token))
	}

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

	var response map[string]any
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return response, nil
}

// Note represents a note on a PagerDuty incident
type Note struct {
	ID        string       `json:"id"`
	User      *ServiceRef  `json:"user"`
	Channel   *NoteChannel `json:"channel"`
	Content   string       `json:"content"`
	CreatedAt string       `json:"created_at"`
}

// NoteChannel represents the channel through which a note was created
type NoteChannel struct {
	Type string `json:"type"`
}

// ListIncidentNotes retrieves all notes for a given incident
func (c *Client) ListIncidentNotes(incidentID string) ([]Note, error) {
	apiURL := fmt.Sprintf("%s/incidents/%s/notes", c.BaseURL, incidentID)
	responseBody, err := c.execRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Notes []Note `json:"notes"`
	}

	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return response.Notes, nil
}

// LogEntry represents a log entry for a PagerDuty incident
type LogEntry struct {
	ID        string      `json:"id"`
	Type      string      `json:"type"`
	Summary   string      `json:"summary"`
	CreatedAt string      `json:"created_at"`
	Agent     *ServiceRef `json:"agent"`
	Channel   *LogChannel `json:"channel"`
}

// LogChannel represents the channel through which a log entry was created
type LogChannel struct {
	Type string `json:"type"`
}

// ListIncidentLogEntries retrieves log entries for a given incident
func (c *Client) ListIncidentLogEntries(incidentID string, limit int) ([]LogEntry, error) {
	apiURL := fmt.Sprintf("%s/incidents/%s/log_entries?limit=%d", c.BaseURL, incidentID, limit)
	responseBody, err := c.execRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		LogEntries []LogEntry `json:"log_entries"`
	}

	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return response.LogEntries, nil
}
