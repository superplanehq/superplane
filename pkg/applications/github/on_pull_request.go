package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
)

type OnPullRequest struct{}

type OnPullRequestMetadata struct {
	Repository *Repository `json:"repository"`
}

type OnPullRequestConfiguration struct {
	Repository string   `json:"repository"`
	Actions    []string `json:"action"`
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
	var metadata OnPullRequestMetadata
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

	config := OnPullRequestConfiguration{}
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

	client, err := NewClient(ctx.AppInstallationContext, appMetadata.GitHubApp.ID, appMetadata.InstallationID)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	repo, _, err := client.Repositories.Get(context.Background(), appMetadata.Owner, config.Repository)
	if err != nil {
		return fmt.Errorf("failed to get repository: %w", err)
	}

	metadata.Repository = &Repository{
		ID:   repo.GetID(),
		Name: repo.GetName(),
		URL:  repo.GetHTMLURL(),
	}

	ctx.MetadataContext.Set(metadata)

	return ctx.AppInstallationContext.RequestWebhook(WebhookConfiguration{
		EventType:  "pull_request",
		Repository: config.Repository,
	})
}

func (p *OnPullRequest) Actions() []core.Action {
	return []core.Action{}
}

func (p *OnPullRequest) HandleAction(ctx core.TriggerActionContext) error {
	return nil
}

func (p *OnPullRequest) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	signature := ctx.Headers.Get("X-Hub-Signature-256")
	if signature == "" {
		return http.StatusForbidden, fmt.Errorf("invalid signature")
	}

	eventType := ctx.Headers.Get("X-GitHub-Event")
	if eventType == "" {
		return http.StatusBadRequest, fmt.Errorf("missing X-GitHub-Event header")
	}

	config := OnPullRequestConfiguration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

	//
	// If event is not a pull_request event, we ignore it.
	//
	if eventType != "pull_request" {
		return http.StatusOK, nil
	}

	signature = strings.TrimPrefix(signature, "sha256=")
	if signature == "" {
		return http.StatusForbidden, fmt.Errorf("invalid signature")
	}

	secret, err := ctx.WebhookContext.GetSecret()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error authenticating request")
	}

	if err := crypto.VerifySignature(secret, ctx.Body, signature); err != nil {
		return http.StatusForbidden, err
	}

	data := map[string]any{}
	err = json.Unmarshal(ctx.Body, &data)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %v", err)
	}

	action, ok := data["action"]
	if !ok {
		return http.StatusBadRequest, fmt.Errorf("missing action")
	}

	if !slices.Contains(config.Actions, action.(string)) {
		return http.StatusOK, nil
	}

	err = ctx.EventContext.Emit(data)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}
