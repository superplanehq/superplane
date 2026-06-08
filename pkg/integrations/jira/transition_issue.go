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

const TransitionIssuePayloadType = "jira.issue"

type TransitionIssue struct{}

type TransitionIssueSpec struct {
	Project      string `json:"project" mapstructure:"project"`
	IssueKey     string `json:"issueKey" mapstructure:"issueKey"`
	TargetStatus string `json:"targetStatus" mapstructure:"targetStatus"`
	Comment      string `json:"comment" mapstructure:"comment"`
	Resolution   string `json:"resolution" mapstructure:"resolution"`
}

func (c *TransitionIssue) Name() string {
	return "jira.transitionIssue"
}

func (c *TransitionIssue) Label() string {
	return "Transition Issue"
}

func (c *TransitionIssue) Description() string {
	return "Move a Jira issue to a reachable workflow status"
}

func (c *TransitionIssue) Documentation() string {
	return `The Transition Issue component moves a Jira issue through its workflow.

## Use Cases

- **Automated triage**: move issues into the next workflow status after a SuperPlane event
- **Cross-tool state sync**: mirror status changes from incident or deployment systems
- **Resolution automation**: close issues with a transition-scoped resolution and comment

## Configuration

- **Project**: Optional Jira project used to narrow the status picker.
- **Issue Key**: Jira issue key, for example ` + "`PROJ-123`" + `.
- **Target Status**: Status to move the issue to. It must be reachable from the issue's current status.
- **Comment**: Optional transition comment.
- **Resolution**: Optional Jira resolution name to set during the transition.

## Output

Returns the refreshed Jira issue after the transition.

## Notes

- Jira does not allow direct status writes. This component finds an available transition whose target status matches the requested status.
- Workflow conditions and validators still apply.`
}

func (c *TransitionIssue) Icon() string {
	return "jira"
}

func (c *TransitionIssue) Color() string {
	return "blue"
}

func (c *TransitionIssue) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *TransitionIssue) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "project",
			Label:       "Project",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Optional project to narrow the status picker",
			Placeholder: "Any project",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{Type: "project"},
			},
		},
		{
			Name:        "issueKey",
			Label:       "Issue Key",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The issue key (e.g. PROJ-123)",
			Placeholder: "PROJ-123",
		},
		{
			Name:        "targetStatus",
			Label:       "Target Status",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Workflow status to transition to",
			Placeholder: "Select a status",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "issueStatus",
					UseNameAsValue: true,
					Parameters: []configuration.ParameterRef{
						{
							Name:      "project",
							ValueFrom: &configuration.ParameterValueFrom{Field: "project"},
						},
					},
				},
			},
		},
		{
			Name:        "comment",
			Label:       "Comment",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Description: "Optional comment to add during the transition",
		},
		{
			Name:        "resolution",
			Label:       "Resolution",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Optional Jira resolution to set during the transition",
			Placeholder: "Leave empty to keep the current resolution",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "resolution",
					UseNameAsValue: true,
				},
			},
		},
	}
}

func (c *TransitionIssue) Setup(ctx core.SetupContext) error {
	spec := TransitionIssueSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if strings.TrimSpace(spec.IssueKey) == "" {
		return fmt.Errorf("issueKey is required")
	}
	if strings.TrimSpace(spec.TargetStatus) == "" {
		return fmt.Errorf("targetStatus is required")
	}

	meta := NodeMetadata{Status: strings.TrimSpace(spec.TargetStatus)}
	if strings.TrimSpace(spec.Project) != "" {
		project, err := requireProject(ctx.HTTP, ctx.Integration, strings.TrimSpace(spec.Project))
		if err != nil {
			return err
		}
		meta.Project = project
	}

	return ctx.Metadata.Set(meta)
}

func (c *TransitionIssue) Execute(ctx core.ExecutionContext) error {
	spec := TransitionIssueSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	issueKey := strings.TrimSpace(spec.IssueKey)
	targetStatus := strings.TrimSpace(spec.TargetStatus)
	if issueKey == "" {
		return fmt.Errorf("issueKey is required")
	}
	if targetStatus == "" {
		return fmt.Errorf("targetStatus is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}

	if err := applyStatusWithOptions(client, issueKey, targetStatus, DoTransitionOptions{
		Comment:    spec.Comment,
		Resolution: spec.Resolution,
	}); err != nil {
		return fmt.Errorf("failed to transition issue: %v", err)
	}

	issue, err := client.GetIssue(issueKey)
	if err != nil {
		return fmt.Errorf("failed to fetch transitioned issue: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		TransitionIssuePayloadType,
		[]any{issue},
	)
}

func (c *TransitionIssue) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *TransitionIssue) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *TransitionIssue) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *TransitionIssue) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *TransitionIssue) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *TransitionIssue) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
