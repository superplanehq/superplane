package cursor

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	GetLastMessagePayloadType = "cursor.getLastMessage.result"
)

type GetLastMessage struct{}

type GetLastMessageSpec struct {
	AgentID string `json:"agentId" mapstructure:"agentId"`
}

type GetLastMessageOutput struct {
	AgentID string               `json:"agentId"`
	Message *ConversationMessage `json:"message"`
}

func (c *GetLastMessage) Name() string {
	return "cursor.getLastMessage"
}

func (c *GetLastMessage) Label() string {
	return "Get Last Message"
}

func (c *GetLastMessage) Description() string {
	return "Retrieves the last message from a Cursor Cloud Agent conversation."
}

func (c *GetLastMessage) Documentation() string {
	return `The Get Last Message component retrieves the last message from a Cursor Cloud Agent's conversation history.

## Use Cases

- **Message tracking**: Get the latest response or prompt from an agent conversation
- **Workflow automation**: Use the last message as input for downstream components
- **Status monitoring**: Check what the agent last communicated

## How It Works

1. Fetches the conversation history for the specified agent ID
2. Extracts the last message from the conversation
3. Returns the message details including ID, type (user_message or assistant_message), and text

## Configuration

- **Agent ID**: The unique identifier for the cloud agent (e.g., bc_abc123)

## Output

The output includes:
- **Agent ID**: The identifier of the agent
- **Message**: The last message object containing:
  - **ID**: Unique message identifier
  - **Type**: Either "user_message" or "assistant_message"
  - **Text**: The message content

## Notes

- Requires a valid Cursor Cloud Agent API key configured in the integration
- If the agent has been deleted, the conversation cannot be accessed
- Returns nil if the conversation has no messages`
}

func (c *GetLastMessage) Icon() string {
	return "message-square"
}

func (c *GetLastMessage) Color() string {
	return "#3B82F6"
}

func (c *GetLastMessage) ExampleOutput() map[string]any {
	return map[string]any{
		"agentId": "bc_abc123",
		"message": map[string]any{
			"id":   "msg_005",
			"type": "assistant_message",
			"text": "I've added a troubleshooting section to the README.",
		},
	}
}

func (c *GetLastMessage) OutputChannels(config any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetLastMessage) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "agentId",
			Label:       "Agent ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: `{{ $["cursor.launchAgent"].data.agentId }}`,
		},
	}
}

func (c *GetLastMessage) Setup(ctx core.SetupContext) error {
	spec := GetLastMessageSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if spec.AgentID == "" {
		return fmt.Errorf("agent ID is required")
	}

	return nil
}

func (c *GetLastMessage) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetLastMessage) Execute(ctx core.ExecutionContext) error {
	spec := GetLastMessageSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if spec.AgentID == "" {
		return fmt.Errorf("agent ID is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create cursor client: %w", err)
	}

	if client.LaunchAgentKey == "" {
		return fmt.Errorf("cloud agent API key is not configured in the integration")
	}

	ctx.Logger.Infof("Fetching conversation for agent %s", spec.AgentID)

	conversation, err := client.GetAgentConversation(spec.AgentID)
	if err != nil {
		return fmt.Errorf("failed to fetch conversation: %w", err)
	}

	output := GetLastMessageOutput{
		AgentID: spec.AgentID,
		Message: nil,
	}

	// Extract the last message from the messages array
	if conversation.Messages != nil && len(conversation.Messages) > 0 {
		lastMessage := conversation.Messages[len(conversation.Messages)-1]
		output.Message = &lastMessage
		ctx.Logger.Infof("Retrieved last message: %s (type: %s)", lastMessage.ID, lastMessage.Type)
	} else {
		ctx.Logger.Infof("No messages found in conversation")
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, GetLastMessagePayloadType, []any{output})
}

func (c *GetLastMessage) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetLastMessage) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *GetLastMessage) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (c *GetLastMessage) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetLastMessage) Cleanup(ctx core.SetupContext) error {
	return nil
}
