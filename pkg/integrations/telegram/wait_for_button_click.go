package telegram

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	ChannelReceived = "received"
	ChannelTimeout  = "timeout"

	ActionButtonClick = "buttonClick"
	ActionTimeout     = "timeout"
)

type WaitForButtonClick struct{}

type WaitForButtonClickConfiguration struct {
	ChatID  string   `json:"chatId" mapstructure:"chatId"`
	Message string   `json:"message" mapstructure:"message"`
	Timeout *int     `json:"timeout,omitempty" mapstructure:"timeout,omitempty"`
	Buttons []Button `json:"buttons" mapstructure:"buttons"`
}

type Button struct {
	Name  string `json:"name" mapstructure:"name"`
	Value string `json:"value" mapstructure:"value"`
}

type ChatInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type WaitForButtonClickMetadata struct {
	Chat              *ChatInfo `json:"chat"`
	MessageID         *int64    `json:"messageID,omitempty"`
	SelectedButton    *string   `json:"selectedButton,omitempty"`
	AppSubscriptionID *string   `json:"appSubscriptionID,omitempty"`
}

func (c *WaitForButtonClick) Name() string {
	return "telegram.waitForButtonClick"
}

func (c *WaitForButtonClick) Label() string {
	return "Wait for Button Click"
}

func (c *WaitForButtonClick) Description() string {
	return "Send a message with buttons and wait for the user to click one"
}

func (c *WaitForButtonClick) Documentation() string {
	return `The Wait for Button Click component sends a message to a Telegram chat with inline keyboard buttons and waits for the user to click one of the configured buttons.

## Use Cases

- **Request approval or input**: Get structured input from a user in Telegram before applying or deploying (e.g., Approve / Reject buttons)
- **Pause a workflow**: Wait until a human selects an option (e.g., Confirm / Cancel)
- **Implement interactive flows**: Create interactive flows that need a structured reply via buttons

## Configuration

- **Chat ID**: Telegram chat ID (user, group, or channel) to post to (required)
- **Message**: Message text (required)
- **Timeout**: Maximum time to wait in seconds (optional)
- **Buttons**: Set of 1–4 items, each with name (label) and value (required)

## Output Channels

- **Received**: Emits when the user clicks a button; payload includes the selected value and clicker info (when available)
- **Timeout**: Emits when no button click is received within the configured timeout

## Behavior

- The message is posted with inline keyboard buttons
- The workflow pauses until a button is clicked or timeout occurs
- Only the first button click is processed; subsequent clicks are ignored
- If timeout is not configured, the component waits indefinitely

## Notes

- The Telegram bot must be added to the chat and have permission to post messages
- Button clicks are processed through Telegram's callback query API`
}

func (c *WaitForButtonClick) Icon() string {
	return "telegram"
}

func (c *WaitForButtonClick) Color() string {
	return "gray"
}

func (c *WaitForButtonClick) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: ChannelReceived, Label: "Received", Description: "Emits when a button is clicked"},
		{Name: ChannelTimeout, Label: "Timeout", Description: "Emits when timeout is reached"},
	}
}

func (c *WaitForButtonClick) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "chatId",
			Label:       "Chat ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Telegram chat ID (user, group, or channel)",
		},
		{
			Name:     "message",
			Label:    "Message",
			Type:     configuration.FieldTypeText,
			Required: true,
		},
		{
			Name:        "timeout",
			Label:       "Timeout",
			Type:        configuration.FieldTypeNumber,
			Description: "Maximum time to wait in seconds (leave empty to wait indefinitely)",
			Required:    false,
			Default:     "3600",
		},
		{
			Name:        "buttons",
			Label:       "Buttons",
			Description: "Set of 1–4 buttons to display. Each button must have a name (label) and value.",
			Type:        configuration.FieldTypeList,
			Required:    true,
			Default:     `[{"name":"Approve","value":"approve"},{"name":"Reject","value":"reject"}]`,
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Button",
					MaxItems:  ptrInt(4),
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:     "name",
								Label:    "Button Label",
								Type:     configuration.FieldTypeString,
								Required: true,
							},
							{
								Name:     "value",
								Label:    "Button Value",
								Type:     configuration.FieldTypeString,
								Required: true,
							},
						},
					},
				},
			},
		},
	}
}

func ptrInt(v int) *int { return &v }

func validateButtons(buttons []Button) error {
	if len(buttons) == 0 {
		return errors.New("at least one button is required")
	}

	if len(buttons) > 4 {
		return errors.New("maximum of 4 buttons allowed")
	}

	for i, button := range buttons {
		if button.Name == "" {
			return fmt.Errorf("button %d: name is required", i)
		}
		if button.Value == "" {
			return fmt.Errorf("button %d: value is required", i)
		}
	}

	buttonValues := make(map[string]bool)
	for i, button := range buttons {
		if buttonValues[button.Value] {
			return fmt.Errorf("button %d: duplicate value '%s' - each button must have a unique value", i, button.Value)
		}
		buttonValues[button.Value] = true
	}

	return nil
}

func (c *WaitForButtonClick) Setup(ctx core.SetupContext) error {
	var config WaitForButtonClickConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.ChatID == "" {
		return errors.New("chatId is required")
	}

	if config.Message == "" {
		return errors.New("message is required")
	}

	if err := validateButtons(config.Buttons); err != nil {
		return err
	}

	client, err := NewClient(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create Telegram client: %w", err)
	}

	chatInfo, err := client.GetChat(config.ChatID)
	if err != nil {
		return fmt.Errorf("chat validation failed: %w", err)
	}

	metadata := WaitForButtonClickMetadata{
		Chat: &ChatInfo{
			ID:   fmt.Sprintf("%d", chatInfo.ID),
			Name: ChatDisplayName(chatInfo),
		},
	}

	return ctx.Metadata.Set(metadata)
}

func (c *WaitForButtonClick) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *WaitForButtonClick) Execute(ctx core.ExecutionContext) error {
	var config WaitForButtonClickConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.ChatID == "" {
		return errors.New("chatId is required")
	}

	if config.Message == "" {
		return errors.New("message is required")
	}

	if err := validateButtons(config.Buttons); err != nil {
		return err
	}

	client, err := NewClient(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create Telegram client: %w", err)
	}

	allButtons := make([]InlineKeyboardButton, 0, len(config.Buttons))
	for _, button := range config.Buttons {
		allButtons = append(allButtons, InlineKeyboardButton{
			Text:         button.Name,
			CallbackData: button.Value,
		})
	}

	response, err := client.SendMessageWithInlineKeyboard(config.ChatID, config.Message, [][]InlineKeyboardButton{allButtons})
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	chatIDString := fmt.Sprintf("%d", response.Chat.ID)

	subscriptionID, err := ctx.Integration.Subscribe(map[string]any{
		"type":         "button_click",
		"message_id":   response.MessageID,
		"chat_id":      chatIDString,
		"execution_id": ctx.ID.String(),
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to button clicks: %w", err)
	}

	var metadata WaitForButtonClickMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	messageID := response.MessageID
	subIDStr := subscriptionID.String()
	metadata.MessageID = &messageID
	metadata.AppSubscriptionID = &subIDStr

	if err := ctx.Metadata.Set(metadata); err != nil {
		return fmt.Errorf("failed to update metadata: %w", err)
	}

	if config.Timeout != nil && *config.Timeout > 0 {
		timeout := time.Duration(*config.Timeout) * time.Second
		if err := ctx.Requests.ScheduleActionCall(ActionTimeout, map[string]any{}, timeout); err != nil {
			return fmt.Errorf("failed to schedule timeout: %w", err)
		}
	}

	return nil
}

func (c *WaitForButtonClick) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *WaitForButtonClick) Actions() []core.Action {
	return []core.Action{
		{Name: ActionButtonClick},
		{Name: ActionTimeout},
	}
}

func (c *WaitForButtonClick) HandleAction(ctx core.ActionContext) error {
	switch ctx.Name {
	case ActionButtonClick:
		return c.handleButtonClick(ctx)
	case ActionTimeout:
		return c.handleTimeout(ctx)
	default:
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}
}

func (c *WaitForButtonClick) handleButtonClick(ctx core.ActionContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	var metadata WaitForButtonClickMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	buttonValue, ok := ctx.Parameters["value"].(string)
	if !ok {
		return errors.New("button value not found in parameters")
	}

	metadata.SelectedButton = &buttonValue
	if err := ctx.Metadata.Set(metadata); err != nil {
		return fmt.Errorf("failed to update metadata: %w", err)
	}

	payload := map[string]any{
		"value":      buttonValue,
		"clicked_at": time.Now().Format(time.RFC3339),
	}
	if clickedBy, ok := ctx.Parameters["clicked_by"].(map[string]any); ok && len(clickedBy) > 0 {
		payload["clicked_by"] = clickedBy
	}

	return ctx.ExecutionState.Emit(
		ChannelReceived,
		"telegram.button.clicked",
		[]any{payload},
	)
}

func (c *WaitForButtonClick) handleTimeout(ctx core.ActionContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	payload := map[string]any{
		"timeout_at": time.Now().Format(time.RFC3339),
	}

	return ctx.ExecutionState.Emit(
		ChannelTimeout,
		"telegram.button.timeout",
		[]any{payload},
	)
}

func (c *WaitForButtonClick) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *WaitForButtonClick) Cleanup(ctx core.SetupContext) error {
	return nil
}
