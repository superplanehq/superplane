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
1.  The "**Create Slack App**" button/link will take you to Slack with the app manifest pre-filled
2.  Review the manifest and click "**Next**", then "**Create**"
3.  **Get Signing Secret**: In "Basic Information" section, copy the "**Signing Secret**"
4.  **Install App**: In OAuth & Permissions, click "**Install to Workspace**" and authorize
5.  **Get Bot Token**: In "OAuth & Permissions", copy the "**Bot User OAuth Token**"
6.  **Update Configuration**: Paste the "Bot User OAuth Token" and "Signing Secret" into the app installation configuration fields in SuperPlane and save
`
)

func init() {
	registry.RegisterApplication("slack", &Slack{})
}

type Slack struct{}

type Metadata struct {
	URL    string `mapstructure:"url" json:"url"`
	TeamID string `mapstructure:"team_id" json:"team_id"`
	Team   string `mapstructure:"team" json:"team"`
	UserID string `mapstructure:"user_id" json:"user_id"`
	User   string `mapstructure:"user" json:"user"`
	BotID  string `mapstructure:"bot_id" json:"bot_id"`
}

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
	return "Send and react to Slack messages and interactions"
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
	return []core.Component{
		&SendTextMessage{},
	}
}

func (s *Slack) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnAppMention{},
	}
}

func (s *Slack) Sync(ctx core.SyncContext) error {
	metadata := Metadata{}
	err := mapstructure.Decode(ctx.AppInstallation.GetMetadata(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to decode metadata: %v", err)
	}

	//
	// If metadata is already set, nothing to do.
	//
	if metadata.URL != "" {
		return nil
	}

	botToken, _ := ctx.AppInstallation.GetConfig("botToken")
	signingSecret, _ := ctx.AppInstallation.GetConfig("signingSecret")

	//
	// If tokens are configured, verify if the auth is working,
	// by using the bot token to send a message to the channel.
	//
	if botToken != nil && signingSecret != nil {
		client, err := NewClient(ctx.AppInstallation)
		if err != nil {
			return err
		}

		result, err := client.AuthTest()
		if err != nil {
			return fmt.Errorf("error verifying slack auth: %v", err)
		}

		ctx.AppInstallation.SetMetadata(Metadata{
			URL:    result.URL,
			TeamID: result.TeamID,
			Team:   result.Team,
			UserID: result.UserID,
			User:   result.User,
			BotID:  result.BotID,
		})

		ctx.AppInstallation.RemoveBrowserAction()
		ctx.AppInstallation.SetState("ready", "")
		return nil
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

	//
	// TODO: a few other options to consider here:
	// features.app_home.*
	// settings.interactivity.optionsLoadURL
	// Verify if we want incoming webhooks and if it's possible to include that in the manifest here.
	//

	// "token_rotation_enabled": true

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
		ctx.Response.WriteHeader(400)
		return
	}

	if strings.HasSuffix(ctx.Request.URL.Path, "/events") {
		s.handleEvent(ctx, body)
		return
	}

	if strings.HasSuffix(ctx.Request.URL.Path, "/interactions") {
		s.handleInteractivity(ctx, body)
		return
	}

	ctx.Logger.Warnf("unknown path: %s", ctx.Request.URL.Path)
	ctx.Response.WriteHeader(http.StatusNotFound)
}

type EventPayload struct {
	Type  string         `json:"type"`
	Event map[string]any `json:"event"`
}

func (s *Slack) handleEvent(ctx core.HTTPRequestContext, body []byte) {
	payload := EventPayload{}
	err := json.Unmarshal(body, &payload)
	if err != nil {
		ctx.Logger.Errorf("error unmarshaling event payload: %v", err)
		ctx.Response.WriteHeader(400)
		return
	}

	if payload.Type == "url_verification" {
		s.handleChallenge(ctx, payload.Event)
		return
	}

	if payload.Type != "event_callback" {
		ctx.Logger.Warnf("ignoring event type: %s", payload.Type)
		return
	}

	eventType, event, err := s.parseEventCallback(payload)
	if err != nil {
		ctx.Logger.Errorf("error parsing event callback: %v", err)
		ctx.Response.WriteHeader(400)
		return
	}

	subscriptions, err := ctx.AppInstallation.ListSubscriptions()
	if err != nil {
		ctx.Logger.Errorf("error listing subscriptions: %v", err)
		ctx.Response.WriteHeader(500)
		return
	}

	for _, subscription := range subscriptions {
		if !s.subscriptionApplies(ctx, subscription, eventType) {
			continue
		}

		err = subscription.SendMessage(event)
		if err != nil {
			ctx.Logger.Errorf("error sending message from app: %v", err)
		}
	}
}

func (s *Slack) handleChallenge(ctx core.HTTPRequestContext, event any) {
	eventMap, ok := event.(map[string]any)
	if !ok {
		return
	}

	challenge, ok := eventMap["challenge"].(string)
	if !ok {
		return
	}

	ctx.Response.WriteHeader(200)
	ctx.Response.Write([]byte(challenge))
}

func (s *Slack) handleInteractivity(ctx core.HTTPRequestContext, body []byte) {
	// TODO
}

func (s *Slack) parseEventCallback(eventPayload EventPayload) (string, any, error) {
	t, ok := eventPayload.Event["type"]
	if !ok {
		return "", nil, fmt.Errorf("type not found in event")
	}

	eventType, ok := t.(string)
	if !ok {
		return "", nil, fmt.Errorf("type is of type %T: %v", t, t)
	}

	return eventType, eventPayload.Event, nil
}

type SubscriptionConfiguration struct {
	EventTypes []string `json:"eventTypes"`
}

func (s *Slack) subscriptionApplies(ctx core.HTTPRequestContext, subscription core.AppSubscriptionContext, eventType string) bool {
	c := SubscriptionConfiguration{}
	err := mapstructure.Decode(subscription.Configuration(), &c)
	if err != nil {
		ctx.Logger.Errorf("error decoding subscription configuration: %v", err)
		return false
	}

	return slices.ContainsFunc(c.EventTypes, func(t string) bool {
		return t == eventType
	})
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

/*
 * All the events we receive from Slack are on the app's HandleWebhook(),
 * so all the Slack components and triggers use app subscriptions,
 * and not webhooks.
 */

func (s *Slack) CompareWebhookConfig(a, b any) (bool, error) {
	return false, nil
}

func (s *Slack) SetupWebhook(ctx core.SetupWebhookContext) (any, error) {
	return nil, nil
}

func (s *Slack) CleanupWebhook(ctx core.CleanupWebhookContext) error {
	return nil
}
