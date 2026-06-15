package jira

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const CreateJiraIncidentPayloadType = "jira.incident.created"

type CreateIncident struct{}

// CreateIncidentSpec configures JSM Incidents API create (see Atlassian JSM Incidents REST reference).
type CreateIncidentSpec struct {
	ServiceDesk            string `json:"serviceDesk" mapstructure:"serviceDesk"`
	ServiceDeskRequestType string `json:"serviceDeskRequestType" mapstructure:"serviceDeskRequestType"`
	Summary                string `json:"summary" mapstructure:"summary"`

	Description      string `json:"description,omitempty" mapstructure:"description"`
	DueDate          string `json:"dueDate,omitempty" mapstructure:"dueDate"`
	Priority         string `json:"priority,omitempty" mapstructure:"priority"`
	Impact           string `json:"impact,omitempty" mapstructure:"impact"`
	Urgency          string `json:"urgency,omitempty" mapstructure:"urgency"`
	OriginalEstimate string `json:"originalEstimate,omitempty" mapstructure:"originalEstimate"`

	AdditionalFields map[string]any `json:"additionalFields,omitempty" mapstructure:"additionalFields"`
	Update           map[string]any `json:"update,omitempty" mapstructure:"update"`
	AlertIDs         []any          `json:"alertIds,omitempty" mapstructure:"alertIds"`

	CustomFieldValues []IncidentCustomFieldRow `json:"customFieldValues,omitempty" mapstructure:"customFieldValues"`
}

// IncidentCustomFieldRow maps one Jira field id to a JSON value (object or string) as expected by the Jira API.
type IncidentCustomFieldRow struct {
	FieldID   string `json:"fieldId" mapstructure:"fieldId"`
	ValueJSON string `json:"valueJson" mapstructure:"valueJson"`
}

func (c *CreateIncident) Name() string {
	return "jira.createIncident"
}

func (c *CreateIncident) Label() string {
	return "Create Incident"
}

func (c *CreateIncident) Description() string {
	return "Create a Jira Service Management incident via the Incidents API"
}

func (c *CreateIncident) Documentation() string {
	return `The Create Incident component opens a new incident in Jira Service Management.

## Use Cases

- **Alert-driven incidents**: Create an incident from a monitoring or ticketing workflow
- **Cross-tool orchestration**: Open a JSM incident when another system reports an outage
- **Responder assignment**: Pass responders and other fields supported by your service desk

## Configuration

- **Service desk**: Choose a Jira Service Management service desk (from your site via the JSM API)
- **Request type**: Choose a request type on that service desk (lists after a service desk is selected)
- **Summary** (required): Short title for the incident; sent as the Jira **summary** field (Jira requires this). You can set this directly, or supply summary only via **Additional fields**.
- **Description** (optional): Plain text stored as Jira description (Atlassian Document Format).
- **Due date** (optional): Jira **duedate** (use your site date format, typically YYYY-MM-DD).
- **Priority** (optional): Jira priority **name** (for example Medium).
- **Impact** (optional): Impact level from your Jira request type (options load after service desk and request type are selected).
- **Urgency** (optional): Urgency level from your Jira request type (options load after service desk and request type are selected).
- **Original estimate** (optional): Jira **timetracking.originalEstimate** (for example 2h, 1d).
- **Custom fields** (optional): List of Jira field ids with JSON values—for affected services, pending reason, or any customfield_*; the value must be valid JSON (object or string) as Jira expects for that field type.
- **Additional fields** (optional): JSON object merged into the API **fields** map (same pattern as other integrations such as Honeycomb **Fields JSON**).
- **Update** (optional): JSON object for the Incidents API **update** property.
- **Alert IDs** (optional): List of alert id strings to associate with the incident.

## Output

Returns **id** (numeric issue id), **key** (e.g. ITSM-30), and **self** (issue REST URL) from the create response.

## Notes

- Requires a Jira Cloud site with Jira Service Management and a synced SuperPlane Jira integration (cloud id is resolved automatically).
- **Request types** must belong to Jira's **Incident management** work category; other request types (for example service requests) are hidden from the picker when Jira returns a practice field on request types. If your site uses an unrecognized practice value, contact support with a sample from GET /rest/servicedeskapi/servicedesk/{serviceDeskId}/requesttype/{requestTypeId} with expand=practice.
- Field ids such as responders are site-specific; use **Custom fields** or **Additional fields** with values that match your JSM configuration.`
}

func (c *CreateIncident) Icon() string {
	return "jira"
}

func (c *CreateIncident) Color() string {
	return "orange"
}

func (c *CreateIncident) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateIncident) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "serviceDesk",
			Label:       "Service Desk",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Jira Service Management service desk for the new incident",
			Placeholder: "Select a service desk",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "serviceDesk",
				},
			},
		},
		{
			Name:        "serviceDeskRequestType",
			Label:       "Service Desk Request Type",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Incident request type on the selected service desk",
			Placeholder: "Select a request type",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "serviceDeskRequestType",
					Parameters: []configuration.ParameterRef{
						{
							Name:      "serviceDesk",
							ValueFrom: &configuration.ParameterValueFrom{Field: "serviceDesk"},
						},
					},
				},
			},
		},
		{
			Name:        "summary",
			Label:       "Summary",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Incident title (Jira summary). Required unless additional fields include summary.",
			Placeholder: "Brief description of the incident",
		},
		{
			Name:        "description",
			Label:       "Description",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Description: "Optional plain-text description (sent as Jira description)",
			Placeholder: "What happened, scope, links…",
		},
		{
			Name:        "dueDate",
			Label:       "Due date",
			Type:        configuration.FieldTypeDate,
			Required:    false,
			Description: "Optional Jira due date",
		},
		{
			Name:        "priority",
			Label:       "Priority",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Default:     "__none__",
			Description: "Optional Jira priority name",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Don't set", Value: "__none__"},
						{Label: "Highest", Value: "Highest"},
						{Label: "High", Value: "High"},
						{Label: "Medium", Value: "Medium"},
						{Label: "Low", Value: "Low"},
						{Label: "Lowest", Value: "Lowest"},
					},
				},
			},
		},
		{
			Name:        "impact",
			Label:       "Impact",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Optional incident impact (options from the selected request type)",
			Placeholder: "Select impact",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "impact",
					Parameters: []configuration.ParameterRef{
						{
							Name:      "serviceDesk",
							ValueFrom: &configuration.ParameterValueFrom{Field: "serviceDesk"},
						},
						{
							Name:      "serviceDeskRequestType",
							ValueFrom: &configuration.ParameterValueFrom{Field: "serviceDeskRequestType"},
						},
					},
				},
			},
		},
		{
			Name:        "urgency",
			Label:       "Urgency",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Optional incident urgency (options from the selected request type)",
			Placeholder: "Select urgency",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "urgency",
					Parameters: []configuration.ParameterRef{
						{
							Name:      "serviceDesk",
							ValueFrom: &configuration.ParameterValueFrom{Field: "serviceDesk"},
						},
						{
							Name:      "serviceDeskRequestType",
							ValueFrom: &configuration.ParameterValueFrom{Field: "serviceDeskRequestType"},
						},
					},
				},
			},
		},
		{
			Name:        "originalEstimate",
			Label:       "Original estimate",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional Jira timetracking original estimate (e.g. 2h, 1d, 30m)",
			Placeholder: "e.g. 2h",
		},
		{
			Name:        "alertIds",
			Label:       "Alert IDs",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Optional alert ids to associate with the incident",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Alert ID",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
		{
			Name:        "customFieldValues",
			Label:       "Custom fields",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Per-field JSON values for affected services, pending reason, or any customfield_* id",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Field",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:        "fieldId",
								Label:       "Field ID",
								Type:        configuration.FieldTypeString,
								Required:    false,
								Placeholder: "customfield_10001",
							},
							{
								Name:        "valueJson",
								Label:       "Value (JSON)",
								Type:        configuration.FieldTypeText,
								Required:    false,
								Description: "JSON value Jira expects for that field",
								Placeholder: `{"name":"High"}`,
								TypeOptions: &configuration.TypeOptions{
									Text: &configuration.TextTypeOptions{
										Language: "json",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			Name:        "additionalFields",
			Label:       "Additional fields",
			Type:        configuration.FieldTypeObject,
			Required:    false,
			Description: "Optional JSON object merged into Jira fields. Use Custom fields above for known customfield ids with structured rows.",
			Default:     "{}",
		},
		{
			Name:        "update",
			Label:       "Update",
			Type:        configuration.FieldTypeObject,
			Required:    false,
			Description: "Optional JSON object for the Incidents API update property",
			Default:     "{}",
		},
	}
}

func (c *CreateIncident) Setup(ctx core.SetupContext) error {
	spec := CreateIncidentSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if _, err := cloudIDFromIntegration(ctx.Integration); err != nil {
		return err
	}

	if strings.TrimSpace(spec.ServiceDesk) == "" {
		return fmt.Errorf("serviceDesk is required")
	}
	if strings.TrimSpace(spec.ServiceDeskRequestType) == "" {
		return fmt.Errorf("serviceDeskRequestType is required")
	}

	nodeMeta := CreateIncidentNodeMetadata{}
	fieldsPreview, err := incidentCreateFieldsFromSpec(spec, nodeMeta)
	if err != nil {
		return err
	}
	if err := validateIncidentSummaryPresent(spec.Summary, fieldsPreview); err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	svcID := strings.TrimSpace(spec.ServiceDesk)
	desks, err := client.ListServiceDesks()
	if err != nil {
		return fmt.Errorf("list service desks (JSM): %w", err)
	}

	var deskName string
	var projectKey string
	if !slices.ContainsFunc(desks, func(d ServiceDesk) bool {
		if d.ID == svcID {
			deskName = serviceDeskDisplayName(d)
			projectKey = strings.TrimSpace(d.ProjectKey)
			return true
		}
		return false
	}) {
		return fmt.Errorf("service desk %q not found or not accessible with these credentials", svcID)
	}

	requestTypes, err := client.ListRequestTypes(svcID)
	if err != nil {
		return fmt.Errorf("list request types: %w", err)
	}

	reqID := strings.TrimSpace(spec.ServiceDeskRequestType)
	if !slices.ContainsFunc(requestTypes, func(rt RequestType) bool { return rt.ID == reqID }) {
		return fmt.Errorf("request type %q not found on service desk %s", reqID, svcID)
	}

	rtDetail, err := client.GetRequestType(svcID, reqID)
	if err != nil {
		return fmt.Errorf("load request type: %w", err)
	}
	practice := strings.TrimSpace(rtDetail.Practice)
	if practice != "" && !IsIncidentManagementRequestPractice(practice) {
		return fmt.Errorf(
			`request type %q (id %s) has practice %q, which is not in Jira's "Incident management" work category; the Incidents API only accepts incident request types — assign it under Space settings > Request management, or choose another type (see https://support.atlassian.com/jira-service-management-cloud/docs/assign-request-types-to-an-it-service-management-category/)`,
			rtDetail.Name,
			reqID,
			practice,
		)
	}

	rtFields, err := client.ListRequestTypeFields(svcID, reqID)
	if err != nil {
		return fmt.Errorf("list request type fields: %w", err)
	}

	nodeMeta = CreateIncidentNodeMetadata{
		ServiceDeskName: deskName,
		RequestTypeName: strings.TrimSpace(rtDetail.Name),
		ImpactFieldID:   resolveIncidentFieldID(client, rtFields, projectKey, "impact"),
		UrgencyFieldID:  resolveIncidentFieldID(client, rtFields, projectKey, "urgency"),
	}

	return ctx.Metadata.Set(nodeMeta)
}

func serviceDeskDisplayName(d ServiceDesk) string {
	if d.ProjectKey != "" {
		return fmt.Sprintf("%s (%s)", d.ProjectName, d.ProjectKey)
	}
	return d.ProjectName
}

func (c *CreateIncident) Execute(ctx core.ExecutionContext) error {
	spec := CreateIncidentSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	cloudID, err := cloudIDFromIntegration(ctx.Integration)
	if err != nil {
		return err
	}

	nodeMeta := CreateIncidentNodeMetadata{}
	if ctx.NodeMetadata != nil {
		_ = mapstructure.Decode(ctx.NodeMetadata.Get(), &nodeMeta)
	}

	fields, err := incidentCreateFieldsFromSpec(spec, nodeMeta)
	if err != nil {
		return err
	}
	summary, err := resolveIncidentSummary(spec.Summary, fields)
	if err != nil {
		return err
	}
	fields["summary"] = summary

	update := incidentUpdateFromSpec(spec)
	alertIDs := incidentAlertIDsFromSpec(spec)

	apiReq := &CreateIncidentAPIRequest{
		ServiceDeskID: strings.TrimSpace(spec.ServiceDesk),
		RequestTypeID: strings.TrimSpace(spec.ServiceDeskRequestType),
		Fields:        fields,
	}
	if len(update) > 0 {
		apiReq.Update = update
	}
	if len(alertIDs) > 0 {
		apiReq.AlertIDs = alertIDs
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	resp, err := client.CreateIncident(cloudID, apiReq)
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "Incident management work category") {
			return fmt.Errorf(
				`failed to create incident: %w — choose a request type assigned to "Incident management" in Jira (see https://support.atlassian.com/jira-service-management-cloud/docs/assign-request-types-to-an-it-service-management-category/)`,
				err,
			)
		}
		if strings.Contains(msg, "summary") && strings.Contains(strings.ToLower(msg), "must specify") {
			return fmt.Errorf("failed to create incident: %w — set the Summary field (it is sent as fields.summary to Jira)", err)
		}
		return fmt.Errorf("failed to create incident: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		CreateJiraIncidentPayloadType,
		[]any{resp},
	)
}

func (c *CreateIncident) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateIncident) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateIncident) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *CreateIncident) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *CreateIncident) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *CreateIncident) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func summaryFromFieldsMap(fields map[string]any) string {
	raw, ok := fields["summary"]
	if !ok || raw == nil {
		return ""
	}
	switch s := raw.(type) {
	case string:
		return strings.TrimSpace(s)
	default:
		return strings.TrimSpace(fmt.Sprint(s))
	}
}

func validateIncidentSummaryPresent(summaryField string, fields map[string]any) error {
	if strings.TrimSpace(summaryField) != "" {
		return nil
	}
	if summaryFromFieldsMap(fields) != "" {
		return nil
	}
	return fmt.Errorf("summary is required; set the Summary field or include summary in Additional fields")
}

func resolveIncidentSummary(summaryField string, fields map[string]any) (string, error) {
	s := strings.TrimSpace(summaryField)
	if s != "" {
		return s, nil
	}
	if v := summaryFromFieldsMap(fields); v != "" {
		return v, nil
	}
	return "", fmt.Errorf("summary is required; set the Summary field or include summary in Additional fields")
}

func mergeStringMaps(dst, src map[string]any) {
	for k, v := range src {
		if k == "" {
			continue
		}
		dst[k] = v
	}
}

func incidentCreateFieldsFromSpec(spec CreateIncidentSpec, meta CreateIncidentNodeMetadata) (map[string]any, error) {
	fields := map[string]any{}

	if len(spec.AdditionalFields) > 0 {
		mergeStringMaps(fields, spec.AdditionalFields)
	}

	if s := strings.TrimSpace(spec.Description); s != "" {
		fields["description"] = WrapInADF(s)
	}
	if s := strings.TrimSpace(spec.DueDate); s != "" {
		fields["duedate"] = s
	}
	if s := strings.TrimSpace(spec.Priority); s != "" && s != "__none__" {
		fields["priority"] = map[string]any{"name": s}
	}
	if s := strings.TrimSpace(spec.OriginalEstimate); s != "" {
		fields["timetracking"] = map[string]any{"originalEstimate": s}
	}

	if fid := strings.TrimSpace(meta.ImpactFieldID); fid != "" {
		if v := jiraOptionFieldValueByID(spec.Impact); v != nil {
			fields[fid] = v
		}
	}
	if fid := strings.TrimSpace(meta.UrgencyFieldID); fid != "" {
		if v := jiraOptionFieldValueByID(spec.Urgency); v != nil {
			fields[fid] = v
		}
	}

	for i, row := range spec.CustomFieldValues {
		fid := strings.TrimSpace(row.FieldID)
		if fid == "" {
			continue
		}
		raw := strings.TrimSpace(row.ValueJSON)
		if raw == "" {
			continue
		}
		var parsed any
		if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
			return nil, fmt.Errorf("customFieldValues[%d].valueJson for %q: %w", i, fid, err)
		}
		fields[fid] = parsed
	}

	return fields, nil
}

func jiraOptionFieldValueByID(raw string) map[string]any {
	s := strings.TrimSpace(raw)
	if s == "" {
		return nil
	}
	return map[string]any{"id": s}
}

func incidentUpdateFromSpec(spec CreateIncidentSpec) map[string]any {
	if len(spec.Update) == 0 {
		return nil
	}
	return spec.Update
}

func incidentAlertIDsFromSpec(spec CreateIncidentSpec) []string {
	return alertIDsFromSlice(spec.AlertIDs)
}

func alertIDsFromSlice(raw []any) []string {
	if len(raw) == 0 {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, e := range raw {
		s := strings.TrimSpace(fmt.Sprint(e))
		if s != "" {
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
