package public

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/authentication"
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
		"",
		"",
		"",
		"test",
		"/app/templates",
		r.AuthService, false,
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

	organization, err := models.FindOrganizationByName("Demo")
	require.NoError(t, err)
	credit, err := billingmodels.DescribeOrganizationLLMCredit(database.Conn(), organization.ID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.CentsToMicros(billingmodels.SignupCreditCents), credit.GrantMicros)

	owner, err := models.FindAccountByEmail("owner@example.com")
	require.NoError(t, err)
	assert.True(t, owner.HasReceivedWelcomeCredit())

	var fleets []models.RunnerFleet
	require.NoError(t, database.Conn().
		Where("scope_type = ?", models.RunnerFleetScopeInstallation).
		Find(&fleets).Error)
	require.Empty(t, fleets)
}

func setupOwnerTestServer(t *testing.T) *Server {
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
		r.AuthService, false,
	)
	require.NoError(t, err)
	return server
}

// submitOwnerForm creates the owner account through the public endpoint and
// returns the account_token cookie issued for the new owner session.
func submitOwnerForm(t *testing.T, server *Server) string {
	body, err := json.Marshal(map[string]any{
		"email":      "owner@example.com",
		"first_name": "Owner",
		"last_name":  "User",
		"password":   "Password1",
	})
	require.NoError(t, err)

	response := execRequest(server, requestParams{
		method:      http.MethodPost,
		path:        "/api/v1/setup-owner",
		body:        body,
		contentType: "application/json",
	})
	require.Equal(t, http.StatusOK, response.Code)

	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == "account_token" {
			return cookie.Value
		}
	}

	t.Fatal("account_token cookie was not issued")
	return ""
}

func getSetupOwnerStatus(t *testing.T, server *Server, authCookie string) map[string]any {
	response := execRequest(server, requestParams{
		method:     http.MethodGet,
		path:       "/api/v1/setup-owner/status",
		authCookie: authCookie,
	})
	require.Equal(t, http.StatusOK, response.Code)

	var status map[string]any
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &status))
	return status
}

func patchSetupOwnerProgress(t *testing.T, server *Server, authCookie string, step string) *httptest.ResponseRecorder {
	body, err := json.Marshal(map[string]any{"step": step})
	require.NoError(t, err)

	return execRequest(server, requestParams{
		method:      http.MethodPatch,
		path:        "/api/v1/setup-owner/progress",
		body:        body,
		contentType: "application/json",
		authCookie:  authCookie,
	})
}

func TestSetupOwnerStatusRequired(t *testing.T) {
	t.Setenv("SUPERPLANE_BEACON_ENABLED", "no")
	server := setupOwnerTestServer(t)

	status := getSetupOwnerStatus(t, server, "")
	assert.Equal(t, setupOwnerStatusRequired, status["status"])
	assert.NotContains(t, status, "step")
	assert.NotContains(t, status, "organization_slug")
	assert.Equal(t, false, status["beacon_enabled"])
}

func TestSetupOwnerStatusReportsBeacon(t *testing.T) {
	t.Setenv("SUPERPLANE_BEACON_ENABLED", "yes")
	t.Setenv("SUPERPLANE_INSTALLATION_TYPE", "kubernetes")
	server := setupOwnerTestServer(t)

	status := getSetupOwnerStatus(t, server, "")
	assert.Equal(t, true, status["beacon_enabled"])
	assert.Equal(t, "kubernetes", status["installation_type"])
}

func TestSetupOwnerPersistsShareOwnerIdentity(t *testing.T) {
	server := setupOwnerTestServer(t)
	body, err := json.Marshal(map[string]any{
		"email":                "owner@example.com",
		"first_name":           "Owner",
		"last_name":            "User",
		"password":             "Password1",
		"share_owner_identity": true,
	})
	require.NoError(t, err)

	response := execRequest(server, requestParams{
		method:      http.MethodPost,
		path:        "/api/v1/setup-owner",
		body:        body,
		contentType: "application/json",
	})
	require.Equal(t, http.StatusOK, response.Code)

	metadata, err := models.GetInstallationMetadata(database.Conn())
	require.NoError(t, err)
	assert.True(t, metadata.ShareOwnerIdentity)
}

func TestSetupOwnerStatusInProgress(t *testing.T) {
	server := setupOwnerTestServer(t)
	authCookie := submitOwnerForm(t, server)

	status := getSetupOwnerStatus(t, server, authCookie)
	assert.Equal(t, setupOwnerStatusInProgress, status["status"])
	assert.Equal(t, models.OwnerSetupStepLicense, status["step"])
	assert.NotContains(t, status, "organization_slug")
}

func TestSetupOwnerProgressUpdate(t *testing.T) {
	server := setupOwnerTestServer(t)
	authCookie := submitOwnerForm(t, server)

	response := patchSetupOwnerProgress(t, server, authCookie, models.OwnerSetupStepSMTP)
	assert.Equal(t, http.StatusOK, response.Code)

	var progress map[string]any
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &progress))
	assert.Equal(t, setupOwnerStatusInProgress, progress["status"])
	assert.Equal(t, models.OwnerSetupStepSMTP, progress["step"])

	status := getSetupOwnerStatus(t, server, authCookie)
	assert.Equal(t, setupOwnerStatusInProgress, status["status"])
	assert.Equal(t, models.OwnerSetupStepSMTP, status["step"])
}

func TestSetupOwnerProgressAcceptsDockerStep(t *testing.T) {
	server := setupOwnerTestServer(t)
	authCookie := submitOwnerForm(t, server)

	response := patchSetupOwnerProgress(t, server, authCookie, models.OwnerSetupStepFleetDocker)
	assert.Equal(t, http.StatusOK, response.Code)

	status := getSetupOwnerStatus(t, server, authCookie)
	assert.Equal(t, models.OwnerSetupStepFleetDocker, status["step"])
}

func TestSetupOwnerProgressRejectsUnknownStep(t *testing.T) {
	server := setupOwnerTestServer(t)
	authCookie := submitOwnerForm(t, server)

	response := patchSetupOwnerProgress(t, server, authCookie, "not-a-step")
	assert.Equal(t, http.StatusBadRequest, response.Code)

	status := getSetupOwnerStatus(t, server, authCookie)
	assert.Equal(t, models.OwnerSetupStepLicense, status["step"])
}

func TestSetupOwnerStatusComplete(t *testing.T) {
	server := setupOwnerTestServer(t)
	authCookie := submitOwnerForm(t, server)

	response := patchSetupOwnerProgress(t, server, authCookie, "")
	assert.Equal(t, http.StatusOK, response.Code)

	status := getSetupOwnerStatus(t, server, authCookie)
	assert.Equal(t, setupOwnerStatusComplete, status["status"])
	assert.NotContains(t, status, "step")
}

func TestSetupOwnerProgressRejectsReopen(t *testing.T) {
	server := setupOwnerTestServer(t)
	authCookie := submitOwnerForm(t, server)

	response := patchSetupOwnerProgress(t, server, authCookie, "")
	require.Equal(t, http.StatusOK, response.Code)

	response = patchSetupOwnerProgress(t, server, authCookie, models.OwnerSetupStepSMTP)
	assert.Equal(t, http.StatusConflict, response.Code)

	response = patchSetupOwnerProgress(t, server, authCookie, "")
	assert.Equal(t, http.StatusConflict, response.Code)

	status := getSetupOwnerStatus(t, server, authCookie)
	assert.Equal(t, setupOwnerStatusComplete, status["status"])
}

func TestSetupOwnerProgressRejectsNonAdmin(t *testing.T) {
	server := setupOwnerTestServer(t)
	submitOwnerForm(t, server)

	account, err := models.CreateAccount("Regular User", "regular@example.com")
	require.NoError(t, err)

	signer := jwt.NewSigner("test-client-secret")
	token, err := authentication.GenerateAccountToken(signer, account.ID.String(), time.Now(), time.Hour)
	require.NoError(t, err)

	response := patchSetupOwnerProgress(t, server, token, models.OwnerSetupStepSMTP)
	assert.Equal(t, http.StatusForbidden, response.Code)
}

func TestSetupOwnerSkipsWelcomeCreditWhenBillingDisabled(t *testing.T) {
	t.Setenv(billingmodels.BillingEnabledEnv, "no")
	server := setupOwnerTestServer(t)
	authCookie := submitOwnerForm(t, server)

	organization, err := models.FindOrganizationByName("Demo")
	require.NoError(t, err)
	credit, err := billingmodels.DescribeOrganizationLLMCredit(database.Conn(), organization.ID)
	require.NoError(t, err)
	assert.Zero(t, credit.GrantMicros)

	plan, err := billingmodels.FindOrganizationBillingPlan(database.Conn(), organization.ID)
	require.NoError(t, err)
	assert.Nil(t, plan)

	owner, err := models.FindAccountByEmail("owner@example.com")
	require.NoError(t, err)
	assert.False(t, owner.HasReceivedWelcomeCredit())

	status := getSetupOwnerStatus(t, server, authCookie)
	assert.Equal(t, setupOwnerStatusInProgress, status["status"])
	assert.Equal(t, models.OwnerSetupStepLicense, status["step"])
}
