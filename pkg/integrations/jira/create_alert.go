package jira

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const CreateJiraAlertPayloadType = "jira.alert.created"

type CreateAlert struct{}

type ResponderPairSpec struct {
	ID   string `json:"id" mapstructure:"id"`
	Type string `json:"type" mapstructure:"type"`
}

// CreateAlertSpec configures JSM Ops Alerts API create (see Atlassian JSM Ops Alerts REST reference).
type CreateAlertSpec struct {
	Message string `json:"message" mapstructure:"message"`

	Description string `json:"description,omitempty" mapstructure:"description"`
	Note        string `json:"note,omitempty" mapstructure:"note"`
	Alias       string `json:"alias,omitempty" mapstructure:"alias"`
	Entity      string `json:"entity,omitempty" mapstructure:"entity"`
	Source      string `json:"source,omitempty" mapstructure:"source"`
	Priority    string `json:"priority,omitempty" mapstructure:"priority"`

	Tags            []any               `json:"tags,omitempty" mapstructure:"tags"`
	Actions         []any               `json:"actions,omitempty" mapstructure:"actions"`
	Responders      []ResponderPairSpec `json:"responders,omitempty" mapstructure:"responders"`
	VisibleTo       []ResponderPairSpec `json:"visibleTo,omitempty" mapstructure:"visibleTo"`
	ExtraProperties map[string]any      `json:"extraProperties,omitempty" mapstructure:"extraProperties"`
}

func (c *CreateAlert) Name() string {
	return "jira.createAlert"
}

func (c *CreateAlert) Label() string {
	return "Create Alert"
}

func (c *CreateAlert) Description() string {
	return "Create a Jira Service Management Ops alert via the Alerts API"
}

func (c *CreateAlert) Documentation() string {
	return `The Create Alert component opens a new alert on Jira Service Management.

## Use Cases

- **Monitoring integrations**: Raise an alert from metrics or logs
- **Automation**: Drive on-call notifications from workflows

## Configuration

- **Message** (required): Alert message text.
- **Description**, **Note**, **Alias**, **Entity**, **Source** (optional): Standard Ops alert fields.
- **Priority** (optional): Typically P1–P5; choose "Don't set" to omit.
- **Tags** / **Actions** (optional): Lists of strings.
- **Responders** / **Visible to** (optional): Rows with **id** and **type** (team, user, escalation, or schedule).
- **Extra properties** (optional): JSON object merged into the API **extraProperties** field.

## Output

After the Ops API accepts create, SuperPlane waits for asynchronous processing via the alerts **request status** API, then **GET**s the resulting alert.

The payload matches **Get Alert** (full Ops alert JSON: **id**, **message**, **status**, etc.).

If Jira delays processing beyond the polling window the step fails — use **Get Alert** with the Ops request id noted in logs if Atlassian exposes it.

## Notes

- Requires Jira Service Management Ops permissions and API scopes for alerts on your Atlassian site; the integration stores your site **cloud id** during sync.`
}

func (c *CreateAlert) Icon() string {
	return "jira"
}

func (c *CreateAlert) Color() string {
	return "orange"
}

func (c *CreateAlert) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateAlert) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "message",
			Label:       "Message",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Alert message (required by the Ops Alerts API)",
			Placeholder: "Short description of the condition",
		},
		{
			Name:        "description",
			Label:       "Description",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Description: "Optional longer description for the alert",
		},
		{
			Name:        "note",
			Label:       "Note",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Description: "Optional note included when the alert is created",
		},
		{
			Name:        "alias",
			Label:       "Alias",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional deduplication alias",
		},
		{
			Name:        "entity",
			Label:       "Entity",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional entity name (for example host or service)",
		},
		{
			Name:        "source",
			Label:       "Source",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional source system name",
		},
		{
			Name:        "priority",
			Label:       "Priority",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Default:     "__none__",
			Description: "Optional Ops alert priority",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Don't set", Value: "__none__"},
						{Label: "P1", Value: "P1"},
						{Label: "P2", Value: "P2"},
						{Label: "P3", Value: "P3"},
						{Label: "P4", Value: "P4"},
						{Label: "P5", Value: "P5"},
					},
				},
			},
		},
		{
			Name:        "tags",
			Label:       "Tags",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Optional tag strings",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Tag",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
		{
			Name:        "actions",
			Label:       "Actions",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Optional custom action names",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Action",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
		{
			Name:        "responders",
			Label:       "Responders",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Optional responders (id + type)",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Responder",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{Name: "id", Label: "ID", Type: configuration.FieldTypeString, Required: true},
							{
								Name: "type", Label: "Type", Type: configuration.FieldTypeSelect, Required: true,
								TypeOptions: &configuration.TypeOptions{
									Select: &configuration.SelectTypeOptions{
										Options: []configuration.FieldOption{
											{Label: "Team", Value: "team"},
											{Label: "User", Value: "user"},
											{Label: "Escalation", Value: "escalation"},
											{Label: "Schedule", Value: "schedule"},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			Name:        "visibleTo",
			Label:       "Visible to",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Optional visibility entries (id + type)",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Entry",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{Name: "id", Label: "ID", Type: configuration.FieldTypeString, Required: true},
							{
								Name: "type", Label: "Type", Type: configuration.FieldTypeSelect, Required: true,
								TypeOptions: &configuration.TypeOptions{
									Select: &configuration.SelectTypeOptions{
										Options: []configuration.FieldOption{
											{Label: "Team", Value: "team"},
											{Label: "User", Value: "user"},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			Name:        "extraProperties",
			Label:       "Extra properties",
			Type:        configuration.FieldTypeObject,
			Required:    false,
			Description: "Optional JSON object merged into extraProperties",
			Default:     "{}",
		},
	}
}

func (c *CreateAlert) Setup(ctx core.SetupContext) error {
	spec := CreateAlertSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	if _, err := cloudIDFromIntegration(ctx.Integration); err != nil {
		return err
	}
	if strings.TrimSpace(spec.Message) == "" {
		return fmt.Errorf("message is required")
	}
	return nil
}

func (c *CreateAlert) Execute(ctx core.ExecutionContext) error {
	spec := CreateAlertSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	cloudID, err := cloudIDFromIntegration(ctx.Integration)
	if err != nil {
		return err
	}
	apiBody := opsCreateAlertRequestFromSpec(spec)
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	resp, err := client.CreateOpsAlert(cloudID, apiBody)
	if err != nil {
		return fmt.Errorf("failed to create alert: %w", err)
	}
	if strings.TrimSpace(resp.RequestID) == "" {
		return fmt.Errorf("create alert API accepted the request but returned no requestId for async tracking")
	}
	resolvedID, err := client.ResolveAlertIDAfterOpsRequest(cloudID, resp.RequestID, "")
	if err != nil {
		return fmt.Errorf("failed to resolve created alert after async processing: %w", err)
	}
	alertDetails, err := client.GetOpsAlert(cloudID, resolvedID)
	if err != nil {
		return fmt.Errorf("failed to load created alert: %w", err)
	}
	if err := validateCreatedOpsAlertMatchesSpec(spec, alertDetails); err != nil {
		return err
	}
	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		CreateJiraAlertPayloadType,
		[]any{alertDetails},
	)
}

func (c *CreateAlert) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateAlert) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateAlert) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *CreateAlert) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *CreateAlert) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *CreateAlert) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

// validateCreatedOpsAlertMatchesSpec ensures the alert returned after create matches what we asked for.
// Jira Ops deduplicates open alerts by alias, which can surface an existing alert instead of a new one.
func validateCreatedOpsAlertMatchesSpec(spec CreateAlertSpec, alert map[string]any) error {
	wantMsg := strings.TrimSpace(spec.Message)
	gotMsg := strings.TrimSpace(opsAlertStringField(alert, "message"))
	if wantMsg == "" || gotMsg == wantMsg {
		return nil
	}

	wantAlias := strings.TrimSpace(spec.Alias)
	gotAlias := strings.TrimSpace(opsAlertStringField(alert, "alias"))
	if wantAlias != "" && gotAlias == wantAlias {
		return fmt.Errorf(
			"Jira Ops did not create a new alert: an open alert with alias %q already exists (its message is %q, not %q). Use a unique alias or close the existing alert",
			wantAlias,
			gotMsg,
			wantMsg,
		)
	}

	return fmt.Errorf(
		"loaded alert message %q does not match the requested message %q; creation may have failed or returned a different alert",
		gotMsg,
		wantMsg,
	)
}

func opsCreateAlertRequestFromSpec(spec CreateAlertSpec) *OpsCreateAlertRequest {
	out := &OpsCreateAlertRequest{
		Message: strings.TrimSpace(spec.Message),
	}
	if s := strings.TrimSpace(spec.Description); s != "" {
		out.Description = s
	}
	if s := strings.TrimSpace(spec.Note); s != "" {
		out.Note = s
	}
	if s := strings.TrimSpace(spec.Alias); s != "" {
		out.Alias = s
	}
	if s := strings.TrimSpace(spec.Entity); s != "" {
		out.Entity = s
	}
	if s := strings.TrimSpace(spec.Source); s != "" {
		out.Source = s
	}
	if p := strings.TrimSpace(spec.Priority); p != "" && p != "__none__" {
		out.Priority = p
	}
	if tags := stringSliceFromAny(spec.Tags); len(tags) > 0 {
		out.Tags = tags
	}
	if actions := stringSliceFromAny(spec.Actions); len(actions) > 0 {
		out.Actions = actions
	}
	if r := filterResponderPairs(spec.Responders); len(r) > 0 {
		out.Responders = r
	}
	if r := filterResponderPairs(spec.VisibleTo); len(r) > 0 {
		out.VisibleTo = r
	}
	if len(spec.ExtraProperties) > 0 {
		out.ExtraProperties = spec.ExtraProperties
	}
	return out
}

func stringSliceFromAny(items []any) []string {
	if len(items) == 0 {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, e := range items {
		s := strings.TrimSpace(fmt.Sprint(e))
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func filterResponderPairs(rows []ResponderPairSpec) []OpsAlertResponder {
	if len(rows) == 0 {
		return nil
	}
	out := make([]OpsAlertResponder, 0, len(rows))
	for _, row := range rows {
		id := strings.TrimSpace(row.ID)
		if id == "" {
			continue
		}
		t := strings.TrimSpace(row.Type)
		if t == "" {
			continue
		}
		out = append(out, OpsAlertResponder{ID: id, Type: t})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
