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

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
)

const (
	appBootstrapDescription = `
To complete the Slack app setup:
1.  The "**Continue**" button will take you to Slack with the app manifest pre-filled
2.  Choose the workspace, and click "**Next**"
3.  Review the manifest, and click "**Create**"
3.  **Get Signing Secret**: In "Basic Information" section, copy the "**Signing Secret**"
4.  **Install App**: In OAuth & Permissions, click "**Install to Workspace**" and authorize
5.  **Get Bot Token**: In "OAuth & Permissions", copy the "**Bot User OAuth Token**"
6.  **Update Configuration**: Paste the "Bot User OAuth Token" and "Signing Secret" into the app installation configuration fields in SuperPlane and save
`
)

func init() {
	registry.RegisterIntegration("slack", &Slack{})
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

func (s *Slack) Instructions() string {
	return `
You can install the Slack app without the **Bot Token** and **Signing Secret**.
After installation, follow the setup prompt to create the Slack app and add those values.
`
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

func (s *Slack) Actions() []core.Action {
	return []core.Action{}
}

func (s *Slack) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}

func (s *Slack) Components() []core.Component {
	return []core.Component{
		&SendTextMessage{},
		&WaitForButtonClick{},
	}
}

func (s *Slack) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnAppMention{},
	}
}

func (s *Slack) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (s *Slack) Sync(ctx core.SyncContext) error {
	metadata := Metadata{}
	err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to decode metadata: %v", err)
	}

	//
	// If metadata is already set, nothing to do.
	//
	if metadata.URL != "" {
		return nil
	}

	botToken, _ := ctx.Integration.GetConfig("botToken")
	signingSecret, _ := ctx.Integration.GetConfig("signingSecret")

	//
	// If tokens are configured, verify if the auth is working,
	// by using the bot token to send a message to the channel.
	//
	if botToken != nil && signingSecret != nil {
		client, err := NewClient(ctx.Integration)
		if err != nil {
			return err
		}

		result, err := client.AuthTest()
		if err != nil {
			return fmt.Errorf("error verifying slack auth: %v", err)
		}

		ctx.Integration.SetMetadata(Metadata{
			URL:    result.URL,
			TeamID: result.TeamID,
			Team:   result.Team,
			UserID: result.UserID,
			User:   result.User,
			BotID:  result.BotID,
		})

		ctx.Integration.RemoveBrowserAction()
		ctx.Integration.Ready()
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

	ctx.Integration.NewBrowserAction(core.BrowserAction{
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
				"request_url": fmt.Sprintf("%s/api/v1/integrations/%s/events", appURL, ctx.Integration.ID().String()),
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
				"request_url": fmt.Sprintf("%s/api/v1/integrations/%s/interactions", appURL, ctx.Integration.ID().String()),
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
	Type      string         `json:"type"`
	Event     map[string]any `json:"event"`
	Challenge string         `json:"challenge"`
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
		s.handleChallenge(ctx, payload)
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

	subscriptions, err := ctx.Integration.ListSubscriptions()
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

func (s *Slack) handleChallenge(ctx core.HTTPRequestContext, payload EventPayload) {
	if payload.Challenge == "" {
		ctx.Logger.Errorf("missing challenge in event payload")
		ctx.Response.WriteHeader(http.StatusBadRequest)
		return
	}

	challenge := payload.Challenge
	_, err := ctx.Response.Write([]byte(challenge))
	if err != nil {
		ctx.Logger.Errorf("error writing challenge: %v", err)
		return
	}
}

type InteractionPayload struct {
	Type        string              `json:"type"`
	User        map[string]any      `json:"user"`
	Container   map[string]any      `json:"container"`
	Actions     []InteractionAction `json:"actions"`
	ResponseURL string              `json:"response_url"`
	Message     map[string]any      `json:"message"`
	Channel     map[string]any      `json:"channel"`
	State       map[string]any      `json:"state"`
	Token       string              `json:"token"`
	APIAppID    string              `json:"api_app_id"`
	Team        map[string]any      `json:"team"`
}

type InteractionAction struct {
	Type     string         `json:"type"`
	ActionID string         `json:"action_id"`
	Value    string         `json:"value"`
	Text     map[string]any `json:"text"`
}

func (s *Slack) handleInteractivity(ctx core.HTTPRequestContext, body []byte) {
	formValues, err := url.ParseQuery(string(body))
	if err != nil {
		ctx.Logger.Errorf("error parsing form data: %v", err)
		ctx.Response.WriteHeader(http.StatusBadRequest)
		return
	}

	payloadStr := formValues.Get("payload")
	if payloadStr == "" {
		ctx.Logger.Errorf("missing payload in form data")
		ctx.Response.WriteHeader(http.StatusBadRequest)
		return
	}

	var payload InteractionPayload
	if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
		ctx.Logger.Errorf("error unmarshaling interaction payload: %v", err)
		ctx.Response.WriteHeader(http.StatusBadRequest)
		return
	}

	if payload.Type != "block_actions" {
		ctx.Logger.Infof("ignoring interaction type: %s", payload.Type)
		ctx.Response.WriteHeader(http.StatusOK)
		return
	}

	if len(payload.Actions) == 0 {
		ctx.Logger.Errorf("no actions in payload")
		ctx.Response.WriteHeader(http.StatusBadRequest)
		return
	}

	action := payload.Actions[0]
	if action.Type != "button" {
		ctx.Logger.Infof("ignoring action type: %s", action.Type)
		ctx.Response.WriteHeader(http.StatusOK)
		return
	}

	messageTS, ok := payload.Container["message_ts"].(string)
	if !ok {
		ctx.Logger.Errorf("message_ts not found in container")
		ctx.Response.WriteHeader(http.StatusBadRequest)
		return
	}

	channelID, ok := payload.Channel["id"].(string)
	if !ok {
		ctx.Logger.Errorf("channel id not found in payload")
		ctx.Response.WriteHeader(http.StatusBadRequest)
		return
	}

	subscription, err := ctx.Integration.FindSubscription(func(sub core.IntegrationSubscriptionContext) bool {
		config, ok := sub.Configuration().(map[string]any)
		if !ok {
			return false
		}

		subscriptionType, ok := config["type"].(string)
		if !ok {
			return false
		}
		subscriptionMessageTS, ok := config["message_ts"].(string)
		if !ok {
			return false
		}
		subscriptionChannelID, ok := config["channel_id"].(string)
		if !ok {
			return false
		}

		return subscriptionType == "button_click" &&
			subscriptionMessageTS == messageTS &&
			subscriptionChannelID == channelID
	})
	if err != nil {
		ctx.Logger.Errorf("error finding subscription: %v", err)
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	if subscription == nil {
		ctx.Logger.Warnf("no subscription found for message_ts %s", messageTS)
		ctx.Response.WriteHeader(http.StatusOK)
		return
	}

	config := subscription.Configuration()
	configMap, ok := config.(map[string]any)
	if !ok {
		ctx.Logger.Errorf("invalid subscription configuration")
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	executionIDStr, ok := configMap["execution_id"].(string)
	if !ok {
		ctx.Logger.Errorf("execution_id not found in subscription configuration")
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	executionID, err := uuid.Parse(executionIDStr)
	if err != nil {
		ctx.Logger.Errorf("invalid execution_id in subscription: %v", err)
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	clickedBy := map[string]any{}
	if userID, ok := payload.User["id"].(string); ok && userID != "" {
		clickedBy["id"] = userID
	}
	if username, ok := payload.User["username"].(string); ok && username != "" {
		clickedBy["username"] = username
	}
	if name, ok := payload.User["name"].(string); ok && name != "" {
		clickedBy["name"] = name
	}
	if realName, ok := payload.User["real_name"].(string); ok && realName != "" {
		clickedBy["real_name"] = realName
	}

	err = s.createButtonClickAction(executionID, action.Value, clickedBy)
	if err != nil {
		ctx.Logger.Errorf("error creating button click action: %v", err)
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	ctx.Response.WriteHeader(http.StatusOK)
}

func (s *Slack) createButtonClickAction(executionID uuid.UUID, buttonValue string, clickedBy map[string]any) error {
	var execution models.CanvasNodeExecution
	err := database.Conn().Where("id = ?", executionID).First(&execution).Error
	if err != nil {
		return fmt.Errorf("failed to find execution: %w", err)
	}

	parameters := map[string]any{
		"value": buttonValue,
	}
	if len(clickedBy) > 0 {
		parameters["clicked_by"] = clickedBy
	}

	runAt := time.Now()
	return execution.CreateRequest(database.Conn(), models.NodeRequestTypeInvokeAction, models.NodeExecutionRequestSpec{
		InvokeAction: &models.InvokeAction{
			ActionName: ActionButtonClick,
			Parameters: parameters,
		},
	}, &runAt)
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

func (s *Slack) subscriptionApplies(ctx core.HTTPRequestContext, subscription core.IntegrationSubscriptionContext, eventType string) bool {
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
	signingSecret, err := ctx.Integration.GetConfig("signingSecret")
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
