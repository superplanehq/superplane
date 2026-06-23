package native

import (
	"strings"

	"github.com/superplanehq/superplane/pkg/agents/native/llm"
)

func boundedHistory(history []llm.Message, maxChars int) ([]llm.Message, error) {
	if maxChars <= 0 {
		return nil, errContextBudgetEmpty
	}
	if messageChars(history) <= maxChars {
		return cloneMessages(history), nil
	}

	bounded := []llm.Message{}
	system, remaining, hasSystem := systemMessageWithinBudget(history, maxChars)
	if hasSystem {
		bounded = append(bounded, system)
		maxChars -= messageChars([]llm.Message{system})
	}
	if maxChars <= 0 {
		return nil, errContextBudgetEmpty
	}

	recent := recentMessagesWithinBudget(remaining, maxChars)
	if len(recent) == len(remaining) {
		bounded = append(bounded, recent...)
		if len(bounded) == 0 {
			return nil, errContextBudgetEmpty
		}
		return bounded, nil
	}

	summaryBudget := min(maxChars/3, 12000)
	if summaryBudget > 200 {
		recentBudget := maxChars - summaryBudget
		recent = recentMessagesWithinBudget(remaining, recentBudget)
		omittedCount := len(remaining) - len(recent)
		if omittedCount > 0 {
			summary := compactedSummaryMessage(remaining[:omittedCount], summaryBudget)
			summarySize := messageChars([]llm.Message{summary})
			if summarySize > 0 && summarySize < maxChars {
				bounded = append(bounded, summary)
				maxChars -= summarySize
				recent = recentMessagesWithinBudget(remaining, maxChars)
			}
		}
	}

	bounded = append(bounded, recent...)
	if len(bounded) == 0 {
		return nil, errContextBudgetEmpty
	}
	return bounded, nil
}

func systemMessageWithinBudget(history []llm.Message, maxChars int) (llm.Message, []llm.Message, bool) {
	if len(history) == 0 || history[0].Role != llm.RoleSystem {
		return llm.Message{}, history, false
	}

	system := cloneMessages(history[:1])[0]
	if messageChars([]llm.Message{system}) <= maxChars {
		return system, history[1:], true
	}

	system = truncateMessage(system, maxChars)
	return system, history[1:], true
}

func recentMessagesWithinBudget(history []llm.Message, maxChars int) []llm.Message {
	selected := []llm.Message{}
	remaining := maxChars
	for i := len(history) - 1; i >= 0; i-- {
		message := cloneMessages(history[i : i+1])[0]
		size := messageChars([]llm.Message{message})
		if size > remaining {
			if len(selected) > 0 {
				break
			}
			message = truncateMessage(message, remaining)
			size = messageChars([]llm.Message{message})
		}
		if size <= 0 || size > remaining {
			break
		}
		selected = append(selected, message)
		remaining -= size
	}

	for i, j := 0, len(selected)-1; i < j; i, j = i+1, j-1 {
		selected[i], selected[j] = selected[j], selected[i]
	}
	return selected
}

func compactedSummaryMessage(history []llm.Message, maxChars int) llm.Message {
	text := compactedSummary(history, maxChars)
	return llm.NewSystemMessage(text)
}

func compactedSummary(history []llm.Message, maxChars int) string {
	if len(history) == 0 || maxChars <= 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("Compacted earlier conversation. Use this as background only; recent messages below are authoritative.\n")
	builder.WriteString("Preserved older highlights:\n")
	for _, message := range history {
		entry := compactedSummaryEntry(message)
		if entry == "" {
			continue
		}
		line := "- " + entry + "\n"
		if builder.Len()+len(line) > maxChars {
			break
		}
		builder.WriteString(line)
	}

	return truncateString(builder.String(), maxChars)
}

func compactedSummaryEntry(message llm.Message) string {
	parts := []string{}
	for _, block := range message.Blocks {
		switch block.Type {
		case llm.BlockTypeText:
			if block.Text != "" {
				parts = append(parts, truncateString(block.Text, 600))
			}
		case llm.BlockTypeToolUse:
			if block.ToolCall != nil {
				parts = append(parts, "tool request "+block.ToolCall.Name+" "+truncateString(block.ToolCall.Input, 300))
			}
		case llm.BlockTypeToolResult:
			if block.ToolResult != nil {
				status := "ok"
				if block.ToolResult.IsError {
					status = "error"
				}
				parts = append(parts, "tool result "+block.ToolResult.Name+" ("+status+") "+truncateString(block.ToolResult.Content, 300))
			}
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return string(message.Role) + ": " + strings.Join(parts, " | ")
}

func truncateMessage(message llm.Message, maxChars int) llm.Message {
	if maxChars <= 0 {
		message.Blocks = nil
		return message
	}

	remaining := maxChars
	for i := range message.Blocks {
		blockSize := blockChars(message.Blocks[i])
		if blockSize <= remaining {
			remaining -= blockSize
			continue
		}

		message.Blocks[i] = truncateBlock(message.Blocks[i], remaining)
		message.Blocks = message.Blocks[:i+1]
		return message
	}
	return message
}

func truncateBlock(block llm.Block, maxChars int) llm.Block {
	switch block.Type {
	case llm.BlockTypeText:
		block.Text = truncateString(block.Text, maxChars)
	case llm.BlockTypeToolUse:
		if block.ToolCall != nil {
			block.ToolCall.Input = truncateString(block.ToolCall.Input, maxChars)
		}
	case llm.BlockTypeToolResult:
		if block.ToolResult != nil {
			block.ToolResult.Content = truncateString(block.ToolResult.Content, maxChars)
		}
	}
	return block
}

func messageChars(messages []llm.Message) int {
	total := 0
	for _, message := range messages {
		for _, block := range message.Blocks {
			total += blockChars(block)
		}
	}
	return total
}

func blockChars(block llm.Block) int {
	switch block.Type {
	case llm.BlockTypeText:
		return len(block.Text)
	case llm.BlockTypeToolUse:
		if block.ToolCall == nil {
			return 0
		}
		return len(block.ToolCall.Name) + len(block.ToolCall.Input)
	case llm.BlockTypeToolResult:
		if block.ToolResult == nil {
			return 0
		}
		return len(block.ToolResult.Name) + len(block.ToolResult.Content)
	default:
		return 0
	}
}

func truncateString(value string, maxChars int) string {
	if maxChars <= 0 {
		return ""
	}
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) <= maxChars {
		return value
	}
	if maxChars <= 3 {
		return string(runes[:maxChars])
	}
	return strings.TrimSpace(string(runes[:maxChars-3])) + "..."
}
