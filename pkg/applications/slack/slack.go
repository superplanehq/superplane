package slack

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const (
	appBootstrapDescription = `
To complete the Slack app setup:
1.  The "**Create Slack App**" button/link will take you to Slack with the app manifest pre-filled.
2.  Review the manifest and click "**Next**", then "**Create**".
3.  **Install App**: On the next page, click "**Install to Workspace**" and authorize.
4.  **Get Bot Token**: Navigate to "OAuth & Permissions" (under Features in the sidebar). Copy the "**Bot User OAuth Token**".
5.  **Get Signing Secret**: Navigate to "Basic Information" (under Settings in the sidebar). Scroll down to "App Credentials" and copy the "**Signing Secret**".
6.  **Update Configuration**: Paste the "Bot User OAuth Token" and "Signing Secret" into this SuperPlane App's configuration fields and save.
`
)

func init() {
	registry.RegisterApplication("slack", &Slack{})
}

type Slack struct{}

func (s *Slack) Name() string {
	return "slack"
}

func (s *Slack) Label() string {
	return "Slack"
}

func (s *Slack) Icon() string {
	return "slack"
}

func (s *Slack) Description() string {
	return "Slack"
}

func (s *Slack) Configuration() []configuration.Field {
	//
	// Both fields are not required, because they will only be filled in after the app is created.
	//
	return []configuration.Field{
		{
			Name:        "botToken",
			Label:       "Bot Token",
			Type:        configuration.FieldTypeString,
			Description: "The bot token for the Slack app",
			Sensitive:   true,
			Required:    false,
		},
		{
			Name:        "signingSecret",
			Label:       "Signing Secret",
			Type:        configuration.FieldTypeString,
			Description: "The signing secret for the Slack app",
			Sensitive:   true,
			Required:    false,
		},
	}
}

func (s *Slack) Components() []core.Component {
	return []core.Component{}
}

func (s *Slack) Triggers() []core.Trigger {
	return []core.Trigger{}
}

func (s *Slack) Sync(ctx core.SyncContext) error {
	// TODO: metadata? user, bot, team ID?

	botToken, err := ctx.AppInstallation.GetConfig("botToken")
	if err != nil {
		return fmt.Errorf("failed to get bot token: %v", err)
	}

	signingSecret, err := ctx.AppInstallation.GetConfig("signingSecret")
	if err != nil {
		return fmt.Errorf("failed to get signing secret: %v", err)
	}

	//
	// If tokens are configured, verify if the auth is working,
	// by using the bot token to send a message to the channel.
	//
	if botToken != nil && signingSecret != nil {
		client, err := NewClient(string(botToken))
		if err != nil {
			return err
		}

		return client.AuthTest()
	}

	return s.createAppCreationPrompt(ctx)
}

func (s *Slack) createAppCreationPrompt(ctx core.SyncContext) error {
	manifestJSON, err := s.appManifest(ctx)
	if err != nil {
		return fmt.Errorf("failed to create manifest: %v", err)
	}

	encodedManifest := url.QueryEscape(string(manifestJSON))
	manifestURL := fmt.Sprintf("https://api.slack.com/apps?new_app=1&manifest_json=%s", encodedManifest)

	ctx.AppInstallation.NewBrowserAction(core.BrowserAction{
		Description: appBootstrapDescription,
		URL:         manifestURL,
		Method:      "GET",
	})
	return nil
}

func (s *Slack) appManifest(ctx core.SyncContext) ([]byte, error) {
	appURL := ctx.WebhooksBaseURL
	if appURL == "" {
		appURL = ctx.BaseURL
	}

	manifest := map[string]any{
		"_metadata": map[string]int{
			"major_version": 1,
			"minor_version": 2,
		},
		"display_information": map[string]string{
			"name":             "SuperPlane Integration",
			"description":      "Integration with SuperPlane",
			"background_color": "#2E2D2D",
		},
		"features": map[string]any{
			"bot_user": map[string]any{
				"display_name":  "SuperPlane Bot",
				"always_online": false,
			},
			"app_home": map[string]any{
				"home_tab_enabled":               false,
				"messages_tab_enabled":           true,
				"messages_tab_read_only_enabled": true,
			},
		},
		"oauth_config": map[string]any{
			"scopes": map[string]any{
				"bot": []string{
					"app_mentions:read",
					"chat:write",
					"chat:write.public",
					"channels:history",
					"groups:history",
					"im:history",
					"mpim:history",
					"reactions:write",
					"reactions:read",
					"usergroups:write",
					"usergroups:read",
					"channels:manage",
					"groups:write",
					"channels:read",
					"groups:read",
					"users:read",
				},
			},
		},
		"settings": map[string]any{
			"event_subscriptions": map[string]any{
				"request_url": fmt.Sprintf("%s/api/v1/apps/%s/events", appURL, ctx.InstallationID),
				"bot_events": []string{
					"app_mention",
					"reaction_added",
					"reaction_removed",
					"message.channels",
					"message.groups",
					"message.im",
					"message.mpim",
				},
			},
			"interactivity": map[string]any{
				"is_enabled":  true,
				"request_url": fmt.Sprintf("%s/api/v1/apps/%s/interactions", appURL, ctx.InstallationID),
			},
			"org_deploy_enabled":  false,
			"socket_mode_enabled": false,
		},
	}

	return json.Marshal(manifest)
}

func (s *Slack) HandleRequest(ctx core.HTTPRequestContext) {
	body, err := s.readAndVerify(ctx)
	if err != nil {
		ctx.Logger.Errorf("error verifying slack request: %v", err)
		return
	}

	if strings.HasSuffix(ctx.Request.URL.Path, "/events") {
		s.handleEvent(ctx, body)
		return
	}

	if strings.HasSuffix(ctx.Request.URL.Path, "/interactions") {
		s.handleInteraction(ctx, body)
		return
	}

	ctx.Logger.Warnf("unknown path: %s", ctx.Request.URL.Path)
	ctx.Response.WriteHeader(http.StatusNotFound)

	// TODO: verify request actually comes from Slack
	// TODO: based on the event type (app_mention, reaction_added, reaction_removed, message, ...),
	//       find app appropriate nodes that listen to it, and forward the event to them.
}

func (s *Slack) readAndVerify(ctx core.HTTPRequestContext) ([]byte, error) {
	signingSecret, err := ctx.AppInstallation.GetConfig("signingSecret")
	if err != nil {
		return nil, fmt.Errorf("error finding signing secret: %v", err)
	}

	if signingSecret == nil {
		return nil, fmt.Errorf("signing secret not configured")
	}

	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading request body: %v", err)
	}

	timestampHeader := ctx.Request.Header.Get("X-Slack-Request-Timestamp")
	if timestampHeader == "" {
		return nil, fmt.Errorf("missing X-Slack-Request-Timestamp header")
	}

	signatureHeader := ctx.Request.Header.Get("X-Slack-Signature")
	if signatureHeader == "" {
		return nil, fmt.Errorf("missing X-Slack-Signature header")
	}

	timestamp, err := strconv.ParseInt(timestampHeader, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid timestamp format: %v", err)
	}

	// Validate timestamp to prevent replay attacks (within 5 minutes)
	requestTime := time.Unix(timestamp, 0)
	timeDiff := time.Since(requestTime)
	if timeDiff < 0 {
		timeDiff = -timeDiff
	}
	if timeDiff > 5*time.Minute {
		return nil, fmt.Errorf("request timestamp too old: %v", timeDiff)
	}

	// Create the signature base string: v0:{timestamp}:{body}
	sigBaseString := fmt.Sprintf("v0:%d:%s", timestamp, string(body))

	// Compute HMAC-SHA256
	h := hmac.New(sha256.New, signingSecret)
	h.Write([]byte(sigBaseString))
	computedSignature := fmt.Sprintf("v0=%s", hex.EncodeToString(h.Sum(nil)))

	// Compare signatures using constant-time comparison
	if !hmac.Equal([]byte(computedSignature), []byte(signatureHeader)) {
		return nil, fmt.Errorf("invalid signature")
	}

	return body, nil
}

func (s *Slack) handleEvent(ctx core.HTTPRequestContext, body []byte) {

}

func (s *Slack) handleInteraction(ctx core.HTTPRequestContext, body []byte) {

}

type WebhookConfiguration struct {
	EventTypes []string `json:"eventTypes"`
}

func (s *Slack) CompareWebhookConfig(a, b any) (bool, error) {
	configA := WebhookConfiguration{}
	configB := WebhookConfiguration{}

	err := mapstructure.Decode(a, &configA)
	if err != nil {
		return false, err
	}

	err = mapstructure.Decode(b, &configB)
	if err != nil {
		return false, err
	}

	return slices.Equal(configA.EventTypes, configB.EventTypes), nil
}

func (s *Slack) SetupWebhook(ctx core.AppInstallationContext, options core.WebhookOptions) (any, error) {
	return nil, nil
}

func (s *Slack) CleanupWebhook(ctx core.AppInstallationContext, options core.WebhookOptions) error {
	return nil
}
