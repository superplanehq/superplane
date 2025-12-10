package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/components"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/triggers"
)

type OnPullRequest struct{}

type OnPullRequestMetadata struct {
	Repository *Repository `json:"repository"`
}

type OnPullRequestConfiguration struct {
	Repository string `json:"repository"`
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
	}
}

func (p *OnPullRequest) Setup(ctx triggers.TriggerContext) error {
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

	//
	// TODO: Check if repository exists
	// TODO: Set up web hook
	//

	return nil
}

func (p *OnPullRequest) Actions() []components.Action {
	return []components.Action{}
}

func (p *OnPullRequest) HandleAction(ctx triggers.TriggerActionContext) error {
	return nil
}

func (p *OnPullRequest) HandleWebhook(ctx triggers.WebhookRequestContext) (int, error) {
	signature := ctx.Headers.Get("X-Hub-Signature-256")
	if signature == "" {
		return http.StatusForbidden, fmt.Errorf("invalid signature")
	}

	eventType := ctx.Headers.Get("X-GitHub-Event")
	if eventType == "" {
		return http.StatusBadRequest, fmt.Errorf("missing X-GitHub-Event header")
	}

	config := Configuration{}
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

	err = ctx.EventContext.Emit(data)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}
