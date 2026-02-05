package circleci

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegration("circleci", &CircleCI{})
}

type CircleCI struct{}

type Configuration struct {
	APIToken string `json:"apiToken"`
}

type Metadata struct {
	Projects []string `json:"projects"`
}

func (c *CircleCI) Name() string {
	return "circleci"
}

func (c *CircleCI) Label() string {
	return "CircleCI"
}

func (c *CircleCI) Icon() string {
	return "circleci"
}

func (c *CircleCI) Description() string {
	return "Run and react to your CircleCI pipelines"
}

func (c *CircleCI) Instructions() string {
	return `## Setup Instructions

1. Go to [CircleCI User Settings](https://app.circleci.com/settings/user/tokens)
2. Click "Create New Token"
3. Give your token a name (e.g., "SuperPlane Integration")
4. Copy the token and paste it below

The token needs permission to trigger pipelines and read project data.`
}

func (c *CircleCI) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "apiToken",
			Label:       "API Token",
			Type:        configuration.FieldTypeString,
			Sensitive:   true,
			Description: "CircleCI Personal API Token",
			Required:    true,
		},
	}
}

func (c *CircleCI) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (c *CircleCI) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("Failed to decode configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	//
	// CircleCI has a /me endpoint to verify the token is valid.
	//
	_, err = client.GetCurrentUser()
	if err != nil {
		return fmt.Errorf("error verifying API token: %v", err)
	}

	ctx.Integration.Ready()
	return nil
}

func (c *CircleCI) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op
}

type WebhookConfiguration struct {
	ProjectSlug string `json:"projectSlug"`
}

func (c *CircleCI) CompareWebhookConfig(a, b any) (bool, error) {
	configA := WebhookConfiguration{}
	if err := mapstructure.Decode(a, &configA); err != nil {
		return false, err
	}

	configB := WebhookConfiguration{}
	if err := mapstructure.Decode(b, &configB); err != nil {
		return false, err
	}

	return configA.ProjectSlug == configB.ProjectSlug, nil
}

func (c *CircleCI) Actions() []core.Action {
	return []core.Action{}
}

func (c *CircleCI) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}

type WebhookMetadata struct {
	WebhookID string `json:"webhookId"`
}

func (c *CircleCI) SetupWebhook(ctx core.SetupWebhookContext) (any, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}

	configuration := WebhookConfiguration{}
	err = mapstructure.Decode(ctx.Webhook.GetConfiguration(), &configuration)
	if err != nil {
		return nil, fmt.Errorf("error decoding configuration: %v", err)
	}

	webhookSecret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return nil, fmt.Errorf("error getting webhook secret: %v", err)
	}

	// Get the project ID from the slug
	project, err := client.GetProject(configuration.ProjectSlug)
	if err != nil {
		return nil, fmt.Errorf("error getting project: %v", err)
	}

	//
	// Create CircleCI webhook to receive pipeline events
	//
	webhook, err := client.CreateWebhook(&CreateWebhookRequest{
		Name:   fmt.Sprintf("superplane-%s", ctx.Webhook.GetID()[:8]),
		URL:    ctx.Webhook.GetURL(),
		Secret: string(webhookSecret),
		Events: []string{"workflow-completed", "job-completed"},
		Scope: WebhookScope{
			ID:   project.ID,
			Type: "project",
		},
		VerifyTLS:     true,
		SigningSecret: string(webhookSecret),
	})
	if err != nil {
		return nil, fmt.Errorf("error creating CircleCI webhook: %v", err)
	}

	return WebhookMetadata{
		WebhookID: webhook.ID,
	}, nil
}

func (c *CircleCI) CleanupWebhook(ctx core.CleanupWebhookContext) error {
	metadata := WebhookMetadata{}
	err := mapstructure.Decode(ctx.Webhook.GetMetadata(), &metadata)
	if err != nil {
		return fmt.Errorf("error decoding webhook metadata: %v", err)
	}

	if metadata.WebhookID == "" {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	return client.DeleteWebhook(metadata.WebhookID)
}

func (c *CircleCI) Components() []core.Component {
	return []core.Component{
		&TriggerPipeline{},
	}
}

func (c *CircleCI) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnWorkflowCompleted{},
	}
}

// VerifyWebhookSignature verifies the CircleCI webhook signature
func VerifyWebhookSignature(secret []byte, body []byte, signature string) error {
	mac := hmac.New(sha256.New, secret)
	mac.Write(body)
	expectedMAC := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(signature), []byte(expectedMAC)) {
		return fmt.Errorf("signature mismatch")
	}

	return nil
}
