package native

import (
	"context"
	"errors"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/agents"
	"github.com/superplanehq/superplane/pkg/agents/native/llm"
)

func TestProviderStreamsFinalTextAndCompletesTurn(t *testing.T) {
	client := llm.NewScriptedClient([]llm.StreamEvent{
		{Type: llm.StreamEventTextDelta, Text: "Hello"},
		{Type: llm.StreamEventTextDelta, Text: ", world"},
	})
	provider := newTestProvider(t, client, 4)
	sessionID := createSession(t, provider)
	require.NoError(t, provider.SendMessage(context.Background(), sessionID, "Hi", agents.SendMessageOptions{ContextPreamble: "ctx"}))

	events := streamEvents(t, provider, sessionID)

	require.Len(t, events, 2)
	assert.Equal(t, agents.ProviderEventAssistantMessage, events[0].Type)
	assert.Equal(t, "Hello, world", events[0].Text)
	assert.Equal(t, agents.ProviderEventTurnCompleted, events[1].Type)

	calls := client.Calls()
	require.Len(t, calls, 1)
	require.Len(t, calls[0].Messages, 2)
	assert.Equal(t, llm.RoleSystem, calls[0].Messages[0].Role)
	assert.Contains(t, calls[0].Messages[0].Blocks[0].Text, "You are a SuperPlane app expert")
	assert.Equal(t, llm.RoleUser, calls[0].Messages[1].Role)
	assert.Contains(t, calls[0].Messages[1].Blocks[0].Text, "ctx")
	assert.NotEmpty(t, calls[0].Tools)
}

func TestProviderUsesSameSuperPlaneCustomToolSchemas(t *testing.T) {
	client := llm.NewScriptedClient([]llm.StreamEvent{{Type: llm.StreamEventTextDelta, Text: "ok"}})
	provider := newTestProvider(t, client, 4)
	sessionID := createSession(t, provider)
	require.NoError(t, provider.SendMessage(context.Background(), sessionID, "Use tools", agents.SendMessageOptions{}))

	_ = streamEvents(t, provider, sessionID)

	calls := client.Calls()
	require.Len(t, calls, 1)
	tools := toolDefinitionsByName(calls[0].Tools)

	appTool, ok := tools["superplane_app"]
	require.True(t, ok)
	actionProperty := appTool.InputSchema["properties"].(map[string]any)["action"].(map[string]any)
	actionEnum := stringSet(actionProperty["enum"].([]string))
	assert.Contains(t, actionEnum, "access")
	assert.Contains(t, actionEnum, "read")
	assert.Contains(t, actionEnum, "read_runtime")
	assert.Contains(t, actionEnum, "create_draft")
	assert.Contains(t, actionEnum, "update_draft")
	assert.Contains(t, actionEnum, "list_integrations")

	_, ok = tools["superplane_component_schema"]
	assert.True(t, ok)
}

func TestProviderPausesForToolResultsAndResumesLoop(t *testing.T) {
	client := llm.NewScriptedClient(
		[]llm.StreamEvent{
			{Type: llm.StreamEventToolCall, ToolCall: &llm.ToolCall{ID: "tool-1", Name: "superplane_app", Input: `{"action":"read"}`}},
		},
		[]llm.StreamEvent{
			{Type: llm.StreamEventTextDelta, Text: "Done"},
		},
	)
	provider := newTestProvider(t, client, 4)
	sessionID := createSession(t, provider)
	require.NoError(t, provider.SendMessage(context.Background(), sessionID, "Read it", agents.SendMessageOptions{}))

	firstTurn := streamEvents(t, provider, sessionID)
	require.Len(t, firstTurn, 2)
	assert.Equal(t, agents.ProviderEventCustomToolUseStarted, firstTurn[0].Type)
	assert.Equal(t, "tool-1", firstTurn[0].CustomToolUse.ID)
	assert.Equal(t, agents.ProviderEventCustomToolResultsRequired, firstTurn[1].Type)
	assert.Equal(t, []string{"tool-1"}, firstTurn[1].CustomToolEventIDs)

	require.NoError(t, provider.SendCustomToolResults(context.Background(), sessionID, []agents.CustomToolResult{
		{CustomToolUseID: "tool-1", Content: `{"ok":true}`},
	}))

	secondTurn := streamEvents(t, provider, sessionID)
	require.Len(t, secondTurn, 2)
	assert.Equal(t, agents.ProviderEventAssistantMessage, secondTurn[0].Type)
	assert.Equal(t, "Done", secondTurn[0].Text)
	assert.Equal(t, agents.ProviderEventTurnCompleted, secondTurn[1].Type)

	calls := client.Calls()
	require.Len(t, calls, 2)
	lastMessages := calls[1].Messages
	require.Len(t, lastMessages, 4)
	assert.Equal(t, llm.RoleTool, lastMessages[3].Role)
	assert.Equal(t, "tool-1", lastMessages[3].Blocks[0].ToolResult.ToolCallID)
}

func TestProviderResumesSessionFromDurableStore(t *testing.T) {
	store := newMemoryStore()
	client := llm.NewScriptedClient(
		[]llm.StreamEvent{
			{Type: llm.StreamEventToolCall, ToolCall: &llm.ToolCall{ID: "tool-1", Name: "superplane_app", Input: `{"action":"read"}`}},
		},
		[]llm.StreamEvent{
			{Type: llm.StreamEventTextDelta, Text: "Recovered"},
		},
	)
	firstProvider := newTestProviderWithConfig(t, Config{Client: client, Model: "fast-test-model", MaxSteps: 4, Store: store})
	sessionID := createSession(t, firstProvider)
	require.NoError(t, firstProvider.SendMessage(context.Background(), sessionID, "Read it", agents.SendMessageOptions{}))

	firstTurn := streamEvents(t, firstProvider, sessionID)
	require.Len(t, firstTurn, 2)
	assert.Equal(t, agents.ProviderEventCustomToolResultsRequired, firstTurn[1].Type)

	restartedProvider := newTestProviderWithConfig(t, Config{Client: client, Model: "fast-test-model", MaxSteps: 4, Store: store})
	require.NoError(t, restartedProvider.SendCustomToolResults(context.Background(), sessionID, []agents.CustomToolResult{
		{CustomToolUseID: "tool-1", Content: `{"ok":true}`},
	}))

	secondTurn := streamEvents(t, restartedProvider, sessionID)
	require.Len(t, secondTurn, 2)
	assert.Equal(t, agents.ProviderEventAssistantMessage, secondTurn[0].Type)
	assert.Equal(t, "Recovered", secondTurn[0].Text)
	assert.Equal(t, agents.ProviderEventTurnCompleted, secondTurn[1].Type)
}

func TestProviderRejectsNewMessageWhileWaitingForToolResults(t *testing.T) {
	client := llm.NewScriptedClient([]llm.StreamEvent{
		{Type: llm.StreamEventToolCall, ToolCall: &llm.ToolCall{ID: "tool-1", Name: "superplane_app", Input: `{"action":"read"}`}},
	})
	provider := newTestProvider(t, client, 4)
	sessionID := createSession(t, provider)
	require.NoError(t, provider.SendMessage(context.Background(), sessionID, "Read it", agents.SendMessageOptions{}))
	_ = streamEvents(t, provider, sessionID)

	err := provider.SendMessage(context.Background(), sessionID, "new turn too early", agents.SendMessageOptions{})

	require.ErrorIs(t, err, agents.ErrSessionBusy)
}

func TestProviderRequiresMultipleToolResultsInModelOrder(t *testing.T) {
	client := llm.NewScriptedClient([]llm.StreamEvent{
		{Type: llm.StreamEventToolCall, ToolCall: &llm.ToolCall{ID: "tool-1", Name: "superplane_app", Input: `{}`}},
		{Type: llm.StreamEventToolCall, ToolCall: &llm.ToolCall{ID: "tool-2", Name: "superplane_component_schema", Input: `{}`}},
	})
	provider := newTestProvider(t, client, 4)
	sessionID := createSession(t, provider)
	require.NoError(t, provider.SendMessage(context.Background(), sessionID, "Use tools", agents.SendMessageOptions{}))

	events := streamEvents(t, provider, sessionID)

	require.Len(t, events, 3)
	assert.Equal(t, "tool-1", events[0].CustomToolUse.ID)
	assert.Equal(t, "tool-2", events[1].CustomToolUse.ID)
	assert.Equal(t, []string{"tool-1", "tool-2"}, events[2].CustomToolEventIDs)
}

func TestProviderStopsAtMaxSteps(t *testing.T) {
	client := llm.NewScriptedClient(
		[]llm.StreamEvent{{Type: llm.StreamEventToolCall, ToolCall: &llm.ToolCall{ID: "tool-1", Name: "superplane_app", Input: `{}`}}},
		[]llm.StreamEvent{{Type: llm.StreamEventToolCall, ToolCall: &llm.ToolCall{ID: "tool-2", Name: "superplane_app", Input: `{"next":true}`}}},
	)
	provider := newTestProvider(t, client, 1)
	sessionID := createSession(t, provider)
	require.NoError(t, provider.SendMessage(context.Background(), sessionID, "Loop", agents.SendMessageOptions{}))
	_ = streamEvents(t, provider, sessionID)
	require.NoError(t, provider.SendCustomToolResults(context.Background(), sessionID, []agents.CustomToolResult{
		{CustomToolUseID: "tool-1", Content: `{}`},
	}))

	var events []agents.ProviderEvent
	err := provider.StreamEvents(context.Background(), sessionID, func(event agents.ProviderEvent) error {
		events = append(events, event)
		return nil
	})

	require.ErrorIs(t, err, errMaxStepsReached)
	assert.Empty(t, events)
}

func TestProviderStopsWhenModelRequestsTooManyToolCalls(t *testing.T) {
	client := llm.NewScriptedClient([]llm.StreamEvent{
		{Type: llm.StreamEventToolCall, ToolCall: &llm.ToolCall{ID: "tool-1", Name: "superplane_app", Input: `{}`}},
		{Type: llm.StreamEventToolCall, ToolCall: &llm.ToolCall{ID: "tool-2", Name: "superplane_app", Input: `{}`}},
	})
	provider := newTestProviderWithConfig(t, Config{
		Client:       client,
		Model:        "fast-test-model",
		MaxSteps:     4,
		MaxToolCalls: 1,
	})
	sessionID := createSession(t, provider)
	require.NoError(t, provider.SendMessage(context.Background(), sessionID, "Too many tools", agents.SendMessageOptions{}))

	err := provider.StreamEvents(context.Background(), sessionID, func(agents.ProviderEvent) error {
		return nil
	})

	require.ErrorIs(t, err, errMaxToolCalls)
}

func TestProviderBoundsHistorySentToLLM(t *testing.T) {
	client := llm.NewScriptedClient(
		[]llm.StreamEvent{{Type: llm.StreamEventTextDelta, Text: "first"}},
		[]llm.StreamEvent{{Type: llm.StreamEventTextDelta, Text: "second"}},
	)
	provider := newTestProviderWithConfig(t, Config{
		Client:          client,
		Model:           "fast-test-model",
		MaxSteps:        4,
		MaxContextChars: 80,
		SystemPrompt:    "system prompt that must remain",
	})
	sessionID := createSession(t, provider)
	require.NoError(t, provider.SendMessage(context.Background(), sessionID, strings.Repeat("old ", 80), agents.SendMessageOptions{}))
	_ = streamEvents(t, provider, sessionID)
	require.NoError(t, provider.SendMessage(context.Background(), sessionID, "new request", agents.SendMessageOptions{}))
	_ = streamEvents(t, provider, sessionID)

	calls := client.Calls()
	require.Len(t, calls, 2)
	lastMessages := calls[1].Messages
	require.NotEmpty(t, lastMessages)
	assert.Equal(t, llm.RoleSystem, lastMessages[0].Role)
	assert.Contains(t, lastMessages[0].Blocks[0].Text, "system prompt")
	assert.LessOrEqual(t, messageChars(lastMessages), 80)
}

func TestBoundedHistoryCompactsOmittedOlderMessages(t *testing.T) {
	history := []llm.Message{
		llm.NewSystemMessage("system prompt"),
		llm.NewUserMessage(strings.Repeat("old requirement ", 120)),
		llm.NewAssistantMessage([]llm.Block{{Type: llm.BlockTypeText, Text: strings.Repeat("old work ", 120)}}),
		llm.NewUserMessage("current request"),
	}

	bounded, err := boundedHistory(history, 900)

	require.NoError(t, err)
	require.Len(t, bounded, 3)
	assert.Equal(t, llm.RoleSystem, bounded[0].Role)
	assert.Contains(t, bounded[1].Blocks[0].Text, "Compacted earlier conversation")
	assert.Equal(t, llm.RoleUser, bounded[2].Role)
	assert.Equal(t, "current request", bounded[2].Blocks[0].Text)
	assert.LessOrEqual(t, messageChars(bounded), 900)
}

func TestProviderStopsRepeatedToolCallDoomLoop(t *testing.T) {
	repeated := llm.StreamEvent{
		Type:     llm.StreamEventToolCall,
		ToolCall: &llm.ToolCall{ID: "tool-1", Name: "superplane_app", Input: `{"action":"read"}`},
	}
	client := llm.NewScriptedClient(
		[]llm.StreamEvent{repeated},
		[]llm.StreamEvent{repeated},
		[]llm.StreamEvent{repeated},
	)
	provider := newTestProvider(t, client, 5)
	sessionID := createSession(t, provider)
	require.NoError(t, provider.SendMessage(context.Background(), sessionID, "Loop", agents.SendMessageOptions{}))

	_ = streamEvents(t, provider, sessionID)
	require.NoError(t, provider.SendCustomToolResults(context.Background(), sessionID, []agents.CustomToolResult{{CustomToolUseID: "tool-1", Content: `{}`}}))
	_ = streamEvents(t, provider, sessionID)
	require.NoError(t, provider.SendCustomToolResults(context.Background(), sessionID, []agents.CustomToolResult{{CustomToolUseID: "tool-1", Content: `{}`}}))

	err := provider.StreamEvents(context.Background(), sessionID, func(agents.ProviderEvent) error {
		return nil
	})

	require.ErrorIs(t, err, errDoomLoop)
}

func TestProviderInterruptPreventsFurtherStreaming(t *testing.T) {
	client := llm.NewScriptedClient([]llm.StreamEvent{{Type: llm.StreamEventTextDelta, Text: "nope"}})
	provider := newTestProvider(t, client, 4)
	sessionID := createSession(t, provider)
	require.NoError(t, provider.SendMessage(context.Background(), sessionID, "Stop", agents.SendMessageOptions{}))
	require.NoError(t, provider.InterruptSession(context.Background(), sessionID))

	err := provider.StreamEvents(context.Background(), sessionID, func(agents.ProviderEvent) error {
		return errors.New("must not stream")
	})

	require.ErrorIs(t, err, agents.ErrSessionAlreadyTerminated)
	assert.Empty(t, client.Calls())
}

func TestProviderFakeLatencySamples(t *testing.T) {
	client := timedClient{
		firstDelay:  time.Millisecond,
		secondDelay: time.Millisecond,
	}
	provider := newTestProvider(t, client, 4)

	firstEventSamples := []time.Duration{}
	totalSamples := []time.Duration{}
	for i := 0; i < 9; i++ {
		sessionID := createSession(t, provider)
		require.NoError(t, provider.SendMessage(context.Background(), sessionID, "Measure", agents.SendMessageOptions{}))

		started := time.Now()
		var firstEventAt time.Time
		err := provider.StreamEvents(context.Background(), sessionID, func(agents.ProviderEvent) error {
			if firstEventAt.IsZero() {
				firstEventAt = time.Now()
			}
			return nil
		})
		require.NoError(t, err)
		require.False(t, firstEventAt.IsZero())
		firstEventSamples = append(firstEventSamples, firstEventAt.Sub(started))
		totalSamples = append(totalSamples, time.Since(started))
	}

	firstP50, firstP95 := percentiles(firstEventSamples)
	totalP50, totalP95 := percentiles(totalSamples)
	t.Logf("native fake latency time_to_first p50=%s p95=%s total p50=%s p95=%s", firstP50, firstP95, totalP50, totalP95)

	assert.GreaterOrEqual(t, firstP95, firstP50)
	assert.GreaterOrEqual(t, totalP95, totalP50)
}

func newTestProvider(t *testing.T, client llm.Client, maxSteps int) *Provider {
	t.Helper()
	return newTestProviderWithConfig(t, Config{Client: client, Model: "fast-test-model", MaxSteps: maxSteps})
}

func newTestProviderWithConfig(t *testing.T, cfg Config) *Provider {
	t.Helper()
	provider, err := New(cfg)
	require.NoError(t, err)
	return provider
}

type timedClient struct {
	firstDelay  time.Duration
	secondDelay time.Duration
}

func (c timedClient) Stream(ctx context.Context, _ llm.StreamRequest, onEvent func(llm.StreamEvent) error) error {
	if err := sleep(ctx, c.firstDelay); err != nil {
		return err
	}
	if err := onEvent(llm.StreamEvent{Type: llm.StreamEventTextDelta, Text: "hello"}); err != nil {
		return err
	}
	return sleep(ctx, c.secondDelay)
}

func sleep(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func percentiles(samples []time.Duration) (p50, p95 time.Duration) {
	ordered := append([]time.Duration(nil), samples...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i] < ordered[j] })
	return ordered[percentileIndex(len(ordered), 0.50)], ordered[percentileIndex(len(ordered), 0.95)]
}

func percentileIndex(length int, percentile float64) int {
	if length <= 1 {
		return 0
	}
	index := int(float64(length-1) * percentile)
	if index < 0 {
		return 0
	}
	if index >= length {
		return length - 1
	}
	return index
}

func toolDefinitionsByName(tools []llm.ToolDefinition) map[string]llm.ToolDefinition {
	byName := map[string]llm.ToolDefinition{}
	for _, tool := range tools {
		byName[tool.Name] = tool
	}
	return byName
}

func stringSet(values []string) map[string]struct{} {
	set := map[string]struct{}{}
	for _, value := range values {
		set[value] = struct{}{}
	}
	return set
}

func createSession(t *testing.T, provider *Provider) string {
	t.Helper()
	session, err := provider.CreateSession(context.Background(), agents.CreateSessionOptions{})
	require.NoError(t, err)
	return session.ProviderSessionID
}

func streamEvents(t *testing.T, provider *Provider, sessionID string) []agents.ProviderEvent {
	t.Helper()
	events := []agents.ProviderEvent{}
	err := provider.StreamEvents(context.Background(), sessionID, func(event agents.ProviderEvent) error {
		events = append(events, event)
		return nil
	})
	require.NoError(t, err)
	return events
}
