package linear

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	ReactionTargetIssue   = "issue"
	ReactionTargetComment = "comment"
)

type AddReaction struct{}

type AddReactionSpec struct {
	Target  string `json:"target" mapstructure:"target"`
	Issue   string `json:"issue" mapstructure:"issue"`
	Comment string `json:"comment" mapstructure:"comment"`
	Emoji   string `json:"emoji" mapstructure:"emoji"`
}

func (c *AddReaction) Name() string {
	return "linear.addReaction"
}

func (c *AddReaction) Label() string {
	return "Add Reaction"
}

func (c *AddReaction) Description() string {
	return "Add an emoji reaction to an issue or comment in Linear"
}

func (c *AddReaction) Documentation() string {
	return `The Add Reaction component adds an emoji reaction to a Linear issue or to a comment on one.

## Use Cases

- **Acknowledge commands**: React with 👀 so the author sees a ` + "`/deploy`" + ` comment was picked up
- **Signal results**: React with ✅ or ❌ when the run finishes, without another notification
- **Quiet status**: Reactions notify nobody, unlike a comment per run

## Configuration

- **Target** (required): React to the issue itself, or to a comment on it. Default: issue.
- **Issue** (required for the issue target): The issue, by identifier (e.g. ENG-142) or ID
- **Comment** (required for the comment target): The comment ID to react to
- **Reaction** (required): The emoji to add. Default: eyes.

## Re-running

Adding a reaction that is already present is treated as success, so this component is safe to
re-run. Linear also normalizes emoji aliases - thumbsup is stored as ` + "`+1`" + ` - which this
component accounts for.

## Output

Returns the created reaction, including its ` + "`id`" + `, ` + "`emoji`" + ` and the
` + "`user`" + ` it is attributed to. Keep the ` + "`id`" + ` if a later step needs to remove it.

## Permissions

The user who authorized the Linear connection must be able to see the issue. SuperPlane's OAuth
connection includes the **write** scope, which covers adding reactions.`
}

func (c *AddReaction) Icon() string {
	return "linear"
}

func (c *AddReaction) Color() string {
	return "indigo"
}

func (c *AddReaction) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *AddReaction) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "target",
			Label:    "Target",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  ReactionTargetIssue,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Issue", Value: ReactionTargetIssue},
						{Label: "Comment", Value: ReactionTargetComment},
					},
				},
			},
		},
		{
			Name:        "issue",
			Label:       "Issue",
			Type:        configuration.FieldTypeString,
			Description: "The issue to react to, by identifier (e.g. ENG-142) or ID",
			Placeholder: "ENG-142",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "target", Values: []string{ReactionTargetIssue}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "target", Values: []string{ReactionTargetIssue}},
			},
		},
		{
			Name:        "comment",
			Label:       "Comment",
			Type:        configuration.FieldTypeString,
			Description: "The ID of the comment to react to",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "target", Values: []string{ReactionTargetComment}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "target", Values: []string{ReactionTargetComment}},
			},
		},
		{
			Name:     "emoji",
			Label:    "Reaction",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "eyes",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					//
					// Values are the forms Linear stores. Sending an alias of an
					// emoji already on the target is rejected as a conflict, and
					// thumbsup/thumbsdown are aliases of +1/-1.
					//
					Options: []configuration.FieldOption{
						{Label: "Eyes 👀", Value: "eyes"},
						{Label: "Thumbs up 👍", Value: "+1"},
						{Label: "Thumbs down 👎", Value: "-1"},
						{Label: "Tada 🎉", Value: "tada"},
						{Label: "Rocket 🚀", Value: "rocket"},
						{Label: "Heart ❤️", Value: "heart"},
						{Label: "Laughing 😄", Value: "laughing"},
						{Label: "Confused 😕", Value: "confused"},
					},
				},
			},
		},
	}
}

func (c *AddReaction) Setup(ctx core.SetupContext) error {
	spec := AddReactionSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	_, err := spec.targetInput()
	if err != nil {
		return err
	}

	if strings.TrimSpace(spec.Emoji) == "" {
		return fmt.Errorf("emoji is required")
	}

	return nil
}

func (c *AddReaction) Execute(ctx core.ExecutionContext) error {
	spec := AddReactionSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	input, err := spec.targetInput()
	if err != nil {
		return err
	}

	emoji := strings.TrimSpace(spec.Emoji)
	if emoji == "" {
		return fmt.Errorf("emoji is required")
	}

	input["emoji"] = emoji

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}

	reaction, err := client.CreateReaction(input)
	if err != nil {
		//
		// The reaction is already on the target, which is the desired end state,
		// so report success rather than failing a re-run. Linear gives back no
		// reaction here, so the emitted payload carries only what is known.
		//
		if !isReactionConflict(err) {
			return fmt.Errorf("failed to add reaction: %v", err)
		}

		ctx.Logger.Infof("Reaction %s is already present - skipping", emoji)
		reaction = &Reaction{Emoji: emoji}
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		ReactionPayloadType,
		[]any{reaction},
	)
}

// targetInput builds the ReactionCreateInput target key for the selected target,
// validating that the matching ID is set. Linear accepts exactly one target.
func (s *AddReactionSpec) targetInput() (map[string]any, error) {
	target := strings.TrimSpace(s.Target)
	if target == "" {
		target = ReactionTargetIssue
	}

	switch target {
	case ReactionTargetIssue:
		issue := strings.TrimSpace(s.Issue)
		if issue == "" {
			return nil, fmt.Errorf("issue is required")
		}

		return map[string]any{"issueId": issue}, nil

	case ReactionTargetComment:
		comment := strings.TrimSpace(s.Comment)
		if comment == "" {
			return nil, fmt.Errorf("comment is required")
		}

		return map[string]any{"commentId": comment}, nil

	default:
		return nil, fmt.Errorf("invalid target %q", target)
	}
}

// isReactionConflict reports whether an error is Linear refusing a reaction that
// is already on the target, which happens when an alias of a present emoji is
// sent. Linear returns no error code for this, so the message is the only signal.
func isReactionConflict(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "conflict on insert of reaction")
}

func (c *AddReaction) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *AddReaction) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
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
