package support

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/agents"
)

func TestAgentProviderCreatesSessionAndStreamsQueuedEvents(t *testing.T) {
	provider := NewAgentProvider()
	provider.SetSessionIDProvider(func() string { return "session-1" })

	result, err := provider.CreateSession(context.Background(), agents.CreateSessionOptions{
		Title:          "Test session",
		InitialContext: "context",
		VaultIDs:       []string{"vault-1"},
		Resources: []agents.FileResource{
			{FileID: "file-1", MountPath: "/workspace/file-1"},
		},
	})
	require.NoError(t, err)
	require.Equal(t, "session-1", result.ProviderSessionID)

	err = provider.QueueEvents(
		result.ProviderSessionID,
		agents.ProviderEvent{ProviderEventID: "event-1", Type: agents.ProviderEventAssistantMessage, Text: "hello"},
		agents.ProviderEvent{ProviderEventID: "event-2", Type: agents.ProviderEventTurnCompleted},
	)
	require.NoError(t, err)

	var events []agents.ProviderEvent
	err = provider.StreamEvents(context.Background(), result.ProviderSessionID, func(event agents.ProviderEvent) error {
		events = append(events, event)
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, []agents.ProviderEvent{
		{ProviderEventID: "event-1", Type: agents.ProviderEventAssistantMessage, Text: "hello"},
		{ProviderEventID: "event-2", Type: agents.ProviderEventTurnCompleted},
	}, events)

	require.Equal(t, []AgentProviderCreateSessionCall{{
		Options: agents.CreateSessionOptions{
			Title:          "Test session",
			InitialContext: "context",
			VaultIDs:       []string{"vault-1"},
			Resources: []agents.FileResource{
				{FileID: "file-1", MountPath: "/workspace/file-1"},
			},
		},
	}}, provider.CreateSessionCalls())
}

func TestAgentProviderSendMessageRecordsCallAndQueuesConfiguredEvents(t *testing.T) {
	provider := NewAgentProvider()
	provider.SetSessionIDProvider(func() string { return "session-1" })
	provider.SetSendMessageEvents(
		agents.ProviderEvent{ProviderEventID: "event-1", Type: agents.ProviderEventAssistantMessage, Text: "reply"},
		agents.ProviderEvent{ProviderEventID: "event-2", Type: agents.ProviderEventTurnCompleted},
	)

	result, err := provider.CreateSession(context.Background(), agents.CreateSessionOptions{})
	require.NoError(t, err)

	err = provider.SendMessage(
		context.Background(),
		result.ProviderSessionID,
		"hello",
		agents.SendMessageOptions{ContextPreamble: "preamble"},
	)
	require.NoError(t, err)

	var events []agents.ProviderEvent
	err = provider.StreamEvents(context.Background(), result.ProviderSessionID, func(event agents.ProviderEvent) error {
		events = append(events, event)
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, []agents.ProviderEvent{
		{ProviderEventID: "event-1", Type: agents.ProviderEventAssistantMessage, Text: "reply"},
		{ProviderEventID: "event-2", Type: agents.ProviderEventTurnCompleted},
	}, events)

	require.Equal(t, []AgentProviderSendMessageCall{{
		ProviderSessionID: "session-1",
		Message:           "hello",
		Options:           agents.SendMessageOptions{ContextPreamble: "preamble"},
	}}, provider.SendMessageCalls())
}

func TestAgentProviderSendMessageHandlerCanBuildDynamicEvents(t *testing.T) {
	provider := NewAgentProvider()
	provider.SetSessionIDProvider(func() string { return "session-1" })
	provider.SetSendMessageHandler(func(call AgentProviderSendMessageCall) ([]agents.ProviderEvent, error) {
		return []agents.ProviderEvent{
			{ProviderEventID: "event-1", Type: agents.ProviderEventAssistantMessage, Text: call.Message},
			{ProviderEventID: "event-2", Type: agents.ProviderEventTurnCompleted},
		}, nil
	})

	result, err := provider.CreateSession(context.Background(), agents.CreateSessionOptions{})
	require.NoError(t, err)
	require.NoError(t, provider.SendMessage(context.Background(), result.ProviderSessionID, "echo me", agents.SendMessageOptions{}))

	var events []agents.ProviderEvent
	err = provider.StreamEvents(context.Background(), result.ProviderSessionID, func(event agents.ProviderEvent) error {
		events = append(events, event)
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, "echo me", events[0].Text)
}

func TestAgentProviderDefineOutcomeRecordsCallAndQueuesConfiguredEvents(t *testing.T) {
	provider := NewAgentProvider()
	provider.SetSessionIDProvider(func() string { return "session-1" })
	provider.SetDefineOutcomeEvents(
		agents.ProviderEvent{ProviderEventID: "event-1", Type: agents.ProviderEventOutcomeEvaluationStart},
		agents.ProviderEvent{
			ProviderEventID: "event-2",
			Type:            agents.ProviderEventOutcomeEvaluation,
			OutcomeResult:   &agents.OutcomeEvaluation{Iteration: 1, Result: "satisfied", Explanation: "done"},
		},
		agents.ProviderEvent{ProviderEventID: "event-3", Type: agents.ProviderEventTurnCompleted},
	)

	result, err := provider.CreateSession(context.Background(), agents.CreateSessionOptions{})
	require.NoError(t, err)

	options := agents.DefineOutcomeOptions{
		Description:     "ship it",
		Rubric:          "works",
		MaxIterations:   2,
		ContextPreamble: "preamble",
	}
	require.NoError(t, provider.DefineOutcome(context.Background(), result.ProviderSessionID, options))

	var events []agents.ProviderEvent
	err = provider.StreamEvents(context.Background(), result.ProviderSessionID, func(event agents.ProviderEvent) error {
		events = append(events, event)
		return nil
	})
	require.NoError(t, err)
	require.Len(t, events, 3)
	require.Equal(t, agents.ProviderEventOutcomeEvaluation, events[1].Type)

	require.Equal(t, []AgentProviderDefineOutcomeCall{{
		ProviderSessionID: "session-1",
		Options:           options,
	}}, provider.DefineOutcomeCalls())
}

func TestAgentProviderStreamReturnsCallbackAndQueuedErrors(t *testing.T) {
	provider := NewAgentProvider()
	provider.SetSessionIDProvider(func() string { return "session-1" })

	result, err := provider.CreateSession(context.Background(), agents.CreateSessionOptions{})
	require.NoError(t, err)

	callbackErr := errors.New("callback failed")
	err = provider.QueueEvents(result.ProviderSessionID, agents.ProviderEvent{
		ProviderEventID: "event-1",
		Type:            agents.ProviderEventAssistantMessage,
		Text:            "hello",
	})
	require.NoError(t, err)

	err = provider.StreamEvents(context.Background(), result.ProviderSessionID, func(agents.ProviderEvent) error {
		return callbackErr
	})
	require.ErrorIs(t, err, callbackErr)

	queuedErr := errors.New("stream failed")
	require.NoError(t, provider.QueueError(result.ProviderSessionID, queuedErr))

	err = provider.StreamEvents(context.Background(), result.ProviderSessionID, func(agents.ProviderEvent) error {
		return nil
	})
	require.ErrorIs(t, err, queuedErr)
}

func TestAgentProviderInterruptAndDeleteUpdateSessionState(t *testing.T) {
	provider := NewAgentProvider()
	provider.SetSessionIDProvider(func() string { return "session-1" })

	result, err := provider.CreateSession(context.Background(), agents.CreateSessionOptions{})
	require.NoError(t, err)

	require.NoError(t, provider.InterruptSession(context.Background(), result.ProviderSessionID))
	session, ok := provider.Session(result.ProviderSessionID)
	require.True(t, ok)
	require.True(t, session.Interrupted)

	require.NoError(t, provider.DeleteSession(context.Background(), result.ProviderSessionID))
	session, ok = provider.Session(result.ProviderSessionID)
	require.True(t, ok)
	require.True(t, session.Deleted)

	err = provider.StreamEvents(context.Background(), result.ProviderSessionID, func(agents.ProviderEvent) error {
		return nil
	})
	require.NoError(t, err)
	require.ErrorIs(t, provider.QueueEvents(result.ProviderSessionID, agents.ProviderEvent{}), agents.ErrSessionAlreadyTerminated)

	require.Equal(t, []AgentProviderInterruptSessionCall{{
		ProviderSessionID: "session-1",
	}}, provider.InterruptSessionCalls())
	require.Equal(t, []AgentProviderDeleteSessionCall{{
		ProviderSessionID: "session-1",
	}}, provider.DeleteSessionCalls())
}
