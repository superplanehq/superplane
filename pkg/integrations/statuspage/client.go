// Package statuspage implements the Atlassian Statuspage integration.
// API reference: https://developer.statuspage.io/
package statuspage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const defaultBaseURL = "https://api.statuspage.io/v1"

type Client struct {
	apiKey  string
	baseURL string
	http    core.HTTPContext
}

// validateBaseURL returns an error if the URL is invalid.
func validateBaseURL(baseURL string) error {
	u, err := url.Parse(baseURL)
	if err != nil {
		return fmt.Errorf("invalid base URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("base URL must use http or https scheme")
	}
	if u.Host == "" {
		return fmt.Errorf("base URL must have a host")
	}
	return nil
}

func NewClient(http core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	apiKey, err := ctx.GetConfig("apiKey")
	if err != nil {
		return nil, fmt.Errorf("api key not found: %w", err)
	}

	baseURL := defaultBaseURL
	if baseURLConfig, err := ctx.GetConfig("baseURL"); err == nil && baseURLConfig != nil && len(baseURLConfig) > 0 {
		baseURL = string(baseURLConfig)
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	if err := validateBaseURL(baseURL); err != nil {
		return nil, err
	}

	return &Client{
		apiKey:  string(apiKey),
		baseURL: baseURL,
		http:    http,
	}, nil
}

func (c *Client) do(method, path string, body io.Reader, contentType string) ([]byte, error) {
	req, err := http.NewRequest(method, c.baseURL+path, body)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Authorization", "OAuth "+c.apiKey)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if res.StatusCode == 420 || res.StatusCode == 429 {
		return nil, fmt.Errorf("rate limited (HTTP %d): %s", res.StatusCode, string(resBody))
	}
	if res.StatusCode == 404 {
		return nil, fmt.Errorf("resource not found (404): %s", string(resBody))
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("request failed with %d: %s", res.StatusCode, string(resBody))
	}

	return resBody, nil
}

type Page struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (c *Client) ListPages() ([]Page, error) {
	body, err := c.do(http.MethodGet, "/pages", nil, "")
	if err != nil {
		return nil, err
	}
	var pages []Page
	if err := json.Unmarshal(body, &pages); err != nil {
		return nil, fmt.Errorf("parsing pages: %w", err)
	}
	return pages, nil
}

type Component struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

func (c *Client) ListComponents(pageID string) ([]Component, error) {
	path := fmt.Sprintf("/pages/%s/components", url.PathEscape(pageID))
	body, err := c.do(http.MethodGet, path, nil, "")
	if err != nil {
		return nil, err
	}
	var components []Component
	if err := json.Unmarshal(body, &components); err != nil {
		return nil, fmt.Errorf("parsing components: %w", err)
	}
	return components, nil
}

// CreateIncidentRequest holds the payload for creating an incident.
// Realtime: name, body, status, impactOverride, components (list of { componentId, status }), deliverNotifications.
// Scheduled: name, body, scheduledFor, scheduledUntil, scheduledRemindPrior, scheduledAutoInProgress, scheduledAutoCompleted, components, deliverNotifications.
type CreateIncidentRequest struct {
	Name                    string            `json:"name"`
	Body                    string            `json:"body"`
	Status                  string            `json:"status"`
	ImpactOverride          string            `json:"impact_override"`
	ComponentIDs            []string          `json:"component_ids"`
	Components              map[string]string `json:"components"` // component_id -> status
	ScheduledFor            string            `json:"scheduled_for"`
	ScheduledUntil          string            `json:"scheduled_until"`
	ScheduledRemindPrior    bool              `json:"scheduled_remind_prior"`
	ScheduledAutoInProgress bool              `json:"scheduled_auto_in_progress"`
	ScheduledAutoCompleted  bool              `json:"scheduled_auto_completed"`
	DeliverNotifications    *bool             `json:"deliver_notifications"`
	Realtime                bool              `json:"-"` // if true, send as realtime; else scheduled
}

// incidentPayload is the JSON body for POST /pages/{page_id}/incidents.
type incidentPayload struct {
	Name                    string            `json:"name"`
	Body                    string            `json:"body,omitempty"`
	Status                  string            `json:"status,omitempty"`
	ImpactOverride          string            `json:"impact_override,omitempty"`
	ComponentIDs            []string          `json:"component_ids,omitempty"`
	Components              map[string]string `json:"components,omitempty"`
	ScheduledFor            string            `json:"scheduled_for,omitempty"`
	ScheduledUntil          string            `json:"scheduled_until,omitempty"`
	ScheduledRemindPrior    *bool             `json:"scheduled_remind_prior,omitempty"`
	ScheduledAutoInProgress *bool             `json:"scheduled_auto_in_progress,omitempty"`
	ScheduledAutoCompleted  *bool             `json:"scheduled_auto_completed,omitempty"`
	DeliverNotifications    *bool             `json:"deliver_notifications,omitempty"`
}

// CreateIncident creates an incident and returns the incident object as map[string]any.
func (c *Client) CreateIncident(pageID string, req CreateIncidentRequest) (map[string]any, error) {
	payload := incidentPayload{
		Name:                 req.Name,
		Body:                 req.Body,
		ComponentIDs:         req.ComponentIDs,
		Components:           req.Components,
		DeliverNotifications: req.DeliverNotifications,
	}

	if req.Realtime {
		if req.Status != "" {
			payload.Status = req.Status
		}
		payload.ImpactOverride = req.ImpactOverride
	} else {
		payload.Status = req.Status
		if payload.Status == "" {
			payload.Status = "scheduled"
		}
		payload.ScheduledFor = req.ScheduledFor
		payload.ScheduledUntil = req.ScheduledUntil
		payload.ScheduledRemindPrior = &req.ScheduledRemindPrior
		payload.ScheduledAutoInProgress = &req.ScheduledAutoInProgress
		payload.ScheduledAutoCompleted = &req.ScheduledAutoCompleted
	}

	body := map[string]any{"incident": payload}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	path := fmt.Sprintf("/pages/%s/incidents", url.PathEscape(pageID))
	resBody, err := c.do(http.MethodPost, path, bytes.NewReader(raw), "application/json")
	if err != nil {
		return nil, err
	}
	return extractIncident(resBody)
}

// Incident is a minimal incident representation for listing.
type Incident struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ListIncidents returns incidents for a page. Use empty q and limit 0 for defaults.
func (c *Client) ListIncidents(pageID string, q string, limit int) ([]Incident, error) {
	path := fmt.Sprintf("/pages/%s/incidents", url.PathEscape(pageID))
	if q != "" || limit > 0 {
		params := url.Values{}
		if q != "" {
			params.Set("q", q)
		}
		if limit > 0 {
			params.Set("limit", fmt.Sprintf("%d", limit))
		}
		path += "?" + params.Encode()
	}
	body, err := c.do(http.MethodGet, path, nil, "")
	if err != nil {
		return nil, err
	}
	var incidents []Incident
	if err := json.Unmarshal(body, &incidents); err != nil {
		return nil, fmt.Errorf("parsing incidents: %w", err)
	}
	return incidents, nil
}

// UpdateIncidentRequest holds the payload for PATCH /pages/{page_id}/incidents/{incident_id}.
type UpdateIncidentRequest struct {
	Status               string            `json:"status,omitempty"`
	Body                 string            `json:"body,omitempty"`
	ImpactOverride       string            `json:"impact_override,omitempty"`
	ComponentIDs         []string          `json:"component_ids,omitempty"`
	Components           map[string]string `json:"components,omitempty"`
	DeliverNotifications *bool             `json:"deliver_notifications,omitempty"`
}

// UpdateIncident updates an incident and returns the full response as map[string]any.
func (c *Client) UpdateIncident(pageID, incidentID string, req UpdateIncidentRequest) (map[string]any, error) {
	if req.Status == "" && req.Body == "" && req.ImpactOverride == "" && len(req.ComponentIDs) == 0 && len(req.Components) == 0 {
		return nil, fmt.Errorf("at least one of status, body, impact override, or components must be provided")
	}

	payload := map[string]any{}
	if req.Status != "" {
		payload["status"] = req.Status
	}
	if req.Body != "" {
		payload["body"] = req.Body
	}
	if req.ImpactOverride != "" {
		payload["impact_override"] = req.ImpactOverride
	}
	if len(req.ComponentIDs) > 0 {
		payload["component_ids"] = req.ComponentIDs
	}
	if len(req.Components) > 0 {
		payload["components"] = req.Components
	}
	if req.DeliverNotifications != nil {
		payload["deliver_notifications"] = *req.DeliverNotifications
	}

	body := map[string]any{"incident": payload}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	path := fmt.Sprintf("/pages/%s/incidents/%s", url.PathEscape(pageID), url.PathEscape(incidentID))
	resBody, err := c.do(http.MethodPatch, path, bytes.NewReader(raw), "application/json")
	if err != nil {
		return nil, err
	}
	return extractIncident(resBody)
}

// extractIncident parses the API response and returns the incident object.
// The Statuspage API returns {"incident": {...}} for create/update/get; we unwrap to return the incident directly.
func extractIncident(resBody []byte) (map[string]any, error) {
	var raw map[string]any
	if err := json.Unmarshal(resBody, &raw); err != nil {
		return nil, fmt.Errorf("parsing incident response: %w", err)
	}
	if inc, ok := raw["incident"].(map[string]any); ok {
		return inc, nil
	}
	return raw, nil
}

// GetIncident fetches a single incident by ID and returns the full response including incident_updates (timeline).
// incident_updates ordering is preserved as returned by the Statuspage API.
func (c *Client) GetIncident(pageID, incidentID string) (map[string]any, error) {
	path := fmt.Sprintf("/pages/%s/incidents/%s", url.PathEscape(pageID), url.PathEscape(incidentID))
	resBody, err := c.do(http.MethodGet, path, nil, "")
	if err != nil {
		return nil, err
	}
	return extractIncident(resBody)
}
