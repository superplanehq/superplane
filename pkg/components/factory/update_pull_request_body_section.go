package factory

import (
	"fmt"
	"maps"
	"time"

	"github.com/go-viper/mapstructure/v2"
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/prbody"
	"github.com/superplanehq/superplane/pkg/registry"
)

const UpdatePullRequestBodySectionComponentName = "updatePullRequestBodySection"
const UpdatePullRequestBodySectionHook = "updatePullRequestBodySection"
const PullRequestBodySectionUpdatedPayloadType = "pullRequest.bodySectionUpdated"

type UpdatePullRequestBodySection struct{}

var disallowSectionKeyExpressions = false

type UpdatePullRequestBodySectionConfiguration struct {
	PullRequestID   string `json:"pullRequestId" mapstructure:"pullRequestId"`
	SectionKey      string `json:"sectionKey" mapstructure:"sectionKey"`
	Body            string `json:"body" mapstructure:"body"`
	ExpectedHeadSHA string `json:"expectedHeadSha,omitempty" mapstructure:"expectedHeadSha"`
	Enabled         *bool  `json:"enabled,omitempty" mapstructure:"enabled"`
}

func (c UpdatePullRequestBodySectionConfiguration) IsEnabled() bool {
	return c.Enabled == nil || *c.Enabled
}

func DecodeBodySectionConfiguration(value any) (UpdatePullRequestBodySectionConfiguration, error) {
	var config UpdatePullRequestBodySectionConfiguration
	if err := mapstructure.Decode(value, &config); err != nil {
		return config, err
	}
	return config, nil
}

func init() {
	registry.RegisterAction(UpdatePullRequestBodySectionComponentName, &UpdatePullRequestBodySection{})
}

func (c *UpdatePullRequestBodySection) Name() string {
	return UpdatePullRequestBodySectionComponentName
}

func (c *UpdatePullRequestBodySection) Label() string { return "Update PR Body Section" }

func (c *UpdatePullRequestBodySection) Description() string {
	return "Update one Markdown section in a factory pull request description"
}

func (c *UpdatePullRequestBodySection) Documentation() string {
	return "Update a marked section on a pull request in the current factory with arbitrary Markdown. The body supports standard canvas expressions, such as `{{ previous().data.summary }}`. Use `markdownText(value)` for generated text in Markdown tables and `markdownLink(url)` for link destinations enclosed in angle brackets. The resolved body is saved with the execution and reused on retries. Use a stable, literal section key (expressions are not supported) with lowercase letters, numbers, and single hyphens, starting with a letter (maximum 64 characters). The section uses `<!-- superplane:KEY:start -->` and `<!-- superplane:KEY:end -->`, where KEY is the section key. Missing sections are appended. Other content is preserved byte for byte. Malformed markers and unchanged content are skipped. The connected GitHub or Bitbucket provider must have an open pull request. Set expectedHeadSha to stop publication after a commit changes. Provider calls run outside database transactions. Temporary failures retry through scheduled requests. Execution details show attempts and skipped outcomes. Newer runs attached to the same pull request supersede older runs for the same section. The enabled setting defaults to true. Disabling the saved or current published node stops pending writes and leaves the existing section in place. This component can only be used in factory-owned apps."
}

func (c *UpdatePullRequestBodySection) Icon() string { return "file-check" }

func (c *UpdatePullRequestBodySection) Color() string { return "blue" }

func (c *UpdatePullRequestBodySection) Configuration() []configuration.Field {
	return []configuration.Field{
		{Name: "pullRequestId", Label: "Pull Request ID", Type: configuration.FieldTypeString, Required: true, Description: "Factory pull request to update. Use the ID from Find Pull Request."},
		// The section key must be a literal. Supersession and enabled checks
		// compare the saved key across runs, so a resolved key cannot match.
		{Name: "sectionKey", Label: "Section Key", Type: configuration.FieldTypeString, Required: true, Description: "Stable marker key, such as merge-confidence. Use lowercase letters, numbers, and single hyphens. Expressions are not supported.", TypeOptions: &configuration.TypeOptions{String: &configuration.StringTypeOptions{AllowExpressions: &disallowSectionKeyExpressions}}},
		{Name: "body", Label: "Body", Type: configuration.FieldTypeText, Required: true, Description: "Markdown to place inside the section markers. Supports expressions such as {{ previous().data.summary }}."},
		{Name: "expectedHeadSha", Label: "Expected Head SHA", Type: configuration.FieldTypeString, Required: false, Description: "Skip the update if the pull request head changed."},
		{Name: "enabled", Label: "Enabled", Type: configuration.FieldTypeBool, Default: true, Description: "Publish this section to the pull request."},
	}
}

func (c *UpdatePullRequestBodySection) ExampleOutput() map[string]any {
	return map[string]any{"type": PullRequestBodySectionUpdatedPayloadType, "timestamp": "2026-11-03T12:00:00Z", "data": map[string]any{"sourceRunId": "123", "status": "published", "attempts": 1}}
}

func (c *UpdatePullRequestBodySection) OutputChannels(any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *UpdatePullRequestBodySection) Setup(ctx core.SetupContext) error {
	config, err := DecodeBodySectionConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}
	if configuration.ExpressionPlaceholderRegex.MatchString(config.SectionKey) {
		return fmt.Errorf("sectionKey does not support expressions; use a literal key such as merge-confidence")
	}
	return prbody.ValidateBodySectionKey(config.SectionKey)
}

func (c *UpdatePullRequestBodySection) Execute(ctx core.ExecutionContext) error {
	if ctx.RunID == uuid.Nil {
		return fmt.Errorf("a current run is required")
	}
	config, err := DecodeBodySectionConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}
	if err := prbody.ValidateBodySectionKey(config.SectionKey); err != nil {
		return err
	}
	if _, err := uuid.Parse(config.PullRequestID); err != nil {
		return fmt.Errorf("invalid pullRequestId: %w", err)
	}
	metadata, _ := ctx.Metadata.Get().(map[string]any)
	metadata = maps.Clone(metadata)
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadata["sourceRunId"], metadata["status"], metadata["attempts"] = ctx.RunID.String(), "queued", 0
	if err := ctx.Metadata.Set(metadata); err != nil {
		return err
	}
	// The request worker calls the publisher after this execution transaction
	// commits, so host calls do not hold an execution lock or connection.
	return ctx.Requests.ScheduleActionCall(UpdatePullRequestBodySectionHook, nil, time.Second)
}

func (c *UpdatePullRequestBodySection) Hooks() []core.Hook {
	return []core.Hook{{Name: UpdatePullRequestBodySectionHook, Type: core.HookTypeInternal}}
}

func (c *UpdatePullRequestBodySection) HandleHook(core.ActionHookContext) error {
	return fmt.Errorf("publication requires the scheduled request dispatcher")
}

func (c *UpdatePullRequestBodySection) HandleWebhook(core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (c *UpdatePullRequestBodySection) Cancel(core.ExecutionContext) error { return nil }

func (c *UpdatePullRequestBodySection) Cleanup(core.SetupContext) error { return nil }
