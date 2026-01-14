package github

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnRelease struct{}

type OnReleaseConfiguration struct {
	Repository string   `json:"repository" mapstructure:"repository"`
	Actions    []string `json:"actions" mapstructure:"actions"`
}

func (r *OnRelease) Name() string {
	return "github.onRelease"
}

func (r *OnRelease) Label() string {
	return "On Release"
}

func (r *OnRelease) Description() string {
	return "Listen to release events"
}

func (r *OnRelease) Icon() string {
	return "github"
}

func (r *OnRelease) Color() string {
	return "gray"
}

func (r *OnRelease) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "repository",
			Label:    "Repository",
			Type:     configuration.FieldTypeAppInstallationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "repository",
					UseNameAsValue: true,
				},
			},
		},
		{
			Name:     "actions",
			Label:    "Actions",
			Type:     configuration.FieldTypeMultiSelect,
			Required: true,
			Default:  []string{"published"},
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Published", Value: "published"},
						{Label: "Unpublished", Value: "unpublished"},
						{Label: "Created", Value: "created"},
						{Label: "Edited", Value: "edited"},
						{Label: "Deleted", Value: "deleted"},
						{Label: "Prereleased", Value: "prereleased"},
						{Label: "Released", Value: "released"},
					},
				},
			},
		},
	}
}

func (r *OnRelease) Setup(ctx core.TriggerContext) error {
	err := ensureRepoInMetadata(
		ctx.Metadata,
		ctx.AppInstallation,
		ctx.Configuration,
	)

	if err != nil {
		return err
	}

	var config OnReleaseConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	return ctx.AppInstallation.RequestWebhook(WebhookConfiguration{
		EventType:  "release",
		Repository: config.Repository,
	})
}

func (r *OnRelease) Actions() []core.Action {
	return []core.Action{}
}

func (r *OnRelease) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (r *OnRelease) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	config := OnReleaseConfiguration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

	code, err := verifySignature(ctx, "release")
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

	err = ctx.Events.Emit("github.release", data)

	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}
