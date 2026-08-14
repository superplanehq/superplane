package eventdistributer_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/public/ws"
	"github.com/superplanehq/superplane/pkg/workers/eventdistributer"
	"google.golang.org/protobuf/proto"
)

func TestHandleFactoryAppUpdated_BroadcastsToTopicSubscribers(t *testing.T) {
	hub := ws.NewHub()
	hub.Run()

	factoryID := "11111111-1111-1111-1111-111111111111"
	appID := "33333333-3333-3333-3333-333333333333"
	topic := eventdistributer.FactoryWebsocketTopic(factoryID)

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

	require.Eventually(t, func() bool {
		return hub.WorkflowSubscriberCount(topic) == 1
	}, 2*time.Second, 5*time.Millisecond, "subscriber never registered on hub")

	payload, err := proto.Marshal(&pb.FactoryAppUpdatedMessage{
		FactoryId: factoryID,
		AppId:     appID,
		Reason:    "app.created",
	})
	require.NoError(t, err)
	require.NoError(t, eventdistributer.HandleFactoryAppUpdated(payload, hub))

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := conn.ReadMessage()
	require.NoError(t, err)

	var got struct {
		Event   string `json:"event"`
		Payload struct {
			FactoryID string `json:"factoryId"`
			AppID     string `json:"appId"`
			Reason    string `json:"reason"`
		} `json:"payload"`
	}
	require.NoError(t, json.Unmarshal(data, &got))
	require.Equal(t, eventdistributer.FactoryAppUpdatedEvent, got.Event)
	require.Equal(t, factoryID, got.Payload.FactoryID)
	require.Equal(t, appID, got.Payload.AppID)
	require.Equal(t, "app.created", got.Payload.Reason)
}

func TestHandleFactoryUpdated_BroadcastsToTopicSubscribers(t *testing.T) {
	hub := ws.NewHub()
	hub.Run()

	factoryID := "11111111-1111-1111-1111-111111111111"
	topic := eventdistributer.FactoryWebsocketTopic(factoryID)

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

	require.Eventually(t, func() bool {
		return hub.WorkflowSubscriberCount(topic) == 1
	}, 2*time.Second, 5*time.Millisecond, "subscriber never registered on hub")

	payload, err := proto.Marshal(&pb.FactoryUpdatedMessage{
		FactoryId: factoryID,
		Reason:    "line.updated",
	})
	require.NoError(t, err)
	require.NoError(t, eventdistributer.HandleFactoryUpdated(payload, hub))

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := conn.ReadMessage()
	require.NoError(t, err)

	var got struct {
		Event   string `json:"event"`
		Payload struct {
			FactoryID string `json:"factoryId"`
			Reason    string `json:"reason"`
		} `json:"payload"`
	}
	require.NoError(t, json.Unmarshal(data, &got))
	require.Equal(t, eventdistributer.FactoryUpdatedEvent, got.Event)
	require.Equal(t, factoryID, got.Payload.FactoryID)
	require.Equal(t, "line.updated", got.Payload.Reason)
}
