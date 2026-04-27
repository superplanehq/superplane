package grafana

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	grafanaIRMRPCBasePath        = "/api/plugins/grafana-irm-app/resources/api/v1"
	incidentDefaultRoomPrefix    = "incident"
	incidentDefaultLabelKey      = "tags"
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

type DeclareIncidentInput struct {
	Title               string
	Severity            string
	Labels              []string
	RoomPrefix          string
	IsDrill             bool
	Status              string
	InitialStatusUpdate string
	StartTime           *time.Time
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

func (c *Client) DeclareIncident(input DeclareIncidentInput) (*Incident, error) {
	roomPrefix := strings.TrimSpace(input.RoomPrefix)
	if roomPrefix == "" {
		roomPrefix = incidentDefaultRoomPrefix
	}

	status := strings.TrimSpace(input.Status)
	if status == "" {
		status = incidentStatusActive
	}

	request := declareIncidentRequest{
		Title:               strings.TrimSpace(input.Title),
		Severity:            strings.TrimSpace(input.Severity),
		Labels:              incidentLabelsFromStrings(input.Labels),
		RoomPrefix:          roomPrefix,
		IsDrill:             input.IsDrill,
		Status:              status,
		InitialStatusUpdate: strings.TrimSpace(input.InitialStatusUpdate),
	}

	response := incidentResponse{}
	if err := c.execGrafanaIRMRPC("IncidentsService.CreateIncident", request, &response); err != nil {
		return nil, err
	}

	if input.StartTime != nil {
		if err := c.updateIncidentEventTime(response.Incident.IncidentID, "incidentStart", *input.StartTime); err != nil {
			return nil, err
		}

		response.Incident.IncidentStart = input.StartTime.UTC().Format(time.RFC3339)
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

func (c *Client) UpdateIncident(id string, title, severity *string, labels []string, isDrill *bool) (*Incident, error) {
	var incident *Incident
	var anyLabelSkipped bool
	normalizedLabels := incidentLabelsFromStrings(labels)
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

	for _, label := range normalizedLabels {
		if err := c.ensureIncidentLabelValue(label); err != nil {
			return nil, err
		}

		updated, err := c.addIncidentLabel(id, label)
		if err != nil {
			if isDuplicateIncidentLabelError(err) {
				anyLabelSkipped = true
				continue
			}
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
		if len(normalizedLabels) > 0 {
			return c.GetIncident(id)
		}
		return nil, fmt.Errorf("at least one incident field must be provided")
	}

	if anyLabelSkipped {
		return c.GetIncident(id)
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

func (c *Client) ensureIncidentLabelValue(label IncidentLabel) error {
	key := strings.TrimSpace(label.Key)
	value := strings.TrimSpace(label.Label)
	if key == "" || value == "" {
		return nil
	}

	response := map[string]any{}
	if err := c.execGrafanaIRMRPC("FieldsService.AddLabelValue", map[string]string{
		"key":   key,
		"value": value,
	}, &response); err != nil {
		if isDuplicateIncidentLabelError(err) {
			return nil
		}
		return err
	}

	return nil
}

func (c *Client) addIncidentLabel(id string, label IncidentLabel) (*Incident, error) {
	response := incidentResponse{}
	if err := c.execGrafanaIRMRPC("IncidentsService.AddLabel", map[string]any{
		"incidentID": strings.TrimSpace(id),
		"label":      label,
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

func (c *Client) updateIncidentEventTime(id, eventName string, eventTime time.Time) error {
	return c.execGrafanaIRMRPC("IncidentsService.UpdateIncidentEventTime", map[string]string{
		"incidentID":       strings.TrimSpace(id),
		"activityItemKind": strings.TrimSpace(eventName),
		"eventName":        strings.TrimSpace(eventName),
		"eventTime":        eventTime.UTC().Format(time.RFC3339),
	}, &struct{}{})
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
		out = append(out, IncidentLabel{Key: incidentDefaultLabelKey, Label: trimmed})
	}
	return dedupeIncidentLabels(out)
}

func dedupeIncidentLabels(labels []IncidentLabel) []IncidentLabel {
	seen := map[string]bool{}
	out := make([]IncidentLabel, 0, len(labels))
	for _, label := range labels {
		key := strings.TrimSpace(label.Key)
		value := strings.TrimSpace(label.Label)
		if value == "" {
			continue
		}

		identity := key + "\x00" + value
		if seen[identity] {
			continue
		}
		seen[identity] = true
		out = append(out, IncidentLabel{Key: key, Label: value})
	}
	return out
}

func isDuplicateIncidentLabelError(err error) bool {
	if err == nil {
		return false
	}

	var apiErr *apiStatusError
	if errors.As(err, &apiErr) && apiErr.StatusCode >= 500 {
		return false
	}

	message := strings.ToLower(err.Error())
	return strings.Contains(message, "already exists") ||
		strings.Contains(message, "already added") ||
		strings.Contains(message, "duplicate")
}
