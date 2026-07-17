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

func TestSetupOwnerIgnoresInstallationSettings(t *testing.T) {
	middleware.ResetOwnerSetupStateForTests()

	r := support.Setup(t)
	require.NoError(t, database.TruncateTables())

	t.Setenv("BASE_URL", "http://localhost:8000")

	server, err := NewServer(
		r.Encryptor,
		r.Registry,
		jwt.NewSigner("test-client-secret"),
		support.NewOIDCProvider(),
		r.GitProvider,
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

	body, err := json.Marshal(map[string]any{
		"email":                        "owner@example.com",
		"first_name":                   "Owner",
		"last_name":                    "User",
		"password":                     "Password1",
		"allow_private_network_access": true,
		"smtp_enabled":                 true,
		"smtp_host":                    "smtp.example.com",
		"smtp_port":                    587,
		"smtp_username":                "smtp-user",
		"smtp_password":                "smtp-pass",
		"smtp_from_name":               "SuperPlane",
		"smtp_from_email":              "noreply@example.com",
		"smtp_use_tls":                 true,
	})
	require.NoError(t, err)

	response := execRequest(server, requestParams{
		method:      http.MethodPost,
		path:        "/api/v1/setup-owner",
		body:        body,
		contentType: "application/json",
	})

	assert.Equal(t, http.StatusOK, response.Code)

	metadata, err := models.GetInstallationMetadata(database.Conn())
	require.NoError(t, err)
	assert.False(t, metadata.AllowPrivateNetworkAccess)

	_, err = models.FindEmailSettings(models.EmailProviderSMTP)
	require.Error(t, err)
}
