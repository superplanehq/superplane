package github

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnIssueCommented struct{}

type OnIssueCommentedConfiguration struct {
	Repository string   `json:"repository" mapstructure:"repository"`
	Actions    []string `json:"actions" mapstructure:"actions"`
}

func (i *OnIssueCommented) Name() string {
	return "github.onIssueCommented"
}

func (i *OnIssueCommented) Label() string {
	return "On Issue Commented"
}

func (i *OnIssueCommented) Description() string {
	return "Listen to issue comment events"
}

func (i *OnIssueCommented) Icon() string {
	return "github"
}

func (i *OnIssueCommented) Color() string {
	return "gray"
}

func (i *OnIssueCommented) Configuration() []configuration.Field {
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

func (i *OnIssueCommented) Setup(ctx core.TriggerContext) error {
	err := ensureRepoInMetadata(
		ctx.Metadata,
		ctx.AppInstallation,
		ctx.Configuration,
	)

	if err != nil {
		return err
	}

	var config OnIssueCommentedConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	return ctx.AppInstallation.RequestWebhook(WebhookConfiguration{
		EventType:  "issue_comment",
		Repository: config.Repository,
	})
}

func (i *OnIssueCommented) Actions() []core.Action {
	return []core.Action{}
}

func (i *OnIssueCommented) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (i *OnIssueCommented) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	config := OnIssueCommentedConfiguration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

	code, err := verifySignature(ctx, "issue_comment")
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

	err = ctx.Events.Emit("github.issueComment", data)

	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}
