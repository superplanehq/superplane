package grafana

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const (
	grafanaIRMRPCBasePath        = "/api/plugins/grafana-irm-app/resources/api/v1"
	incidentDefaultRoomPrefix    = "incident"
	incidentStatusActive         = "active"
	incidentStatusResolved       = "resolved"
	incidentActivityKindUserNote = "userNote"
)

type IncidentLabel struct {
	Key         string `json:"key,omitempty"`
	Label       string `json:"label,omitempty"`
	Description string `json:"description,omitempty"`
	ColorHex    string `json:"colorHex,omitempty"`
}

type Incident struct {
	IncidentID      string          `json:"incidentID"`
	Title           string          `json:"title"`
	Summary         string          `json:"summary,omitempty"`
	Severity        string          `json:"severity"`
	Status          string          `json:"status"`
	Labels          []IncidentLabel `json:"labels,omitempty"`
	IsDrill         bool            `json:"isDrill"`
	CreatedTime     string          `json:"createdTime,omitempty"`
	ModifiedTime    string          `json:"modifiedTime,omitempty"`
	ClosedTime      string          `json:"closedTime,omitempty"`
	IncidentStart   string          `json:"incidentStart,omitempty"`
	IncidentEnd     string          `json:"incidentEnd,omitempty"`
	IncidentType    string          `json:"incidentType,omitempty"`
	OverviewURL     string          `json:"overviewURL,omitempty"`
	IncidentURL     string          `json:"incidentUrl,omitempty"`
	DurationSeconds int64           `json:"durationSeconds,omitempty"`
}

type IncidentPreview struct {
	IncidentID   string `json:"incidentID"`
	Title        string `json:"title"`
	Severity     string `json:"severity"`
	Status       string `json:"status"`
	IsDrill      bool   `json:"isDrill"`
	CreatedTime  string `json:"createdTime,omitempty"`
	ModifiedTime string `json:"modifiedTime,omitempty"`
	ClosedTime   string `json:"closedTime,omitempty"`
	OverviewURL  string `json:"overviewURL,omitempty"`
	IncidentURL  string `json:"incidentUrl,omitempty"`
}

type IncidentActivityItem struct {
	ActivityItemID string         `json:"activityItemID"`
	IncidentID     string         `json:"incidentID"`
	ActivityKind   string         `json:"activityKind"`
	Body           string         `json:"body"`
	CreatedTime    string         `json:"createdTime,omitempty"`
	EventTime      string         `json:"eventTime,omitempty"`
	URL            string         `json:"url,omitempty"`
	Tags           []string       `json:"tags,omitempty"`
	FieldValues    map[string]any `json:"fieldValues,omitempty"`
	Relevance      string         `json:"relevance,omitempty"`
}

type declareIncidentRequest struct {
	Title               string          `json:"title"`
	Severity            string          `json:"severity"`
	Labels              []IncidentLabel `json:"labels,omitempty"`
	RoomPrefix          string          `json:"roomPrefix"`
	IsDrill             bool            `json:"isDrill"`
	Status              string          `json:"status"`
	InitialStatusUpdate string          `json:"initialStatusUpdate,omitempty"`
}

type incidentResponse struct {
	Incident Incident `json:"incident"`
	Error    string   `json:"error,omitempty"`
}

type queryIncidentPreviewsRequest struct {
	Query struct {
		Limit          int    `json:"limit,omitempty"`
		OrderField     string `json:"orderField,omitempty"`
		OrderDirection string `json:"orderDirection,omitempty"`
	} `json:"query"`
}

type queryIncidentPreviewsResponse struct {
	IncidentPreviews []IncidentPreview `json:"incidentPreviews"`
	Error            string            `json:"error,omitempty"`
}

type addIncidentActivityRequest struct {
	IncidentID   string `json:"incidentID"`
	ActivityKind string `json:"activityKind"`
	Body         string `json:"body"`
}

type addIncidentActivityResponse struct {
	ActivityItem IncidentActivityItem `json:"activityItem"`
	Error        string               `json:"error,omitempty"`
}

func (c *Client) DeclareIncident(title, severity, initialStatusUpdate string, labels []string, isDrill bool) (*Incident, error) {
	request := declareIncidentRequest{
		Title:               strings.TrimSpace(title),
		Severity:            strings.TrimSpace(severity),
		Labels:              incidentLabelsFromStrings(labels),
		RoomPrefix:          incidentDefaultRoomPrefix,
		IsDrill:             isDrill,
		Status:              incidentStatusActive,
		InitialStatusUpdate: strings.TrimSpace(initialStatusUpdate),
	}

	response := incidentResponse{}
	if err := c.execGrafanaIRMRPC("IncidentsService.CreateIncident", request, &response); err != nil {
		return nil, err
	}

	c.decorateIncident(&response.Incident)
	return &response.Incident, nil
}

func (c *Client) GetIncident(id string) (*Incident, error) {
	response := incidentResponse{}
	if err := c.execGrafanaIRMRPC("IncidentsService.GetIncident", map[string]string{
		"incidentID": strings.TrimSpace(id),
	}, &response); err != nil {
		return nil, err
	}

	c.decorateIncident(&response.Incident)
	return &response.Incident, nil
}

func (c *Client) UpdateIncident(id string, title, severity *string, isDrill *bool) (*Incident, error) {
	var incident *Incident
	if title != nil {
		updated, err := c.updateIncidentTitle(id, *title)
		if err != nil {
			return nil, err
		}
		incident = updated
	}

	if severity != nil {
		updated, err := c.updateIncidentSeverity(id, *severity)
		if err != nil {
			return nil, err
		}
		incident = updated
	}

	if isDrill != nil {
		updated, err := c.updateIncidentIsDrill(id, *isDrill)
		if err != nil {
			return nil, err
		}
		incident = updated
	}

	if incident == nil {
		return nil, fmt.Errorf("at least one incident field must be provided")
	}

	return incident, nil
}

func (c *Client) ResolveIncident(id string) (*Incident, error) {
	response := incidentResponse{}
	if err := c.execGrafanaIRMRPC("IncidentsService.UpdateStatus", map[string]string{
		"incidentID": strings.TrimSpace(id),
		"status":     incidentStatusResolved,
	}, &response); err != nil {
		return nil, err
	}

	c.decorateIncident(&response.Incident)
	return &response.Incident, nil
}

func (c *Client) AddIncidentActivity(id, body string) (*IncidentActivityItem, error) {
	response := addIncidentActivityResponse{}
	if err := c.execGrafanaIRMRPC("ActivityService.AddActivity", addIncidentActivityRequest{
		IncidentID:   strings.TrimSpace(id),
		ActivityKind: incidentActivityKindUserNote,
		Body:         strings.TrimSpace(body),
	}, &response); err != nil {
		return nil, err
	}

	return &response.ActivityItem, nil
}

func (c *Client) ListIncidents(limit int) ([]IncidentPreview, error) {
	if limit <= 0 {
		limit = 100
	}

	request := queryIncidentPreviewsRequest{}
	request.Query.Limit = limit
	request.Query.OrderField = "createdTime"
	request.Query.OrderDirection = "DESC"

	response := queryIncidentPreviewsResponse{}
	if err := c.execGrafanaIRMRPC("IncidentsService.QueryIncidentPreviews", request, &response); err != nil {
		return nil, err
	}

	for i := range response.IncidentPreviews {
		c.decorateIncidentPreview(&response.IncidentPreviews[i])
	}
	return response.IncidentPreviews, nil
}

func (c *Client) updateIncidentTitle(id, title string) (*Incident, error) {
	response := incidentResponse{}
	if err := c.execGrafanaIRMRPC("IncidentsService.UpdateTitle", map[string]string{
		"incidentID": strings.TrimSpace(id),
		"title":      strings.TrimSpace(title),
	}, &response); err != nil {
		return nil, err
	}

	c.decorateIncident(&response.Incident)
	return &response.Incident, nil
}

func (c *Client) updateIncidentSeverity(id, severity string) (*Incident, error) {
	response := incidentResponse{}
	if err := c.execGrafanaIRMRPC("IncidentsService.UpdateSeverity", map[string]string{
		"incidentID": strings.TrimSpace(id),
		"severity":   strings.TrimSpace(severity),
	}, &response); err != nil {
		return nil, err
	}

	c.decorateIncident(&response.Incident)
	return &response.Incident, nil
}

func (c *Client) updateIncidentIsDrill(id string, isDrill bool) (*Incident, error) {
	response := incidentResponse{}
	if err := c.execGrafanaIRMRPC("IncidentsService.UpdateIncidentIsDrill", map[string]any{
		"incidentID": strings.TrimSpace(id),
		"isDrill":    isDrill,
	}, &response); err != nil {
		return nil, err
	}

	c.decorateIncident(&response.Incident)
	return &response.Incident, nil
}

func (c *Client) execGrafanaIRMRPC(operation string, payload any, response any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error marshaling Grafana IRM payload: %v", err)
	}

	responseBody, status, err := c.execRequest(
		http.MethodPost,
		fmt.Sprintf("%s/%s", grafanaIRMRPCBasePath, operation),
		bytes.NewReader(body),
		"application/json; charset=utf-8",
	)
	if err != nil {
		return fmt.Errorf("error calling Grafana IRM %s: %v", operation, err)
	}
	if status < 200 || status >= 300 {
		return newAPIStatusError(fmt.Sprintf("grafana IRM %s", operation), status, responseBody)
	}

	if err := json.Unmarshal(responseBody, response); err != nil {
		return fmt.Errorf("error parsing Grafana IRM %s response: %v", operation, err)
	}

	var rpcError struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(responseBody, &rpcError); err == nil && strings.TrimSpace(rpcError.Error) != "" {
		return fmt.Errorf("grafana IRM %s failed: %s", operation, strings.TrimSpace(rpcError.Error))
	}

	return nil
}

func (c *Client) decorateIncident(incident *Incident) {
	if incident == nil {
		return
	}

	if strings.TrimSpace(incident.OverviewURL) != "" {
		incident.OverviewURL = c.resolveURL(incident.OverviewURL)
	}
	if strings.TrimSpace(incident.IncidentURL) == "" && strings.TrimSpace(incident.IncidentID) != "" {
		incident.IncidentURL = c.incidentWebURL(incident.IncidentID)
	}
}

func (c *Client) decorateIncidentPreview(incident *IncidentPreview) {
	if incident == nil {
		return
	}

	if strings.TrimSpace(incident.OverviewURL) != "" {
		incident.OverviewURL = c.resolveURL(incident.OverviewURL)
	}
	if strings.TrimSpace(incident.IncidentURL) == "" && strings.TrimSpace(incident.IncidentID) != "" {
		incident.IncidentURL = c.incidentWebURL(incident.IncidentID)
	}
}

func (c *Client) incidentWebURL(id string) string {
	return fmt.Sprintf(
		"%s/a/grafana-irm-app/incidents/%s",
		strings.TrimSuffix(c.BaseURL, "/"),
		url.PathEscape(strings.TrimSpace(id)),
	)
}

func incidentLabelsFromStrings(labels []string) []IncidentLabel {
	out := make([]IncidentLabel, 0, len(labels))
	for _, label := range labels {
		trimmed := strings.TrimSpace(label)
		if trimmed == "" {
			continue
		}
		out = append(out, IncidentLabel{Label: trimmed})
	}
	return out
}
