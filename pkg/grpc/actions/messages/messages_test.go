package messages

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	usagepb "github.com/superplanehq/superplane/pkg/protos/usage"
)

func TestConstructorsMapFields(t *testing.T) {
	canvasID := uuid.NewString()
	orgID := uuid.NewString()

	created := NewCanvasCreatedMessage(canvasID, orgID)
	require.Equal(t, canvasID, created.message.Id)
	require.Equal(t, canvasID, created.message.CanvasId)
	require.Equal(t, orgID, created.message.OrganizationId)
	require.NotNil(t, created.message.Timestamp)

	staging := NewCanvasStagingMessage(canvasID, "user-1")
	require.Equal(t, canvasID, staging.message.CanvasId)
	require.Equal(t, "user-1", staging.message.UserId)

	runID := uuid.New()
	eventID := uuid.New()
	terminal := NewCanvasEventTerminalMessage(uuid.MustParse(canvasID), runID, eventID)
	require.Equal(t, eventID.String(), terminal.message.EventId)
	require.Equal(t, canvasID, terminal.message.CanvasId)
	require.Equal(t, runID.String(), terminal.message.RunId)

	org := NewOrganizationCreatedMessage(orgID)
	require.Equal(t, orgID, org.message.OrganizationId)

	limits := &usagepb.OrganizationLimits{MaxCanvases: 5}
	plan := NewOrganizationPlanChangedMessage(orgID, "pro", limits)
	require.Equal(t, orgID, plan.message.OrganizationId)
	require.Equal(t, "pro", plan.message.PlanName)
	require.Equal(t, limits, plan.message.Limits)

	agent := NewAgentRunFinishedMessage(orgID, "chat-1", "gpt", "idem-1", "session-1", 1, 2, 3, 4, 5)
	require.Equal(t, orgID, agent.message.OrganizationId)
	require.Equal(t, "chat-1", agent.message.ChatId)
	require.Equal(t, "gpt", agent.message.Model)
	require.Equal(t, "idem-1", agent.message.IdempotencyKey)
	require.Equal(t, "session-1", agent.message.SessionId)
	require.Equal(t, int64(3), agent.message.TotalTokens)
}

func TestPublishWithoutRabbitMQURLErrors(t *testing.T) {
	t.Setenv("RABBITMQ_URL", "")
	err := Publish(CanvasExchange, CanvasCreatedRoutingKey, []byte("body"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "RABBITMQ_URL not set")
}

func TestAgentSessionEventWireShape(t *testing.T) {
	now := time.Now().UTC()
	payload := AgentSessionEventMessage{
		SessionID: "session-1",
		Event:     "message",
		MessageID: "msg-1",
		Message: &AgentMessage{
			ID:        "msg-1",
			Role:      "assistant",
			Content:   "hello",
			CreatedAt: &now,
		},
	}

	body, err := json.Marshal(payload)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(body, &decoded))
	require.Equal(t, "session-1", decoded["sessionId"])
	require.Equal(t, "msg-1", decoded["messageId"])

	message, ok := decoded["message"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "assistant", message["role"])
	require.Contains(t, message, "createdAt")

	createdAt, err := time.Parse(time.RFC3339, message["createdAt"].(string))
	require.NoError(t, err)
	require.WithinDuration(t, now, createdAt, time.Second)
}

func TestAgentStreamRequestWireShape(t *testing.T) {
	body, err := json.Marshal(AgentStreamRequest{
		SessionID:      "session-1",
		OrganizationID: "org-1",
		UserID:         "user-1",
	})
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(body, &decoded))
	require.Equal(t, "session-1", decoded["session_id"])
	require.Equal(t, "org-1", decoded["organization_id"])
	require.NotContains(t, decoded, "lock_retry_count")
}
