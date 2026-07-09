package gitlab

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

//go:embed example_output_add_reaction.json
var exampleOutputAddReaction []byte

type AddReaction struct{}

const (
	ReactionTargetMergeRequest = "mergeRequest"
	ReactionTargetNote         = "note"
)

type AddReactionConfiguration struct {
	Project         string `mapstructure:"project"`
	MergeRequestIID string `mapstructure:"mergeRequestIid"`
	Target          string `mapstructure:"target"`
	NoteID          string `mapstructure:"noteId"`
	Content         string `mapstructure:"content"`
}

func (c *AddReaction) Name() string {
	return "gitlab.addReaction"
}

func (c *AddReaction) Label() string {
	return "Add Reaction"
}

func (c *AddReaction) Description() string {
	return "Add an award emoji reaction to a GitLab merge request or comment"
}

func (c *AddReaction) Documentation() string {
	return `The Add Reaction component adds an award emoji reaction to a GitLab merge request or to a comment (note) on a merge request.

## Use Cases

- **Acknowledge commands**: Add eyes to merge request comments to indicate automation saw them
- **Workflow feedback**: React with thumbsup or rocket on success paths
- **Fast triage signals**: Use reactions to show status without posting extra comments

## Configuration

- **Project**: Select the GitLab project
- **Merge Request IID**: The internal ID (IID) of the merge request (supports expressions)
- **Target**: Choose whether to react to the merge request itself or to a comment on it
- **Note ID**: The comment (note) ID to react to. Required when the target is a comment.
- **Reaction**: The award emoji name

## Output

Returns the created GitLab award emoji object, including id, name, user, and timestamp.`
}

func (c *AddReaction) Icon() string {
	return "gitlab"
}

func (c *AddReaction) Color() string {
	return "orange"
}

func (c *AddReaction) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *AddReaction) ExampleOutput() map[string]any {
	var example map[string]any
	if err := json.Unmarshal(exampleOutputAddReaction, &example); err != nil {
		return map[string]any{}
	}
	return example
}

func (c *AddReaction) Configuration() []configuration.Field {
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
			Description: "The internal ID (IID) of the merge request",
		},
		{
			Name:     "target",
			Label:    "Target",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  ReactionTargetMergeRequest,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Merge request", Value: ReactionTargetMergeRequest},
						{Label: "Comment", Value: ReactionTargetNote},
					},
				},
			},
		},
		{
			Name:        "noteId",
			Label:       "Note ID",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "ID of the comment to react to. Required when the target is a comment.",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "target", Values: []string{ReactionTargetNote}},
			},
		},
		{
			Name:     "content",
			Label:    "Reaction",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "eyes",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "thumbsup", Value: "thumbsup"},
						{Label: "thumbsdown", Value: "thumbsdown"},
						{Label: "laughing", Value: "laughing"},
						{Label: "confused", Value: "confused"},
						{Label: "heart", Value: "heart"},
						{Label: "tada", Value: "tada"},
						{Label: "rocket", Value: "rocket"},
						{Label: "eyes", Value: "eyes"},
					},
				},
			},
		},
	}
}

func (c *AddReaction) Setup(ctx core.SetupContext) error {
	var config AddReactionConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Project == "" {
		return errors.New("project is required")
	}

	if config.MergeRequestIID == "" {
		return errors.New("merge request IID is required")
	}

	if config.Content == "" {
		return errors.New("reaction content is required")
	}

	if config.Target != ReactionTargetMergeRequest && config.Target != ReactionTargetNote {
		return fmt.Errorf("invalid target: %s", config.Target)
	}

	if config.Target == ReactionTargetNote && strings.TrimSpace(config.NoteID) == "" {
		return errors.New("note ID is required when target is a comment")
	}

	return ensureProjectInMetadata(
		ctx.Metadata,
		ctx.Integration,
		config.Project,
	)
}

func (c *AddReaction) Execute(ctx core.ExecutionContext) error {
	var config AddReactionConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Target != ReactionTargetMergeRequest && config.Target != ReactionTargetNote {
		return fmt.Errorf("invalid target: %s", config.Target)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to initialize GitLab client: %w", err)
	}

	req := &CreateAwardEmojiRequest{Name: config.Content}

	var awardEmoji *AwardEmoji
	switch config.Target {
	case ReactionTargetMergeRequest:
		awardEmoji, err = client.CreateMergeRequestAwardEmoji(context.Background(), config.Project, config.MergeRequestIID, req)
		if err != nil {
			return fmt.Errorf("failed to create merge request reaction: %w", err)
		}
	case ReactionTargetNote:
		if strings.TrimSpace(config.NoteID) == "" {
			return errors.New("note ID is required when target is a comment")
		}
		awardEmoji, err = client.CreateMergeRequestNoteAwardEmoji(context.Background(), config.Project, config.MergeRequestIID, config.NoteID, req)
		if err != nil {
			return fmt.Errorf("failed to create comment reaction: %w", err)
		}
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"gitlab.addReaction",
		[]any{awardEmoji},
	)
}

func (c *AddReaction) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *AddReaction) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (c *AddReaction) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *AddReaction) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *AddReaction) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *AddReaction) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
