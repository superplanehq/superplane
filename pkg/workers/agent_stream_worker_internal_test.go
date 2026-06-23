package workers

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/agents"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestHandleProviderEvent_PublishesTurnUsageWhenSessionAlreadyReset(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)

	session := &models.AgentSession{
		OrganizationID:    r.Organization.ID,
		UserID:            r.User,
		CanvasID:          canvas.ID,
		Provider:          "test",
		ProviderSessionID: "upstream-session",
		Status:            models.AgentSessionStatusIdle,
	}
	require.NoError(t, database.Conn().Create(session).Error)

	published := 0
	originalPublisher := publishAgentRunFinished
	publishAgentRunFinished = func(gotSession *models.AgentSession, evt agents.ProviderEvent) error {
		published++
		assert.Equal(t, session.ID, gotSession.ID)
		assert.Equal(t, "claude-sonnet-4-5", evt.Model)
		require.NotNil(t, evt.Usage)
		assert.Equal(t, int64(42), evt.Usage.TotalTokens)
		return nil
	}
	t.Cleanup(func() {
		publishAgentRunFinished = originalPublisher
	})

	var streamErr error
	err := handleProviderEvent(
		session,
		agents.ProviderEvent{
			Type:  agents.ProviderEventTurnCompleted,
			Model: "claude-sonnet-4-5",
			Usage: &agents.TokenUsage{TotalTokens: 42},
		},
		func(messages.AgentSessionEventMessage) {},
		&streamErr,
		newCustomToolTurnState(),
	)

	assert.True(t, errors.Is(err, errSessionAlreadyReset))
	assert.Equal(t, 1, published)
	assert.NoError(t, streamErr)
}
