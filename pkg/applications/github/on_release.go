package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnRelease struct{}

type OnReleaseMetadata struct {
	Repository *Repository `json:"repository"`
}

type OnReleaseConfiguration struct {
	Repository string   `json:"repository"`
	Actions    []string `json:"action"`
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
			Type:     configuration.FieldTypeString,
			Required: true,
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
	var metadata OnReleaseMetadata
	err := mapstructure.Decode(ctx.MetadataContext.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to parse metadata: %w", err)
	}

	//
	// If metadata is set, it means the trigger was already setup
	//
	if metadata.Repository != nil {
		return nil
	}

	config := OnReleaseConfiguration{}
	err = mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Repository == "" {
		return fmt.Errorf("repository is required")
	}

	appMetadata := Metadata{}
	err = mapstructure.Decode(ctx.AppInstallationContext.GetMetadata(), &appMetadata)
	if err != nil {
		return fmt.Errorf("error decoding app installation metadata: %v", err)
	}

	repoIndex := slices.IndexFunc(appMetadata.Repositories, func(r Repository) bool {
		return r.Name == config.Repository
	})

	if repoIndex == -1 {
		return fmt.Errorf("repository %s is not accessible to app installation", config.Repository)
	}

	metadata.Repository = &appMetadata.Repositories[repoIndex]
	err = ctx.MetadataContext.Set(metadata)
	if err != nil {
		return fmt.Errorf("error setting metadata: %v", err)
	}

	return ctx.AppInstallationContext.RequestWebhook(WebhookConfiguration{
		EventType:  "release",
		Repository: config.Repository,
	})
}

func (r *OnRelease) Actions() []core.Action {
	return []core.Action{}
}

func (r *OnRelease) HandleAction(ctx core.TriggerActionContext) error {
	return nil
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

	err = ctx.EventContext.Emit(data)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}
