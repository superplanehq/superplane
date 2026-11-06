package sentry

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

// EnrichIssue fetches details after a workflow has selected an issue.
type EnrichIssue struct{}

func (c *EnrichIssue) Name() string  { return "sentry.enrichIssue" }
func (c *EnrichIssue) Label() string { return "Enrich Issue" }
func (c *EnrichIssue) Description() string {
	return "Fetch issue details and event data from Sentry"
}
func (c *EnrichIssue) Documentation() string {
	return `Enrich Issue fetches the latest issue details and a preferred event from Sentry. Add it after filters so rejected issues do not make API requests.

New Sentry factory intakes use HTTP Request nodes after filters to fetch the issue and latest event. They request the recommended event only when the latest event has no stack frames.

The sentryEventHasStack(event) expression returns true when a Sentry event body contains exception or standalone stack frames. Missing or invalid bodies return false. A skipped recommended request does not prevent task creation.

New and reset intakes use this conditional request. Saved intakes keep their existing workflow until reset. Deploy workers with this expression before the API publishes or resets intake templates.

The sentryIssueDescription(issue, fetchedIssue, latestEvent, recommendedEvent) expression uses the same description assembly as manual imports. It prefers the latest event with a stack trace, then a recommended event with a stack trace or more entries. API failures keep the original issue and available event data.

SuperPlane stores and acknowledges webhook deliveries before these API requests. Existing workflows can continue to use Enrich Issue.

## Configuration

- **Issue**: The issue object from On Issue Event, usually ` + "`{{ root().data.data.issue }}`" + `

## Output

- **description**: Markdown with the issue link, details, and available event data. If Sentry is unavailable, the action uses the original issue object and continues.`
}
func (c *EnrichIssue) Icon() string  { return "bug" }
func (c *EnrichIssue) Color() string { return "gray" }
func (c *EnrichIssue) ExampleOutput() map[string]any {
	return map[string]any{
		"type":      "sentry.enrichedIssue",
		"timestamp": "2026-04-22T10:05:00Z",
		"data":      map[string]any{"description": "[View in Sentry](https://sentry.io/issues/123/)"},
	}
}
func (c *EnrichIssue) OutputChannels(any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}
func (c *EnrichIssue) Configuration() []configuration.Field {
	return []configuration.Field{{
		Name:        "issue",
		Label:       "Issue",
		Type:        configuration.FieldTypeObject,
		Required:    true,
		Description: "Issue object from the On Issue Event trigger",
	}}
}

func (c *EnrichIssue) Setup(ctx core.SetupContext) error {
	_, err := configuredIssue(ctx.Configuration)
	return err
}

func (c *EnrichIssue) Execute(ctx core.ExecutionContext) error {
	issue, err := configuredIssue(ctx.Configuration)
	if err != nil {
		return err
	}
	if ctx.Integration == nil {
		return fmt.Errorf("Sentry integration is not connected")
	}
	if ctx.HTTP == nil {
		return fmt.Errorf("Sentry HTTP client is unavailable")
	}

	description := IssueDescription(issue, nil)
	client, clientErr := NewClient(ctx.HTTP, ctx.Integration)
	if clientErr != nil {
		if ctx.Logger != nil {
			ctx.Logger.Warnf("failed to create Sentry client for issue enrichment: %v", clientErr)
		}
	} else {
		description = FetchedIssueDescription(client, issue, ctx.Logger)
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "sentry.enrichedIssue", []any{
		map[string]any{"description": description},
	})
}

func configuredIssue(configuration any) (any, error) {
	values, ok := configuration.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("issue configuration is required")
	}
	issue, exists := values["issue"]
	if !exists || issue == nil {
		return nil, fmt.Errorf("issue is required")
	}
	if text, isString := issue.(string); isString && strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("issue is required")
	}
	return issue, nil
}

func (c *EnrichIssue) HandleWebhook(core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}
func (c *EnrichIssue) Cancel(core.ExecutionContext) error { return nil }
func (c *EnrichIssue) Cleanup(core.SetupContext) error    { return nil }
func (c *EnrichIssue) Hooks() []core.Hook                 { return nil }
func (c *EnrichIssue) HandleHook(core.ActionHookContext) error {
	return nil
}
