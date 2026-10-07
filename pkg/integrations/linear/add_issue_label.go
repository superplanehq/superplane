package linear

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type AddIssueLabel struct{}

type AddIssueLabelSpec struct {
	Team      string   `json:"team" mapstructure:"team"`
	Issue     string   `json:"issue" mapstructure:"issue"`
	Labels    []string `json:"labels" mapstructure:"labels"`
	NewLabels []string `json:"newLabels" mapstructure:"newLabels"`
}

func (c *AddIssueLabel) Name() string {
	return "linear.addIssueLabel"
}

func (c *AddIssueLabel) Label() string {
	return "Add Issue Label"
}

func (c *AddIssueLabel) Description() string {
	return "Add labels to an issue in Linear"
}

func (c *AddIssueLabel) Documentation() string {
	return `The Add Issue Label component adds one or more labels to an existing Linear issue,
keeping any labels already on it.

## Use Cases

- **Triage automation**: Label issues automatically based on workflow signals
- **Status tracking**: Add labels as an issue moves through a workflow
- **Cross-tool sync**: Apply labels from another system onto a Linear issue

## Configuration

- **Team** (required): The Linear team whose labels populate the picker. Select the team the
  issue belongs to.
- **Issue** (required): The issue to label, by identifier (e.g. ENG-142) or ID. Supports expressions.
- **Labels** (optional): One or more existing labels to add. Existing labels on the issue are kept.
- **New Labels** (optional): Names of labels to create if they don't exist yet, then add. A name
  that already matches a team or workspace label is reused instead of creating a duplicate. New
  labels are created in the selected team.

At least one label must be provided across the two fields.

## Output

Returns the updated issue, including its ` + "`identifier`" + `, ` + "`url`" + `, ` + "`title`" + `,
` + "`state`" + `, ` + "`team`" + `, ` + "`assignee`" + `, ` + "`priorityLabel`" + ` and the full
set of ` + "`labels`" + ` after the addition.

## Permissions

The user who authorized the Linear connection must be able to edit the issue. SuperPlane's OAuth
connection includes the **write** scope, which covers adding and creating labels. Labels are added
one at a time, so a mid-way failure may leave some already applied.`
}

func (c *AddIssueLabel) Icon() string {
	return "linear"
}

func (c *AddIssueLabel) Color() string {
	return "indigo"
}

func (c *AddIssueLabel) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *AddIssueLabel) Configuration() []configuration.Field {
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
			Description: "The issue to label, by identifier (e.g. ENG-142) or ID",
			Placeholder: "ENG-142",
		},
		{
			Name:        "labels",
			Label:       "Labels",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Existing labels to add to the issue's current labels",
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
			Name:        "newLabels",
			Label:       "New Labels",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Names of labels to create if missing, then add. Existing labels with the same name are reused.",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Label name",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
	}
}

func (c *AddIssueLabel) Setup(ctx core.SetupContext) error {
	spec := AddIssueLabelSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if spec.Team == "" {
		return fmt.Errorf("team is required")
	}

	if strings.TrimSpace(spec.Issue) == "" {
		return fmt.Errorf("issue is required")
	}

	if len(trimmedLabels(spec.Labels)) == 0 && len(trimmedLabels(spec.NewLabels)) == 0 {
		return fmt.Errorf("at least one label is required")
	}

	team, err := requireTeam(ctx.Integration, spec.Team)
	if err != nil {
		return err
	}

	return ctx.Metadata.Set(NodeMetadata{Team: team})
}

func (c *AddIssueLabel) Execute(ctx core.ExecutionContext) error {
	spec := AddIssueLabelSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	issueID := strings.TrimSpace(spec.Issue)
	if issueID == "" {
		return fmt.Errorf("issue is required")
	}

	existing := trimmedLabels(spec.Labels)
	newNames := trimmedLabels(spec.NewLabels)
	if len(existing) == 0 && len(newNames) == 0 {
		return fmt.Errorf("at least one label is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}

	created, err := resolveNewLabels(client, spec.Team, newNames)
	if err != nil {
		return err
	}

	// Linear adds one label per call, so apply them in turn and emit the issue
	// as it stands after the last addition.
	var issue *Issue
	for _, labelID := range dedupeStrings(append(existing, created...)) {
		issue, err = client.AddIssueLabel(issueID, labelID)
		if err != nil {
			return fmt.Errorf("failed to add label: %v", err)
		}
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		IssuePayloadType,
		[]any{issue},
	)
}

// resolveNewLabels turns free-text label names into label IDs, reusing an
// existing team or workspace label with the same name and creating one only
// when none exists - Linear rejects a duplicate name within a team.
func resolveNewLabels(client *Client, teamID string, names []string) ([]string, error) {
	if len(names) == 0 {
		return nil, nil
	}

	existing, err := client.ListLabels(teamID)
	if err != nil {
		return nil, fmt.Errorf("failed to list labels: %v", err)
	}

	idByName := map[string]string{}
	for _, label := range existing {
		idByName[strings.ToLower(label.Name)] = label.ID
	}

	ids := make([]string, 0, len(names))
	for _, name := range names {
		key := strings.ToLower(name)
		if id, ok := idByName[key]; ok {
			ids = append(ids, id)
			continue
		}

		label, err := client.CreateLabel(name, teamID)
		if err != nil {
			return nil, fmt.Errorf("failed to create label %q: %v", name, err)
		}

		idByName[key] = label.ID
		ids = append(ids, label.ID)
	}

	return ids, nil
}

// trimmedLabels drops blank entries so an empty picker slot never reaches Linear.
func trimmedLabels(labels []string) []string {
	trimmed := []string{}
	for _, label := range labels {
		if l := strings.TrimSpace(label); l != "" {
			trimmed = append(trimmed, l)
		}
	}

	return trimmed
}

// dedupeStrings removes duplicate label IDs so a label picked and also typed as
// a new label is not added twice.
func dedupeStrings(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}

	return result
}

func (c *AddIssueLabel) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *AddIssueLabel) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *AddIssueLabel) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *AddIssueLabel) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *AddIssueLabel) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
