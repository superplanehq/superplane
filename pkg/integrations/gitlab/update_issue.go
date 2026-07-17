package gitlab

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

//go:embed example_output_update_issue.json
var exampleOutputUpdateIssue []byte

type UpdateIssue struct{}

const (
	IssueStateEventClose  = "close"
	IssueStateEventReopen = "reopen"
)

type UpdateIssueConfiguration struct {
	Project   string   `mapstructure:"project"`
	IssueIID  string   `mapstructure:"issueIid"`
	Title     string   `mapstructure:"title"`
	Body      string   `mapstructure:"body"`
	State     string   `mapstructure:"state"`
	Labels    []string `mapstructure:"labels"`
	Assignees []string `mapstructure:"assignees"`
	Milestone string   `mapstructure:"milestone"`
}

// updateIssueToggles tracks which optional fields were explicitly turned on via
// their UI toggle, independent of whether the decoded value ended up empty.
// mapstructure decodes both "never toggled on" and "toggled on but cleared"
// to the same Go zero value, so this reads the raw configuration map instead
// to tell them apart - e.g. clearing all labels (toggled on, empty list)
// must still be sent to GitLab, not treated as "no change".
type updateIssueToggles struct {
	Title     bool
	Body      bool
	State     bool
	Labels    bool
	Assignees bool
	Milestone bool
}

func newUpdateIssueToggles(raw map[string]any) updateIssueToggles {
	enabled := func(field string) bool {
		v, ok := raw[field]
		return ok && v != nil
	}
	return updateIssueToggles{
		Title:     enabled("title"),
		Body:      enabled("body"),
		State:     enabled("state"),
		Labels:    enabled("labels"),
		Assignees: enabled("assignees"),
		Milestone: enabled("milestone"),
	}
}

func (t updateIssueToggles) hasUpdates() bool {
	return t.Title || t.Body || t.State || t.Labels || t.Assignees || t.Milestone
}

func (c *UpdateIssue) Name() string {
	return "gitlab.updateIssue"
}

func (c *UpdateIssue) Label() string {
	return "Update Issue"
}

func (c *UpdateIssue) Description() string {
	return "Update an existing GitLab issue"
}

func (c *UpdateIssue) Documentation() string {
	return `The Update Issue component modifies an existing GitLab issue: its title, description, state, labels, assignees, or milestone.

## Use Cases

- **Status updates**: Close or reopen an issue based on workflow results
- **Label management**: Apply labels to an issue automatically
- **Assignee updates**: Assign an issue to team members automatically
- **Content updates**: Update the issue title or description with new information

## Configuration

- **Project** (required): The GitLab project containing the issue
- **Issue IID** (required): The internal ID (IID) of the issue to update (supports expressions)
- **Title** (toggle): New title for the issue
- **Description** (toggle): New description for the issue
- **State** (toggle): Close or reopen the issue
- **Labels** (toggle): Labels to set on the issue, replacing any existing labels
- **Assignees** (toggle): Users to assign the issue to, replacing any existing assignees
- **Milestone** (toggle): Milestone to associate with the issue

Each field besides Project and Issue IID is toggled on individually, so only the fields you enable are sent in the update. At least one must be enabled. Enabling a field with an empty value clears it - e.g. toggling on Labels or Assignees with nothing selected removes all of them, and toggling on Milestone with nothing selected unassigns the milestone. Title is the exception: GitLab does not allow blank titles, so it must have a value when enabled.

## Output

Returns the updated issue object.`
}

func (c *UpdateIssue) Icon() string {
	return "gitlab"
}

func (c *UpdateIssue) Color() string {
	return "orange"
}

func (c *UpdateIssue) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *UpdateIssue) ExampleOutput() map[string]any {
	var example map[string]any
	if err := json.Unmarshal(exampleOutputUpdateIssue, &example); err != nil {
		return map[string]any{}
	}
	return example
}

func (c *UpdateIssue) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "project",
			Label:    "Project",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeProject,
				},
			},
		},
		{
			Name:        "issueIid",
			Label:       "Issue IID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The internal ID (IID) of the issue to update",
		},
		{
			Name:      "title",
			Label:     "Title",
			Type:      configuration.FieldTypeString,
			Required:  false,
			Togglable: true,
		},
		{
			Name:      "body",
			Label:     "Description",
			Type:      configuration.FieldTypeText,
			Required:  false,
			Togglable: true,
		},
		{
			Name:      "state",
			Label:     "State",
			Type:      configuration.FieldTypeSelect,
			Required:  false,
			Togglable: true,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Close", Value: IssueStateEventClose},
						{Label: "Reopen", Value: IssueStateEventReopen},
					},
				},
			},
		},
		{
			Name:      "labels",
			Label:     "Labels",
			Type:      configuration.FieldTypeList,
			Required:  false,
			Togglable: true,
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Label",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
		{
			Name:      "assignees",
			Label:     "Assignees",
			Type:      configuration.FieldTypeIntegrationResource,
			Required:  false,
			Togglable: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:  ResourceTypeMember,
					Multi: true,
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
			Name:      "milestone",
			Label:     "Milestone",
			Type:      configuration.FieldTypeIntegrationResource,
			Required:  false,
			Togglable: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeMilestone,
					Parameters: []configuration.ParameterRef{
						{
							Name:      "project",
							ValueFrom: &configuration.ParameterValueFrom{Field: "project"},
						},
					},
				},
			},
		},
	}
}

func (c *UpdateIssue) Setup(ctx core.SetupContext) error {
	var config UpdateIssueConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Project == "" {
		return errors.New("project is required")
	}

	if config.IssueIID == "" {
		return errors.New("issue IID is required")
	}

	if config.State != "" && config.State != IssueStateEventClose && config.State != IssueStateEventReopen {
		return fmt.Errorf("invalid state: %s", config.State)
	}

	raw, _ := ctx.Configuration.(map[string]any)
	toggles := newUpdateIssueToggles(raw)
	if !toggles.hasUpdates() {
		return errors.New("at least one field must be enabled to update")
	}

	if toggles.Title && config.Title == "" {
		return errors.New("title cannot be empty")
	}

	return ensureProjectInMetadata(
		ctx.Metadata,
		ctx.Integration,
		config.Project,
	)
}

func (c *UpdateIssue) Execute(ctx core.ExecutionContext) error {
	var config UpdateIssueConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.State != "" && config.State != IssueStateEventClose && config.State != IssueStateEventReopen {
		return fmt.Errorf("invalid state: %s", config.State)
	}

	raw, _ := ctx.Configuration.(map[string]any)
	toggles := newUpdateIssueToggles(raw)
	if !toggles.hasUpdates() {
		return errors.New("at least one field must be enabled to update")
	}

	if toggles.Title && config.Title == "" {
		return errors.New("title cannot be empty")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to initialize GitLab client: %w", err)
	}

	req := &UpdateIssueRequest{}

	if toggles.Title {
		req.Title = &config.Title
	}

	if toggles.Body {
		req.Description = &config.Body
	}

	if toggles.State {
		req.StateEvent = &config.State
	}

	if toggles.Labels {
		labels := strings.Join(config.Labels, ",")
		req.Labels = &labels
	}

	if toggles.Assignees {
		assigneeIDs := make([]int, 0, len(config.Assignees))
		for _, idStr := range config.Assignees {
			id, err := strconv.Atoi(idStr)
			if err != nil {
				return fmt.Errorf("invalid assignee id %q: %w", idStr, err)
			}
			assigneeIDs = append(assigneeIDs, id)
		}
		req.AssigneeIDs = &assigneeIDs
	}

	if toggles.Milestone {
		if config.Milestone == "" {
			// GitLab clears the milestone when milestone_id is explicitly set to 0.
			unset := 0
			req.MilestoneID = &unset
		} else {
			id, err := strconv.Atoi(config.Milestone)
			if err != nil {
				return fmt.Errorf("invalid milestone id %q: %w", config.Milestone, err)
			}
			req.MilestoneID = &id
		}
	}

	issue, err := client.UpdateIssue(context.Background(), config.Project, config.IssueIID, req)
	if err != nil {
		return fmt.Errorf("failed to update issue: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"gitlab.updateIssue",
		[]any{issue},
	)
}

func (c *UpdateIssue) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *UpdateIssue) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (c *UpdateIssue) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *UpdateIssue) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *UpdateIssue) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *UpdateIssue) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
