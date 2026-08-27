package factory

import (
	"net/http"

	"github.com/go-viper/mapstructure/v2"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const UpdatePullRequestComponentName = "updatePullRequest"

func init() {
	registry.RegisterAction(UpdatePullRequestComponentName, &UpdatePullRequest{})
}

type UpdatePullRequest struct{}

type UpdatePullRequestConfiguration struct {
	PullRequestID string `json:"pullRequestId" mapstructure:"pullRequestId"`
	ExternalID    string `json:"externalId" mapstructure:"externalId"`
	Repository    string `json:"repository" mapstructure:"repository"`
	URL           string `json:"url" mapstructure:"url"`
	Title         string `json:"title" mapstructure:"title"`
	State         any    `json:"state,omitempty" mapstructure:"state,omitempty"`
	Merged        any    `json:"merged,omitempty" mapstructure:"merged,omitempty"`
	Draft         any    `json:"draft,omitempty" mapstructure:"draft,omitempty"`
	MergedAt      string `json:"mergedAt" mapstructure:"mergedAt"`
	ClosedAt      string `json:"closedAt" mapstructure:"closedAt"`
}

func (c *UpdatePullRequest) Name() string {
	return UpdatePullRequestComponentName
}

func (c *UpdatePullRequest) Label() string {
	return "Update Pull Request"
}

func (c *UpdatePullRequest) Description() string {
	return "Update a factory pull request that is already attached to a work order"
}

func (c *UpdatePullRequest) Documentation() string {
	return `The Update Pull Request component updates a factory pull request by ` + "`pullRequestId`" + `. External-event canvases should run Find Pull Request first and pass its output id.

Mutable fields: repository, external id, URL, title, state, merged timestamp, and closed timestamp. The component does not move the pull request to a different work order.

` + "`state`" + `, ` + "`merged`" + `, and ` + "`draft`" + ` accept expressions. A GitHub-shaped ` + "`state: \"closed\"`" + ` + ` + "`merged: true`" + ` becomes SuperPlane state ` + "`merged`" + `. This component can only be used in factory-owned apps.`
}

func (c *UpdatePullRequest) Icon() string {
	return "factory"
}

func (c *UpdatePullRequest) Color() string {
	return "blue"
}

func (c *UpdatePullRequest) ExampleOutput() map[string]any {
	return map[string]any{
		"timestamp": "2026-01-01T00:00:00Z",
		"type":      "workOrder.pullRequestUpdated",
		"data": map[string]any{
			"pullRequest": map[string]any{
				"id":         "2b3cf24d-c0e2-4d42-bbe7-4c30ff2cb2a4",
				"provider":   "github",
				"repository": "acme/app",
				"number":     42,
				"url":        "https://github.com/acme/app/pull/42",
				"state":      "merged",
			},
		},
	}
}

func (c *UpdatePullRequest) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *UpdatePullRequest) Configuration() []configuration.Field {
	fields := []configuration.Field{
		{
			Name:        "pullRequestId",
			Label:       "Pull Request ID",
			Description: "Factory pull request to update. Use the id from Find Pull Request.",
			Type:        configuration.FieldTypeString,
			Required:    true,
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
			Description: "Repository in owner/name format.",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
		},
		{
			Name:        "url",
			Label:       "URL",
			Description: "Absolute http(s) pull request URL.",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
		},
		{
			Name:        "title",
			Label:       "Title",
			Description: "New title. Leave unset to keep the existing title.",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
		},
	}

	fields = append(fields, prArtifactLifecycleFields(prArtifactLifecycleFieldOptions{
		StateTogglable: true,
	})...)

	return append(fields,
		configuration.Field{
			Name:        "mergedAt",
			Label:       "Merged At",
			Description: "Optional RFC3339 merge timestamp — usually {{ event.data.pull_request.merged_at }}.",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
		},
		configuration.Field{
			Name:        "closedAt",
			Label:       "Closed At",
			Description: "Optional RFC3339 close timestamp — usually {{ event.data.pull_request.closed_at }}.",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
		},
	)
}

func (c *UpdatePullRequest) Execute(ctx core.ExecutionContext) error {
	config := UpdatePullRequestConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return err
	}

	mergedAt, err := parseOptionalRFC3339(config.MergedAt)
	if err != nil {
		return err
	}
	closedAt, err := parseOptionalRFC3339(config.ClosedAt)
	if err != nil {
		return err
	}

	params := core.UpdatePullRequestParams{
		PullRequestID: config.PullRequestID,
		MergedAt:      mergedAt,
		ClosedAt:      closedAt,
	}
	if raw, ok := ctx.Configuration.(map[string]any); ok {
		if _, present := raw["externalId"]; present {
			params.ExternalID = optionalStringPointer(config.ExternalID, true)
		}
		if _, present := raw["repository"]; present {
			params.Repository = optionalStringPointer(config.Repository, true)
		}
		if _, present := raw["url"]; present {
			params.URL = optionalStringPointer(config.URL, true)
		}
		if _, present := raw["title"]; present {
			params.Title = optionalStringPointer(config.Title, true)
		}
	}

	state := resolvePrArtifactState(config.State, config.Merged, config.Draft)
	if err := validatePrArtifactState(state); err != nil {
		return err
	}
	if state != "" {
		params.State = &state
	}

	pullRequest, err := ctx.Factory.UpdatePullRequest(params)
	if err != nil {
		return err
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"workOrder.pullRequestUpdated",
		[]any{map[string]any{
			"pullRequest": pullRequest,
		}},
	)
}

func (c *UpdatePullRequest) Setup(ctx core.SetupContext) error {
	return nil
}

func (c *UpdatePullRequest) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *UpdatePullRequest) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *UpdatePullRequest) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *UpdatePullRequest) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *UpdatePullRequest) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
