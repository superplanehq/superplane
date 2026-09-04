package factory

import (
	"net/http"

	"github.com/go-viper/mapstructure/v2"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const ListPullRequestsComponentName = "listPullRequests"
const listPullRequestsEventType = "pullRequest.listed"

func init() {
	registry.RegisterAction(ListPullRequestsComponentName, &ListPullRequests{})
}

type ListPullRequests struct{}

type ListPullRequestsConfiguration struct {
	Repository string   `json:"repository" mapstructure:"repository"`
	States     []string `json:"states" mapstructure:"states"`
}

func (c *ListPullRequests) Name() string {
	return ListPullRequestsComponentName
}

func (c *ListPullRequests) Label() string {
	return "List Pull Requests"
}

func (c *ListPullRequests) Description() string {
	return "List factory pull requests for a repository from the database"
}

func (c *ListPullRequests) Documentation() string {
	return `The List Pull Requests component reads factory pull requests for one repository from the SuperPlane database. It does not call GitHub.

Use ` + "`states`" + ` to select which pull request states to include. Leave it empty to list open and draft pull requests, the common case for rechecking mergeability after a base branch push.

Pair this component with ` + "`forEach`" + ` to start one downstream path per listed pull request.

This component can only be used in factory-owned apps.

## Output Channels

- **Default**: The matching pull requests, oldest work order first`
}

func (c *ListPullRequests) Icon() string {
	return "factory"
}

func (c *ListPullRequests) Color() string {
	return "blue"
}

func (c *ListPullRequests) ExampleOutput() map[string]any {
	return map[string]any{
		"timestamp": "2026-01-01T00:00:00Z",
		"type":      listPullRequestsEventType,
		"data": map[string]any{
			"pullRequests": []any{
				map[string]any{
					"id":          "2b3cf24d-c0e2-4d42-bbe7-4c30ff2cb2a4",
					"workOrderId": "9ac921bc-68e4-46dc-a78b-60b955f3bbf2",
					"provider":    "github",
					"repository":  "acme/app",
					"number":      42,
					"url":         "https://github.com/acme/app/pull/42",
					"title":       "Fix retry handling",
					"state":       "open",
				},
			},
		},
	}
}

func (c *ListPullRequests) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *ListPullRequests) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "repository",
			Label:       "Repository",
			Description: "Repository in owner/name format.",
			Type:        configuration.FieldTypeString,
			Required:    true,
		},
		{
			Name:        "states",
			Label:       "States",
			Description: "Pull request states to include. Leave empty for open and draft.",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "State",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
	}
}

func (c *ListPullRequests) Execute(ctx core.ExecutionContext) error {
	config := ListPullRequestsConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return err
	}

	pullRequests, err := ctx.Factory.ListPullRequests(core.ListPullRequestsParams{
		Repository: config.Repository,
		States:     config.States,
	})
	if err != nil {
		return err
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		listPullRequestsEventType,
		[]any{map[string]any{
			"pullRequests": pullRequests,
		}},
	)
}

func (c *ListPullRequests) Setup(ctx core.SetupContext) error {
	return nil
}

func (c *ListPullRequests) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *ListPullRequests) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *ListPullRequests) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *ListPullRequests) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *ListPullRequests) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
