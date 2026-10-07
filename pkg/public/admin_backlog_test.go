package public

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
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

func TestAdminResetOrganizationBacklogDefaults(t *testing.T) {
	server, r, token := setupAdminTestServer(t)
	path := "/admin/api/organizations/" + r.Organization.ID.String() + "/backlog-defaults/reset"

	t.Run("non-admin gets 404", func(t *testing.T) {
		account, err := models.CreateAccount("Regular User", "regular-reset@example.com")
		require.NoError(t, err)
		signer := jwt.NewSigner("test-client-secret")
		regularToken, err := authentication.GenerateAccountToken(signer, account.ID.String(), time.Now(), time.Hour)
		require.NoError(t, err)

		response := execRequest(server, requestParams{
			method:     http.MethodPost,
			path:       path,
			authCookie: regularToken,
		})
		assert.Equal(t, http.StatusNotFound, response.Code)
	})

	t.Run("unknown organization is 404", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:     http.MethodPost,
			path:       "/admin/api/organizations/" + uuid.NewString() + "/backlog-defaults/reset",
			authCookie: token,
		})
		assert.Equal(t, http.StatusNotFound, response.Code)
	})

	t.Run("admin resets Backlog automations in the organization", func(t *testing.T) {
		server.WebhooksBaseURL = "http://localhost:8000"
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
			path:       path,
			authCookie: token,
		})
		assert.Equal(t, http.StatusOK, response.Code)

		var result factoryactions.ResetOrganizationBacklogResult
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &result))
		assert.Equal(t, 1, result.Reset)
		assert.Empty(t, result.Failures)
	})
}
