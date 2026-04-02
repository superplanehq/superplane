package public

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	"github.com/superplanehq/superplane/test/support"
)

func TestSetupOwnerPersistsInstallationNetworkSettings(t *testing.T) {
	middleware.ResetOwnerSetupStateForTests()

	r := support.Setup(t)
	require.NoError(t, database.TruncateTables())

	t.Setenv("BASE_URL", "http://localhost:8000")

	server, err := NewServer(
		r.Encryptor,
		r.Registry,
		jwt.NewSigner("test-client-secret"),
		support.NewOIDCProvider(),
		"",
		"",
		"",
		"test",
		"/app/templates",
		r.AuthService,
		nil,
		false,
	)
	require.NoError(t, err)

	body, err := json.Marshal(SetupOwnerRequest{
		Email:                     "owner@example.com",
		FirstName:                 "Owner",
		LastName:                  "User",
		Password:                  "Password1!",
		AllowPrivateNetworkAccess: true,
	})
	require.NoError(t, err)

	response := execRequest(server, requestParams{
		method:      http.MethodPost,
		path:        "/api/v1/setup-owner",
		body:        body,
		contentType: "application/json",
	})

	assert.Equal(t, http.StatusOK, response.Code)

	metadata, err := models.GetInstallationMetadata()
	require.NoError(t, err)
	assert.True(t, metadata.AllowPrivateNetworkAccess)
}
