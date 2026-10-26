package billinghttp

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	eebilling "github.com/superplanehq/superplane/ee/billing"
	"github.com/superplanehq/superplane/ee/billing/polar"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/billing"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public"
	"github.com/superplanehq/superplane/test/support"
)

type testServer struct {
	*public.Server
	billing *Server
}

func init() {
	billing.Bind(eebilling.New(polar.NewRuntime()))
}

func clearHostedProviderDefaults(t *testing.T) {
	t.Helper()
	db := database.DB(t.Context())
	original, err := models.GetInstallationLLMSettings(db)
	require.NoError(t, err)
	t.Cleanup(func() {
		// Restore the saved selection even if its provider no longer exists.
		// Fixture cleanup must not validate or replace the previous snapshot.
		require.NoError(t, database.DB(context.WithoutCancel(t.Context())).
			Model(&models.InstallationLLMSettings{}).Where("id = ?", original.ID).
			Updates(map[string]any{
				"default_hosted_provider": original.DefaultHostedProvider,
				"default_hosted_model":    original.DefaultHostedModel,
				"updated_at":              original.UpdatedAt,
			}).Error)
	})
	_, err = models.UpdateInstallationLLMSettings(db, models.InstallationLLMSettings{})
	require.NoError(t, err)
}

func setupAdminTestServer(t *testing.T) (*testServer, *support.ResourceRegistry, string) {
	t.Helper()
	r := support.Setup(t)
	t.Setenv("GITHUB_CLIENT_ID", "test-client-id")
	t.Setenv("GITHUB_CLIENT_SECRET", "test-client-secret")
	t.Setenv("GOOGLE_CLIENT_ID", "test-google-client-id")
	t.Setenv("GOOGLE_CLIENT_SECRET", "test-google-client-secret")
	t.Setenv("BASE_URL", "http://localhost:8000")
	signer := jwt.NewSigner("test-client-secret")
	server, err := public.NewServer(r.Encryptor, r.Registry, signer, support.NewOIDCProvider(), "", "", "", "test", "/app/templates", r.AuthService, false)
	require.NoError(t, err)
	handlers := New(r.Encryptor, r.Registry, func() (string, string) { return server.BaseURL, server.WebhooksBaseURL })
	server.SetBillingHTTPRegistrar(handlers)
	require.NoError(t, models.PromoteToInstallationAdmin(r.Account.ID.String()))
	token, err := authentication.GenerateAccountToken(signer, r.Account.ID.String(), time.Now(), time.Hour)
	require.NoError(t, err)
	return &testServer{Server: server, billing: handlers}, r, token
}

type requestParams struct {
	method      string
	path        string
	body        []byte
	authToken   string
	authCookie  string
	contentType string
	headers     map[string]string
}

func execRequest(server *testServer, params requestParams) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(params.method, params.path, bytes.NewReader(params.body))
	if params.contentType != "" {
		req.Header.Set("Content-Type", params.contentType)
	}
	if params.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+params.authToken)
	}
	if params.authCookie != "" {
		req.AddCookie(&http.Cookie{Name: "account_token", Value: params.authCookie})
	}
	for key, value := range params.headers {
		req.Header.Set(key, value)
	}
	response := httptest.NewRecorder()
	server.Router.ServeHTTP(response, req)
	return response
}
