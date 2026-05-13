package anthropic

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/agents"
)

func newTestProvider(t *testing.T, server *httptest.Server) *Provider {
	t.Helper()
	p, err := New(Config{
		APIKey:        "test-key",
		AgentID:       "agent-123",
		EnvironmentID: "env-456",
		BaseURL:       server.URL,
	})
	require.NoError(t, err)
	return p
}

func TestNew_RequiresAllFields(t *testing.T) {
	cases := []struct {
		name string
		cfg  Config
	}{
		{"missing api key", Config{AgentID: "a", EnvironmentID: "e"}},
		{"missing agent id", Config{APIKey: "k", EnvironmentID: "e"}},
		{"missing env id", Config{APIKey: "k", AgentID: "a"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := New(c.cfg)
			require.Error(t, err)
		})
	}
}

func TestCreateSession_SendsCorrectRequest(t *testing.T) {
	var capturedBody map[string]any
	var capturedHeaders http.Header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/sessions", r.URL.Path)
		capturedHeaders = r.Header.Clone()
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &capturedBody)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"sesn_abc","status":"idle"}`))
	}))
	defer server.Close()

	p := newTestProvider(t, server)
	res, err := p.CreateSession(context.Background(), agents.CreateSessionOptions{})
	require.NoError(t, err)
	assert.Equal(t, "sesn_abc", res.ProviderSessionID)
	assert.Equal(t, "agent-123", capturedBody["agent"])
	assert.Equal(t, "env-456", capturedBody["environment_id"])
	assert.Equal(t, "test-key", capturedHeaders.Get("x-api-key"))
	assert.Equal(t, managedAgentsBeta, capturedHeaders.Get("anthropic-beta"))
}

func TestCreateSession_PropagatesAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"bad"}`))
	}))
	defer server.Close()

	p := newTestProvider(t, server)
	_, err := p.CreateSession(context.Background(), agents.CreateSessionOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "400")
}

func TestSendMessage_PrependsPreamble(t *testing.T) {
	var capturedBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/sessions/sesn_abc/events", r.URL.Path)
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &capturedBody)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{}"))
	}))
	defer server.Close()

	p := newTestProvider(t, server)
	err := p.SendMessage(context.Background(), "sesn_abc", "do the thing", agents.SendMessageOptions{
		ContextPreamble: "[ctx]",
	})
	require.NoError(t, err)

	events := capturedBody["events"].([]any)
	require.Len(t, events, 1)
	content := events[0].(map[string]any)["content"].([]any)
	text := content[0].(map[string]any)["text"].(string)
	assert.Equal(t, "[ctx]\n\ndo the thing", text)
}

func TestSendMessage_NoPreamble(t *testing.T) {
	var capturedBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &capturedBody)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{}"))
	}))
	defer server.Close()

	p := newTestProvider(t, server)
	require.NoError(t, p.SendMessage(context.Background(), "sesn_abc", "hi", agents.SendMessageOptions{}))

	events := capturedBody["events"].([]any)
	content := events[0].(map[string]any)["content"].([]any)
	text := content[0].(map[string]any)["text"].(string)
	assert.Equal(t, "hi", text)
}

func TestSendMessage_RequiresSessionID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("server should not be hit")
	}))
	defer server.Close()
	p := newTestProvider(t, server)
	require.Error(t, p.SendMessage(context.Background(), "", "hi", agents.SendMessageOptions{}))
}

func TestArchiveSession(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		assert.Equal(t, "/sessions/sesn_abc/archive", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{}"))
	}))
	defer server.Close()

	p := newTestProvider(t, server)
	require.NoError(t, p.ArchiveSession(context.Background(), "sesn_abc"))
	assert.True(t, called)
}

func TestStreamEvents_MapsKnownTypes(t *testing.T) {
	const sse = "data: {\"id\":\"e1\",\"type\":\"agent.message\",\"content\":[{\"type\":\"text\",\"text\":\"Hello\"}]}\n\n" +
		"data: {\"id\":\"e2\",\"type\":\"agent.tool_use\",\"name\":\"bash\",\"input\":{\"command\":\"ls -la\"}}\n\n" +
		"data: {\"id\":\"e3\",\"type\":\"agent.tool_result\",\"name\":\"search\"}\n\n" +
		"data: {\"type\":\"session.status_idle\"}\n\n" +
		// Anything past status_idle must be ignored because the
		// provider returns once the turn closes.
		"data: {\"type\":\"agent.message\",\"content\":[{\"type\":\"text\",\"text\":\"after idle\"}]}\n\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/sessions/sesn_abc/events/stream", r.URL.Path)
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, sse)
	}))
	defer server.Close()

	p := newTestProvider(t, server)
	var received []agents.ProviderEvent
	err := p.StreamEvents(context.Background(), "sesn_abc", func(e agents.ProviderEvent) error {
		received = append(received, e)
		return nil
	})
	require.NoError(t, err)

	require.Len(t, received, 4)
	assert.Equal(t, agents.ProviderEventAssistantMessage, received[0].Type)
	assert.Equal(t, "Hello", received[0].Text)
	assert.Equal(t, agents.ProviderEventToolUseStarted, received[1].Type)
	assert.Equal(t, "bash", received[1].ToolName)
	assert.Equal(t, "ls -la", received[1].ToolInput, "bash-style tools surface the `command` field as the input")
	assert.Equal(t, agents.ProviderEventToolUseFinished, received[2].Type)
	assert.Equal(t, agents.ProviderEventTurnCompleted, received[3].Type)
}

func TestStreamEvents_RedactsJWTsInToolCommands(t *testing.T) {
	// Realistic bash command the agent writes when materialising the CLI
	// config from the first-turn preamble. The token must be replaced.
	const sse = "data: {\"id\":\"e1\",\"type\":\"agent.tool_use\",\"name\":\"bash\"," +
		"\"input\":{\"command\":\"cat > ~/.superplane.yaml <<EOF\\napiToken: eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ1In0.abc\\nEOF\"}}\n\n" +
		"data: {\"type\":\"session.status_idle\"}\n\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, sse)
	}))
	defer server.Close()

	p := newTestProvider(t, server)
	var received []agents.ProviderEvent
	require.NoError(t, p.StreamEvents(context.Background(), "sesn_abc", func(e agents.ProviderEvent) error {
		received = append(received, e)
		return nil
	}))
	require.GreaterOrEqual(t, len(received), 1)
	assert.NotContains(t, received[0].ToolInput, "eyJhbGciOiJIUzI1NiJ9", "JWT must not survive into the broadcast payload")
	assert.Contains(t, received[0].ToolInput, "<redacted>")
}

func TestStreamEvents_RedactsJWTsInAssistantMessages(t *testing.T) {
	const sse = "data: {\"id\":\"e1\",\"type\":\"agent.message\",\"content\":[{\"type\":\"text\"," +
		"\"text\":\"I set the token to eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ1In0.xyz for you\"}]}\n\n" +
		"data: {\"type\":\"session.status_idle\"}\n\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, sse)
	}))
	defer server.Close()

	p := newTestProvider(t, server)
	var received []agents.ProviderEvent
	require.NoError(t, p.StreamEvents(context.Background(), "sesn_abc", func(e agents.ProviderEvent) error {
		received = append(received, e)
		return nil
	}))
	require.GreaterOrEqual(t, len(received), 1)
	assert.NotContains(t, received[0].Text, "eyJhbGciOiJIUzI1NiJ9")
	assert.Contains(t, received[0].Text, "<redacted>")
}

func TestStreamEvents_ToolInputFallsBackToJSON(t *testing.T) {
	const sse = "data: {\"id\":\"e1\",\"type\":\"agent.tool_use\",\"name\":\"search\",\"input\":{\"query\":\"weather\",\"limit\":5}}\n\n" +
		"data: {\"type\":\"session.status_idle\"}\n\n"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, sse)
	}))
	defer server.Close()

	p := newTestProvider(t, server)
	var received []agents.ProviderEvent
	require.NoError(t, p.StreamEvents(context.Background(), "sesn_abc", func(e agents.ProviderEvent) error {
		received = append(received, e)
		return nil
	}))
	require.GreaterOrEqual(t, len(received), 1)
	assert.Contains(t, received[0].ToolInput, "weather")
	assert.Contains(t, received[0].ToolInput, "limit")
}

func TestStreamEvents_SessionFailed(t *testing.T) {
	const sse = "data: {\"type\":\"session.error\",\"error\":{\"message\":\"boom\"}}\n\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, sse)
	}))
	defer server.Close()

	p := newTestProvider(t, server)
	var received []agents.ProviderEvent
	require.NoError(t, p.StreamEvents(context.Background(), "sesn_abc", func(e agents.ProviderEvent) error {
		received = append(received, e)
		return nil
	}))
	require.Len(t, received, 1)
	assert.Equal(t, agents.ProviderEventSessionFailed, received[0].Type)
	assert.Equal(t, "boom", received[0].ErrorMessage)
}

func TestStreamEvents_SkipsMalformed(t *testing.T) {
	const sse = "data: not-json\n\n" +
		"data: {\"type\":\"session.status_idle\"}\n\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, sse)
	}))
	defer server.Close()

	p := newTestProvider(t, server)
	count := 0
	require.NoError(t, p.StreamEvents(context.Background(), "sesn_abc", func(e agents.ProviderEvent) error {
		count++
		return nil
	}))
	assert.Equal(t, 1, count)
}

func TestStreamEvents_StopsOnCallbackError(t *testing.T) {
	const sse = "data: {\"id\":\"e1\",\"type\":\"agent.message\",\"content\":[{\"type\":\"text\",\"text\":\"Hi\"}]}\n\n" +
		"data: {\"type\":\"session.status_idle\"}\n\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, sse)
	}))
	defer server.Close()

	p := newTestProvider(t, server)
	count := 0
	err := p.StreamEvents(context.Background(), "sesn_abc", func(e agents.ProviderEvent) error {
		count++
		return io.ErrUnexpectedEOF
	})
	require.Error(t, err)
	assert.Equal(t, 1, count)
}

func TestStreamEvents_PropagatesHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = io.WriteString(w, "down")
	}))
	defer server.Close()

	p := newTestProvider(t, server)
	err := p.StreamEvents(context.Background(), "sesn_abc", func(agents.ProviderEvent) error { return nil })
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "503"))
}
