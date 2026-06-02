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
	name          string
	events        []agents.ProviderEvent
	err           error
	afterOpenSeen bool
}

func (p *scriptedProvider) Name() string { return p.name }
func (p *scriptedProvider) CreateSession(context.Context, agents.CreateSessionOptions) (*agents.CreateSessionResult, error) {
	return nil, errors.New("not used")
}
func (p *scriptedProvider) SendMessage(context.Context, string, string, agents.SendMessageOptions) error {
	return errors.New("not used")
}
func (p *scriptedProvider) InterruptSession(context.Context, string) error {
	return errors.New("not used")
}
func (p *scriptedProvider) DefineOutcome(context.Context, string, agents.DefineOutcomeOptions) error {
	return errors.New("not used")
}
func (p *scriptedProvider) StreamEvents(ctx context.Context, _ string, afterOpen func(context.Context) error, cb func(agents.ProviderEvent) error) error {
	if afterOpen != nil {
		if err := afterOpen(ctx); err != nil {
			return err
		}
		p.afterOpenSeen = true
	}
	for _, e := range p.events {
		if err := cb(e); err != nil {
			return err
		}
	}
	return p.err
}
func (p *scriptedProvider) ArchiveSession(context.Context, string) error { return nil }

type recordingTurnSender struct {
	sessionID uuid.UUID
	messageID uuid.UUID
	mode      string
	err       error
}

func (s *recordingTurnSender) SendPersistedMessage(_ context.Context, session *models.AgentSession, messageID uuid.UUID, mode string) error {
	s.sessionID = session.ID
	s.messageID = messageID
	s.mode = mode
	return s.err
}

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

func TestAgentStreamWorker_UpdatesStreamedAssistantMessage(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)

	session := mustCreateSession(t, r, canvas.ID)

	provider := &scriptedProvider{
		name: testProvider,
		events: []agents.ProviderEvent{
			{ProviderEventID: "msg-1", Type: agents.ProviderEventAssistantMessage, Text: "Hello"},
			{ProviderEventID: "msg-1", Type: agents.ProviderEventAssistantMessage, Text: "Hello, world"},
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

	stored, err := models.ListAgentSessionMessagesPage(session.ID, nil, 100)
	require.NoError(t, err)
	require.Len(t, stored, 1, "streamed assistant text must update one row so the UI can repaint the current response")
	assert.Equal(t, models.AgentMessageRoleAssistant, stored[0].Role)
	assert.Equal(t, "Hello, world", stored[0].Content)

	refreshed, err := models.FindAgentSession(session.ID)
	require.NoError(t, err)
	assert.Equal(t, models.AgentSessionStatusIdle, refreshed.Status)
}

func TestAgentStreamWorker_SendsQueuedUserMessageAfterStreamOpens(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	session := mustCreateSession(t, r, canvas.ID)

	message := &models.AgentSessionMessage{
		SessionID: session.ID,
		Role:      models.AgentMessageRoleUser,
		Content:   "Build this",
	}
	require.NoError(t, models.AppendAgentSessionMessage(message))

	provider := &scriptedProvider{
		name: testProvider,
		events: []agents.ProviderEvent{
			{ProviderEventID: "msg-1", Type: agents.ProviderEventAssistantMessage, Text: "Done"},
			{Type: agents.ProviderEventTurnCompleted},
		},
	}
	sender := &recordingTurnSender{}

	w := workers.NewAgentStreamWorker(provider, "amqp://ignored", sender)
	body, _ := json.Marshal(messages.AgentStreamRequest{
		SessionID:      session.ID.String(),
		OrganizationID: r.Organization.ID.String(),
		UserID:         r.User.String(),
		UserMessageID:  message.ID.String(),
		Mode:           string(agents.ModeBuilder),
	})

	require.NoError(t, w.Handle(context.Background(), body))
	assert.True(t, provider.afterOpenSeen)
	assert.Equal(t, session.ID, sender.sessionID)
	assert.Equal(t, message.ID, sender.messageID)
	assert.Equal(t, string(agents.ModeBuilder), sender.mode)
}

func TestAgentStreamWorker_MarksFailedWhenQueuedUserMessageCannotSend(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	session := mustCreateSession(t, r, canvas.ID)

	message := &models.AgentSessionMessage{
		SessionID: session.ID,
		Role:      models.AgentMessageRoleUser,
		Content:   "Build this",
	}
	require.NoError(t, models.AppendAgentSessionMessage(message))

	provider := &scriptedProvider{name: testProvider}
	sender := &recordingTurnSender{err: errors.New("provider send failed")}

	w := workers.NewAgentStreamWorker(provider, "amqp://ignored", sender)
	body, _ := json.Marshal(messages.AgentStreamRequest{
		SessionID:     session.ID.String(),
		UserMessageID: message.ID.String(),
	})

	require.NoError(t, w.Handle(context.Background(), body))

	refreshed, err := models.FindAgentSession(session.ID)
	require.NoError(t, err)
	assert.Equal(t, models.AgentSessionStatusFailed, refreshed.Status)
}

func TestAgentStreamWorker_PersistsDistinctAssistantMessages(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	session := mustCreateSession(t, r, canvas.ID)

	provider := &scriptedProvider{
		name: testProvider,
		events: []agents.ProviderEvent{
			{ProviderEventID: "msg-1", Type: agents.ProviderEventAssistantMessage, Text: "First"},
			{ProviderEventID: "msg-2", Type: agents.ProviderEventAssistantMessage, Text: "Second"},
			{Type: agents.ProviderEventTurnCompleted},
		},
	}

	w := workers.NewAgentStreamWorker(provider, "amqp://ignored")
	body, _ := json.Marshal(messages.AgentStreamRequest{SessionID: session.ID.String()})
	require.NoError(t, w.Handle(context.Background(), body))

	stored, err := models.ListAgentSessionMessagesPage(session.ID, nil, 100)
	require.NoError(t, err)
	require.Len(t, stored, 2)
	assert.Equal(t, "First", stored[0].Content)
	assert.Equal(t, "Second", stored[1].Content)
}

func TestAgentStreamWorker_PersistsToolEvents(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	session := mustCreateSession(t, r, canvas.ID)

	provider := &scriptedProvider{
		name: testProvider,
		events: []agents.ProviderEvent{
			{ProviderEventID: "tool-1", Type: agents.ProviderEventToolUseStarted, ToolName: "search", ToolCallID: "call-1", ToolInput: "rg --files"},
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

	stored, err := models.ListAgentSessionMessagesPage(session.ID, nil, 100)
	require.NoError(t, err)
	require.Len(t, stored, 1, "matching tool start+finish must collapse into a single row (provider_event_id is unique)")
	assert.Equal(t, models.AgentMessageRoleTool, stored[0].Role)
	assert.Equal(t, "search", stored[0].ToolName)
	assert.Equal(t, models.AgentToolStatusFinished, stored[0].ToolStatus)
	assert.Equal(t, "rg --files", stored[0].Content,
		"tool_finished must not wipe the input captured by tool_started — the UI's expandable command relies on it")
}

func TestAgentStreamWorker_ParallelToolsTrackedIndependently(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	session := mustCreateSession(t, r, canvas.ID)

	// Agents commonly run tools in parallel — interleaved use/result events
	// keyed by tool_use_id must each upsert into their own row, leaving
	// any tool whose result hasn't arrived yet still in "started" state
	// until the turn-end sweep catches it.
	provider := &scriptedProvider{
		name: testProvider,
		events: []agents.ProviderEvent{
			{ProviderEventID: "toolu_1", Type: agents.ProviderEventToolUseStarted, ToolName: "bash"},
			{ProviderEventID: "toolu_2", Type: agents.ProviderEventToolUseStarted, ToolName: "bash"},
			{ProviderEventID: "toolu_3", Type: agents.ProviderEventToolUseStarted, ToolName: "bash"},
			{ProviderEventID: "toolu_2", Type: agents.ProviderEventToolUseFinished, ToolName: "bash"},
			{ProviderEventID: "toolu_1", Type: agents.ProviderEventToolUseFinished, ToolName: "bash"},
			{Type: agents.ProviderEventTurnCompleted},
		},
	}

	w := workers.NewAgentStreamWorker(provider, "amqp://ignored")
	body, _ := json.Marshal(messages.AgentStreamRequest{SessionID: session.ID.String()})
	require.NoError(t, w.Handle(context.Background(), body))

	stored, err := models.ListAgentSessionMessagesPage(session.ID, nil, 100)
	require.NoError(t, err)
	require.Len(t, stored, 3, "one row per tool_use_id, not duplicated per event")

	// All three end up finished — toolu_1/2 via their explicit
	// tool_result, toolu_3 via the turn-end sweep.
	statuses := map[string]string{}
	for _, m := range stored {
		statuses[m.ProviderEventID] = m.ToolStatus
	}
	assert.Equal(t, models.AgentToolStatusFinished, statuses["toolu_1"])
	assert.Equal(t, models.AgentToolStatusFinished, statuses["toolu_2"])
	assert.Equal(t, models.AgentToolStatusFinished, statuses["toolu_3"], "tools without a result close on turn end")
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

	stored, err := models.ListAgentSessionMessagesPage(session.ID, nil, 100)
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

func TestAgentStreamWorker_CleanupFailsStuckStreamingSessions(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	staleCanvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	freshCanvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)

	stale := mustCreateSession(t, r, staleCanvas.ID)
	require.NoError(t, database.Conn().Model(&models.AgentSession{}).
		Where("id = ?", stale.ID).
		Update("updated_at", time.Now().Add(-2*time.Hour)).Error)
	fresh := mustCreateSession(t, r, freshCanvas.ID)

	closed, err := models.FailStuckStreamingSessions(time.Now().Add(-30 * time.Minute))
	require.NoError(t, err)
	require.Len(t, closed, 1)
	assert.Equal(t, stale.ID, closed[0].ID)

	staleAfter, err := models.FindAgentSession(stale.ID)
	require.NoError(t, err)
	assert.Equal(t, models.AgentSessionStatusFailed, staleAfter.Status)

	freshAfter, err := models.FindAgentSession(fresh.ID)
	require.NoError(t, err)
	assert.Equal(t, models.AgentSessionStatusStreaming, freshAfter.Status)
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
