package jira

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

// IncidentEventPayloadType is the event type emitted for every matching incident webhook.
const IncidentEventPayloadType = "jira.incident"

type OnIncident struct{}

type OnIncidentConfiguration struct {
	ServiceDesk  string   `json:"serviceDesk" mapstructure:"serviceDesk"`
	RequestTypes []string `json:"requestTypes" mapstructure:"requestTypes"`
	Events       []string `json:"events" mapstructure:"events"`
}

type OnIncidentMetadata struct {
	ServiceDesk  *ServiceDesk `json:"serviceDesk,omitempty" mapstructure:"serviceDesk,omitempty"`
	IssueTypeIDs []string     `json:"issueTypeIds,omitempty" mapstructure:"issueTypeIds,omitempty"`
}

func (t *OnIncident) Name() string {
	return "jira.onIncident"
}

func (t *OnIncident) Label() string {
	return "On Incident"
}

func (t *OnIncident) Description() string {
	return "Listen to incident created, updated, or deleted events in Jira Service Management"
}

func (t *OnIncident) Documentation() string {
	return `The On Incident trigger starts a workflow execution when an incident is created, updated, or deleted on a Jira Service Management service desk.

An incident is a Jira issue on a request type that belongs to the Incident management work
category (the same request types ` + "`jira.createIncident`" + ` can open) - this trigger reuses the
project-wide issue webhook every ` + "`jira.onIssue`" + ` trigger shares, and matches events to the
request types you select here by their underlying issue type.

## Use Cases

- **Incident response automation**: Page a team or open a channel when a new incident is created
- **Status sync**: Mirror incident updates into another tool
- **Reporting**: Track incident volume and resolution over time

## Configuration

- **Service Desk**: The Jira Service Management service desk to listen for incidents on
- **Request Types**: One or more incident request types on that service desk to listen for
- **Events**: Which incident events to listen for (Created, Updated, Deleted)

## Webhook Setup

This is provisioned automatically, sharing the same webhook registration ` + "`jira.onIssue`" + ` uses - see that trigger's documentation for how Jira's dynamic webhook registration works.

## Output

Emits one event per matching incident webhook with:
- **action**: ` + "`created`" + `, ` + "`updated`" + `, or ` + "`deleted`" + `
- **issue**: The full issue (id, key, self, fields)
- **user**: The user who triggered the event
- **changelog**: The list of changed fields (only present for updates)`
}

func (t *OnIncident) Icon() string {
	return "jira"
}

func (t *OnIncident) Color() string {
	return "orange"
}

func (t *OnIncident) ExampleData() map[string]any {
	return onIncidentExampleData()
}

func (t *OnIncident) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "serviceDesk",
			Label:       "Service Desk",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Jira Service Management service desk to listen for incidents on",
			Placeholder: "Select a service desk",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "serviceDesk",
				},
			},
		},
		{
			Name:        "requestTypes",
			Label:       "Request Types",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Incident request types to listen for on the selected service desk",
			Placeholder: "Select one or more request types",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:  "serviceDeskRequestType",
					Multi: true,
					Parameters: []configuration.ParameterRef{
						{
							Name:      "serviceDesk",
							ValueFrom: &configuration.ParameterValueFrom{Field: "serviceDesk"},
						},
					},
				},
			},
		},
		issueEventsField("Which incident events to listen for"),
	}
}

func (t *OnIncident) Setup(ctx core.TriggerContext) error {
	config := OnIncidentConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.ServiceDesk == "" {
		return fmt.Errorf("service desk is required")
	}
	if len(config.RequestTypes) == 0 {
		return fmt.Errorf("at least one request type must be selected")
	}
	if len(config.Events) == 0 {
		return fmt.Errorf("at least one event must be selected")
	}

	desk, err := requireServiceDesk(ctx.HTTP, ctx.Integration, config.ServiceDesk)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	issueTypeIDs := make([]string, 0, len(config.RequestTypes))
	for _, requestTypeID := range config.RequestTypes {
		requestType, err := client.GetRequestType(config.ServiceDesk, requestTypeID)
		if err != nil {
			return fmt.Errorf("failed to resolve request type %s: %w", requestTypeID, err)
		}
		// A request type with no issue type would silently never match anything in HandleWebhook,
		// leaving the trigger looking configured but permanently inert - fail loudly instead.
		if requestType.IssueTypeID == "" {
			return fmt.Errorf("request type %s has no underlying issue type", requestTypeID)
		}
		issueTypeIDs = append(issueTypeIDs, requestType.IssueTypeID)
	}

	if err := ctx.Metadata.Set(OnIncidentMetadata{ServiceDesk: desk, IssueTypeIDs: issueTypeIDs}); err != nil {
		return fmt.Errorf("failed to update metadata: %w", err)
	}

	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		Events: []string{issueEventCreated, issueEventUpdated, issueEventDeleted},
	})
}

func (t *OnIncident) Hooks() []core.Hook {
	return []core.Hook{}
}

func (t *OnIncident) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnIncident) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	config := OnIncidentConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	metadata := OnIncidentMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to decode metadata: %w", err)
	}

	payload := IssueWebhookPayload{}
	if err := json.Unmarshal(ctx.Body, &payload); err != nil {
		return http.StatusBadRequest, nil, fmt.Errorf("error parsing request body: %w", err)
	}

	action, ok := issueEventAction(payload.WebhookEvent)
	if !ok {
		ctx.Logger.Infof("Ignoring event - unsupported webhookEvent %q", payload.WebhookEvent)
		return http.StatusOK, nil, nil
	}

	if !slices.Contains(config.Events, action) {
		ctx.Logger.Infof("Ignoring event - action %q is not configured", action)
		return http.StatusOK, nil, nil
	}

	if payload.Issue == nil {
		ctx.Logger.Info("Ignoring event - missing issue")
		return http.StatusOK, nil, nil
	}

	// Issue type ids are global to the site, not scoped to a project, so a matching type alone
	// doesn't prove the issue belongs to this trigger's service desk - check the project too,
	// failing closed when the payload doesn't carry one to compare.
	if metadata.ServiceDesk != nil && !strings.EqualFold(issueProjectKey(payload.Issue), metadata.ServiceDesk.ProjectKey) {
		ctx.Logger.Infof("Ignoring event - project does not match %q", metadata.ServiceDesk.ProjectKey)
		return http.StatusOK, nil, nil
	}

	// The webhook is shared by every trigger on the integration (see JiraWebhookHandler), so this
	// issue-type check is the only thing keeping one incident trigger from reacting to an
	// unrelated issue - fail closed when the payload doesn't carry an issue type to compare.
	if !slices.Contains(metadata.IssueTypeIDs, issueTypeID(payload.Issue)) {
		ctx.Logger.Info("Ignoring event - issue type does not match a configured request type")
		return http.StatusOK, nil, nil
	}

	event := IssueEvent{
		Action:    action,
		Issue:     payload.Issue,
		User:      payload.User,
		Changelog: payload.Changelog,
	}

	if err := ctx.Events.Emit(IncidentEventPayloadType, event); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("error emitting event: %w", err)
	}

	return http.StatusOK, nil, nil
}

// Cleanup does nothing: the shared Jira webhook registered via RequestWebhook is torn down by
// the platform's webhook cleanup worker (through JiraWebhookHandler.Cleanup) once the last
// trigger referencing it is removed.
func (t *OnIncident) Cleanup(ctx core.TriggerContext) error {
	return nil
}

func issueTypeID(issue *Issue) string {
	if issue == nil || issue.Fields == nil {
		return ""
	}
	issuetype, ok := issue.Fields["issuetype"].(map[string]any)
	if !ok {
		return ""
	}
	id, _ := issuetype["id"].(string)
	return id
}
