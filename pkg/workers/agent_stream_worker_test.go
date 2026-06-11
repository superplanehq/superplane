package workers_test

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
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

const (
	testProvider       = "test"
	testCustomToolName = "test_custom_tool"
)

type scriptedProvider struct {
	name         string
	events       []agents.ProviderEvent
	eventBatches [][]agents.ProviderEvent
	streamCalls  int
	sentResults  []agents.CustomToolResult
	err          error
	sendErr      error
	streamOnce   sync.Once
	streamReady  chan struct{}
	release      chan struct{}
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
func (p *scriptedProvider) StreamEvents(ctx context.Context, _ string, cb func(agents.ProviderEvent) error) error {
	if p.streamReady != nil {
		p.streamOnce.Do(func() { close(p.streamReady) })
	}
	if p.release != nil {
		select {
		case <-p.release:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	events := p.events
	if len(p.eventBatches) > 0 {
		index := p.streamCalls
		if index >= len(p.eventBatches) {
			index = len(p.eventBatches) - 1
		}
		events = p.eventBatches[index]
	}
	p.streamCalls++
	for _, e := range events {
		if err := cb(e); err != nil {
			return err
		}
	}
	return p.err
}
func (p *scriptedProvider) SendCustomToolResults(_ context.Context, _ string, results []agents.CustomToolResult) error {
	if p.sendErr != nil {
		return p.sendErr
	}
	p.sentResults = append(p.sentResults, results...)
	return nil
}
func (p *scriptedProvider) ArchiveSession(context.Context, string) error { return nil }

type fakeCustomToolExecutor struct {
	seen []agents.CustomToolUse
}

func (e *fakeCustomToolExecutor) ExecuteCustomTool(_ context.Context, session agents.AgentSessionContext, toolUse agents.CustomToolUse) agents.CustomToolResult {
	e.seen = append(e.seen, toolUse)
	return agents.CustomToolResult{
		CustomToolUseID: toolUse.ID,
		Content:         `{"ok":true,"canvas_id":"` + session.CanvasID + `"}`,
	}
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

func TestAgentStreamWorker_ContinuesWhenSessionChangesDuringStream(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	session := mustCreateSession(t, r, canvas.ID)

	provider := &scriptedProvider{
		name:        testProvider,
		streamReady: make(chan struct{}),
		release:     make(chan struct{}),
		eventBatches: [][]agents.ProviderEvent{
			{
				{Type: agents.ProviderEventTurnCompleted},
			},
			{
				{ProviderEventID: "msg-follow-up", Type: agents.ProviderEventAssistantMessage, Text: "second turn"},
				{Type: agents.ProviderEventTurnCompleted},
			},
		},
	}

	w := workers.NewAgentStreamWorker(provider, "amqp://ignored")
	body, _ := json.Marshal(messages.AgentStreamRequest{SessionID: session.ID.String()})

	firstDone := make(chan error, 1)
	go func() {
		firstDone <- w.Handle(context.Background(), body)
	}()
	<-provider.streamReady

	require.NoError(t, models.UpdateAgentSessionStatus(session.ID, models.AgentSessionStatusStreaming))

	close(provider.release)
	require.NoError(t, <-firstDone)
	assert.Equal(t, 2, provider.streamCalls)

	stored, err := models.ListAgentSessionMessagesPage(session.ID, nil, 100)
	require.NoError(t, err)
	require.Len(t, stored, 1)
	assert.Equal(t, "second turn", stored[0].Content)
}

func TestAgentStreamWorker_PersistsAssistantTurn(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)

	session := mustCreateSession(t, r, canvas.ID)

	provider := &scriptedProvider{
		name: testProvider,
		events: []agents.ProviderEvent{
			{ProviderEventID: "msg-1", Type: agents.ProviderEventAssistantMessage, Text: "Hello"},
			{ProviderEventID: "msg-2", Type: agents.ProviderEventAssistantMessage, Text: ", world"},
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
	require.Len(t, stored, 2, "each assistant text block must persist as its own row so chronology lines up with interleaved tool rows")
	assert.Equal(t, models.AgentMessageRoleAssistant, stored[0].Role)
	assert.Equal(t, "Hello", stored[0].Content)
	assert.Equal(t, models.AgentMessageRoleAssistant, stored[1].Role)
	assert.Equal(t, ", world", stored[1].Content)

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

func TestAgentStreamWorker_ExecutesCustomToolsAndResumesStream(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	session := mustCreateSession(t, r, canvas.ID)

	provider := &scriptedProvider{
		name: testProvider,
		eventBatches: [][]agents.ProviderEvent{
			{
				{
					ProviderEventID: "custom-1",
					Type:            agents.ProviderEventCustomToolUseStarted,
					ToolName:        testCustomToolName,
					ToolCallID:      "custom-1",
					ToolInput:       `{"action":"read"}`,
					CustomToolUse:   &agents.CustomToolUse{ID: "custom-1", Name: testCustomToolName, Input: `{"action":"read"}`},
				},
				{Type: agents.ProviderEventCustomToolResultsRequired, CustomToolEventIDs: []string{"custom-1"}},
			},
			{
				{ProviderEventID: "msg-1", Type: agents.ProviderEventAssistantMessage, Text: "Done"},
				{Type: agents.ProviderEventTurnCompleted},
			},
		},
	}
	executor := &fakeCustomToolExecutor{}

	w := workers.NewAgentStreamWorker(provider, "amqp://ignored", executor)
	body, _ := json.Marshal(messages.AgentStreamRequest{SessionID: session.ID.String()})
	require.NoError(t, w.Handle(context.Background(), body))

	require.Equal(t, 2, provider.streamCalls)
	require.Len(t, executor.seen, 1)
	require.Len(t, provider.sentResults, 1)
	assert.Equal(t, "custom-1", provider.sentResults[0].CustomToolUseID)

	stored, err := models.ListAgentSessionMessagesPage(session.ID, nil, 100)
	require.NoError(t, err)
	require.Len(t, stored, 2)
	assert.Equal(t, models.AgentMessageRoleTool, stored[0].Role)
	assert.Equal(t, models.AgentToolStatusFinished, stored[0].ToolStatus)
	assert.Contains(t, stored[0].Content, `"ok":true`)
	assert.Equal(t, models.AgentMessageRoleAssistant, stored[1].Role)
	assert.Equal(t, "Done", stored[1].Content)
}

func TestAgentStreamWorker_SendsErrorResultWhenCustomToolExecutorMissing(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	session := mustCreateSession(t, r, canvas.ID)

	provider := &scriptedProvider{
		name: testProvider,
		eventBatches: [][]agents.ProviderEvent{
			{
				{
					ProviderEventID: "custom-1",
					Type:            agents.ProviderEventCustomToolUseStarted,
					ToolName:        testCustomToolName,
					ToolCallID:      "custom-1",
					ToolInput:       `{"action":"read"}`,
					CustomToolUse:   &agents.CustomToolUse{ID: "custom-1", Name: testCustomToolName, Input: `{"action":"read"}`},
				},
				{Type: agents.ProviderEventCustomToolResultsRequired, CustomToolEventIDs: []string{"custom-1"}},
			},
			{
				{ProviderEventID: "msg-1", Type: agents.ProviderEventAssistantMessage, Text: "Recovered"},
				{Type: agents.ProviderEventTurnCompleted},
			},
		},
	}

	w := workers.NewAgentStreamWorker(provider, "amqp://ignored")
	body, _ := json.Marshal(messages.AgentStreamRequest{SessionID: session.ID.String()})
	require.NoError(t, w.Handle(context.Background(), body))

	require.Len(t, provider.sentResults, 1)
	assert.Equal(t, "custom-1", provider.sentResults[0].CustomToolUseID)
	assert.True(t, provider.sentResults[0].IsError)
	assert.Contains(t, provider.sentResults[0].Content, "custom tool executor is not configured")

	stored, err := models.ListAgentSessionMessagesPage(session.ID, nil, 100)
	require.NoError(t, err)
	require.Len(t, stored, 2)
	assert.Equal(t, models.AgentToolStatusFailed, stored[0].ToolStatus)
	assert.Equal(t, models.AgentMessageRoleAssistant, stored[1].Role)
	assert.Equal(t, "Recovered", stored[1].Content)

	refreshed, err := models.FindAgentSession(session.ID)
	require.NoError(t, err)
	assert.Equal(t, models.AgentSessionStatusIdle, refreshed.Status)
}

func TestAgentStreamWorker_DoesNotPersistFinishedCustomToolWhenResultSendFails(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	session := mustCreateSession(t, r, canvas.ID)

	provider := &scriptedProvider{
		name:    testProvider,
		sendErr: errors.New("provider unavailable"),
		events: []agents.ProviderEvent{
			{
				ProviderEventID: "custom-1",
				Type:            agents.ProviderEventCustomToolUseStarted,
				ToolName:        testCustomToolName,
				ToolCallID:      "custom-1",
				ToolInput:       `{"action":"read"}`,
				CustomToolUse:   &agents.CustomToolUse{ID: "custom-1", Name: testCustomToolName, Input: `{"action":"read"}`},
			},
			{Type: agents.ProviderEventCustomToolResultsRequired, CustomToolEventIDs: []string{"custom-1"}},
		},
	}

	w := workers.NewAgentStreamWorker(provider, "amqp://ignored", &fakeCustomToolExecutor{})
	body, _ := json.Marshal(messages.AgentStreamRequest{SessionID: session.ID.String()})
	require.NoError(t, w.Handle(context.Background(), body))

	stored, err := models.ListAgentSessionMessagesPage(session.ID, nil, 100)
	require.NoError(t, err)
	require.Len(t, stored, 1)
	assert.Equal(t, models.AgentToolStatusFinished, stored[0].ToolStatus)
	assert.Equal(t, `{"action":"read"}`, stored[0].Content)
	assert.NotContains(t, stored[0].Content, `"ok":true`)
	assert.Empty(t, provider.sentResults)

	refreshed, err := models.FindAgentSession(session.ID)
	require.NoError(t, err)
	assert.Equal(t, models.AgentSessionStatusFailed, refreshed.Status)
}

func TestAgentStreamWorker_DoesNotResendResolvedCustomToolResults(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	session := mustCreateSession(t, r, canvas.ID)

	provider := &scriptedProvider{
		name: testProvider,
		eventBatches: [][]agents.ProviderEvent{
			{
				{
					ProviderEventID: "custom-1",
					Type:            agents.ProviderEventCustomToolUseStarted,
					ToolName:        testCustomToolName,
					ToolCallID:      "custom-1",
					ToolInput:       `{"action":"read"}`,
					CustomToolUse:   &agents.CustomToolUse{ID: "custom-1", Name: testCustomToolName, Input: `{"action":"read"}`},
				},
				{Type: agents.ProviderEventCustomToolResultsRequired, CustomToolEventIDs: []string{"custom-1"}},
			},
			{
				{Type: agents.ProviderEventCustomToolResultsRequired, CustomToolEventIDs: []string{"custom-1"}},
				{ProviderEventID: "msg-1", Type: agents.ProviderEventAssistantMessage, Text: "Done after replay"},
				{Type: agents.ProviderEventTurnCompleted},
			},
		},
	}
	executor := &fakeCustomToolExecutor{}

	w := workers.NewAgentStreamWorker(provider, "amqp://ignored", executor)
	body, _ := json.Marshal(messages.AgentStreamRequest{SessionID: session.ID.String()})
	require.NoError(t, w.Handle(context.Background(), body))

	require.Equal(t, 2, provider.streamCalls)
	require.Len(t, executor.seen, 1)
	require.Len(t, provider.sentResults, 1)
	assert.Equal(t, "custom-1", provider.sentResults[0].CustomToolUseID)

	stored, err := models.ListAgentSessionMessagesPage(session.ID, nil, 100)
	require.NoError(t, err)
	require.Len(t, stored, 2)
	assert.Equal(t, models.AgentMessageRoleAssistant, stored[1].Role)
	assert.Equal(t, "Done after replay", stored[1].Content)
}

func TestAgentStreamWorker_DoesNotResendPersistedCustomToolResultsOnReplay(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	session := mustCreateSession(t, r, canvas.ID)
	require.NoError(t, models.AppendAgentSessionMessage(&models.AgentSessionMessage{
		SessionID:       session.ID,
		ProviderEventID: "custom-1",
		Role:            models.AgentMessageRoleTool,
		Content:         `{"ok":true}`,
		ToolName:        testCustomToolName,
		ToolStatus:      models.AgentToolStatusFinished,
	}))

	provider := &scriptedProvider{
		name: testProvider,
		events: []agents.ProviderEvent{
			{
				ProviderEventID: "custom-1",
				Type:            agents.ProviderEventCustomToolUseStarted,
				ToolName:        testCustomToolName,
				ToolCallID:      "custom-1",
				ToolInput:       `{"action":"read"}`,
				CustomToolUse:   &agents.CustomToolUse{ID: "custom-1", Name: testCustomToolName, Input: `{"action":"read"}`},
			},
			{Type: agents.ProviderEventCustomToolResultsRequired, CustomToolEventIDs: []string{"custom-1"}},
			{ProviderEventID: "msg-1", Type: agents.ProviderEventAssistantMessage, Text: "Recovered message"},
			{Type: agents.ProviderEventTurnCompleted},
		},
	}
	executor := &fakeCustomToolExecutor{}

	w := workers.NewAgentStreamWorker(provider, "amqp://ignored", executor)
	body, _ := json.Marshal(messages.AgentStreamRequest{SessionID: session.ID.String()})
	require.NoError(t, w.Handle(context.Background(), body))

	require.Len(t, executor.seen, 0)
	require.Len(t, provider.sentResults, 0)

	stored, err := models.ListAgentSessionMessagesPage(session.ID, nil, 100)
	require.NoError(t, err)
	require.Len(t, stored, 2)
	assert.Equal(t, models.AgentMessageRoleTool, stored[0].Role)
	assert.Equal(t, models.AgentToolStatusFinished, stored[0].ToolStatus)
	assert.Equal(t, `{"ok":true}`, stored[0].Content)
	assert.Equal(t, models.AgentMessageRoleAssistant, stored[1].Role)
	assert.Equal(t, "Recovered message", stored[1].Content)
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

func TestAgentStreamWorker_DoesNotOverwriteIdleWithFailedAfterInterrupt(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	session := mustCreateSession(t, r, canvas.ID)

	// Stream blocks on `release`, then returns an error — simulating
	// "provider 500'd after the user already hit Stop". The interrupt
	// path (UpdateAgentSessionStatus → idle) bumps updated_at, and the
	// worker's failed-path must respect that and not flip the row from
	// idle back to failed.
	provider := &scriptedProvider{
		name:        testProvider,
		streamReady: make(chan struct{}),
		release:     make(chan struct{}),
		err:         errors.New("provider blew up after interrupt"),
	}

	w := workers.NewAgentStreamWorker(provider, "amqp://ignored")
	body, _ := json.Marshal(messages.AgentStreamRequest{SessionID: session.ID.String()})

	done := make(chan error, 1)
	go func() { done <- w.Handle(context.Background(), body) }()

	<-provider.streamReady
	require.NoError(t, models.UpdateAgentSessionStatus(session.ID, models.AgentSessionStatusIdle))
	close(provider.release)
	require.NoError(t, <-done)

	refreshed, err := models.FindAgentSession(session.ID)
	require.NoError(t, err)
	assert.Equal(t, models.AgentSessionStatusIdle, refreshed.Status,
		"a stream error that arrives after Stop must not overwrite the user's interrupt")
}

func TestAgentStreamWorker_DropsAssistantEventsAfterInterrupt(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	session := mustCreateSession(t, r, canvas.ID)

	// The user reported seeing a new message appear in the transcript
	// AFTER clicking Stop. This guards the race: InterruptSession flips
	// the row to idle, then late events arrive from the still-open SSE
	// (Anthropic flushing already-generated content). handleProviderEvent
	// must check status per event and exit the stream without persisting
	// or broadcasting anything to the UI.
	provider := &scriptedProvider{
		name:        testProvider,
		streamReady: make(chan struct{}),
		release:     make(chan struct{}),
		events: []agents.ProviderEvent{
			{ProviderEventID: "msg-late", Type: agents.ProviderEventAssistantMessage, Text: "late content that must not appear"},
			{Type: agents.ProviderEventTurnCompleted},
		},
	}

	w := workers.NewAgentStreamWorker(provider, "amqp://ignored")
	body, _ := json.Marshal(messages.AgentStreamRequest{SessionID: session.ID.String()})

	done := make(chan error, 1)
	go func() { done <- w.Handle(context.Background(), body) }()

	<-provider.streamReady
	require.NoError(t, models.UpdateAgentSessionStatus(session.ID, models.AgentSessionStatusIdle))
	close(provider.release)
	require.NoError(t, <-done)

	stored, err := models.ListAgentSessionMessagesPage(session.ID, nil, 100)
	require.NoError(t, err)
	assert.Empty(t, stored, "no assistant rows must be persisted once the session has been reset; late SSE content must be discarded")

	refreshed, err := models.FindAgentSession(session.ID)
	require.NoError(t, err)
	assert.Equal(t, models.AgentSessionStatusIdle, refreshed.Status,
		"the worker must exit cleanly without overwriting the user's stop with an idle/failed transition of its own")
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

	// Heartbeated row whose last heartbeat is past the tight cutoff:
	// flag as leaked.
	stale := mustCreateSession(t, r, staleCanvas.ID)
	require.NoError(t, database.Conn().Model(&models.AgentSession{}).
		Where("id = ?", stale.ID).
		UpdateColumn("heartbeat_at", time.Now().Add(-10*time.Minute)).Error)
	fresh := mustCreateSession(t, r, freshCanvas.ID)
	require.NoError(t, database.Conn().Model(&models.AgentSession{}).
		Where("id = ?", fresh.ID).
		UpdateColumn("heartbeat_at", time.Now()).Error)

	heartbeatCutoff := time.Now().Add(-2 * time.Minute)
	legacyCutoff := time.Now().Add(-30 * time.Minute)
	closed, err := models.FailStuckStreamingSessions(heartbeatCutoff, legacyCutoff)
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

func TestAgentStreamWorker_CleanupRespectsLegacyGraceForRowsWithoutHeartbeat(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	youngLegacyCanvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	oldLegacyCanvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)

	// A row without heartbeat_at is "legacy": owned by a binary that
	// doesn't write heartbeats yet (e.g. mid-rolling-deploy). Cleanup
	// must fall back to the loose updated_at cutoff so a healthy long
	// turn isn't force-failed before the worker finishes.
	young := mustCreateSession(t, r, youngLegacyCanvas.ID)
	require.NoError(t, database.Conn().Model(&models.AgentSession{}).
		Where("id = ?", young.ID).
		UpdateColumn("updated_at", time.Now().Add(-10*time.Minute)).Error)
	old := mustCreateSession(t, r, oldLegacyCanvas.ID)
	require.NoError(t, database.Conn().Model(&models.AgentSession{}).
		Where("id = ?", old.ID).
		UpdateColumn("updated_at", time.Now().Add(-45*time.Minute)).Error)

	heartbeatCutoff := time.Now().Add(-2 * time.Minute)
	legacyCutoff := time.Now().Add(-30 * time.Minute)
	closed, err := models.FailStuckStreamingSessions(heartbeatCutoff, legacyCutoff)
	require.NoError(t, err)
	require.Len(t, closed, 1, "only the row past the legacy cutoff should be flagged")
	assert.Equal(t, old.ID, closed[0].ID)

	youngAfter, err := models.FindAgentSession(young.ID)
	require.NoError(t, err)
	assert.Equal(t, models.AgentSessionStatusStreaming, youngAfter.Status,
		"a 10-min legacy turn is within the loose cutoff and must survive — covers rolling-deploy safety")
}

func TestAgentStreamWorker_HeartbeatKeepsSessionAlive(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	session := mustCreateSession(t, r, canvas.ID)

	// Pre-stamp heartbeat_at well past the heartbeat cutoff. A live
	// heartbeat must bump it forward so cleanup no longer flags the
	// row — without that, a healthy worker's session would be wrongly
	// failed under load.
	require.NoError(t, database.Conn().Model(&models.AgentSession{}).
		Where("id = ?", session.ID).
		UpdateColumn("heartbeat_at", time.Now().Add(-10*time.Minute)).Error)

	require.NoError(t, models.TouchAgentSessionHeartbeat(session.ID))

	heartbeatCutoff := time.Now().Add(-2 * time.Minute)
	legacyCutoff := time.Now().Add(-30 * time.Minute)
	closed, err := models.FailStuckStreamingSessions(heartbeatCutoff, legacyCutoff)
	require.NoError(t, err)
	assert.Empty(t, closed, "a fresh heartbeat must lift the row out of the stuck-cleanup window")

	refreshed, err := models.FindAgentSession(session.ID)
	require.NoError(t, err)
	assert.Equal(t, models.AgentSessionStatusStreaming, refreshed.Status)
}

func TestAgentStreamWorker_StatusTransitionClearsHeartbeat(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	session := mustCreateSession(t, r, canvas.ID)

	// Stale heartbeat lingering from a previous turn. Once the row goes
	// idle and then back to streaming for a new turn, cleanup must NOT
	// see the old heartbeat — otherwise a healthy queued turn gets
	// failed before the worker's first heartbeat lands.
	require.NoError(t, database.Conn().Model(&models.AgentSession{}).
		Where("id = ?", session.ID).
		UpdateColumn("heartbeat_at", time.Now().Add(-10*time.Minute)).Error)

	require.NoError(t, models.UpdateAgentSessionStatus(session.ID, models.AgentSessionStatusIdle))
	require.NoError(t, models.UpdateAgentSessionStatus(session.ID, models.AgentSessionStatusStreaming))

	refreshed, err := models.FindAgentSession(session.ID)
	require.NoError(t, err)
	assert.Nil(t, refreshed.HeartbeatAt, "status transitions must clear heartbeat_at so a new turn starts in the legacy cutoff branch")

	heartbeatCutoff := time.Now().Add(-2 * time.Minute)
	legacyCutoff := time.Now().Add(-30 * time.Minute)
	closed, err := models.FailStuckStreamingSessions(heartbeatCutoff, legacyCutoff)
	require.NoError(t, err)
	assert.Empty(t, closed, "a freshly-restarted streaming row must not be flagged on the strength of its previous turn's heartbeat")
}

func TestAgentStreamWorker_HeartbeatSkipsNonStreamingRows(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	session := mustCreateSession(t, r, canvas.ID)

	// Once a session is no longer streaming (interrupted, failed, etc.),
	// a goroutine that lived past the reset must not be able to plant a
	// heartbeat_at on the new state.
	require.NoError(t, models.UpdateAgentSessionStatus(session.ID, models.AgentSessionStatusIdle))

	require.NoError(t, models.TouchAgentSessionHeartbeat(session.ID))

	after, err := models.FindAgentSession(session.ID)
	require.NoError(t, err)
	assert.Nil(t, after.HeartbeatAt,
		"heartbeat must be a no-op when status != 'streaming' — the row has never been streamed by a heartbeat-aware worker")
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
