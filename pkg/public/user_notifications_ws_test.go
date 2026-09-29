package public

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/workers/eventdistributer"
	"github.com/superplanehq/superplane/test/support"
)

func TestUserNotificationsWebSocketRequiresAuth(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	server, _, _ := setupTestServer(r, t)

	response := execRequest(server, requestParams{
		method: http.MethodGet,
		path:   "/ws/users/notifications?organization_id=" + r.Organization.ID.String(),
	})

	assert.Equal(t, http.StatusUnauthorized, response.Code)
}

func TestUserNotificationsWebSocketSubscribesToCallerTopicOnly(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	server, _, token := setupTestServer(r, t)
	server.WebsocketHub().Run()

	httpServer := httptest.NewServer(server.Router)
	defer httpServer.Close()

	u, err := url.Parse(httpServer.URL)
	require.NoError(t, err)
	wsURL := "ws://" + u.Host + "/ws/users/notifications?organization_id=" + r.Organization.ID.String()

	header := http.Header{}
	header.Set("Origin", "http://localhost:8000")
	header.Set("Cookie", "account_token="+token)
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
	require.NoError(t, err)
	require.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
	defer conn.Close()

	ownTopic := eventdistributer.UserNotificationWebsocketTopic(r.User.String())
	otherTopic := eventdistributer.UserNotificationWebsocketTopic("11111111-1111-1111-1111-111111111111")

	require.Eventually(t, func() bool {
		return server.WebsocketHub().WorkflowSubscriberCount(ownTopic) == 1
	}, 2*time.Second, 5*time.Millisecond, "caller never subscribed to own user topic")
	assert.Equal(t, 0, server.WebsocketHub().WorkflowSubscriberCount(otherTopic))
}
