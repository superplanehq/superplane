package factory

import (
	"errors"
	"net/http"

	"github.com/go-viper/mapstructure/v2"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const FindPullRequestComponentName = "findPullRequest"
const FindPullRequestChannelNameFound = "found"
const FindPullRequestChannelNameNotFound = "notFound"

func init() {
	registry.RegisterAction(FindPullRequestComponentName, &FindPullRequest{})
}

type FindPullRequest struct{}

type FindPullRequestConfiguration struct {
	Provider   string `json:"provider" mapstructure:"provider"`
	ExternalID string `json:"externalId" mapstructure:"externalId"`
	Repository string `json:"repository" mapstructure:"repository"`
	Number     string `json:"number" mapstructure:"number"`
	URL        string `json:"url" mapstructure:"url"`
}

func (c *FindPullRequest) Name() string {
	return FindPullRequestComponentName
}

func (c *FindPullRequest) Label() string {
	return "Find Pull Request"
}

func (c *FindPullRequest) Description() string {
	return "Look up a factory pull request by provider identity or URL"
}

func (c *FindPullRequest) Documentation() string {
	return `The Find Pull Request component looks up a pull request in the current factory. It does not create a pull request and it does not link the current run.

Supply at least one complete identity:

- Provider and external id
- Provider, repository, and number
- URL

On a match, emits ` + "`pullRequest.found`" + ` with the pull request and its work order on the ` + "`found`" + ` channel. When nothing matches, emits ` + "`pullRequest.notFound`" + ` on the ` + "`notFound`" + ` channel instead of failing the run. This component can only be used in factory-owned apps.

## Output Channels

- **Found**: A pull request matched the lookup
- **Not Found**: No pull request matched the lookup`
}

func (c *FindPullRequest) Icon() string {
	return "factory"
}

func (c *FindPullRequest) Color() string {
	return "blue"
}

func (c *FindPullRequest) ExampleOutput() map[string]any {
	return map[string]any{
		"timestamp": "2026-01-01T00:00:00Z",
		"type":      "pullRequest.found",
		"data": map[string]any{
			"pullRequest": map[string]any{
				"id":     "2b3cf24d-c0e2-4d42-bbe7-4c30ff2cb2a4",
				"number": 42,
				"url":    "https://github.com/acme/app/pull/42",
			},
			"workOrder": map[string]any{
				"id":     "9ac921bc-68e4-46dc-a78b-60b955f3bbf2",
				"number": 123,
				"key":    "SP-123",
			},
		},
	}
}

func (c *FindPullRequest) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: FindPullRequestChannelNameFound, Label: "Found"},
		{Name: FindPullRequestChannelNameNotFound, Label: "Not Found"},
	}
}

func (c *FindPullRequest) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "provider",
			Label:       "Provider",
			Description: "Source control provider. Defaults to github.",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Default:     "github",
		},
		{
			Name:        "externalId",
			Label:       "External ID",
			Description: "Provider pull request id.",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
		},
		{
			Name:        "repository",
			Label:       "Repository",
			Description: "Repository in owner/name format. Required when you look up by number.",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
		},
		{
			Name:        "number",
			Label:       "Number",
			Description: "Pull request number.",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
		},
		{
			Name:        "url",
			Label:       "URL",
			Description: "Absolute http(s) pull request URL. Used when other identity fields are absent.",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
		},
	}
}

func (c *FindPullRequest) Execute(ctx core.ExecutionContext) error {
	config := FindPullRequestConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return err
	}

	number, err := parseOptionalInt64(config.Number)
	if err != nil {
		return err
	}

	match, err := ctx.Factory.FindPullRequest(core.FindPullRequestParams{
		Provider:   config.Provider,
		ExternalID: config.ExternalID,
		Repository: config.Repository,
		Number:     number,
		URL:        config.URL,
	})
	if err != nil {
		if errors.Is(err, core.ErrPullRequestNotFound) {
			return ctx.ExecutionState.Emit(
				FindPullRequestChannelNameNotFound,
				"pullRequest.notFound",
				[]any{map[string]any{
					"provider":   config.Provider,
					"externalId": config.ExternalID,
					"repository": config.Repository,
					"number":     config.Number,
					"url":        config.URL,
				}},
			)
		}
		return err
	}

	return ctx.ExecutionState.Emit(
		FindPullRequestChannelNameFound,
		"pullRequest.found",
		[]any{map[string]any{
			"pullRequest": match.PullRequest,
			"workOrder":   match.WorkOrder,
		}},
	)
}

func (c *FindPullRequest) Setup(ctx core.SetupContext) error {
	return nil
}

func (c *FindPullRequest) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *FindPullRequest) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *FindPullRequest) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *FindPullRequest) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *FindPullRequest) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
