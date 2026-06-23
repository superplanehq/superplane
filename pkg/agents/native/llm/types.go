package llm

import "context"

type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

type BlockType string

const (
	BlockTypeText       BlockType = "text"
	BlockTypeToolUse    BlockType = "tool_use"
	BlockTypeToolResult BlockType = "tool_result"
)

type Message struct {
	Role   Role
	Blocks []Block
}

type Block struct {
	Type       BlockType
	Text       string
	ToolCall   *ToolCall
	ToolResult *ToolResult
}

type ToolDefinition struct {
	Name        string
	Description string
	InputSchema map[string]any
}

type ToolCall struct {
	ID    string
	Name  string
	Input string
}

type ToolResult struct {
	ToolCallID string
	Name       string
	Content    string
	IsError    bool
}

type StreamRequest struct {
	SessionID string
	Model     string
	Messages  []Message
	Tools     []ToolDefinition
}

type StreamEventType string

const (
	StreamEventTextDelta StreamEventType = "text_delta"
	StreamEventToolCall  StreamEventType = "tool_call"
)

type StreamEvent struct {
	Type     StreamEventType
	Text     string
	ToolCall *ToolCall
}

type Client interface {
	Stream(ctx context.Context, req StreamRequest, onEvent func(StreamEvent) error) error
}

func NewUserMessage(text string) Message {
	return Message{
		Role:   RoleUser,
		Blocks: []Block{{Type: BlockTypeText, Text: text}},
	}
}

func NewSystemMessage(text string) Message {
	return Message{
		Role:   RoleSystem,
		Blocks: []Block{{Type: BlockTypeText, Text: text}},
	}
}

func NewAssistantMessage(blocks []Block) Message {
	return Message{Role: RoleAssistant, Blocks: blocks}
}

func NewToolResultMessage(results []ToolResult) Message {
	blocks := make([]Block, 0, len(results))
	for _, result := range results {
		result := result
		blocks = append(blocks, Block{
			Type:       BlockTypeToolResult,
			ToolResult: &result,
		})
	}
	return Message{Role: RoleTool, Blocks: blocks}
}
