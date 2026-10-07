package public

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	factoryactions "github.com/superplanehq/superplane/pkg/grpc/actions/factories"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"

	_ "github.com/superplanehq/superplane/pkg/registryimports"
)

func TestAdminFactoryTemplates(t *testing.T) {
	server, r, token := setupAdminTestServer(t)
	server.WebhooksBaseURL = "http://localhost:8000"
	listPath := "/admin/api/installation/factory-templates"

	t.Run("non-admin gets 404", func(t *testing.T) {
		account, err := models.CreateAccount("Regular User", "regular-factory-templates@example.com")
		require.NoError(t, err)
		signer := jwt.NewSigner("test-client-secret")
		regularToken, err := authentication.GenerateAccountToken(signer, account.ID.String(), time.Now(), time.Hour)
		require.NoError(t, err)

		listResponse := execRequest(server, requestParams{
			method:     http.MethodGet,
			path:       listPath,
			authCookie: regularToken,
		})
		assert.Equal(t, http.StatusNotFound, listResponse.Code)

		resetResponse := execRequest(server, requestParams{
			method:     http.MethodPost,
			path:       listPath + "/backlog/reset",
			authCookie: regularToken,
		})
		assert.Equal(t, http.StatusNotFound, resetResponse.Code)
	})

	t.Run("admin lists onboarding templates", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:     http.MethodGet,
			path:       listPath,
			authCookie: token,
		})
		assert.Equal(t, http.StatusOK, response.Code)

		var payload adminFactoryTemplatesResponse
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &payload))
		require.Len(t, payload.Templates, 4)
		assert.Equal(t, "backlog", payload.Templates[0].ID)
		assert.Equal(t, "line-implementation", payload.Templates[1].ID)
		assert.Equal(t, "pr-closure", payload.Templates[2].ID)
		assert.Equal(t, "intake", payload.Templates[3].ID)
	})

	t.Run("unknown template is 404", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:     http.MethodPost,
			path:       listPath + "/risk-score/reset",
			authCookie: token,
		})
		assert.Equal(t, http.StatusNotFound, response.Code)
	})

	t.Run("admin resets Backlog automations on the installation", func(t *testing.T) {
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		factoryModel, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, "Reset Target", "", "")
		require.NoError(t, err)
		_, err = factoryactions.CreateFactoryIntake(ctx, factoryactions.IntakeDependencies{
			Registry:       r.Registry,
			Encryptor:      r.Encryptor,
			AuthService:    r.AuthService,
			WebhookBaseURL: "http://localhost:8000",
		}, r.Organization.ID.String(), &pb.CreateFactoryIntakeRequest{
			FactoryId: factoryModel.ID.String(),
			Source:    pb.FactoryIntake_SOURCE_GITHUB_ISSUES,
		})
		require.NoError(t, err)

		response := execRequest(server, requestParams{
			method:     http.MethodPost,
			path:       listPath + "/backlog/reset",
			authCookie: token,
		})
		assert.Equal(t, http.StatusOK, response.Code)

		var result factoryactions.ResetFactoryTemplateResult
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &result))
		assert.Equal(t, 1, result.Reset)
		assert.Empty(t, result.Failures)
	})
}
