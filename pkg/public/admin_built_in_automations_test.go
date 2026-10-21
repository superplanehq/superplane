package public

import (
	"encoding/json"
	"net/http"
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

	_ "github.com/superplanehq/superplane/pkg/registryimports"
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
	t.Run("admin sees the automation and invalid setting without a partial reset", func(t *testing.T) {
		db := database.DB(t.Context())
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		canvas := support.CreateFactoryCanvas(t, r, factory.ID, "Implement")
		original := *canvas.LiveVersionID
		require.NoError(t, db.Model(&models.CanvasVersion{}).Where("id = ?", original).Update("nodes", datatypes.NewJSONSlice([]models.Node{
			{ID: "onrun-implement", Name: "On Run", Type: models.NodeTypeTrigger, Ref: models.NodeRef{Trigger: &models.TriggerRef{Name: "onRun"}}},
			{ID: "implementation-agent-no-issue", Name: "Implement task", Type: models.NodeTypeComponent, Ref: models.NodeRef{Component: &models.ComponentRef{Name: models.SuperPlaneRunnerComponent}}, Configuration: map[string]any{"executionTimeoutSeconds": -1}},
		})).Error)
		response := execRequest(server, requestParams{
			method:     "POST",
			path:       "/admin/api/organizations/" + r.Organization.ID.String() + "/built-in-automations/reset",
			authCookie: token,
		})
		assert.Equal(t, http.StatusBadRequest, response.Code)
		assert.Contains(t, response.Body.String(), "Implement")
		assert.Contains(t, response.Body.String(), "executionTimeoutSeconds")
		assert.Contains(t, response.Body.String(), "No automations were changed.")
		reloaded, err := models.FindCanvasInTransaction(db, r.Organization.ID, canvas.ID)
		require.NoError(t, err)
		assert.Equal(t, original, *reloaded.LiveVersionID)
	})

}
