package agents

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
)

const (
	rewindMessageLimit  = 30
	rewindMaxChars      = 20_000
	rewindEntryMaxChars = 2_000
	rewindToolMaxChars  = 800
)

func (s *Service) messageWithRewind(session *models.AgentSession, content string) (string, bool, error) {
	if session.ContextReplayedAt != nil {
		return content, false, nil
	}

	rewind, err := buildConversationRewind(session.ID)
	if err != nil {
		return "", false, fmt.Errorf("build conversation rewind: %w", err)
	}
	if rewind == "" {
		return content, true, nil
	}

	return rewind + "\n\n[Current user request]\n" + content, true, nil
}

func buildConversationRewind(sessionID uuid.UUID) (string, error) {
	rows, err := models.ListAgentSessionMessagesPage(sessionID, nil, rewindMessageLimit)
	if err != nil {
		return "", err
	}

	entries := make([]string, 0, len(rows))
	remaining := rewindMaxChars
	for i := len(rows) - 1; i >= 0; i-- {
		entry := formatRewindEntry(rows[i])
		if entry == "" {
			continue
		}
		if len(entry) > remaining && len(entries) > 0 {
			break
		}
		if len(entry) > remaining {
			entry = truncateText(entry, remaining)
		}
		entries = append(entries, entry)
		remaining -= len(entry)
	}
	if len(entries) == 0 {
		return "", nil
	}

	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}

	return "[SuperPlane conversation rewind]\n" +
		"The provider session was recreated. Treat this as prior conversation context, not as a new request. Do not answer or summarize the rewind; answer only the current user request that follows.\n\n" +
		"Recent conversation:\n" +
		strings.Join(entries, "\n\n"), nil
}

func formatRewindEntry(message models.AgentSessionMessage) string {
	content := strings.TrimSpace(message.Content)
	switch message.Role {
	case models.AgentMessageRoleUser:
		return "User: " + userRewindText(content, len(message.Images))
	case models.AgentMessageRoleAssistant:
		return "Assistant: " + truncateText(content, rewindEntryMaxChars)
	case models.AgentMessageRoleSystem:
		return "System note: " + truncateText(strings.TrimPrefix(content, "@@system: "), rewindEntryMaxChars)
	case models.AgentMessageRoleTool:
		return formatToolRewindEntry(message, content)
	default:
		return ""
	}
}

func userRewindText(content string, imageCount int) string {
	text := truncateText(content, rewindEntryMaxChars)
	note := imageAttachmentNote(imageCount)
	switch {
	case note == "":
		return text
	case text == "":
		return note
	default:
		return text + " " + note
	}
}

func imageAttachmentNote(count int) string {
	switch {
	case count <= 0:
		return ""
	case count == 1:
		return "[1 image attachment shared earlier]"
	default:
		return fmt.Sprintf("[%d image attachments shared earlier]", count)
	}
}

func formatToolRewindEntry(message models.AgentSessionMessage, content string) string {
	name := strings.TrimSpace(message.ToolName)
	if name == "" {
		name = "tool"
	}
	status := strings.TrimSpace(message.ToolStatus)
	if status == "" {
		status = "finished"
	}
	if content == "" {
		return fmt.Sprintf("Tool %s %s.", name, status)
	}
	return fmt.Sprintf("Tool %s %s: %s", name, status, truncateText(content, rewindToolMaxChars))
}

func truncateText(text string, limit int) string {
	if limit <= 0 {
		return ""
	}
	text = strings.TrimSpace(text)
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	if limit <= 3 {
		return string(runes[:limit])
	}
	return strings.TrimSpace(string(runes[:limit-3])) + "..."
}
