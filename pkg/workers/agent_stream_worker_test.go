package workers_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/agents"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/workers"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

const testProvider = "test"

type scriptedProvider struct {
	name   string
	events []agents.ProviderEvent
	err    error
}

func (p *scriptedProvider) Name() string { return p.name }
func (p *scriptedProvider) CreateSession(context.Context, agents.CreateSessionOptions) (*agents.CreateSessionResult, error) {
	return nil, errors.New("not used")
}
func (p *scriptedProvider) SendMessage(context.Context, string, string, agents.SendMessageOptions) error {
	return errors.New("not used")
}
func (p *scriptedProvider) StreamEvents(ctx context.Context, _ string, cb func(agents.ProviderEvent) error) error {
	for _, e := range p.events {
		if err := cb(e); err != nil {
			return err
		}
	}
	return p.err
}
func (p *scriptedProvider) ArchiveSession(context.Context, string) error { return nil }

func mustCreateSession(t *testing.T, r *support.ResourceRegistry, canvasID uuid.UUID) *models.AgentSession {
	t.Helper()
	session := &models.AgentSession{
		OrganizationID:    r.Organization.ID,
		UserID:            r.User,
		CanvasID:          canvasID,
		Provider:          testProvider,
		ProviderSessionID: "upstream-" + uuid.NewString(),
		Status:            models.AgentSessionStatusStreaming,
	}
	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		return models.CreateAgentSessionInTransaction(tx, session)
	}))
	return session
}

func TestAgentStreamWorker_PersistsAssistantTurn(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)

	session := mustCreateSession(t, r, canvas.ID)

	provider := &scriptedProvider{
		name: testProvider,
		events: []agents.ProviderEvent{
			{Type: agents.ProviderEventAssistantMessage, Text: "Hello"},
			{Type: agents.ProviderEventAssistantMessage, Text: ", world"},
			{Type: agents.ProviderEventTurnCompleted},
		},
	}

	w := workers.NewAgentStreamWorker(provider, "amqp://ignored")
	body, _ := json.Marshal(messages.AgentStreamRequest{
		SessionID:      session.ID.String(),
		OrganizationID: r.Organization.ID.String(),
		UserID:         r.User.String(),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, w.Handle(ctx, body))

	stored, err := models.ListAgentSessionMessages(session.ID)
	require.NoError(t, err)
	require.Len(t, stored, 1, "the streamed text deltas must be coalesced into a single assistant message")
	assert.Equal(t, models.AgentMessageRoleAssistant, stored[0].Role)
	assert.Equal(t, "Hello, world", stored[0].Content)

	refreshed, err := models.FindAgentSession(session.ID)
	require.NoError(t, err)
	assert.Equal(t, models.AgentSessionStatusIdle, refreshed.Status)
}

func TestAgentStreamWorker_PersistsToolEvents(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	session := mustCreateSession(t, r, canvas.ID)

	provider := &scriptedProvider{
		name: testProvider,
		events: []agents.ProviderEvent{
			{ProviderEventID: "tool-1", Type: agents.ProviderEventToolUseStarted, ToolName: "search", ToolCallID: "call-1"},
			{ProviderEventID: "tool-1", Type: agents.ProviderEventToolUseFinished, ToolName: "search", ToolCallID: "call-1"},
			{Type: agents.ProviderEventTurnCompleted},
		},
	}

	w := workers.NewAgentStreamWorker(provider, "amqp://ignored")
	body, _ := json.Marshal(messages.AgentStreamRequest{
		SessionID:      session.ID.String(),
		OrganizationID: r.Organization.ID.String(),
		UserID:         r.User.String(),
	})
	require.NoError(t, w.Handle(context.Background(), body))

	stored, err := models.ListAgentSessionMessages(session.ID)
	require.NoError(t, err)
	require.Len(t, stored, 1, "matching tool start+finish must collapse into a single row (provider_event_id is unique)")
	assert.Equal(t, models.AgentMessageRoleTool, stored[0].Role)
	assert.Equal(t, "search", stored[0].ToolName)
	assert.Equal(t, models.AgentToolStatusFinished, stored[0].ToolStatus)
}

func TestAgentStreamWorker_ForceClosesOpenToolsOnTurnEnd(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	session := mustCreateSession(t, r, canvas.ID)

	// Provider emits tool_use but never tool_result — then turn_completed.
	// Mirrors the real-world case where Anthropic emits idle before a
	// tool_result for one of the built-in toolset tools.
	provider := &scriptedProvider{
		name: testProvider,
		events: []agents.ProviderEvent{
			{ProviderEventID: "tool-stuck", Type: agents.ProviderEventToolUseStarted, ToolName: "bash", ToolCallID: "call-stuck"},
			{Type: agents.ProviderEventTurnCompleted},
		},
	}

	w := workers.NewAgentStreamWorker(provider, "amqp://ignored")
	body, _ := json.Marshal(messages.AgentStreamRequest{SessionID: session.ID.String()})
	require.NoError(t, w.Handle(context.Background(), body))

	stored, err := models.ListAgentSessionMessages(session.ID)
	require.NoError(t, err)
	require.Len(t, stored, 1)
	assert.Equal(t, models.AgentToolStatusFinished, stored[0].ToolStatus, "open tool must be force-closed when the turn ends")
}

func TestAgentStreamWorker_MarksFailedOnSessionError(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	session := mustCreateSession(t, r, canvas.ID)

	provider := &scriptedProvider{
		name: testProvider,
		events: []agents.ProviderEvent{
			{Type: agents.ProviderEventSessionFailed, ErrorMessage: "explosion"},
		},
	}

	w := workers.NewAgentStreamWorker(provider, "amqp://ignored")
	body, _ := json.Marshal(messages.AgentStreamRequest{
		SessionID: session.ID.String(),
	})
	require.NoError(t, w.Handle(context.Background(), body))

	refreshed, err := models.FindAgentSession(session.ID)
	require.NoError(t, err)
	assert.Equal(t, models.AgentSessionStatusFailed, refreshed.Status)
}

func TestAgentStreamWorker_SkipsUnknownProvider(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	session := mustCreateSession(t, r, canvas.ID)

	provider := &scriptedProvider{name: "different-provider"}

	w := workers.NewAgentStreamWorker(provider, "amqp://ignored")
	body, _ := json.Marshal(messages.AgentStreamRequest{
		SessionID: session.ID.String(),
	})
	require.NoError(t, w.Handle(context.Background(), body))

	// Status untouched — the worker should leave this for another replica.
	refreshed, err := models.FindAgentSession(session.ID)
	require.NoError(t, err)
	assert.Equal(t, models.AgentSessionStatusStreaming, refreshed.Status)
}

func TestAgentStreamWorker_DropsUnknownSession(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	provider := &scriptedProvider{name: testProvider}
	w := workers.NewAgentStreamWorker(provider, "amqp://ignored")

	body, _ := json.Marshal(messages.AgentStreamRequest{
		SessionID:      uuid.NewString(),
		OrganizationID: r.Organization.ID.String(),
	})
	require.NoError(t, w.Handle(context.Background(), body), "unknown sessions must not crash the worker")
}
