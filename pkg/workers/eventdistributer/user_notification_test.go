package eventdistributer_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/public/ws"
	"github.com/superplanehq/superplane/pkg/workers/eventdistributer"
)

func TestUserNotificationWebsocketTopic_IsStable(t *testing.T) {
	assert.Equal(t, "user:abc", eventdistributer.UserNotificationWebsocketTopic("abc"))
}

func TestHandleUserNotification_RejectsMalformedPayload(t *testing.T) {
	hub := ws.NewHub()
	hub.Run()
	require.Error(t, eventdistributer.HandleUserNotification([]byte("not json"), hub))
}

func TestHandleUserNotification_RejectsMissingUserID(t *testing.T) {
	hub := ws.NewHub()
	hub.Run()
	payload, err := json.Marshal(messages.UserNotificationMessage{OrderID: "order-1"})
	require.NoError(t, err)
	require.Error(t, eventdistributer.HandleUserNotification(payload, hub))
}

func TestHandleUserNotification_BroadcastsToUserTopic(t *testing.T) {
	hub := ws.NewHub()
	hub.Run()

	userID := "11111111-1111-1111-1111-111111111111"
	otherUserID := "22222222-2222-2222-2222-222222222222"
	topic := eventdistributer.UserNotificationWebsocketTopic(userID)
	otherTopic := eventdistributer.UserNotificationWebsocketTopic(otherUserID)

	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade failed: %v", err)
			return
		}
		switch r.URL.Path {
		case "/other":
			hub.NewClient(conn, otherTopic)
		default:
			hub.NewClient(conn, topic)
		}
	}))
	defer server.Close()

	u, err := url.Parse(server.URL)
	require.NoError(t, err)
	wsURL := "ws://" + u.Host
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	otherConn, _, err := websocket.DefaultDialer.Dial(wsURL+"/other", nil)
	require.NoError(t, err)
	defer otherConn.Close()

	require.Eventually(t, func() bool {
		return hub.WorkflowSubscriberCount(topic) == 1 && hub.WorkflowSubscriberCount(otherTopic) == 1
	}, 2*time.Second, 5*time.Millisecond, "subscribers never registered on hub")

	payload, err := json.Marshal(messages.UserNotificationMessage{
		UserID:         userID,
		OrganizationID: "org-1",
		FactoryID:      "factory-1",
		FactoryKey:     "SP",
		OrderID:        "order-1",
		OrderKey:       "SP-1",
		EventType:      "work_order_comment_owned",
		Title:          "[SP-1] New comment",
		Body:           "Ada commented on SP-1.",
		URLPath:        "/org-1/workspaces/SP/work-order/1",
	})
	require.NoError(t, err)
	require.NoError(t, eventdistributer.HandleUserNotification(payload, hub))

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := conn.ReadMessage()
	require.NoError(t, err)

	var got struct {
		Event   string                           `json:"event"`
		Payload messages.UserNotificationMessage `json:"payload"`
	}
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, eventdistributer.UserNotificationEvent, got.Event)
	assert.Equal(t, userID, got.Payload.UserID)
	assert.Equal(t, "SP-1", got.Payload.OrderKey)
	assert.Equal(t, "[SP-1] New comment", got.Payload.Title)
	assert.Equal(t, "Ada commented on SP-1.", got.Payload.Body)
	assert.Equal(t, "/org-1/workspaces/SP/work-order/1", got.Payload.URLPath)

	otherConn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	_, _, err = otherConn.ReadMessage()
	require.Error(t, err)
}
