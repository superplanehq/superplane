package linear

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type CreateAttachment struct{}

type CreateAttachmentSpec struct {
	Issue    string `json:"issue" mapstructure:"issue"`
	Title    string `json:"title" mapstructure:"title"`
	URL      string `json:"url" mapstructure:"url"`
	Subtitle string `json:"subtitle" mapstructure:"subtitle"`
	IconURL  string `json:"iconUrl" mapstructure:"iconUrl"`
}

func (c *CreateAttachment) Name() string {
	return "linear.createAttachment"
}

func (c *CreateAttachment) Label() string {
	return "Create Attachment"
}

func (c *CreateAttachment) Description() string {
	return "Attach a link to an issue in Linear"
}

func (c *CreateAttachment) Documentation() string {
	return `The Create Attachment component adds a link attachment to an existing Linear issue.
Attachments render as cards on the issue instead of messages in the comment thread.

## Use Cases

- **Status tiles**: Attach a deploy or pipeline run and keep it current as the run progresses
- **Cross-linking**: Attach the pull request, dashboard, or alert behind an issue
- **Traceability**: Attach the SuperPlane run that acted on the issue

## Configuration

- **Issue** (required): The issue to attach to, by identifier (e.g. ENG-142) or ID. Supports
  expressions.
- **Title** (required): The attachment's headline. Supports expressions.
- **URL** (required): The link the attachment points to. Supports expressions.
- **Subtitle** (optional): Secondary line shown under the title. Supports expressions.
- **Icon URL** (optional): Icon shown on the attachment card. Linear downloads the image when the
  attachment is created, so the URL must be publicly reachable — an unreachable one fails the whole
  step, not just the icon. Linear does not expose the icon on the attachment afterwards, so it is not
  part of this component's output.

## Deduplication

Linear deduplicates attachments by URL within an issue. Creating an attachment with a URL that is
already attached **updates that attachment in place** rather than adding a second one, and Linear
sends an ` + "`update`" + ` webhook action rather than ` + "`create`" + `. This makes the component
safe to run repeatedly: keep the URL stable and vary the title or subtitle to show progress, without
the notification noise a new comment per run would cause.

## Output

Returns the created attachment, including its ` + "`id`" + `, ` + "`title`" + `, ` + "`subtitle`" + `,
` + "`url`" + `, the ` + "`creator`" + `, and the ` + "`issue`" + ` it belongs to. Keep the
` + "`id`" + ` if a later step needs to delete the attachment.

## Permissions

The user who authorized the Linear connection must be able to edit the issue. SuperPlane's OAuth
connection includes the **write** scope, which covers creating attachments.`
}

func (c *CreateAttachment) Icon() string {
	return "linear"
}

func (c *CreateAttachment) Color() string {
	return "indigo"
}

func (c *CreateAttachment) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateAttachment) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "issue",
			Label:       "Issue",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The issue to attach to, by identifier (e.g. ENG-142) or ID",
			Placeholder: "ENG-142",
		},
		{
			Name:        "title",
			Label:       "Title",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The attachment's headline",
			Placeholder: "Deploy - staging",
		},
		{
			Name:        "url",
			Label:       "URL",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The link the attachment points to. Reusing a URL updates that attachment instead of adding another",
			Placeholder: "https://example.com/runs/123",
		},
		{
			Name:        "subtitle",
			Label:       "Subtitle",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Secondary line shown under the title",
			Placeholder: "Succeeded in 4m 12s",
		},
		{
			Name:        "iconUrl",
			Label:       "Icon URL",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Icon shown on the attachment card. Must be publicly reachable - Linear downloads it",
		},
	}
}

func (c *CreateAttachment) Setup(ctx core.SetupContext) error {
	spec := CreateAttachmentSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	return spec.validate()
}

func (c *CreateAttachment) Execute(ctx core.ExecutionContext) error {
	spec := CreateAttachmentSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if err := spec.validate(); err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}

	// issueId accepts the human identifier (e.g. ENG-142), so no lookup is needed.
	input := map[string]any{
		"issueId": strings.TrimSpace(spec.Issue),
		"title":   strings.TrimSpace(spec.Title),
		"url":     strings.TrimSpace(spec.URL),
	}

	if subtitle := strings.TrimSpace(spec.Subtitle); subtitle != "" {
		input["subtitle"] = subtitle
	}

	if iconURL := strings.TrimSpace(spec.IconURL); iconURL != "" {
		input["iconUrl"] = iconURL
	}

	attachment, err := client.CreateAttachment(input)
	if err != nil {
		return fmt.Errorf("failed to create attachment: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		AttachmentPayloadType,
		[]any{attachment},
	)
}

func (s *CreateAttachmentSpec) validate() error {
	if strings.TrimSpace(s.Issue) == "" {
		return fmt.Errorf("issue is required")
	}

	if strings.TrimSpace(s.Title) == "" {
		return fmt.Errorf("title is required")
	}

	if strings.TrimSpace(s.URL) == "" {
		return fmt.Errorf("url is required")
	}

	return nil
}

func (c *CreateAttachment) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateAttachment) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *CreateAttachment) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *CreateAttachment) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *CreateAttachment) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
