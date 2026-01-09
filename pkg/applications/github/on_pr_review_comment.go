package github

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnPullRequestReviewComment struct{}

type OnPullRequestReviewCommentConfiguration struct {
	Repository string   `json:"repository" mapstructure:"repository"`
	Actions    []string `json:"actions" mapstructure:"actions"`
}

func (p *OnPullRequestReviewComment) Name() string {
	return "github.onPullRequestReviewComment"
}

func (p *OnPullRequestReviewComment) Label() string {
	return "On PR Review Comment"
}

func (p *OnPullRequestReviewComment) Description() string {
	return "Listen to pull request review comment events"
}

func (p *OnPullRequestReviewComment) Icon() string {
	return "github"
}

func (p *OnPullRequestReviewComment) Color() string {
	return "gray"
}

func (p *OnPullRequestReviewComment) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "repository",
			Label:    "Repository",
			Type:     configuration.FieldTypeString,
			Required: true,
		},
		{
			Name:     "actions",
			Label:    "Actions",
			Type:     configuration.FieldTypeMultiSelect,
			Required: true,
			Default:  []string{"created"},
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Created", Value: "created"},
						{Label: "Edited", Value: "edited"},
						{Label: "Deleted", Value: "deleted"},
					},
				},
			},
		},
	}
}

func (p *OnPullRequestReviewComment) Setup(ctx core.TriggerContext) error {
	err := ensureRepoInMetadata(
		ctx.Metadata,
		ctx.AppInstallation,
		ctx.Configuration,
	)

	if err != nil {
		return err
	}

	var config OnPullRequestReviewCommentConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	return ctx.AppInstallation.RequestWebhook(WebhookConfiguration{
		EventType:  "pull_request_review_comment",
		Repository: config.Repository,
	})
}

func (p *OnPullRequestReviewComment) Actions() []core.Action {
	return []core.Action{}
}

func (p *OnPullRequestReviewComment) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (p *OnPullRequestReviewComment) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	config := OnPullRequestReviewCommentConfiguration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

	code, err := verifySignature(ctx, "pull_request_review_comment")
	if err != nil {
		return code, err
	}

	data := map[string]any{}
	err = json.Unmarshal(ctx.Body, &data)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %v", err)
	}

	if !whitelistedAction(data, config.Actions) {
		return http.StatusOK, nil
	}

	err = ctx.Events.Emit("github.pullRequestReviewComment", data)

	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}
