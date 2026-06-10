package models

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
)

func TestAppendAgentSessionMessageInTransaction_PreservesToolNameOnNamelessResult(t *testing.T) {
	session := &AgentSession{
		ID:                uuid.New(),
		OrganizationID:    uuid.New(),
		UserID:            uuid.New(),
		CanvasID:          uuid.New(),
		Provider:          "anthropic",
		ProviderSessionID: "sesn_test",
		Status:            AgentSessionStatusStreaming,
	}
	require.NoError(t, CreateAgentSessionInTransaction(database.Conn(), session))
	t.Cleanup(func() {
		_ = database.Conn().Delete(&AgentSession{}, "id = ?", session.ID).Error
	})

	require.NoError(t, AppendAgentSessionMessage(&AgentSessionMessage{
		SessionID:       session.ID,
		ProviderEventID: "toolu_1",
		Role:            AgentMessageRoleTool,
		Content:         `{"file_path":"/tmp/spec.md"}`,
		ToolCallID:      "toolu_1",
		ToolName:        "read",
		ToolStatus:      AgentToolStatusStarted,
	}))

	require.NoError(t, AppendAgentSessionMessage(&AgentSessionMessage{
		SessionID:       session.ID,
		ProviderEventID: "toolu_1",
		Role:            AgentMessageRoleTool,
		ToolCallID:      "toolu_1",
		ToolStatus:      AgentToolStatusFinished,
	}))

	var stored AgentSessionMessage
	require.NoError(t, database.Conn().
		Where("session_id = ? AND provider_event_id = ?", session.ID, "toolu_1").
		First(&stored).Error)

	assert.Equal(t, "read", stored.ToolName)
	assert.Equal(t, AgentToolStatusFinished, stored.ToolStatus)
	assert.Equal(t, `{"file_path":"/tmp/spec.md"}`, stored.Content)
}

func TestAppendAgentSessionMessageInTransaction_DoesNotDowngradeFinishedToolOnReplay(t *testing.T) {
	session := &AgentSession{
		ID:                uuid.New(),
		OrganizationID:    uuid.New(),
		UserID:            uuid.New(),
		CanvasID:          uuid.New(),
		Provider:          "anthropic",
		ProviderSessionID: "sesn_test",
		Status:            AgentSessionStatusStreaming,
	}
	require.NoError(t, CreateAgentSessionInTransaction(database.Conn(), session))
	t.Cleanup(func() {
		_ = database.Conn().Delete(&AgentSession{}, "id = ?", session.ID).Error
	})

	require.NoError(t, AppendAgentSessionMessage(&AgentSessionMessage{
		SessionID:       session.ID,
		ProviderEventID: "toolu_1",
		Role:            AgentMessageRoleTool,
		Content:         `{"ok":true}`,
		ToolCallID:      "toolu_1",
		ToolName:        "superplane_app",
		ToolStatus:      AgentToolStatusFinished,
	}))

	require.NoError(t, AppendAgentSessionMessage(&AgentSessionMessage{
		SessionID:       session.ID,
		ProviderEventID: "toolu_1",
		Role:            AgentMessageRoleTool,
		Content:         `{"action":"read"}`,
		ToolCallID:      "toolu_1",
		ToolName:        "superplane_app",
		ToolStatus:      AgentToolStatusStarted,
	}))

	var stored AgentSessionMessage
	require.NoError(t, database.Conn().
		Where("session_id = ? AND provider_event_id = ?", session.ID, "toolu_1").
		First(&stored).Error)

	assert.Equal(t, "superplane_app", stored.ToolName)
	assert.Equal(t, AgentToolStatusFinished, stored.ToolStatus)
	assert.Equal(t, `{"ok":true}`, stored.Content)
}
