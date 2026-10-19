package public

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestAdminResetBuiltInAutomations(t *testing.T) {
	server, r, token := setupAdminTestServer(t)

	t.Run("non-admin gets 404", func(t *testing.T) {
		account, err := models.CreateAccount("Regular User", "regular-reset@example.com")
		require.NoError(t, err)
		signer := jwt.NewSigner("test-client-secret")
		regularToken, err := authentication.GenerateAccountToken(signer, account.ID.String(), time.Now(), time.Hour)
		require.NoError(t, err)

		response := execRequest(server, requestParams{
			method:     "POST",
			path:       "/admin/api/organizations/" + r.Organization.ID.String() + "/built-in-automations/reset",
			authCookie: regularToken,
		})
		assert.Equal(t, http.StatusNotFound, response.Code)
	})

	t.Run("admin resets the organization", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:     "POST",
			path:       "/admin/api/organizations/" + r.Organization.ID.String() + "/built-in-automations/reset",
			authCookie: token,
		})
		assert.Equal(t, http.StatusOK, response.Code)

		var body struct {
			Reset int `json:"reset"`
		}
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
		assert.Equal(t, 0, body.Reset)
	})
}
