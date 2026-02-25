package telegram

import (
	"encoding/json"
	"fmt"
	"io"
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

func init() {
	registry.RegisterIntegration("telegram", &Telegram{})
}

type Telegram struct{}

type Configuration struct {
	BotToken string `json:"botToken" mapstructure:"botToken"`
}

type Metadata struct {
	BotID     int64  `json:"botId" mapstructure:"botId"`
	Username  string `json:"username" mapstructure:"username"`
	FirstName string `json:"firstName" mapstructure:"firstName"`
}

type SubscriptionConfiguration struct {
	EventTypes []string `json:"eventTypes" mapstructure:"eventTypes"`
}

func (t *Telegram) Name() string {
	return "telegram"
}

func (t *Telegram) Label() string {
	return "Telegram"
}

func (t *Telegram) Icon() string {
	return "telegram"
}

func (t *Telegram) Description() string {
	return "Send messages and react to events via Telegram bots"
}

func (t *Telegram) Instructions() string {
	return `To set up Telegram integration:

1. Get a bot token from @BotFather and paste it in the field below
2. Disable privacy mode so the bot can receive messages in groups: send /setprivacy to @BotFather, select your bot, and choose Disable
3. Add the bot to your group or channel

Note: if the bot was already in a group before disabling privacy mode, remove and re-add it for the change to take effect.`
}

func (t *Telegram) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "botToken",
			Label:       "Bot Token",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Telegram bot token from BotFather",
		},
	}
}

func (t *Telegram) Components() []core.Component {
	return []core.Component{
		&SendMessage{},
		&WaitForButtonClick{},
	}
}

func (t *Telegram) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnMention{},
	}
}

func (t *Telegram) Sync(ctx core.SyncContext) error {
	// Get the decrypted bot token
	botTokenBytes, err := ctx.Integration.GetConfig("botToken")
	if err != nil {
		return fmt.Errorf("botToken is required")
	}

	botToken := string(botTokenBytes)
	if botToken == "" {
		return fmt.Errorf("botToken is required")
	}

	// Verify the bot token is valid by getting bot info
	client, err := NewClient(ctx.Integration)
	if err != nil {
		return err
	}

	botUser, err := client.GetMe()
	if err != nil {
		return fmt.Errorf("failed to verify bot token: %v", err)
	}

	// Set webhook URL for receiving updates
	webhookURL := ctx.WebhooksBaseURL
	if webhookURL == "" {
		webhookURL = ctx.BaseURL
	}
	webhookURL = fmt.Sprintf("%s/api/v1/integrations/%s/events", webhookURL, ctx.Integration.ID().String())

	err = client.SetWebhook(webhookURL)
	if err != nil {
		return fmt.Errorf("failed to set webhook: %v", err)
	}

	ctx.Integration.SetMetadata(Metadata{
		BotID:     botUser.ID,
		Username:  botUser.Username,
		FirstName: botUser.FirstName,
	})

	ctx.Integration.Ready()
	return nil
}

func (t *Telegram) HandleRequest(ctx core.HTTPRequestContext) {
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		ctx.Logger.Errorf("error reading request body: %v", err)
		ctx.Response.WriteHeader(400)
		return
	}

	var update Update
	if err := json.Unmarshal(body, &update); err != nil {
		ctx.Logger.Errorf("error unmarshaling update: %v", err)
		ctx.Response.WriteHeader(400)
		return
	}

	if update.CallbackQuery != nil {
		t.handleCallbackQuery(ctx, update)
		return
	}

	// Only process messages
	if update.Message == nil {
		ctx.Response.WriteHeader(200)
		return
	}

	// Check if this is a bot mention
	metadata := Metadata{}
	m := ctx.Integration.GetMetadata()
	if m != nil {
		mMap, ok := m.(map[string]any)
		if ok {
			if botID, ok := mMap["botId"].(float64); ok {
				metadata.BotID = int64(botID)
			}
			if username, ok := mMap["username"].(string); ok {
				metadata.Username = username
			}
			if firstName, ok := mMap["firstName"].(string); ok {
				metadata.FirstName = firstName
			}
		}
	}

	isMention := false
	botUsername := "@" + metadata.Username

	// Check for mentions in entities
	for _, entity := range update.Message.Entities {
		if entity.Type == "mention" {
			// Extract the mentioned username from the message text
			mentionedText := update.Message.Text[entity.Offset : entity.Offset+entity.Length]
			if strings.EqualFold(mentionedText, botUsername) {
				isMention = true
				break
			}
		}
	}

	if !isMention {
		ctx.Response.WriteHeader(200)
		return
	}

	// Get subscriptions and dispatch
	subscriptions, err := ctx.Integration.ListSubscriptions()
	if err != nil {
		ctx.Logger.Errorf("error listing subscriptions: %v", err)
		ctx.Response.WriteHeader(500)
		return
	}

	for _, subscription := range subscriptions {
		config := SubscriptionConfiguration{}
		if err := mapstructure.Decode(subscription.Configuration(), &config); err != nil {
			ctx.Logger.Errorf("error decoding subscription configuration: %v", err)
			continue
		}

		if !slices.ContainsFunc(config.EventTypes, func(t string) bool {
			return t == "message_mention"
		}) {
			continue
		}

		// Convert message to map for serialization
		messageMap := map[string]any{
			"message_id": update.Message.MessageID,
			"text":       update.Message.Text,
			"date":       update.Message.Date,
		}

		if update.Message.From != nil {
			messageMap["from"] = map[string]any{
				"id":         update.Message.From.ID,
				"is_bot":     update.Message.From.IsBot,
				"first_name": update.Message.From.FirstName,
				"username":   update.Message.From.Username,
			}
		}

		messageMap["chat"] = map[string]any{
			"id":    update.Message.Chat.ID,
			"type":  update.Message.Chat.Type,
			"title": update.Message.Chat.Title,
		}

		if len(update.Message.Entities) > 0 {
			entities := make([]map[string]any, len(update.Message.Entities))
			for i, entity := range update.Message.Entities {
				entities[i] = map[string]any{
					"type":   entity.Type,
					"offset": entity.Offset,
					"length": entity.Length,
				}
			}
			messageMap["entities"] = entities
		}

		err = subscription.SendMessage(messageMap)
		if err != nil {
			ctx.Logger.Errorf("error sending message from integration: %v", err)
		}
	}

	ctx.Response.WriteHeader(200)
}

func (t *Telegram) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (t *Telegram) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	return []core.IntegrationResource{}, nil
}

func (t *Telegram) Actions() []core.Action {
	return []core.Action{}
}

func (t *Telegram) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}

func (t *Telegram) handleCallbackQuery(ctx core.HTTPRequestContext, update Update) {
	query := update.CallbackQuery
	if query.Message == nil {
		ctx.Response.WriteHeader(200)
		return
	}

	messageID := query.Message.MessageID
	chatID := query.Message.Chat.ID
	chatIDStr := strconv.FormatInt(chatID, 10)

	subscriptions, err := ctx.Integration.ListSubscriptions()
	if err != nil {
		ctx.Logger.Errorf("error listing subscriptions: %v", err)
		ctx.Response.WriteHeader(500)
		return
	}

	var matchedSubscription core.IntegrationSubscriptionContext
	for _, sub := range subscriptions {
		cfg, ok := sub.Configuration().(map[string]any)
		if !ok {
			continue
		}

		subType, _ := cfg["type"].(string)
		subChatID, _ := cfg["chat_id"].(string)

		var subMessageID int64
		switch v := cfg["message_id"].(type) {
		case int64:
			subMessageID = v
		case float64:
			subMessageID = int64(v)
		}

		if subType == "button_click" && subMessageID == messageID && subChatID == chatIDStr {
			matchedSubscription = sub
			break
		}
	}

	if matchedSubscription == nil {
		ctx.Response.WriteHeader(200)
		return
	}

	cfg, _ := matchedSubscription.Configuration().(map[string]any)
	executionIDStr, _ := cfg["execution_id"].(string)
	executionID, err := uuid.Parse(executionIDStr)
	if err != nil {
		ctx.Logger.Errorf("error parsing execution ID: %v", err)
		ctx.Response.WriteHeader(200)
		return
	}

	var clickedBy map[string]any
	if query.From != nil {
		clickedBy = map[string]any{
			"id":       query.From.ID,
			"username": query.From.Username,
		}
	}

	if err := t.createButtonClickAction(executionID, query.Data, clickedBy); err != nil {
		ctx.Logger.Errorf("error creating button click action: %v", err)
		ctx.Response.WriteHeader(500)
		return
	}

	client, err := NewClient(ctx.Integration)
	if err != nil {
		ctx.Logger.Errorf("error creating Telegram client: %v", err)
		ctx.Response.WriteHeader(200)
		return
	}

	if err := client.AnswerCallbackQuery(query.ID); err != nil {
		ctx.Logger.Errorf("error answering callback query: %v", err)
	}

	ctx.Response.WriteHeader(200)
}

func (t *Telegram) createButtonClickAction(executionID uuid.UUID, buttonValue string, clickedBy map[string]any) error {
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

// Telegram API types for parsing webhook updates

type Update struct {
	UpdateID      int64            `json:"update_id"`
	Message       *TelegramMessage `json:"message,omitempty"`
	CallbackQuery *CallbackQuery   `json:"callback_query,omitempty"`
}

type CallbackQuery struct {
	ID      string           `json:"id"`
	From    *User            `json:"from,omitempty"`
	Message *TelegramMessage `json:"message,omitempty"`
	Data    string           `json:"data"`
}

type TelegramMessage struct {
	MessageID int64           `json:"message_id"`
	From      *User           `json:"from,omitempty"`
	Chat      Chat            `json:"chat"`
	Text      string          `json:"text,omitempty"`
	Entities  []MessageEntity `json:"entities,omitempty"`
	Date      int64           `json:"date"`
}

type User struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	Username  string `json:"username,omitempty"`
}

type Chat struct {
	ID    int64  `json:"id"`
	Type  string `json:"type"`
	Title string `json:"title,omitempty"`
}

type MessageEntity struct {
	Type   string `json:"type"`
	Offset int    `json:"offset"`
	Length int    `json:"length"`
}
