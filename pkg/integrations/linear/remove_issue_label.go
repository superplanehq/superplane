package linear

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type RemoveIssueLabel struct{}

type RemoveIssueLabelSpec struct {
	Team           string   `json:"team" mapstructure:"team"`
	Issue          string   `json:"issue" mapstructure:"issue"`
	Labels         []string `json:"labels" mapstructure:"labels"`
	FailIfNotFound bool     `json:"failIfNotFound" mapstructure:"failIfNotFound"`
}

func (c *RemoveIssueLabel) Name() string {
	return "linear.removeIssueLabel"
}

func (c *RemoveIssueLabel) Label() string {
	return "Remove Issue Label"
}

func (c *RemoveIssueLabel) Description() string {
	return "Remove labels from an issue in Linear"
}

func (c *RemoveIssueLabel) Documentation() string {
	return `The Remove Issue Label component removes one or more labels from an existing Linear issue,
leaving its other labels untouched.

## Use Cases

- **Status transitions**: Clear ` + "`in-progress`" + ` once a workflow finishes
- **Triage cleanup**: Strip a temporary triage label after processing
- **Undo**: Remove a label a previous step added

## Configuration

- **Team** (required): The Linear team whose labels populate the picker below
- **Issue** (required): The issue to remove labels from, by identifier (e.g. ENG-142) or ID. Supports
  expressions.
- **Labels** (required): One or more labels to remove. Other labels on the issue are kept.
- **Fail if not found** (optional): By default a label that is not on the issue is skipped, so the
  component is safe to re-run. Turn this on to fail the execution instead.

## Output

Returns the updated issue, including its full remaining set of ` + "`labels`" + `.

## Permissions

The user who authorized the Linear connection must be able to edit the issue. SuperPlane's OAuth
connection includes the **write** scope, which covers removing labels.`
}

func (c *RemoveIssueLabel) Icon() string {
	return "linear"
}

func (c *RemoveIssueLabel) Color() string {
	return "indigo"
}

func (c *RemoveIssueLabel) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *RemoveIssueLabel) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "team",
			Label:       "Team",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Linear team whose labels populate the picker below",
			Placeholder: "Select a team",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeTeam,
				},
			},
		},
		{
			Name:        "issue",
			Label:       "Issue",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The issue to remove labels from, by identifier (e.g. ENG-142) or ID",
			Placeholder: "ENG-142",
		},
		{
			Name:        "labels",
			Label:       "Labels",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Labels to remove from the issue. Its other labels are kept",
			Placeholder: "Select labels",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:  ResourceTypeLabel,
					Multi: true,
					Parameters: []configuration.ParameterRef{
						{
							Name:      "team",
							ValueFrom: &configuration.ParameterValueFrom{Field: "team"},
						},
					},
				},
			},
		},
		{
			Name:        "failIfNotFound",
			Label:       "Fail if not found",
			Type:        configuration.FieldTypeBool,
			Description: "Fail the execution when a label is not on the issue, instead of skipping it",
			Default:     false,
		},
	}
}

func (c *RemoveIssueLabel) Setup(ctx core.SetupContext) error {
	spec := RemoveIssueLabelSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if spec.Team == "" {
		return fmt.Errorf("team is required")
	}

	if strings.TrimSpace(spec.Issue) == "" {
		return fmt.Errorf("issue is required")
	}

	if len(trimmedLabels(spec.Labels)) == 0 {
		return fmt.Errorf("at least one label is required")
	}

	team, err := requireTeam(ctx.Integration, spec.Team)
	if err != nil {
		return err
	}

	return ctx.Metadata.Set(NodeMetadata{Team: team})
}

func (c *RemoveIssueLabel) Execute(ctx core.ExecutionContext) error {
	spec := RemoveIssueLabelSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	issueID := strings.TrimSpace(spec.Issue)
	if issueID == "" {
		return fmt.Errorf("issue is required")
	}

	labels := dedupeStrings(trimmedLabels(spec.Labels))
	if len(labels) == 0 {
		return fmt.Errorf("at least one label is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}

	//
	// Linear removes one label per call. A label already off the issue is the
	// desired end state, so skipping it keeps re-runs and retries working -
	// unless the user asked to hear about it.
	//
	var issue *Issue
	for _, labelID := range labels {
		updated, err := client.RemoveIssueLabel(issueID, labelID)
		if err != nil {
			if spec.FailIfNotFound || !isLabelNotOnIssue(err) {
				return fmt.Errorf("failed to remove label: %v", err)
			}

			ctx.Logger.Infof("Label %s is not on issue %s - skipping", labelID, issueID)
			continue
		}

		issue = updated
	}

	//
	// Every label was already absent, so nothing came back to emit. Fetch the
	// issue so the output shape stays the same either way.
	//
	if issue == nil {
		issue, err = client.GetIssue(issueID)
		if err != nil {
			return fmt.Errorf("failed to get issue: %v", err)
		}
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		IssuePayloadType,
		[]any{issue},
	)
}

// isLabelNotOnIssue reports whether an error is Linear refusing to remove a
// label the issue does not carry. Linear returns no error code for this, so the
// message is the only signal.
func isLabelNotOnIssue(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "label not on issue")
}

func (c *RemoveIssueLabel) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *RemoveIssueLabel) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *RemoveIssueLabel) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *RemoveIssueLabel) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *RemoveIssueLabel) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *RemoveIssueLabel) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
