package public

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestWebhookSetupWebSocketAPIKeyCanvasScope(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	server, _, _ := setupTestServer(r, t)
	server.WebsocketHub().Run()
	httpServer := httptest.NewServer(server.Router)
	defer httpServer.Close()

	for _, test := range []struct {
		name       string
		canvasIDs  []string
		wantStatus int
	}{
		{name: "unrestricted", wantStatus: http.StatusSwitchingProtocols},
		{name: "canvas restricted", canvasIDs: []string{uuid.NewString()}, wantStatus: http.StatusForbidden},
	} {
		t.Run(test.name, func(t *testing.T) {
			key, err := models.CreateAPIKey(database.DB(t.Context()), r.Organization.ID, test.name, nil, r.User, nil, test.canvasIDs)
			require.NoError(t, err)
			token, err := crypto.Base64String(32)
			require.NoError(t, err)
			require.NoError(t, key.UpdateTokenHash(crypto.HashToken(token)))
			require.NoError(t, r.AuthService.AssignRole(key.ID.String(), models.RoleOrgAdmin, r.Organization.ID.String(), models.DomainTypeOrganization))

			header := http.Header{}
			header.Set("Origin", "http://localhost:8000")
			header.Set("Authorization", "Bearer "+token)
			url := "ws" + strings.TrimPrefix(httpServer.URL, "http") + "/ws/integrations/webhook-setup?organization_id=" + r.Organization.ID.String()
			conn, response, err := websocket.DefaultDialer.Dial(url, header)
			if conn != nil {
				defer conn.Close()
			}
			if response != nil {
				defer response.Body.Close()
			}
			if test.wantStatus == http.StatusSwitchingProtocols {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, websocket.ErrBadHandshake)
			}
			require.NotNil(t, response)
			require.Equal(t, test.wantStatus, response.StatusCode)
		})
	}
}
