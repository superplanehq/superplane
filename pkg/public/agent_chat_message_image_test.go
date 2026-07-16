package public

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func TestAgentChatMessageImage(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	signer := jwt.NewSigner("test")
	server, err := NewServer(
		r.Encryptor, r.Registry, signer, support.NewOIDCProvider(), r.GitProvider,
		"", "http://localhost", "http://localhost", "test", "/app/templates", r.AuthService, nil, false,
	)
	require.NoError(t, err)
	registerTestGRPCGateway(t, server, r.AuthService, r.Registry, r.Encryptor, support.NewOIDCProvider(), r.GitProvider, nil)

	token, err := authentication.GenerateAccountToken(signer, r.Account.ID.String(), "", time.Now(), time.Hour)
	require.NoError(t, err)

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	session := &models.AgentSession{
		OrganizationID:    r.Organization.ID,
		UserID:            r.User,
		CanvasID:          canvas.ID,
		Provider:          "anthropic",
		ProviderSessionID: "provider-session",
		Status:            models.AgentSessionStatusIdle,
	}
	require.NoError(t, models.CreateAgentSessionInTransaction(database.Conn(), session))

	raw := []byte("pretend image bytes")
	message := &models.AgentSessionMessage{
		SessionID: session.ID,
		Role:      models.AgentMessageRoleUser,
		Content:   "look at this",
		Images: datatypes.JSONSlice[models.AgentSessionImage]{
			{MediaType: "image/png", Data: base64.StdEncoding.EncodeToString(raw)},
		},
	}
	require.NoError(t, models.AppendAgentSessionMessage(message))

	base := "/api/v1/agents/chats/" + session.ID.String() + "/messages/" + message.ID.String() + "/images/"
	org := "?organization_id=" + r.Organization.ID.String()

	get := func(path string, withAuth bool) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		if withAuth {
			req.AddCookie(&http.Cookie{Name: "account_token", Value: token})
		}
		rec := httptest.NewRecorder()
		server.Router.ServeHTTP(rec, req)
		return rec
	}

	t.Run("serves the decoded image bytes", func(t *testing.T) {
		res := get(base+"0"+org, true)

		require.Equal(t, http.StatusOK, res.Code)
		assert.Equal(t, "image/png", res.Header().Get("Content-Type"))
		assert.Equal(t, raw, res.Body.Bytes())
	})

	t.Run("out-of-range index returns 404", func(t *testing.T) {
		res := get(base+"9"+org, true)

		assert.Equal(t, http.StatusNotFound, res.Code)
	})

	t.Run("unauthenticated request is rejected", func(t *testing.T) {
		res := get(base+"0"+org, false)

		assert.NotEqual(t, http.StatusOK, res.Code)
	})
}
