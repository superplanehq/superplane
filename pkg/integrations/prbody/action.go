// Package prbody implements marked pull request sections for Git integrations.
package prbody

import (
	"fmt"
	"maps"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-viper/mapstructure/v2"
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const HookName = "updatePullRequestBodySection"
const PayloadType = "pullRequest.bodySectionUpdated"

func IsComponent(name string) bool {
	return name == "github.updatePullRequestBodySection" || name == "bitbucket.updatePullRequestBodySection" || name == "updatePullRequestBodySection"
}

type Configuration struct {
	Repository      string `json:"repository" mapstructure:"repository"`
	PullNumber      string `json:"pullNumber" mapstructure:"pullNumber"`
	SectionKey      string `json:"sectionKey" mapstructure:"sectionKey"`
	Body            string `json:"body" mapstructure:"body"`
	ExpectedHeadSHA string `json:"expectedHeadSha,omitempty" mapstructure:"expectedHeadSha"`
	Enabled         *bool  `json:"enabled,omitempty" mapstructure:"enabled"`
	// Ordered accepts only the newest source run and attempt for this section.
	Ordered              bool  `json:"ordered,omitempty" mapstructure:"ordered"`
	ActivationGeneration int64 `json:"activationGeneration,omitempty" mapstructure:"activationGeneration"`
}

func Decode(value any) (Configuration, error) {
	var config Configuration
	if err := mapstructure.WeakDecode(value, &config); err != nil {
		return config, err
	}
	config.Repository = strings.TrimSpace(config.Repository)
	config.PullNumber = strings.TrimSpace(config.PullNumber)
	return config, nil
}

func (c Configuration) IsEnabled() bool { return c.Enabled == nil || *c.Enabled }

// PublicationKey is unique within one integration. Writers of the same section
// share it, so ordering applies across canvases.
func (c Configuration) PublicationKey() string {
	return fmt.Sprintf("pull-request-body/%s/%s/%s", strings.ToLower(c.Repository), c.PullNumber, c.SectionKey)
}

func (c Configuration) Number() (int64, error) {
	number, err := strconv.ParseInt(c.PullNumber, 10, 64)
	if err != nil || number <= 0 {
		return 0, fmt.Errorf("pull request number must be a positive integer")
	}
	return number, nil
}

func (c Configuration) Validate() error {
	if c.Repository == "" {
		return fmt.Errorf("repository is required")
	}
	if c.PullNumber == "" {
		return fmt.Errorf("pull request number is required")
	}
	if !configuration.ExpressionPlaceholderRegex.MatchString(c.PullNumber) {
		if _, err := c.Number(); err != nil {
			return err
		}
	}
	if c.ActivationGeneration < 0 {
		return fmt.Errorf("activation generation must be positive")
	}
	return ValidateBodySectionKey(c.SectionKey)
}

// Action owns the configuration and scheduled lifecycle shared by both providers.
type Action struct{}

func (Action) Label() string { return "Update PR Body Section" }
func (Action) Documentation() string {
	return `Update one marked section in a pull request description. Select the integration, repository, and pull request number.

Use a stable section key with lowercase letters, numbers, and single hyphens. Start with a letter. The maximum length is 64 characters. Expressions are not supported in the section key.

The section uses ` + "`<!-- superplane:KEY:start -->`" + ` and ` + "`<!-- superplane:KEY:end -->`" + ` markers. Missing sections are appended. Content outside the section is preserved byte for byte. Invalid markers produce skipped. Identical content produces unchanged when publication is still eligible. Writes produce published.

The body supports canvas expressions. Use ` + "`markdownText(value)`" + ` in Markdown tables and ` + "`markdownLink(url)`" + ` in link destinations. The resolved body is saved with the execution and reused on retries.

Set **Expected Head SHA** to skip publication if the pull request commit changes. Closed pull requests are skipped. **Enabled** defaults to true.

Temporary failures retry through scheduled requests. Each attempt reads the latest description. Provider calls run outside database transactions. Execution details show attempts and publication outcomes.

In factory apps, disabling the published node stops pending writes. Newer runs for the same pull request and section supersede older runs.

Turn on **Ordered** to order writers by their persisted source run and attempt. The source is the broadcast source run, or the current run for other triggers. An older source or attempt cannot replace the section, including on the same commit and across canvases. Each attempt rechecks the order, the pull request commit, and the saved node at the write. Ordered publication fails the step when the host rejects the write or retries are exhausted.

**Activation Generation** fences saved activations. Increase it when you enable the node again. Attempts that started with another generation do not write.`
}
func (Action) Configuration() []configuration.Field {
	literalKey := false
	return []configuration.Field{
		{Name: "repository", Label: "Repository", Type: configuration.FieldTypeIntegrationResource, Required: true, TypeOptions: &configuration.TypeOptions{Resource: &configuration.ResourceTypeOptions{Type: "repository", UseNameAsValue: true}}},
		{Name: "pullNumber", Label: "Pull Request Number", Type: configuration.FieldTypeString, Required: true, Description: "Pull request number to update. Supports expressions."},
		{Name: "sectionKey", Label: "Section Key", Type: configuration.FieldTypeString, Required: true, Description: "Stable marker key, such as merge-confidence. Expressions are not supported.", TypeOptions: &configuration.TypeOptions{String: &configuration.StringTypeOptions{AllowExpressions: &literalKey}}},
		{Name: "body", Label: "Body", Type: configuration.FieldTypeText, Required: true, Description: "Markdown to place inside the section markers. Supports expressions."},
		{Name: "expectedHeadSha", Label: "Expected Head SHA", Type: configuration.FieldTypeString, Description: "Skip the update if the pull request commit changed."},
		{Name: "enabled", Label: "Enabled", Type: configuration.FieldTypeBool, Default: true, Description: "Publish this section to the pull request."},
		{Name: "ordered", Label: "Ordered", Type: configuration.FieldTypeBool, Default: false, Description: "Skip publication when a newer source run or attempt owns this section."},
		{Name: "activationGeneration", Label: "Activation Generation", Type: configuration.FieldTypeNumber, Default: 1, Description: "Increase when you enable publication again. Older attempts do not write."},
	}
}
func ExampleOutput() map[string]any {
	return map[string]any{"type": PayloadType, "timestamp": "2026-11-03T12:00:00Z", "data": map[string]any{"status": "published", "attempts": 1}}
}
func (Action) OutputChannels(any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}
func (Action) ValidateNodeConfiguration(value map[string]any) error {
	config, err := Decode(value)
	if err != nil {
		return err
	}
	return config.Validate()
}
func (Action) Execute(ctx core.ExecutionContext) error {
	if ctx.RunID == uuid.Nil {
		return fmt.Errorf("a current run is required")
	}
	config, err := Decode(ctx.Configuration)
	if err != nil {
		return err
	}
	if err := config.Validate(); err != nil {
		return err
	}
	if _, err := config.Number(); err != nil {
		return err
	}
	metadata, _ := ctx.Metadata.Get().(map[string]any)
	metadata = maps.Clone(metadata)
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadata["sourceRunId"], metadata["status"], metadata["attempts"] = ctx.RunID.String(), "queued", 0
	if config.Ordered && config.IsEnabled() {
		accepted, err := AcceptOrderedWrite(ctx, config.PublicationKey())
		if err != nil {
			return err
		}
		if accepted == nil {
			metadata["status"], metadata["reason"] = string(StatusSkipped), SupersededReason
			if err := ctx.Metadata.Set(metadata); err != nil {
				return err
			}
			return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, PayloadType, []any{Skipped(SupersededReason)})
		}
		metadata["publication"] = accepted
	}
	if err := ctx.Metadata.Set(metadata); err != nil {
		return err
	}
	return ctx.Requests.ScheduleActionCall(HookName, nil, time.Second)
}

// SupersededReason explains an ordered write that lost to a newer source or attempt.
const SupersededReason = "A newer source run or attempt owns this publication."

// AcceptOrderedWrite claims the key for the current execution. It returns nil
// when a newer source run or attempt already owns the key.
func AcceptOrderedWrite(ctx core.ExecutionContext, key string) (*core.OrderedWrite, error) {
	if ctx.Publications == nil {
		return nil, fmt.Errorf("ordered publication requires a publication context")
	}
	canvasID, err := uuid.Parse(ctx.WorkflowID)
	if err != nil {
		return nil, fmt.Errorf("ordered publication requires a canvas: %w", err)
	}
	accepted, ok, err := core.AcceptOrderedWrite(ctx.Publications, key, canvasID, ctx.NodeID)
	if err != nil || !ok {
		return nil, err
	}
	return &accepted, nil
}

func (Action) Hooks() []core.Hook { return []core.Hook{{Name: HookName, Type: core.HookTypeInternal}} }
func (Action) HandleWebhook(core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusNotFound, nil, nil
}
func (Action) Cancel(core.ExecutionContext) error { return nil }
func (Action) Cleanup(core.SetupContext) error    { return nil }
