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

//go:embed example_output_update_merge_request.json
var exampleOutputUpdateMergeRequest []byte

type UpdateMergeRequest struct{}

const (
	MergeRequestStateEventClose  = "close"
	MergeRequestStateEventReopen = "reopen"
)

type UpdateMergeRequestConfiguration struct {
	Project         string   `mapstructure:"project"`
	MergeRequestIID string   `mapstructure:"mergeRequestIid"`
	Title           string   `mapstructure:"title"`
	Description     string   `mapstructure:"description"`
	TargetBranch    string   `mapstructure:"targetBranch"`
	State           string   `mapstructure:"state"`
	Labels          []string `mapstructure:"labels"`
	Assignees       []string `mapstructure:"assignees"`
}

// updateMergeRequestToggles tracks which optional fields were explicitly turned
// on via their UI toggle, independent of whether the decoded value ended up
// empty - clearing a field (toggled on, empty value) must still be sent, unlike
// a field that was never toggled on. See update_issue.go for the same pattern.
type updateMergeRequestToggles struct {
	Title        bool
	Description  bool
	TargetBranch bool
	State        bool
	Labels       bool
	Assignees    bool
}

func newUpdateMergeRequestToggles(raw map[string]any) updateMergeRequestToggles {
	enabled := func(field string) bool {
		v, ok := raw[field]
		return ok && v != nil
	}
	return updateMergeRequestToggles{
		Title:        enabled("title"),
		Description:  enabled("description"),
		TargetBranch: enabled("targetBranch"),
		State:        enabled("state"),
		Labels:       enabled("labels"),
		Assignees:    enabled("assignees"),
	}
}

func (t updateMergeRequestToggles) hasUpdates() bool {
	return t.Title || t.Description || t.TargetBranch || t.State || t.Labels || t.Assignees
}

func (c *UpdateMergeRequest) Name() string {
	return "gitlab.updateMergeRequest"
}

func (c *UpdateMergeRequest) Label() string {
	return "Update Merge Request"
}

func (c *UpdateMergeRequest) Description() string {
	return "Update an existing merge request in a GitLab project"
}

func (c *UpdateMergeRequest) Documentation() string {
	return `The Update Merge Request component modifies an existing GitLab merge request: its title, description, target branch, state, labels, or assignees.

## Use Cases

- **Retitle/redescribe automation**: Update a merge request's title or description as a workflow progresses
- **Open/close automation**: Close a merge request when it is superseded, or reopen one automatically
- **Retargeting**: Change the target branch a merge request merges into
- **Label and assignee management**: Replace a merge request's labels or assignees from a workflow

## Configuration

- **Project** (required): The GitLab project containing the merge request
- **Merge Request IID** (required): The internal ID (IID) of the merge request to update (supports expressions)
- **Title** (toggle): New title for the merge request
- **Description** (toggle): New description for the merge request
- **Target Branch** (toggle): Retarget the merge request onto a different branch
- **State** (toggle): Close or reopen the merge request
- **Labels** (toggle): Labels to set, replacing any existing labels
- **Assignees** (toggle): Users to assign, replacing any existing assignees

Each field besides Project and Merge Request IID is toggled on individually, so only the fields you enable are sent in the update. At least one must be enabled. Enabling Labels or Assignees with nothing selected clears them. Title and Target Branch are the exception: GitLab rejects a blank title, and an empty target branch cannot resolve to a real branch, so both must have a value when enabled.

## Permissions

The connected user needs at least the **Developer** role on the project (or be the merge request author) to update a merge request.

## Output

Returns the updated merge request object.`
}

func (c *UpdateMergeRequest) Icon() string {
	return "gitlab"
}

func (c *UpdateMergeRequest) Color() string {
	return "orange"
}

func (c *UpdateMergeRequest) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *UpdateMergeRequest) ExampleOutput() map[string]any {
	var example map[string]any
	if err := json.Unmarshal(exampleOutputUpdateMergeRequest, &example); err != nil {
		return map[string]any{}
	}
	return example
}

func (c *UpdateMergeRequest) Configuration() []configuration.Field {
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
			Name:        "mergeRequestIid",
			Label:       "Merge Request IID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "42 or {{event.data.object_attributes.iid}}",
			Description: "The internal ID (IID) of the merge request to update",
		},
		{
			Name:      "title",
			Label:     "Title",
			Type:      configuration.FieldTypeString,
			Required:  false,
			Togglable: true,
		},
		{
			Name:      "description",
			Label:     "Description",
			Type:      configuration.FieldTypeText,
			Required:  false,
			Togglable: true,
		},
		{
			Name:        "targetBranch",
			Label:       "Target Branch",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
			Description: "Retarget the merge request onto a different branch",
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
						{Label: "Close", Value: MergeRequestStateEventClose},
						{Label: "Reopen", Value: MergeRequestStateEventReopen},
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
	}
}

func (c *UpdateMergeRequest) Setup(ctx core.SetupContext) error {
	var config UpdateMergeRequestConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Project == "" {
		return errors.New("project is required")
	}

	if config.MergeRequestIID == "" {
		return errors.New("merge request IID is required")
	}

	if err := validateUpdateMergeRequest(ctx.Configuration, config); err != nil {
		return err
	}

	return ensureProjectInMetadata(
		ctx.Metadata,
		ctx.Integration,
		config.Project,
	)
}

func (c *UpdateMergeRequest) Execute(ctx core.ExecutionContext) error {
	var config UpdateMergeRequestConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if err := validateUpdateMergeRequest(ctx.Configuration, config); err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to initialize GitLab client: %w", err)
	}

	raw, _ := ctx.Configuration.(map[string]any)
	toggles := newUpdateMergeRequestToggles(raw)
	req := &UpdateMergeRequestRequest{}

	if toggles.Title {
		req.Title = &config.Title
	}

	if toggles.Description {
		req.Description = &config.Description
	}

	if toggles.TargetBranch {
		req.TargetBranch = &config.TargetBranch
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
		for _, id := range config.Assignees {
			parsed, err := strconv.Atoi(id)
			if err != nil {
				return fmt.Errorf("invalid assignee id %q: %w", id, err)
			}
			assigneeIDs = append(assigneeIDs, parsed)
		}
		req.AssigneeIDs = &assigneeIDs
	}

	mergeRequest, err := client.UpdateMergeRequest(context.Background(), config.Project, config.MergeRequestIID, req)
	if err != nil {
		return fmt.Errorf("failed to update merge request: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"gitlab.mergeRequest",
		[]any{mergeRequest},
	)
}

func (c *UpdateMergeRequest) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *UpdateMergeRequest) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (c *UpdateMergeRequest) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *UpdateMergeRequest) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *UpdateMergeRequest) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *UpdateMergeRequest) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func validateUpdateMergeRequest(rawConfig any, config UpdateMergeRequestConfiguration) error {
	if config.State != "" && config.State != MergeRequestStateEventClose && config.State != MergeRequestStateEventReopen {
		return fmt.Errorf("invalid state: %s", config.State)
	}

	raw, _ := rawConfig.(map[string]any)
	toggles := newUpdateMergeRequestToggles(raw)
	if !toggles.hasUpdates() {
		return errors.New("at least one field must be enabled to update")
	}

	if toggles.Title && config.Title == "" {
		return errors.New("title cannot be empty")
	}

	if toggles.TargetBranch && config.TargetBranch == "" {
		return errors.New("target branch cannot be empty")
	}

	return nil
}
