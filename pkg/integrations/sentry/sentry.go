package sentry

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/registry"
)

const MaxWebhookBodySize int64 = 1 * 1024 * 1024 // 1MB

func init() {
	registry.RegisterIntegration("sentry", &Sentry{})
}

type Sentry struct{}

type Configuration struct {
	Organization string `json:"organization"`
	AuthToken    string `json:"authToken"`
	ClientSecret string `json:"clientSecret"`
}

type Metadata struct {
	Projects      []Project `json:"projects"`
	SentryAppSlug string    `json:"sentryAppSlug,omitempty" mapstructure:"sentryAppSlug"`
	SentryAppUUID string    `json:"sentryAppUuid,omitempty" mapstructure:"sentryAppUuid"`
	WebhookURL    string    `json:"webhookUrl,omitempty" mapstructure:"webhookUrl"`
	ClientSecret  string    `json:"clientSecret,omitempty" mapstructure:"clientSecret"`
}

type SubscriptionConfiguration struct{}

type Project struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
	Name string `json:"name"`
}

func (s *Sentry) Name() string {
	return "sentry"
}

func (s *Sentry) Label() string {
	return "Sentry"
}

func (s *Sentry) Icon() string {
	return "sentry"
}

func (s *Sentry) Description() string {
	return "Monitor and manage your Sentry issues"
}

func (s *Sentry) Instructions() string {
	return `To connect Sentry:

1. In Sentry, go to **Settings → Auth Tokens** and create a token with scopes: ` + "`org:read`" + `, ` + "`org:admin`" + `, ` + "`event:read`" + `, ` + "`event:write`" + `.
2. Find your organization slug from your Sentry URL (e.g. sentry.io/organizations/**my-org**/).
3. Enter the **Organization** slug and **Auth Token** below and save. SuperPlane will automatically create a Sentry Internal Integration.
4. (Optional) For webhook signature verification, go to **Settings → Developer Settings**, open the auto-created **SuperPlane-XXXXXXXX** integration, copy the **Client Secret**, paste it here, and save again.`
}

func (s *Sentry) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "organization",
			Label:       "Organization",
			Type:        configuration.FieldTypeString,
			Description: "Your Sentry organization slug (e.g., 'acme-corp')",
			Placeholder: "e.g. acme-corp",
			Required:    true,
		},
		{
			Name:        "authToken",
			Label:       "Auth Token",
			Type:        configuration.FieldTypeString,
			Sensitive:   true,
			Description: "Sentry Auth Token with project:read, event:read, and event:write scopes",
			Required:    true,
		},
		{
			Name:        "clientSecret",
			Label:       "Client Secret",
			Type:        configuration.FieldTypeString,
			Sensitive:   true,
			Description: "Client Secret for webhook signature verification. After saving, go to Sentry > Settings > Developer Settings > Your Integration > Credentials and copy the Client Secret.",
			Required:    false,
		},
	}
}

func (s *Sentry) Cleanup(ctx core.IntegrationCleanupContext) error {
	metadata := Metadata{}
	if existing := ctx.Integration.GetMetadata(); existing != nil {
		if err := mapstructure.Decode(existing, &metadata); err != nil {
			return nil
		}
	}

	if metadata.SentryAppSlug != "" {
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil
		}

		_ = client.DeleteSentryApp(metadata.SentryAppSlug)
	}

	return nil
}

func (s *Sentry) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}

	if _, err := client.GetOrganization(); err != nil {
		return fmt.Errorf("error verifying connection: %w", err)
	}

	webhookURL := fmt.Sprintf("%s/api/v1/integrations/%s/webhook",
		strings.TrimRight(ctx.WebhooksBaseURL, "/"),
		ctx.Integration.ID().String(),
	)

	metadata := Metadata{}
	if existing := ctx.Integration.GetMetadata(); existing != nil {
		_ = mapstructure.Decode(existing, &metadata)
	}

	if metadata.SentryAppSlug == "" {
		appName := fmt.Sprintf("SuperPlane-%s", ctx.Integration.ID().String()[:8])

		app, err := client.CreateSentryApp(appName, webhookURL, []string{"issue"})
		if err != nil {
			ctx.Integration.Error(fmt.Sprintf("failed to create Sentry App: %v", err))
			return err
		}

		metadata.SentryAppSlug = app.Slug
		metadata.SentryAppUUID = app.UUID
	} else {
		if _, err := client.GetSentryApp(metadata.SentryAppSlug); err != nil {
			appName := fmt.Sprintf("SuperPlane-%s", ctx.Integration.ID().String()[:8])

			app, err := client.CreateSentryApp(appName, webhookURL, []string{"issue"})
			if err != nil {
				ctx.Integration.Error(fmt.Sprintf("failed to recreate Sentry App: %v", err))
				return err
			}

			metadata.SentryAppSlug = app.Slug
			metadata.SentryAppUUID = app.UUID
		}
	}

	metadata.WebhookURL = webhookURL
	ctx.Integration.SetMetadata(metadata)
	ctx.Integration.Ready()
	return nil
}

func (s *Sentry) HandleRequest(ctx core.HTTPRequestContext) {
	if !strings.HasSuffix(ctx.Request.URL.Path, "/webhook") {
		ctx.Response.WriteHeader(http.StatusNotFound)
		return
	}

	if ctx.Request.Method != http.MethodPost {
		ctx.Response.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	ctx.Request.Body = http.MaxBytesReader(ctx.Response, ctx.Request.Body, MaxWebhookBodySize)
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		ctx.Logger.Errorf("failed to read sentry webhook body: %v", err)
		ctx.Response.WriteHeader(http.StatusBadRequest)
		return
	}

	metadata := Metadata{}
	if existing := ctx.Integration.GetMetadata(); existing != nil {
		_ = mapstructure.Decode(existing, &metadata)
	}

	if metadata.ClientSecret != "" {
		signature := ctx.Request.Header.Get("Sentry-Hook-Signature")
		if signature == "" {
			ctx.Logger.Errorf("missing Sentry-Hook-Signature header")
			ctx.Response.WriteHeader(http.StatusForbidden)
			return
		}

		if err := crypto.VerifySignature([]byte(metadata.ClientSecret), body, signature); err != nil {
			ctx.Logger.Errorf("webhook signature verification failed: %v", err)
			ctx.Response.WriteHeader(http.StatusForbidden)
			return
		}
	}

	payload := map[string]any{}
	if err := json.Unmarshal(body, &payload); err != nil {
		ctx.Logger.Errorf("failed to parse sentry webhook body: %v", err)
		ctx.Response.WriteHeader(http.StatusBadRequest)
		return
	}

	subscriptions, err := ctx.Integration.ListSubscriptions()
	if err != nil {
		ctx.Logger.Errorf("failed to list sentry subscriptions: %v", err)
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	for _, subscription := range subscriptions {
		if err := subscription.SendMessage(payload); err != nil {
			ctx.Logger.Errorf("failed to send sentry event to subscription: %v", err)
		}
	}

	ctx.Response.WriteHeader(http.StatusOK)
}

func (s *Sentry) Actions() []core.Action {
	return []core.Action{}
}

func (s *Sentry) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}

func (s *Sentry) Components() []core.Component {
	return []core.Component{
		&UpdateIssue{},
	}
}

func (s *Sentry) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnIssueEvent{},
	}
}
