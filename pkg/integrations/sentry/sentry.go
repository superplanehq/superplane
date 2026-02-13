package sentry

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegrationWithWebhookHandler("sentry", &Sentry{}, &SentryWebhookHandler{})
}

type Sentry struct{}

type Configuration struct {
	AuthToken string `json:"authToken"`
	BaseURL   string `json:"baseURL"`
}

func (s *Sentry) Name() string {
	return "sentry"
}

func (s *Sentry) Label() string {
	return "Sentry"
}

func (s *Sentry) Icon() string {
	return "alert-triangle"
}

func (s *Sentry) Description() string {
	return "Trigger workflows from Sentry issue events and update issues from workflows"
}

func (s *Sentry) Instructions() string {
	return `After you click Connect: go to Settings → Integrations → SuperPlane and install the Sentry app.`
}

func (s *Sentry) Configuration() []configuration.Field {
	return []configuration.Field{}
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

func (s *Sentry) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (s *Sentry) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("decode config: %w", err)
	}

	// Public Integration OAuth flow stores tokens in integration secrets.
	// When the integration is first created, it may not have either a personal
	// token or OAuth tokens yet (waiting for /sentry/setup attach). In that case,
	// don't fail Sync and don't mark the integration as errored.
	if strings.TrimSpace(config.AuthToken) == "" {
		if secrets, err := ctx.Integration.GetSecrets(); err == nil {
			for _, secret := range secrets {
				if secret.Name == "sentryPublicAccessToken" && strings.TrimSpace(string(secret.Value)) != "" {
					goto hasToken
				}
			}
		}
		return nil
	}

hasToken:

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}
	if err := client.ValidateToken(); err != nil {
		// If this integration is using OAuth tokens, try refresh.
		if refreshed, refreshErr := s.tryRefreshOAuthToken(ctx); refreshErr == nil && refreshed {
			client, err = NewClient(ctx.HTTP, ctx.Integration)
			if err != nil {
				return err
			}
			if err := client.ValidateToken(); err != nil {
				return fmt.Errorf("invalid Sentry token after refresh: %w", err)
			}
		} else {
			return fmt.Errorf("invalid Sentry token: %w", err)
		}
	}

	// Keep OAuth tokens fresh.
	_ = ctx.Integration.ScheduleResync(7 * time.Hour)
	ctx.Integration.Ready()
	return nil
}

func (s *Sentry) tryRefreshOAuthToken(ctx core.SyncContext) (bool, error) {
	installationID := ""
	if md, ok := ctx.Integration.GetMetadata().(map[string]any); ok {
		if v, ok := md["sentryInstallationID"].(string); ok {
			installationID = strings.TrimSpace(v)
		}
	}
	if installationID == "" {
		return false, fmt.Errorf("missing sentryInstallationID")
	}

	clientID := strings.TrimSpace(os.Getenv("SENTRY_PUBLIC_CLIENT_ID"))
	clientSecret := strings.TrimSpace(os.Getenv("SENTRY_PUBLIC_CLIENT_SECRET"))
	if clientID == "" || clientSecret == "" {
		return false, fmt.Errorf("SENTRY_PUBLIC_CLIENT_ID and SENTRY_PUBLIC_CLIENT_SECRET must be set")
	}

	refreshToken := ""
	secrets, err := ctx.Integration.GetSecrets()
	if err != nil {
		return false, err
	}
	for _, s := range secrets {
		if s.Name == "sentryPublicRefreshToken" {
			refreshToken = strings.TrimSpace(string(s.Value))
			break
		}
	}
	if refreshToken == "" {
		return false, fmt.Errorf("missing sentryPublicRefreshToken")
	}

	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("SENTRY_PUBLIC_BASE_URL")), "/")
	if baseURL == "" {
		baseURL = "https://sentry.io"
	}

	auth, err := refreshSentryAuthorizationToken(context.Background(), ctx.HTTP, baseURL, installationID, refreshToken, clientID, clientSecret)
	if err != nil {
		return false, err
	}

	if err := ctx.Integration.SetSecret("sentryPublicAccessToken", []byte(auth.Token)); err != nil {
		return false, err
	}
	if strings.TrimSpace(auth.RefreshToken) != "" {
		_ = ctx.Integration.SetSecret("sentryPublicRefreshToken", []byte(auth.RefreshToken))
	}
	return true, nil
}

func (s *Sentry) HandleRequest(ctx core.HTTPRequestContext) {}

func (s *Sentry) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	return nil, nil
}

func (s *Sentry) Actions() []core.Action {
	return nil
}

func (s *Sentry) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}
