package grafana

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	grafanaIRMRPCBasePath         = "/api/plugins/grafana-irm-app/resources/api/v1"
	incidentDefaultRoomPrefix     = "incident"
	incidentDefaultLabelKey       = "tags"
	incidentStatusActive          = "active"
	incidentStatusResolved        = "resolved"
	incidentActivityKindUserNote  = "userNote"
	incidentFieldDomainIncident   = "incident"
	incidentFieldSlugDebrief      = "debrief_status"
	incidentFieldTypeString       = "string"
	incidentFieldTypeSingleSelect = "single-select"
	incidentTargetKind            = "incident"
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
	DebriefStatus       string
	InitialStatusUpdate string
	StartTime           *time.Time
}

type IncidentCustomField struct {
	UUID          string                            `json:"uuid"`
	Name          string                            `json:"name,omitempty"`
	Slug          string                            `json:"slug"`
	Type          string                            `json:"type,omitempty"`
	Archived      bool                              `json:"archived,omitempty"`
	SelectOptions []IncidentCustomFieldSelectOption `json:"selectoptions,omitempty"`
}

type IncidentCustomFieldSelectOption struct {
	UUID  string `json:"uuid,omitempty"`
	Value string `json:"value"`
	Label string `json:"label"`
}

type getIncidentCustomFieldsRequest struct {
	DomainName string `json:"domainName,omitempty"`
}

type getIncidentCustomFieldsResponse struct {
	Fields []IncidentCustomField `json:"fields"`
	Error  string                `json:"error,omitempty"`
}

type recordIncidentFieldValueRequest struct {
	FieldUUID  string `json:"fieldUUID"`
	Value      string `json:"value,omitempty"`
	TargetKind string `json:"targetKind"`
	TargetID   string `json:"targetID"`
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

	if err := c.setIncidentDebriefStatus(response.Incident.IncidentID, input.DebriefStatus); err != nil {
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

func (c *Client) UpdateIncident(id string, title, severity *string, labels []string, isDrill *bool) (*Incident, error) {
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

	for _, label := range incidentLabelsFromStrings(labels) {
		if err := c.ensureIncidentLabelValue(label); err != nil {
			return nil, err
		}

		updated, err := c.addIncidentLabel(id, label)
		if err != nil {
			if isDuplicateIncidentLabelError(err) {
				if incident == nil {
					incident = &Incident{IncidentID: strings.TrimSpace(id)}
					c.decorateIncident(incident)
				}
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

func (c *Client) setIncidentDebriefStatus(id, value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	field, err := c.getIncidentCustomFieldBySlug(incidentFieldSlugDebrief)
	if err != nil {
		return err
	}

	normalizedValue, err := normalizeIncidentCustomFieldValue(field, value)
	if err != nil {
		return err
	}

	return c.execGrafanaIRMRPC("FieldsService.RecordFieldValue", recordIncidentFieldValueRequest{
		FieldUUID:  strings.TrimSpace(field.UUID),
		Value:      normalizedValue,
		TargetKind: incidentTargetKind,
		TargetID:   strings.TrimSpace(id),
	}, &struct{}{})
}

func (c *Client) getIncidentCustomFieldBySlug(slug string) (IncidentCustomField, error) {
	response := getIncidentCustomFieldsResponse{}
	if err := c.execGrafanaIRMRPC("FieldsService.GetFields", getIncidentCustomFieldsRequest{
		DomainName: incidentFieldDomainIncident,
	}, &response); err != nil {
		return IncidentCustomField{}, err
	}

	slug = strings.TrimSpace(slug)
	for _, field := range response.Fields {
		if field.Archived {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(field.Slug), slug) {
			return field, nil
		}
	}

	return IncidentCustomField{}, fmt.Errorf("grafana incident custom field %q was not found", slug)
}

func normalizeIncidentCustomFieldValue(field IncidentCustomField, value string) (string, error) {
	value = strings.TrimSpace(value)
	switch strings.TrimSpace(field.Type) {
	case "", incidentFieldTypeString:
		return value, nil
	case incidentFieldTypeSingleSelect:
		for _, option := range field.SelectOptions {
			optionUUID := strings.TrimSpace(option.UUID)
			optionValue := strings.TrimSpace(option.Value)
			optionLabel := strings.TrimSpace(option.Label)
			if strings.EqualFold(value, optionValue) || strings.EqualFold(value, optionLabel) {
				if optionUUID == "" {
					return "", fmt.Errorf(
						"grafana incident custom field %q option %q is missing a UUID",
						strings.TrimSpace(field.Slug),
						optionValue,
					)
				}
				return optionUUID, nil
			}
		}

		options := make([]string, 0, len(field.SelectOptions))
		for _, option := range field.SelectOptions {
			optionValue := strings.TrimSpace(option.Value)
			optionLabel := strings.TrimSpace(option.Label)
			if optionLabel != "" && !strings.EqualFold(optionLabel, optionValue) {
				options = append(options, fmt.Sprintf("%s (%s)", optionLabel, optionValue))
				continue
			}
			if optionValue != "" {
				options = append(options, optionValue)
			}
		}

		if len(options) == 0 {
			return "", fmt.Errorf("grafana incident custom field %q does not define any selectable options", strings.TrimSpace(field.Slug))
		}

		return "", fmt.Errorf(
			"invalid value %q for grafana incident custom field %q; expected one of: %s",
			value,
			strings.TrimSpace(field.Slug),
			strings.Join(options, ", "),
		)
	default:
		return "", fmt.Errorf(
			"grafana incident custom field %q must be type %q or %q, got %q",
			strings.TrimSpace(field.Slug),
			incidentFieldTypeString,
			incidentFieldTypeSingleSelect,
			strings.TrimSpace(field.Type),
		)
	}
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

	message := strings.ToLower(err.Error())
	return strings.Contains(message, "already exists") ||
		strings.Contains(message, "already added") ||
		strings.Contains(message, "duplicate")
}
