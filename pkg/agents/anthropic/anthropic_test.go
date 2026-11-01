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

func TestDefaultAgentPrompt_IsEmbedded(t *testing.T) {
	prompt := DefaultAgentPrompt()
	assert.Contains(t, prompt, "You are a SuperPlane app expert")
	assert.Contains(t, prompt, "## App Update Rules")
}

func defaultAgentToolsJSON(t *testing.T) string {
	t.Helper()
	tools, err := json.Marshal(defaultAgentTools())
	require.NoError(t, err)
	return string(tools)
}

func TestSyncAgentPrompt_SkipsUpdateWhenCurrent(t *testing.T) {
	postCount := 0
	tools := defaultAgentToolsJSON(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/agents/agent-123", r.URL.Path)
		assert.Equal(t, "test-key", r.Header.Get("x-api-key"))

		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"system":"current prompt\n\n","version":7,"tools":` + tools + `}`))
			return
		}

		postCount++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	err := SyncAgentPrompt(context.Background(), Config{
		APIKey:  "test-key",
		AgentID: "agent-123",
		BaseURL: server.URL,
	}, "current prompt\n")
	require.NoError(t, err)
	assert.Equal(t, 0, postCount)
}

func TestSyncAgentPrompt_UpdatesWhenCurrentAndToolsOmittedOnRead(t *testing.T) {
	postCount := 0
	tools := defaultAgentToolsJSON(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/agents/agent-123", r.URL.Path)

		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"system":"current prompt\n","version":7}`))
			return
		}

		postCount++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"system":"current prompt\n","version":8,"tools":` + tools + `}`))
	}))
	defer server.Close()

	err := SyncAgentPrompt(context.Background(), Config{
		APIKey:  "test-key",
		AgentID: "agent-123",
		BaseURL: server.URL,
	}, "current prompt\n")
	require.NoError(t, err)
	assert.Equal(t, 1, postCount)
}

func TestSyncAgentPrompt_SkipsUpdateWhenCurrentToolsHaveDifferentOrder(t *testing.T) {
	postCount := 0
	reversedTools, err := json.Marshal([]map[string]any{
		defaultAgentTools()[2],
		defaultAgentTools()[1],
		defaultAgentTools()[0],
	})
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/agents/agent-123", r.URL.Path)

		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"system":"current prompt\n","version":7,"tools":` + string(reversedTools) + `}`))
			return
		}

		postCount++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	err = SyncAgentPrompt(context.Background(), Config{
		APIKey:  "test-key",
		AgentID: "agent-123",
		BaseURL: server.URL,
	}, "current prompt\n")
	require.NoError(t, err)
	assert.Equal(t, 0, postCount)
}

func TestToolsEqual_IgnoresProviderMetadata(t *testing.T) {
	currentTools := defaultAgentTools()
	currentTools[1]["provider_created_at"] = "2026-06-10T00:00:00Z"
	currentTools[1]["cache_control"] = nil
	currentTools[1]["input_schema"].(map[string]any)["provider_schema_id"] = "schema_123"

	current, err := json.Marshal(currentTools)
	require.NoError(t, err)

	assert.True(t, toolsEqual(current, defaultAgentTools()))
}

func TestToolsEqual_FailsWhenExpectedToolFieldsDiffer(t *testing.T) {
	currentTools := defaultAgentTools()
	currentTools[1]["description"] = "different"

	current, err := json.Marshal(currentTools)
	require.NoError(t, err)

	assert.False(t, toolsEqual(current, defaultAgentTools()))
}

func TestSyncAgentPrompt_UpdatesWhenDifferent(t *testing.T) {
	var capturedBody map[string]any
	tools := defaultAgentToolsJSON(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/agents/agent-123", r.URL.Path)

		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"system":"old prompt","version":9}`))
		case http.MethodPost:
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &capturedBody)

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"system":"new prompt","version":10,"tools":` + tools + `}`))
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	err := SyncAgentPrompt(context.Background(), Config{
		APIKey:  "test-key",
		AgentID: "agent-123",
		BaseURL: server.URL,
	}, "new prompt")
	require.NoError(t, err)
	assert.Equal(t, "new prompt", capturedBody["system"])
	assert.Equal(t, float64(9), capturedBody["version"])
	require.NotEmpty(t, capturedBody["tools"])
}

func TestSyncAgentPrompt_AcceptsUpdatedPromptWithNormalizedTrailingNewlines(t *testing.T) {
	tools := defaultAgentToolsJSON(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"system":"old prompt","version":9}`))
		case http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"system":"new prompt","version":10,"tools":` + tools + `}`))
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	err := SyncAgentPrompt(context.Background(), Config{
		APIKey:  "test-key",
		AgentID: "agent-123",
		BaseURL: server.URL,
	}, "new prompt\n\n")
	require.NoError(t, err)
}

func TestSyncAgentPrompt_ErrorsWhenUpdatedToolsDiffer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"system":"old prompt","version":9}`))
		case http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"system":"new prompt","version":10,"tools":[{"type":"agent_toolset_20260401"}]}`))
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	err := SyncAgentPrompt(context.Background(), Config{
		APIKey:  "test-key",
		AgentID: "agent-123",
		BaseURL: server.URL,
	}, "new prompt")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "provider returned different tools")
}

func TestSyncAgentPrompt_ErrorsWhenUpdatedToolsAreOmitted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"system":"old prompt","version":9}`))
		case http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"system":"new prompt","version":10}`))
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	err := SyncAgentPrompt(context.Background(), Config{
		APIKey:  "test-key",
		AgentID: "agent-123",
		BaseURL: server.URL,
	}, "new prompt")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "provider returned different tools")
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

func TestSendMessage_IncludesImageContentBlocks(t *testing.T) {
	var capturedBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &capturedBody)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{}"))
	}))
	defer server.Close()

	p := newTestProvider(t, server)
	err := p.SendMessage(context.Background(), "sesn_abc", "look at this", agents.SendMessageOptions{
		Images: []agents.MessageImage{{MediaType: "image/png", Data: "aGVsbG8="}},
	})
	require.NoError(t, err)

	events := capturedBody["events"].([]any)
	content := events[0].(map[string]any)["content"].([]any)
	require.Len(t, content, 2)
	assert.Equal(t, "look at this", content[0].(map[string]any)["text"].(string))

	imageBlock := content[1].(map[string]any)
	assert.Equal(t, "image", imageBlock["type"].(string))
	source := imageBlock["source"].(map[string]any)
	assert.Equal(t, "base64", source["type"].(string))
	assert.Equal(t, "image/png", source["media_type"].(string))
	assert.Equal(t, "aGVsbG8=", source["data"].(string))
}

func TestSendMessage_RequiresSessionID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("server should not be hit")
	}))
	defer server.Close()
	p := newTestProvider(t, server)
	require.Error(t, p.SendMessage(context.Background(), "", "hi", agents.SendMessageOptions{}))
}

func TestSendMessage_MapsWaitingOnToolResultsToBusySession(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"type":"error","error":{"type":"invalid_request_error","message":"Invalid user.message event at events[0]: waiting on responses to events [sevt_012]; only user.tool_result may be sent"}}`))
	}))
	defer server.Close()

	p := newTestProvider(t, server)
	err := p.SendMessage(context.Background(), "sesn_abc", "hi", agents.SendMessageOptions{})
	require.Error(t, err)
	assert.ErrorIs(t, err, agents.ErrSessionBusy)
}

func TestSendMessage_MapsArchivedSessionToUnavailableSession(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"type":"error","error":{"type":"not_found_error","message":"session has been archived"}}`))
	}))
	defer server.Close()

	p := newTestProvider(t, server)
	err := p.SendMessage(context.Background(), "sesn_archived", "hi", agents.SendMessageOptions{})
	require.Error(t, err)
	assert.ErrorIs(t, err, agents.ErrProviderSessionUnavailable)
}

func TestSendMessage_MapsDeletedSessionBadRequestToUnavailableSession(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"type":"error","error":{"type":"invalid_request_error","message":"Session sesn_deleted does not exist"}}`))
	}))
	defer server.Close()

	p := newTestProvider(t, server)
	err := p.SendMessage(context.Background(), "sesn_deleted", "hi", agents.SendMessageOptions{})
	require.Error(t, err)
	assert.ErrorIs(t, err, agents.ErrProviderSessionUnavailable)
}

func TestSendMessage_MapsPayloadTooLargeToInvalidRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		_, _ = w.Write([]byte(`{"type":"error","error":{"type":"invalid_request_error","message":"request payload is too large"}}`))
	}))
	defer server.Close()

	p := newTestProvider(t, server)
	err := p.SendMessage(context.Background(), "sesn_live", "hi", agents.SendMessageOptions{})
	require.Error(t, err)
	assert.ErrorIs(t, err, agents.ErrInvalidRequest)
}

func TestSendMessage_MapsImageSizeLimitErrorToInvalidRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"type":"error","error":{"type":"invalid_request_error","message":"image content exceeds the model limit"}}`))
	}))
	defer server.Close()

	p := newTestProvider(t, server)
	err := p.SendMessage(context.Background(), "sesn_live", "hi", agents.SendMessageOptions{
		Images: []agents.MessageImage{{MediaType: "image/png", Data: "aGVsbG8="}},
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, agents.ErrInvalidRequest)
}

func TestSendMessage_DoesNotMapUnrelatedImageErrorToInvalidRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"type":"error","error":{"type":"invalid_request_error","message":"image service temporarily unavailable"}}`))
	}))
	defer server.Close()

	p := newTestProvider(t, server)
	err := p.SendMessage(context.Background(), "sesn_live", "hi", agents.SendMessageOptions{
		Images: []agents.MessageImage{{MediaType: "image/png", Data: "aGVsbG8="}},
	})
	require.Error(t, err)
	assert.NotErrorIs(t, err, agents.ErrInvalidRequest)
	assert.Contains(t, err.Error(), "anthropic: send message")
}

func TestSendMessage_DoesNotTreatBadRequestNotFoundMessageAsUnavailableSession(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"type":"error","error":{"type":"invalid_request_error","message":"tool target not found"}}`))
	}))
	defer server.Close()

	p := newTestProvider(t, server)
	err := p.SendMessage(context.Background(), "sesn_live", "hi", agents.SendMessageOptions{})
	require.Error(t, err)
	assert.NotErrorIs(t, err, agents.ErrProviderSessionUnavailable)
	assert.Contains(t, err.Error(), "anthropic: send message")
}

func TestSendMessage_DoesNotTreatUnrelatedSessionNotFoundMessageAsUnavailableSession(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"type":"error","error":{"type":"invalid_request_error","message":"validation failed: session variable not found"}}`))
	}))
	defer server.Close()

	p := newTestProvider(t, server)
	err := p.SendMessage(context.Background(), "sesn_live", "hi", agents.SendMessageOptions{})
	require.Error(t, err)
	assert.NotErrorIs(t, err, agents.ErrProviderSessionUnavailable)
	assert.Contains(t, err.Error(), "anthropic: send message")
}

func TestDefineOutcome_PrependsPreambleToDescription(t *testing.T) {
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
	err := p.DefineOutcome(context.Background(), "sesn_abc", agents.DefineOutcomeOptions{
		Description:     "Build the workflow",
		Rubric:          "- Done",
		MaxIterations:   3,
		ContextPreamble: "[ctx]",
	})
	require.NoError(t, err)

	events := capturedBody["events"].([]any)
	require.Len(t, events, 1)
	event := events[0].(map[string]any)
	assert.Equal(t, "[ctx]\n\nBuild the workflow", event["description"])
}

func TestSendCustomToolResults_SendsUserCustomToolResultEvents(t *testing.T) {
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
	err := p.SendCustomToolResults(context.Background(), "sesn_abc", []agents.CustomToolResult{
		{CustomToolUseID: "evt_1", Content: `{"ok":true}`},
	})
	require.NoError(t, err)

	events := capturedBody["events"].([]any)
	require.Len(t, events, 1)
	event := events[0].(map[string]any)
	assert.Equal(t, "user.custom_tool_result", event["type"])
	assert.Equal(t, "evt_1", event["custom_tool_use_id"])
	content := event["content"].([]any)
	assert.Equal(t, `{"ok":true}`, content[0].(map[string]any)["text"])
}

func TestDeleteSession_SendsCorrectRequest(t *testing.T) {
	var capturedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/sessions/sesn_abc", r.URL.Path)
		capturedHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{}"))
	}))
	defer server.Close()

	p := newTestProvider(t, server)
	require.NoError(t, p.DeleteSession(context.Background(), "sesn_abc"))
	assert.Equal(t, "test-key", capturedHeaders.Get("x-api-key"))
	assert.Equal(t, managedAgentsBeta, capturedHeaders.Get("anthropic-beta"))
}

func TestDeleteSession_PropagatesAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"error":"running"}`))
	}))
	defer server.Close()

	p := newTestProvider(t, server)
	err := p.DeleteSession(context.Background(), "sesn_abc")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "409")
}

func TestArchiveSession_SendsCorrectRequest(t *testing.T) {
	var capturedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/sessions/sesn_abc/archive", r.URL.Path)
		capturedHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{}"))
	}))
	defer server.Close()

	p := newTestProvider(t, server)
	require.NoError(t, p.ArchiveSession(context.Background(), "sesn_abc"))
	assert.Equal(t, "test-key", capturedHeaders.Get("x-api-key"))
	assert.Equal(t, managedAgentsBeta, capturedHeaders.Get("anthropic-beta"))
}

func TestArchiveSession_MapsUnavailableProviderSession(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"message":"session has been archived"}}`))
	}))
	defer server.Close()

	p := newTestProvider(t, server)
	err := p.ArchiveSession(context.Background(), "sesn_abc")
	require.Error(t, err)
	assert.ErrorIs(t, err, agents.ErrProviderSessionUnavailable)
}

func TestToolSchemaRevision_ReturnsStableRevision(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("schema revision must not call provider API")
	}))
	defer server.Close()

	p := newTestProvider(t, server)
	first := p.ToolSchemaRevision()
	second := p.ToolSchemaRevision()

	assert.Equal(t, first, second)
	assert.Contains(t, first, "agent-tools-v1.2.0:")
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
		switch r.URL.Path {
		case "/sessions/sesn_abc/events/stream":
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = io.WriteString(w, sse)
		case "/sessions/sesn_abc":
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"id":"sesn_abc","status":"idle"}`)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
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

func TestStreamEvents_TreatsUnknownIdleStopReasonAsTurnCompleted(t *testing.T) {
	const sse = "data: {\"type\":\"session.status_idle\",\"stop_reason\":{\"type\":\"new_provider_reason\"}}\n\n"

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
	assert.Equal(t, agents.ProviderEventTurnCompleted, received[0].Type)
}

func TestStreamEvents_IgnoresUnknownMidTurnEvents(t *testing.T) {
	const sse = "data: {\"id\":\"think-1\",\"type\":\"agent.thinking\"}\n\n" +
		"data: {\"id\":\"msg-1\",\"type\":\"agent.message\",\"content\":[{\"type\":\"text\",\"text\":\"after thinking\"}]}\n\n" +
		"data: {\"type\":\"session.status_idle\"}\n\n"

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

	require.Len(t, received, 2)
	assert.Equal(t, agents.ProviderEventAssistantMessage, received[0].Type)
	assert.Equal(t, "after thinking", received[0].Text)
	assert.Equal(t, agents.ProviderEventTurnCompleted, received[1].Type)
}

func TestStreamEvents_StopsWhenCallbackRequestsCustomToolHandling(t *testing.T) {
	const sse = "data: {\"id\":\"evt_custom\",\"type\":\"agent.custom_tool_use\",\"name\":\"superplane_app\",\"input\":{\"action\":\"read\"}}\n\n" +
		"data: {\"type\":\"session.status_idle\",\"stop_reason\":{\"type\":\"requires_action\",\"event_ids\":[\"evt_custom\"]}}\n\n" +
		"data: {\"id\":\"message_after_pause\",\"type\":\"agent.message\",\"content\":[{\"type\":\"text\",\"text\":\"after pause\"}]}\n\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, sse)
	}))
	defer server.Close()

	p := newTestProvider(t, server)
	var received []agents.ProviderEvent
	err := p.StreamEvents(context.Background(), "sesn_abc", func(e agents.ProviderEvent) error {
		received = append(received, e)
		if e.Type == agents.ProviderEventCustomToolResultsRequired {
			return io.ErrUnexpectedEOF
		}
		return nil
	})

	require.ErrorIs(t, err, io.ErrUnexpectedEOF)
	require.Len(t, received, 2)
	assert.Equal(t, agents.ProviderEventCustomToolUseStarted, received[0].Type)
	assert.Equal(t, agents.ProviderEventCustomToolResultsRequired, received[1].Type)
	assert.Equal(t, []string{"evt_custom"}, received[1].CustomToolEventIDs)
}

func TestStreamEvents_PairsToolUseAndResultByToolUseID(t *testing.T) {
	// Tool use and its matching result have different event ids but share
	// the same tool_use_id. The mapper must surface tool_use_id as the
	// ProviderEventID so the worker upserts both into one DB row.
	const sse = "data: {\"id\":\"evt_A\",\"type\":\"agent.tool_use\",\"tool_use_id\":\"toolu_1\",\"name\":\"bash\",\"input\":{\"command\":\"ls\"}}\n\n" +
		"data: {\"id\":\"evt_B\",\"type\":\"agent.tool_result\",\"tool_use_id\":\"toolu_1\",\"name\":\"bash\"}\n\n" +
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
	require.Len(t, received, 3)
	assert.Equal(t, "toolu_1", received[0].ProviderEventID, "tool_use must key on tool_use_id")
	assert.Equal(t, "toolu_1", received[1].ProviderEventID, "tool_result must key on the same tool_use_id")
}

func TestStreamEvents_KeysCustomToolUseByEventID(t *testing.T) {
	// Managed Agents require user.custom_tool_result.custom_tool_use_id to
	// reference the agent.custom_tool_use event id from requires_action.
	const sse = "data: {\"id\":\"evt_custom\",\"type\":\"agent.custom_tool_use\",\"tool_use_id\":\"toolu_custom\",\"name\":\"superplane_app\",\"input\":{\"action\":\"read\"}}\n\n" +
		"data: {\"type\":\"session.status_idle\",\"stop_reason\":{\"type\":\"requires_action\",\"event_ids\":[\"evt_custom\"]}}\n\n"

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

	require.Len(t, received, 2)
	assert.Equal(t, "evt_custom", received[0].ProviderEventID)
	assert.Equal(t, "evt_custom", received[0].ToolCallID)
	require.NotNil(t, received[0].CustomToolUse)
	assert.Equal(t, "evt_custom", received[0].CustomToolUse.ID)
	assert.Equal(t, []string{"evt_custom"}, received[1].CustomToolEventIDs)
}

func TestStreamEvents_MapsAlternateToolNameField(t *testing.T) {
	const sse = "data: {\"id\":\"evt_A\",\"type\":\"agent.tool_use\",\"tool_use_id\":\"toolu_1\",\"tool_name\":\"read\",\"input\":{\"file_path\":\"/tmp/spec.md\"}}\n\n" +
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
	require.Len(t, received, 2)
	assert.Equal(t, "read", received[0].ToolName)
	assert.Contains(t, received[0].ToolInput, "/tmp/spec.md")
}

func TestStreamEvents_MapsCustomToolUseAndRequiresAction(t *testing.T) {
	const sse = "data: {\"id\":\"evt_custom\",\"type\":\"agent.custom_tool_use\",\"name\":\"superplane_app\",\"input\":{\"action\":\"read\"}}\n\n" +
		"data: {\"type\":\"session.status_idle\",\"stop_reason\":{\"type\":\"requires_action\",\"event_ids\":[\"evt_custom\"]}}\n\n" +
		"data: {\"id\":\"message_after_pause\",\"type\":\"agent.message\",\"content\":[{\"type\":\"text\",\"text\":\"after pause\"}]}\n\n"

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

	require.Len(t, received, 3)
	assert.Equal(t, agents.ProviderEventCustomToolUseStarted, received[0].Type)
	require.NotNil(t, received[0].CustomToolUse)
	assert.Equal(t, "evt_custom", received[0].CustomToolUse.ID)
	assert.Equal(t, agents.ProviderEventCustomToolResultsRequired, received[1].Type)
	assert.Equal(t, []string{"evt_custom"}, received[1].CustomToolEventIDs)
	assert.Equal(t, agents.ProviderEventAssistantMessage, received[2].Type)
	assert.Equal(t, "after pause", received[2].Text)
}

func TestStreamEvents_ParsesCustomToolInputLargerThanDefaultScannerLimit(t *testing.T) {
	consoleYAML := strings.Repeat("x", 70*1024)
	input := map[string]any{
		"action":       "patch_staging",
		"version_id":   "draft-version",
		"console_yaml": consoleYAML,
	}
	event := map[string]any{
		"id":    "evt_custom",
		"type":  "agent.custom_tool_use",
		"name":  "superplane_app",
		"input": input,
	}
	payload, err := json.Marshal(event)
	require.NoError(t, err)
	require.Greater(t, len(payload), 64*1024)

	sse := "data: " + string(payload) + "\n\n" +
		"data: {\"type\":\"session.status_idle\",\"stop_reason\":{\"type\":\"requires_action\",\"event_ids\":[\"evt_custom\"]}}\n\n"

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

	require.Len(t, received, 2)
	assert.Equal(t, agents.ProviderEventCustomToolUseStarted, received[0].Type)
	require.NotNil(t, received[0].CustomToolUse)
	assert.Equal(t, "evt_custom", received[0].CustomToolUse.ID)
	assert.Contains(t, received[0].CustomToolUse.Input, consoleYAML)
	assert.Equal(t, agents.ProviderEventCustomToolResultsRequired, received[1].Type)
}

func TestStreamEvents_EndTurnStopReasonCompletesTurn(t *testing.T) {
	const sse = "data: {\"type\":\"session.status_idle\",\"stop_reason\":{\"type\":\"end_turn\"}}\n\n"

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

	require.Len(t, received, 1)
	assert.Equal(t, agents.ProviderEventTurnCompleted, received[0].Type)
}

func TestStreamEvents_MapsTokenUsageOnCompletedTurn(t *testing.T) {
	const sse = "data: {\"type\":\"session.status_idle\",\"model\":\"claude-sonnet-4-5\",\"usage\":{\"input_tokens\":10,\"output_tokens\":5,\"cache_read_input_tokens\":3,\"cache_creation_input_tokens\":2}}\n\n"

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

	require.Len(t, received, 1)
	assert.Equal(t, agents.ProviderEventTurnCompleted, received[0].Type)
	assert.Equal(t, "claude-sonnet-4-5", received[0].Model)
	require.NotNil(t, received[0].Usage)
	assert.Equal(t, int64(10), received[0].Usage.InputTokens)
	assert.Equal(t, int64(5), received[0].Usage.OutputTokens)
	assert.Equal(t, int64(3), received[0].Usage.CacheReadTokens)
	assert.Equal(t, int64(2), received[0].Usage.CacheWriteTokens)
	assert.Equal(t, int64(20), received[0].Usage.TotalTokens)
}

func TestStreamEvents_DoesNotFetchSessionUsageWhenIdleEventHasNoUsage(t *testing.T) {
	var sessionFetched bool
	const sse = "data: {\"type\":\"session.status_idle\",\"model\":\"claude-sonnet-4-5\",\"stop_reason\":{\"type\":\"end_turn\"}}\n\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/sessions/sesn_abc/events/stream":
			_, _ = io.WriteString(w, sse)
		case r.Method == http.MethodGet && r.URL.Path == "/sessions/sesn_abc":
			sessionFetched = true
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"id":"sesn_abc","status":"idle","usage":{"input_tokens":6,"output_tokens":6,"cache_read_input_tokens":3,"cache_creation":{"ephemeral_5m_input_tokens":20,"ephemeral_1h_input_tokens":7}}}`)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	p := newTestProvider(t, server)
	var received []agents.ProviderEvent
	require.NoError(t, p.StreamEvents(context.Background(), "sesn_abc", func(e agents.ProviderEvent) error {
		received = append(received, e)
		return nil
	}))

	require.False(t, sessionFetched)
	require.Len(t, received, 1)
	assert.Equal(t, agents.ProviderEventTurnCompleted, received[0].Type)
	assert.Nil(t, received[0].Usage)
}

func TestRetrieveSessionUsage_MapsCompletedSessionUsage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/sessions/sesn_abc", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":"sesn_abc","status":"idle","usage":{"input_tokens":6,"output_tokens":6,"cache_read_input_tokens":3,"cache_creation":{"ephemeral_5m_input_tokens":20,"ephemeral_1h_input_tokens":7}}}`)
	}))
	defer server.Close()

	p := newTestProvider(t, server)
	usage, err := p.RetrieveSessionUsage(context.Background(), "sesn_abc")
	require.NoError(t, err)
	require.NotNil(t, usage)
	assert.Equal(t, int64(6), usage.InputTokens)
	assert.Equal(t, int64(6), usage.OutputTokens)
	assert.Equal(t, int64(3), usage.CacheReadTokens)
	assert.Equal(t, int64(27), usage.CacheWriteTokens)
	assert.Equal(t, int64(42), usage.TotalTokens)
}

func TestStreamEvents_FallsBackToEventIDWhenToolUseIDMissing(t *testing.T) {
	const sse = "data: {\"id\":\"evt_fallback\",\"type\":\"agent.tool_use\",\"name\":\"search\"}\n\n" +
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
	assert.Equal(t, "evt_fallback", received[0].ProviderEventID)
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
	const sse = "data: {\"type\":\"session.status_terminated\",\"error\":{\"message\":\"boom\"}}\n\n"

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

func TestStreamEvents_SessionErrorDoesNotStopStream(t *testing.T) {
	const sse = "data: {\"type\":\"session.error\",\"error\":{\"message\":\"An internal service error occurred\"}}\n\n" +
		"data: {\"id\":\"e1\",\"type\":\"agent.message\",\"content\":[{\"type\":\"text\",\"text\":\"Recovered\"}]}\n\n" +
		"data: {\"type\":\"session.status_idle\"}\n\n"

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

	// session.error surfaces as a non-terminal notice; the stream keeps going
	// and the recovered assistant message + turn completion still arrive.
	require.Len(t, received, 3)
	assert.Equal(t, agents.ProviderEventSessionNotice, received[0].Type)
	assert.Equal(t, "An internal service error occurred", received[0].ErrorMessage)
	assert.Equal(t, agents.ProviderEventAssistantMessage, received[1].Type)
	assert.Equal(t, "Recovered", received[1].Text)
	assert.Equal(t, agents.ProviderEventTurnCompleted, received[2].Type)
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
