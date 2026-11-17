package public

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/test/support"
)

func TestCanvasWebSocketCanvasScopes(t *testing.T) {
	t.Setenv("ALLOWED_WS_ORIGINS", "http://localhost")
	r := support.Setup(t)
	defer r.Close()
	server, signer := mustRunnerLiveLogServer(t, r)
	server.RegisterWebSocketRoutes()
	server.WebsocketHub().Run()
	canvasA, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	canvasB, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	credentials := canvasReadCredentials(t, r, signer, canvasA.ID)
	httpServer := httptest.NewServer(server.Router)
	defer httpServer.Close()
	for _, credential := range credentials {
		t.Run(credential.name, func(t *testing.T) {
			queries := observeCanvasReadQueries(t)
			for _, item := range []struct {
				id      uuid.UUID
				allowed bool
			}{{canvasA.ID, credential.allowedA}, {canvasB.ID, credential.allowedB}} {
				*queries = nil
				header := http.Header{}
				header.Set("Origin", "http://localhost")
				header.Set("Authorization", "Bearer "+credential.token)
				conn, response, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(httpServer.URL, "http")+"/ws/"+item.id.String(), header)
				if conn != nil {
					t.Cleanup(func() { _ = conn.Close() })
				}
				if !item.allowed {
					require.Error(t, err)
					require.NotNil(t, response)
					require.Equal(t, http.StatusForbidden, response.StatusCode)
					require.NoError(t, response.Body.Close())
					require.Zero(t, server.WebsocketHub().WorkflowSubscriberCount(item.id.String()))
					require.Empty(t, *queries)
					continue
				}
				require.NoError(t, err)
				require.Equal(t, http.StatusSwitchingProtocols, response.StatusCode)
				require.Eventually(t, func() bool { return server.WebsocketHub().WorkflowSubscriberCount(item.id.String()) == 1 }, time.Second, 5*time.Millisecond)
				require.NoError(t, conn.Close())
				require.Eventually(t, func() bool { return server.WebsocketHub().WorkflowSubscriberCount(item.id.String()) == 0 }, time.Second, 5*time.Millisecond)
			}
		})
	}
	t.Run("cookie reader", func(t *testing.T) {
		token, err := authentication.GenerateAccountToken(signer, r.Account.ID.String(), time.Now(), time.Hour)
		require.NoError(t, err)
		header := http.Header{}
		header.Set("Origin", "http://localhost")
		header.Set("Cookie", "account_token="+token)
		conn, response, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(httpServer.URL, "http")+"/ws/"+canvasB.ID.String()+"?organization_id="+r.Organization.ID.String(), header)
		require.NoError(t, err)
		defer conn.Close()
		require.Equal(t, http.StatusSwitchingProtocols, response.StatusCode)
		require.Eventually(t, func() bool { return server.WebsocketHub().WorkflowSubscriberCount(canvasB.ID.String()) == 1 }, time.Second, 5*time.Millisecond)
	})
	foreignOrg := support.CreateOrganization(t, r, r.User)
	foreignCanvas, _ := support.CreateCanvas(t, foreignOrg.ID, r.User, nil, nil)
	response := canvasReadGET(server, credentials[len(credentials)-1].token, "/ws/"+foreignCanvas.ID.String())
	require.Equal(t, http.StatusNotFound, response.Code)
	request := httptest.NewRequest(http.MethodGet, "/ws/"+canvasA.ID.String(), nil)
	recorder := httptest.NewRecorder()
	server.Router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusUnauthorized, recorder.Code)
}
