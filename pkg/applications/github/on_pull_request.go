package github

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"slices"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnPullRequest struct{}

type OnPullRequestConfiguration struct {
	Repository string   `json:"repository" mapstructure:"repository"`
	Actions    []string `json:"actions" mapstructure:"actions"`
}

func (p *OnPullRequest) Name() string {
	return "github.onPullRequest"
}

func (p *OnPullRequest) Label() string {
	return "On Pull Request"
}

func (p *OnPullRequest) Description() string {
	return "Listen to pull request events"
}

func (p *OnPullRequest) Icon() string {
	return "github"
}

func (p *OnPullRequest) Color() string {
	return "gray"
}

func (p *OnPullRequest) Configuration() []configuration.Field {
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
			Default:  []string{"opened"},
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Assigned", Value: "assigned"},
						{Label: "Unassigned", Value: "unassigned"},
						{Label: "Opened", Value: "opened"},
						{Label: "Closed", Value: "closed"},
						{Label: "Labeled", Value: "labeled"},
						{Label: "Unlabeled", Value: "unlabeled"},
						{Label: "Reopened", Value: "reopened"},
						{Label: "Synchronize", Value: "synchronize"},
					},
				},
			},
		},
	}
}

func (p *OnPullRequest) Setup(ctx core.TriggerContext) error {
	err := ensureRepoInMetadata(
		ctx.MetadataContext,
		ctx.AppInstallationContext,
		ctx.Configuration,
	)

	if err != nil {
		return err
	}

	var config OnPullRequestConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	return ctx.AppInstallationContext.RequestWebhook(WebhookConfiguration{
		EventType:  "pull_request",
		Repository: config.Repository,
	})
}

func (p *OnPullRequest) Actions() []core.Action {
	return []core.Action{}
}

func (p *OnPullRequest) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (p *OnPullRequest) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	config := OnPullRequestConfiguration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

	code, err := verifySignature(ctx, "pull_request")
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

	err = ctx.EventContext.Emit("github.pullRequest", data)

	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}

func whitelistedAction(data map[string]any, allowed []string) bool {
	action, ok := data["action"]
	if !ok {
		return false
	}

	log.Printf("Allowed: %v, action: %v", allowed, action)

	return slices.Contains(allowed, action.(string))
}
