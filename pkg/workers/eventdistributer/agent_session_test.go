package eventdistributer_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/public/ws"
	"github.com/superplanehq/superplane/pkg/workers/eventdistributer"
)

func TestAgentSessionWebsocketTopic_IsStable(t *testing.T) {
	assert.Equal(t, "agent-session:abc", eventdistributer.AgentSessionWebsocketTopic("abc"))
}

func TestHandleAgentSessionEvent_RejectsMalformedPayload(t *testing.T) {
	hub := ws.NewHub()
	hub.Run()
	require.Error(t, eventdistributer.HandleAgentSessionEvent([]byte("not json"), hub))
}

func TestHandleAgentSessionEvent_RejectsMissingSessionID(t *testing.T) {
	hub := ws.NewHub()
	hub.Run()
	payload, err := json.Marshal(messages.AgentSessionEventMessage{Event: "assistant_message"})
	require.NoError(t, err)
	require.Error(t, eventdistributer.HandleAgentSessionEvent(payload, hub))
}

func TestHandleAgentSessionEvent_BroadcastsToTopicSubscribers(t *testing.T) {
	hub := ws.NewHub()
	hub.Run()

	sessionID := "11111111-1111-1111-1111-111111111111"
	topic := eventdistributer.AgentSessionWebsocketTopic(sessionID)

	// Stand up a tiny HTTP server that hands the upgraded connection to
	// the hub on the chosen topic. This is the lightest way to verify the
	// broadcast end-to-end without rewriting the hub.
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade failed: %v", err)
			return
		}
		hub.NewClient(conn, topic)
	}))
	defer server.Close()

	u, err := url.Parse(server.URL)
	require.NoError(t, err)
	wsURL := "ws://" + u.Host
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	// Give the hub time to register the client before we publish.
	require.Eventually(t, func() bool { return strings.TrimSpace("ready") != "" }, time.Second, time.Millisecond)

	payload, err := json.Marshal(messages.AgentSessionEventMessage{
		SessionID: sessionID,
		Event:     "assistant_message",
		Message: &messages.AgentMessage{
			ID:      "msg-1",
			Role:    "assistant",
			Content: "hello world",
		},
	})
	require.NoError(t, err)

	// The hub registers asynchronously, so retry the publish a few times
	// until the subscriber is wired up.
	var got messages.AgentSessionEventMessage
	require.Eventually(t, func() bool {
		_ = eventdistributer.HandleAgentSessionEvent(payload, hub)
		conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		_, data, err := conn.ReadMessage()
		if err != nil {
			return false
		}
		return json.Unmarshal(data, &got) == nil
	}, 2*time.Second, 50*time.Millisecond)

	assert.Equal(t, sessionID, got.SessionID)
	assert.Equal(t, "assistant_message", got.Event)
	require.NotNil(t, got.Message)
	assert.Equal(t, "hello world", got.Message.Content)
}
