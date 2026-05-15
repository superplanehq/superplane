package jira

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// OpsAlertResponder is a responder or visibleTo entry for the Jira Service Management Ops Alerts API.
type OpsAlertResponder struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

// OpsCreateAlertRequest is the JSON body for POST /v1/alerts.
type OpsCreateAlertRequest struct {
	Message         string              `json:"message"`
	Responders      []OpsAlertResponder `json:"responders,omitempty"`
	VisibleTo       []OpsAlertResponder `json:"visibleTo,omitempty"`
	Note            string              `json:"note,omitempty"`
	Alias           string              `json:"alias,omitempty"`
	Entity          string              `json:"entity,omitempty"`
	Source          string              `json:"source,omitempty"`
	Tags            []string            `json:"tags,omitempty"`
	Actions         []string            `json:"actions,omitempty"`
	Description     string              `json:"description,omitempty"`
	Priority        string              `json:"priority,omitempty"`
	ExtraProperties map[string]any      `json:"extraProperties,omitempty"`
}

// OpsAsyncSuccessResponse is returned by create/delete and several mutating Ops alert endpoints (often with HTTP 202).
type OpsAsyncSuccessResponse struct {
	Result    string  `json:"result"`
	RequestID string  `json:"requestId"`
	Took      float64 `json:"took"`
}

func (c *Client) opsAlertsBasePath(cloudID string) string {
	return fmt.Sprintf(
		"https://api.atlassian.com/jsm/ops/api/%s/v1/alerts",
		url.PathEscape(strings.TrimSpace(cloudID)),
	)
}

func (c *Client) execOpsAlertJSON(method, fullURL string, body io.Reader) ([]byte, error) {
	return c.execRequest(method, fullURL, body)
}

func parseOpsAsyncSuccess(body []byte) (*OpsAsyncSuccessResponse, error) {
	var out OpsAsyncSuccessResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("parse ops alert response: %w", err)
	}
	return &out, nil
}

// CreateOpsAlert creates an alert via the JSM Ops REST API (asynchronous processing; see SuccessResponse).
func (c *Client) CreateOpsAlert(cloudID string, req *OpsCreateAlertRequest) (*OpsAsyncSuccessResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("create alert request is required")
	}
	u := c.opsAlertsBasePath(cloudID)
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal create alert body: %w", err)
	}
	body, err := c.execOpsAlertJSON(http.MethodPost, u, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	return parseOpsAsyncSuccess(body)
}

// GetOpsAlert returns one alert by id (GET /v1/alerts/{id}).
func (c *Client) GetOpsAlert(cloudID, alertID string) (map[string]any, error) {
	alertID = strings.TrimSpace(alertID)
	if alertID == "" {
		return nil, fmt.Errorf("alert id is required")
	}
	u := fmt.Sprintf("%s/%s", c.opsAlertsBasePath(cloudID), url.PathEscape(alertID))
	body, err := c.execOpsAlertJSON(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("parse get alert response: %w", err)
	}
	return out, nil
}

// DeleteOpsAlert deletes an alert by id (DELETE /v1/alerts/{id}).
func (c *Client) DeleteOpsAlert(cloudID, alertID string) (*OpsAsyncSuccessResponse, error) {
	alertID = strings.TrimSpace(alertID)
	if alertID == "" {
		return nil, fmt.Errorf("alert id is required")
	}
	u := fmt.Sprintf("%s/%s", c.opsAlertsBasePath(cloudID), url.PathEscape(alertID))
	body, err := c.execOpsAlertJSON(http.MethodDelete, u, nil)
	if err != nil {
		return nil, err
	}
	return parseOpsAsyncSuccess(body)
}

// AcknowledgeOpsAlert acknowledges an alert (POST /v1/alerts/{id}/acknowledge).
func (c *Client) AcknowledgeOpsAlert(cloudID, alertID string) (*OpsAsyncSuccessResponse, error) {
	u := fmt.Sprintf("%s/%s/acknowledge", c.opsAlertsBasePath(cloudID), url.PathEscape(strings.TrimSpace(alertID)))
	body, err := c.execOpsAlertJSON(http.MethodPost, u, nil)
	if err != nil {
		return nil, err
	}
	return parseOpsAsyncSuccess(body)
}

// CloseOpsAlert closes an alert (POST /v1/alerts/{id}/close).
func (c *Client) CloseOpsAlert(cloudID, alertID string) (*OpsAsyncSuccessResponse, error) {
	u := fmt.Sprintf("%s/%s/close", c.opsAlertsBasePath(cloudID), url.PathEscape(strings.TrimSpace(alertID)))
	body, err := c.execOpsAlertJSON(http.MethodPost, u, nil)
	if err != nil {
		return nil, err
	}
	return parseOpsAsyncSuccess(body)
}

// PatchOpsAlertPriority updates alert priority (PATCH /v1/alerts/{id}/priority).
func (c *Client) PatchOpsAlertPriority(cloudID, alertID, priority string) (*OpsAsyncSuccessResponse, error) {
	payload, err := json.Marshal(map[string]string{"priority": strings.TrimSpace(priority)})
	if err != nil {
		return nil, err
	}
	u := fmt.Sprintf("%s/%s/priority", c.opsAlertsBasePath(cloudID), url.PathEscape(strings.TrimSpace(alertID)))
	body, err := c.execOpsAlertJSON(http.MethodPatch, u, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	return parseOpsAsyncSuccess(body)
}

// PatchOpsAlertMessage updates the alert message (PATCH /v1/alerts/{id}/message).
func (c *Client) PatchOpsAlertMessage(cloudID, alertID, message string) (*OpsAsyncSuccessResponse, error) {
	payload, err := json.Marshal(map[string]string{"message": message})
	if err != nil {
		return nil, err
	}
	u := fmt.Sprintf("%s/%s/message", c.opsAlertsBasePath(cloudID), url.PathEscape(strings.TrimSpace(alertID)))
	body, err := c.execOpsAlertJSON(http.MethodPatch, u, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	return parseOpsAsyncSuccess(body)
}

// PatchOpsAlertDescription updates the alert description (PATCH /v1/alerts/{id}/description).
func (c *Client) PatchOpsAlertDescription(cloudID, alertID, description string) (*OpsAsyncSuccessResponse, error) {
	payload, err := json.Marshal(map[string]string{"description": description})
	if err != nil {
		return nil, err
	}
	u := fmt.Sprintf("%s/%s/description", c.opsAlertsBasePath(cloudID), url.PathEscape(strings.TrimSpace(alertID)))
	body, err := c.execOpsAlertJSON(http.MethodPatch, u, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	return parseOpsAsyncSuccess(body)
}

// AddOpsAlertNote adds a note (POST /v1/alerts/{id}/notes).
func (c *Client) AddOpsAlertNote(cloudID, alertID, note string) (map[string]any, error) {
	payload, err := json.Marshal(map[string]string{"note": note})
	if err != nil {
		return nil, err
	}
	u := fmt.Sprintf("%s/%s/notes", c.opsAlertsBasePath(cloudID), url.PathEscape(strings.TrimSpace(alertID)))
	body, err := c.execOpsAlertJSON(http.MethodPost, u, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("parse add alert note response: %w", err)
	}
	return out, nil
}

// PatchOpsAlertNote updates an existing note (PATCH /v1/alerts/{alertId}/notes/{id}).
func (c *Client) PatchOpsAlertNote(cloudID, alertID, noteID, note string) (map[string]any, error) {
	payload, err := json.Marshal(map[string]string{"note": note})
	if err != nil {
		return nil, err
	}
	base := c.opsAlertsBasePath(cloudID)
	u := fmt.Sprintf(
		"%s/%s/notes/%s",
		base,
		url.PathEscape(strings.TrimSpace(alertID)),
		url.PathEscape(strings.TrimSpace(noteID)),
	)
	body, err := c.execOpsAlertJSON(http.MethodPatch, u, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("parse update alert note response: %w", err)
	}
	return out, nil
}
