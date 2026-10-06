package public

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func setupAdminTestServer(t *testing.T) (*Server, *support.ResourceRegistry, string) {
	r := support.Setup(t)
	server, _, token := setupTestServer(r, t)

	// Promote the test account to installation admin
	require.NoError(t, models.PromoteToInstallationAdmin(r.Account.ID.String()))

	return server, r, token
}

func TestAdminListOrganizations(t *testing.T) {
	server, r, token := setupAdminTestServer(t)

	t.Run("non-admin gets 404", func(t *testing.T) {
		// Create a non-admin account
		account, err := models.CreateAccount("Regular User", "regular@example.com")
		require.NoError(t, err)
		signer := jwt.NewSigner("test-client-secret")
		regularToken, err := authentication.GenerateAccountToken(signer, account.ID.String(), time.Now(), time.Hour)
		require.NoError(t, err)

		response := execRequest(server, requestParams{
			method:     "GET",
			path:       "/admin/api/organizations",
			authCookie: regularToken,
		})
		assert.Equal(t, http.StatusNotFound, response.Code)
	})

	t.Run("admin can list all organizations", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:     "GET",
			path:       "/admin/api/organizations",
			authCookie: token,
		})

		assert.Equal(t, http.StatusOK, response.Code)

		var page struct {
			Items  []map[string]any `json:"items"`
			Total  int64            `json:"total"`
			Limit  int              `json:"limit"`
			Offset int              `json:"offset"`
		}
		err := json.Unmarshal(response.Body.Bytes(), &page)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(page.Items), 1)
		assert.GreaterOrEqual(t, page.Total, int64(1))
		assert.Equal(t, 50, page.Limit)

		found := false
		for _, org := range page.Items {
			if org["id"] == r.Organization.ID.String() {
				found = true
				break
			}
		}
		assert.True(t, found, "should include the test organization")
	})

	t.Run("supports search filter", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:     "GET",
			path:       "/admin/api/organizations?search=" + r.Organization.Name[:3],
			authCookie: token,
		})

		assert.Equal(t, http.StatusOK, response.Code)

		var page struct {
			Items []map[string]any `json:"items"`
			Total int64            `json:"total"`
		}
		err := json.Unmarshal(response.Body.Bytes(), &page)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, page.Total, int64(1))
	})

	t.Run("supports pagination params", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:     "GET",
			path:       "/admin/api/organizations?limit=1&offset=0",
			authCookie: token,
		})

		assert.Equal(t, http.StatusOK, response.Code)

		var page struct {
			Items []map[string]any `json:"items"`
			Total int64            `json:"total"`
			Limit int              `json:"limit"`
		}
		err := json.Unmarshal(response.Body.Bytes(), &page)
		require.NoError(t, err)
		assert.LessOrEqual(t, len(page.Items), 1)
		assert.Equal(t, 1, page.Limit)
	})

	t.Run("unauthenticated request is rejected", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method: "GET",
			path:   "/admin/api/organizations",
		})
		assert.NotEqual(t, http.StatusOK, response.Code)
	})
}

func TestAdminGetOrganization(t *testing.T) {
	server, r, token := setupAdminTestServer(t)
	path := "/admin/api/organizations/" + r.Organization.ID.String()

	t.Run("admin can load organization", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:     "GET",
			path:       path,
			authCookie: token,
		})
		assert.Equal(t, http.StatusOK, response.Code)

		var org adminOrgItem
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &org))
		assert.Equal(t, r.Organization.ID.String(), org.ID)
		assert.Equal(t, r.Organization.Name, org.Name)
		assert.Equal(t, r.Organization.Slug, org.Slug)
		assert.Equal(t, r.Organization.Description, org.Description)
		assert.GreaterOrEqual(t, org.MemberCount, int64(0))
		assert.GreaterOrEqual(t, org.CanvasCount, int64(0))
		assert.GreaterOrEqual(t, org.TaskCount, int64(0))
		assert.GreaterOrEqual(t, org.DoneTaskCount, int64(0))
		require.NotNil(t, org.CreatedAt)
		require.NotNil(t, org.UpdatedAt)
	})

	t.Run("returns exact task counts", func(t *testing.T) {
		org, err := models.CreateOrganization("Counted Tasks Org", "counts")
		require.NoError(t, err)
		db := database.DB(t.Context())
		factory, err := models.CreateFactory(db, org.ID, "Factory", "", "")
		require.NoError(t, err)
		_, err = factory.CreateWorkOrder(db, "Open task", "", nil, nil, nil)
		require.NoError(t, err)
		done, err := factory.CreateWorkOrder(db, "Done task", "", nil, nil, nil)
		require.NoError(t, err)
		require.NoError(t, db.Model(done).Updates(map[string]any{
			"state":  models.FactoryWorkOrderStateClosed,
			"result": models.FactoryWorkOrderResultCompleted,
		}).Error)
		rejected, err := factory.CreateWorkOrder(db, "Rejected task", "", nil, nil, nil)
		require.NoError(t, err)
		require.NoError(t, db.Model(rejected).Updates(map[string]any{
			"state":  models.FactoryWorkOrderStateClosed,
			"result": models.FactoryWorkOrderResultRejected,
		}).Error)

		response := execRequest(server, requestParams{
			method:     "GET",
			path:       "/admin/api/organizations/" + org.ID.String(),
			authCookie: token,
		})
		assert.Equal(t, http.StatusOK, response.Code)

		var body adminOrgItem
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
		assert.Equal(t, int64(3), body.TaskCount)
		assert.Equal(t, int64(1), body.DoneTaskCount)
	})

	t.Run("returns 404 for unknown organization", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:     "GET",
			path:       "/admin/api/organizations/00000000-0000-0000-0000-000000000000",
			authCookie: token,
		})
		assert.Equal(t, http.StatusNotFound, response.Code)
	})

	t.Run("returns 404 for deleted organization", func(t *testing.T) {
		deleted, err := models.CreateOrganization("Deleted Admin Org", "gone")
		require.NoError(t, err)
		require.NoError(t, models.SoftDeleteOrganization(deleted.ID.String()))

		response := execRequest(server, requestParams{
			method:     "GET",
			path:       "/admin/api/organizations/" + deleted.ID.String(),
			authCookie: token,
		})
		assert.Equal(t, http.StatusNotFound, response.Code)
	})

	t.Run("non-admin gets 404", func(t *testing.T) {
		account, err := models.CreateAccount("Regular User", "regular-org-get@example.com")
		require.NoError(t, err)
		signer := jwt.NewSigner("test-client-secret")
		regularToken, err := authentication.GenerateAccountToken(signer, account.ID.String(), time.Now(), time.Hour)
		require.NoError(t, err)

		response := execRequest(server, requestParams{
			method:     "GET",
			path:       path,
			authCookie: regularToken,
		})
		assert.Equal(t, http.StatusNotFound, response.Code)
	})
}

func TestAdminOrganizationPins(t *testing.T) {
	server, r, token := setupAdminTestServer(t)

	t.Run("pins and unpins from the list and detail responses", func(t *testing.T) {
		org, err := models.CreateOrganization("Pinned Customer", "customer")
		require.NoError(t, err)

		pinResponse := execRequest(server, requestParams{
			method:     http.MethodPut,
			path:       adminOrganizationPinPath(org.ID),
			authCookie: token,
		})
		require.Equal(t, http.StatusOK, pinResponse.Code)
		assertPinState(t, pinResponse, true)

		page := listAdminOrganizations(t, server, token, "")
		assert.Equal(t, []string{org.ID.String()}, adminOrganizationIDs(page.Pinned))
		assert.NotContains(t, adminOrganizationIDs(page.Items), org.ID.String())
		assert.Equal(t, page.MatchTotal-1, page.Total)

		detail := getAdminOrganization(t, server, token, org.ID)
		assert.True(t, detail.Pinned)

		reloaded := listAdminOrganizations(t, server, token, "")
		assert.Equal(t, adminOrganizationIDs(page.Pinned), adminOrganizationIDs(reloaded.Pinned))

		unpinResponse := execRequest(server, requestParams{
			method:     http.MethodDelete,
			path:       adminOrganizationPinPath(org.ID),
			authCookie: token,
		})
		require.Equal(t, http.StatusOK, unpinResponse.Code)
		assertPinState(t, unpinResponse, false)

		afterUnpin := listAdminOrganizations(t, server, token, "")
		assert.NotContains(t, adminOrganizationIDs(afterUnpin.Pinned), org.ID.String())
		assert.Contains(t, adminOrganizationIDs(afterUnpin.Items), org.ID.String())
		assert.False(t, getAdminOrganization(t, server, token, org.ID).Pinned)
	})

	t.Run("keeps the original pin position when pinned again", func(t *testing.T) {
		first, err := models.CreateOrganization("First Pin", "")
		require.NoError(t, err)
		second, err := models.CreateOrganization("Second Pin", "")
		require.NoError(t, err)

		require.Equal(t, http.StatusOK, pinAdminOrganization(t, server, token, first.ID).Code)
		time.Sleep(20 * time.Millisecond)
		require.Equal(t, http.StatusOK, pinAdminOrganization(t, server, token, second.ID).Code)
		time.Sleep(20 * time.Millisecond)
		require.Equal(t, http.StatusOK, pinAdminOrganization(t, server, token, first.ID).Code)

		page := listAdminOrganizations(t, server, token, "sort_by=name&sort_direction=asc")
		assert.Equal(t, []string{second.ID.String(), first.ID.String()}, leadingOrganizationIDs(page.Pinned, 2))
		assert.NotContains(t, adminOrganizationIDs(page.Items), first.ID.String())
		assert.NotContains(t, adminOrganizationIDs(page.Items), second.ID.String())
	})

	t.Run("keeps a non-matching pin in the pinned section", func(t *testing.T) {
		matchingPin, err := models.CreateOrganization("Search Pin Match", "")
		require.NoError(t, err)
		unpinnedMatch, err := models.CreateOrganization("Search Row Match", "")
		require.NoError(t, err)
		other, err := models.CreateOrganization("Search Other", "")
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, pinAdminOrganization(t, server, token, matchingPin.ID).Code)
		require.Equal(t, http.StatusOK, pinAdminOrganization(t, server, token, other.ID).Code)

		page := listAdminOrganizations(t, server, token, "search=Match&sort_by=name&sort_direction=asc")
		assert.Contains(t, adminOrganizationIDs(page.Pinned), matchingPin.ID.String())
		assert.Contains(t, adminOrganizationIDs(page.Pinned), other.ID.String())
		assert.Equal(t, []string{unpinnedMatch.ID.String()}, adminOrganizationIDs(page.Items))
		assert.Equal(t, int64(1), page.Total)
		assert.Equal(t, int64(2), page.MatchTotal)
	})

	t.Run("excludes pinned organizations from every page", func(t *testing.T) {
		before := listAdminOrganizations(t, server, token, "")
		first, err := models.CreateOrganization("Page One", "")
		require.NoError(t, err)
		_, err = models.CreateOrganization("Page Two", "")
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, pinAdminOrganization(t, server, token, first.ID).Code)

		firstPage := listAdminOrganizations(t, server, token, "limit=1&offset=0")
		secondPage := listAdminOrganizations(t, server, token, "limit=1&offset=1")

		assert.Equal(t, before.MatchTotal+2, firstPage.MatchTotal)
		assert.Equal(t, before.Total+1, firstPage.Total)
		assert.Equal(t, firstPage.Total, secondPage.Total)
		assert.Contains(t, adminOrganizationIDs(firstPage.Pinned), first.ID.String())
		assert.Contains(t, adminOrganizationIDs(secondPage.Pinned), first.ID.String())
		assert.NotContains(t, adminOrganizationIDs(firstPage.Items), first.ID.String())
		assert.NotContains(t, adminOrganizationIDs(secondPage.Items), first.ID.String())
		assert.LessOrEqual(t, len(firstPage.Items), 1)
		assert.LessOrEqual(t, len(secondPage.Items), 1)
	})

	t.Run("does not show pins to another admin", func(t *testing.T) {
		otherAccount, err := models.CreateAccount("Other Admin", "other-admin-pins@example.com")
		require.NoError(t, err)
		require.NoError(t, models.PromoteToInstallationAdmin(otherAccount.ID.String()))
		signer := jwt.NewSigner("test-client-secret")
		otherToken, err := authentication.GenerateAccountToken(signer, otherAccount.ID.String(), time.Now(), time.Hour)
		require.NoError(t, err)

		require.Equal(t, http.StatusOK, pinAdminOrganization(t, server, token, r.Organization.ID).Code)

		otherPage := listAdminOrganizations(t, server, otherToken, "")
		assert.NotContains(t, adminOrganizationIDs(otherPage.Pinned), r.Organization.ID.String())
		assert.Contains(t, adminOrganizationIDs(otherPage.Items), r.Organization.ID.String())
		assert.False(t, getAdminOrganization(t, server, otherToken, r.Organization.ID).Pinned)
		assert.True(t, getAdminOrganization(t, server, token, r.Organization.ID).Pinned)
	})

	t.Run("unpin of an organization that is not pinned succeeds", func(t *testing.T) {
		org, err := models.CreateOrganization("Never Pinned", "")
		require.NoError(t, err)

		response := execRequest(server, requestParams{
			method:     http.MethodDelete,
			path:       adminOrganizationPinPath(org.ID),
			authCookie: token,
		})
		require.Equal(t, http.StatusOK, response.Code)
		assertPinState(t, response, false)
	})

	t.Run("returns 404 for unknown or deleted organizations", func(t *testing.T) {
		unknown := execRequest(server, requestParams{
			method:     http.MethodPut,
			path:       "/admin/api/organizations/00000000-0000-0000-0000-000000000000/pin",
			authCookie: token,
		})
		assert.Equal(t, http.StatusNotFound, unknown.Code)

		deleted, err := models.CreateOrganization("Deleted Pin Org", "")
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, pinAdminOrganization(t, server, token, deleted.ID).Code)
		require.NoError(t, models.SoftDeleteOrganization(deleted.ID.String()))

		page := listAdminOrganizations(t, server, token, "")
		assert.NotContains(t, adminOrganizationIDs(page.Pinned), deleted.ID.String())

		pinDeleted := execRequest(server, requestParams{
			method:     http.MethodPut,
			path:       adminOrganizationPinPath(deleted.ID),
			authCookie: token,
		})
		assert.Equal(t, http.StatusNotFound, pinDeleted.Code)

		unpinDeleted := execRequest(server, requestParams{
			method:     http.MethodDelete,
			path:       adminOrganizationPinPath(deleted.ID),
			authCookie: token,
		})
		assert.Equal(t, http.StatusNotFound, unpinDeleted.Code)
	})

	t.Run("non-admin cannot pin an organization", func(t *testing.T) {
		account, err := models.CreateAccount("Regular Pin User", "regular-pin@example.com")
		require.NoError(t, err)
		signer := jwt.NewSigner("test-client-secret")
		regularToken, err := authentication.GenerateAccountToken(signer, account.ID.String(), time.Now(), time.Hour)
		require.NoError(t, err)

		response := execRequest(server, requestParams{
			method:     http.MethodPut,
			path:       adminOrganizationPinPath(r.Organization.ID),
			authCookie: regularToken,
		})
		assert.Equal(t, http.StatusNotFound, response.Code)
	})
}

func adminOrganizationPinPath(orgID uuid.UUID) string {
	return "/admin/api/organizations/" + orgID.String() + "/pin"
}

func pinAdminOrganization(t *testing.T, server *Server, token string, orgID uuid.UUID) *httptest.ResponseRecorder {
	t.Helper()
	return execRequest(server, requestParams{
		method:     http.MethodPut,
		path:       adminOrganizationPinPath(orgID),
		authCookie: token,
	})
}

func listAdminOrganizations(t *testing.T, server *Server, token, query string) adminOrganizationListResponse {
	t.Helper()
	path := "/admin/api/organizations"
	if query != "" {
		path += "?" + query
	}
	response := execRequest(server, requestParams{
		method:     http.MethodGet,
		path:       path,
		authCookie: token,
	})
	require.Equal(t, http.StatusOK, response.Code)

	var page adminOrganizationListResponse
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &page))
	return page
}

func getAdminOrganization(t *testing.T, server *Server, token string, orgID uuid.UUID) adminOrgItem {
	t.Helper()
	response := execRequest(server, requestParams{
		method:     http.MethodGet,
		path:       "/admin/api/organizations/" + orgID.String(),
		authCookie: token,
	})
	require.Equal(t, http.StatusOK, response.Code)

	var organization adminOrgItem
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &organization))
	return organization
}

func assertPinState(t *testing.T, response *httptest.ResponseRecorder, pinned bool) {
	t.Helper()
	var state adminOrganizationPinState
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &state))
	assert.Equal(t, pinned, state.Pinned)
}

func adminOrganizationIDs(organizations []adminOrgItem) []string {
	ids := make([]string, 0, len(organizations))
	for _, organization := range organizations {
		ids = append(ids, organization.ID)
	}
	return ids
}

func leadingOrganizationIDs(organizations []adminOrgItem, count int) []string {
	if len(organizations) < count {
		return adminOrganizationIDs(organizations)
	}
	return adminOrganizationIDs(organizations[:count])
}

func TestAdminInstallationNetworkSettings(t *testing.T) {
	unsetEnvForAdminTest(t, "BLOCKED_HTTP_HOSTS")
	unsetEnvForAdminTest(t, "BLOCKED_PRIVATE_IP_RANGES")

	server, _, token := setupAdminTestServer(t)

	t.Run("admin can read installation network settings", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:     "GET",
			path:       "/admin/api/installation/network-settings",
			authCookie: token,
		})

		assert.Equal(t, http.StatusOK, response.Code)

		var result installationSettingsResponse
		err := json.Unmarshal(response.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.False(t, result.AllowPrivateNetworkAccess)
		assert.True(t, result.SignupsEnabled)
		assert.NotEmpty(t, result.EffectiveBlockedHTTPHosts)
		assert.NotEmpty(t, result.EffectivePrivateIPRanges)
		assert.False(t, result.SMTPEnabled)
	})

	t.Run("admin can update installation network settings", func(t *testing.T) {
		body, err := json.Marshal(map[string]bool{
			"allow_private_network_access": true,
		})
		require.NoError(t, err)

		response := execRequest(server, requestParams{
			method:      "PATCH",
			path:        "/admin/api/installation/network-settings",
			body:        body,
			authCookie:  token,
			contentType: "application/json",
		})

		assert.Equal(t, http.StatusOK, response.Code)

		metadata, err := models.GetInstallationMetadata(database.Conn())
		require.NoError(t, err)
		assert.True(t, metadata.AllowPrivateNetworkAccess)

		var result installationSettingsResponse
		err = json.Unmarshal(response.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.True(t, result.AllowPrivateNetworkAccess)
		assert.Empty(t, result.EffectiveBlockedHTTPHosts)
		assert.Empty(t, result.EffectivePrivateIPRanges)
	})

	t.Run("admin can disable signups through installation settings", func(t *testing.T) {
		body, err := json.Marshal(map[string]bool{
			"signups_enabled": false,
		})
		require.NoError(t, err)

		response := execRequest(server, requestParams{
			method:      "PATCH",
			path:        "/admin/api/installation/network-settings",
			body:        body,
			authCookie:  token,
			contentType: "application/json",
		})

		assert.Equal(t, http.StatusOK, response.Code)

		metadata, err := models.GetInstallationMetadata(database.Conn())
		require.NoError(t, err)
		assert.False(t, metadata.SignupsEnabled)

		var result installationSettingsResponse
		err = json.Unmarshal(response.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.False(t, result.SignupsEnabled)
	})

	t.Run("admin can read existing smtp settings", func(t *testing.T) {
		require.NoError(t, models.UpsertEmailSettings(&models.EmailSettings{
			Provider:      models.EmailProviderSMTP,
			SMTPHost:      "smtp.example.com",
			SMTPPort:      587,
			SMTPUsername:  "smtp-user",
			SMTPPassword:  []byte("smtp-pass"),
			SMTPFromName:  "SuperPlane",
			SMTPFromEmail: "noreply@example.com",
			SMTPUseTLS:    true,
		}))

		response := execRequest(server, requestParams{
			method:     "GET",
			path:       "/admin/api/installation/network-settings",
			authCookie: token,
		})

		assert.Equal(t, http.StatusOK, response.Code)

		var result installationSettingsResponse
		err := json.Unmarshal(response.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.True(t, result.SMTPEnabled)
		assert.Equal(t, "smtp.example.com", result.SMTPHost)
		assert.Equal(t, 587, result.SMTPPort)
		assert.Equal(t, "smtp-user", result.SMTPUsername)
		assert.Equal(t, "SuperPlane", result.SMTPFromName)
		assert.Equal(t, "noreply@example.com", result.SMTPFromEmail)
		assert.True(t, result.SMTPUseTLS)
		assert.True(t, result.SMTPPasswordConfigured)
	})

	t.Run("admin can update smtp settings through installation settings", func(t *testing.T) {
		body, err := json.Marshal(map[string]any{
			"smtp_enabled":    true,
			"smtp_host":       "smtp.internal",
			"smtp_port":       2525,
			"smtp_username":   "mailer",
			"smtp_password":   "smtp-secret",
			"smtp_from_name":  "SuperPlane Admin",
			"smtp_from_email": "admin@example.com",
			"smtp_use_tls":    false,
		})
		require.NoError(t, err)

		response := execRequest(server, requestParams{
			method:      "PATCH",
			path:        "/admin/api/installation/network-settings",
			body:        body,
			authCookie:  token,
			contentType: "application/json",
		})

		assert.Equal(t, http.StatusOK, response.Code)

		settings, err := models.FindEmailSettings(models.EmailProviderSMTP)
		require.NoError(t, err)
		assert.Equal(t, "smtp.internal", settings.SMTPHost)
		assert.Equal(t, 2525, settings.SMTPPort)
		assert.Equal(t, "mailer", settings.SMTPUsername)
		assert.Equal(t, []byte("smtp-secret"), settings.SMTPPassword)
		assert.Equal(t, "SuperPlane Admin", settings.SMTPFromName)
		assert.Equal(t, "admin@example.com", settings.SMTPFromEmail)
		assert.False(t, settings.SMTPUseTLS)
	})

	t.Run("admin can disable smtp settings through installation settings", func(t *testing.T) {
		require.NoError(t, models.UpsertEmailSettings(&models.EmailSettings{
			Provider:      models.EmailProviderSMTP,
			SMTPHost:      "smtp.example.com",
			SMTPPort:      587,
			SMTPFromEmail: "noreply@example.com",
			SMTPUseTLS:    true,
		}))

		body, err := json.Marshal(map[string]bool{
			"smtp_enabled": false,
		})
		require.NoError(t, err)

		response := execRequest(server, requestParams{
			method:      "PATCH",
			path:        "/admin/api/installation/network-settings",
			body:        body,
			authCookie:  token,
			contentType: "application/json",
		})

		assert.Equal(t, http.StatusOK, response.Code)

		_, err = models.FindEmailSettings(models.EmailProviderSMTP)
		require.Error(t, err)
		assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
	})

	t.Run("admin installation settings updates are atomic", func(t *testing.T) {
		require.NoError(t, models.DeleteEmailSettings(models.EmailProviderSMTP))

		metadata, err := models.GetInstallationMetadata(database.Conn())
		require.NoError(t, err)
		metadata.AllowPrivateNetworkAccess = false
		metadata.SignupsEnabled = true
		metadata.UpdatedAt = time.Now()
		require.NoError(t, models.UpdateInstallationMetadata(database.Conn(), metadata))

		body, err := json.Marshal(map[string]any{
			"allow_private_network_access": true,
			"signups_enabled":              false,
			"smtp_enabled":                 true,
			"smtp_host":                    "smtp.internal",
			"smtp_port":                    2525,
			"smtp_username":                "mailer",
			"smtp_from_name":               "SuperPlane Admin",
			"smtp_from_email":              "admin@example.com",
			"smtp_use_tls":                 true,
		})
		require.NoError(t, err)

		response := execRequest(server, requestParams{
			method:      "PATCH",
			path:        "/admin/api/installation/network-settings",
			body:        body,
			authCookie:  token,
			contentType: "application/json",
		})

		assert.Equal(t, http.StatusBadRequest, response.Code)

		metadata, err = models.GetInstallationMetadata(database.Conn())
		require.NoError(t, err)
		assert.False(t, metadata.AllowPrivateNetworkAccess)
		assert.True(t, metadata.SignupsEnabled)

		_, err = models.FindEmailSettings(models.EmailProviderSMTP)
		require.Error(t, err)
		assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
	})
}

func unsetEnvForAdminTest(t *testing.T, key string) {
	t.Helper()

	previousValue, hadValue := os.LookupEnv(key)
	require.NoError(t, os.Unsetenv(key))

	t.Cleanup(func() {
		if !hadValue {
			require.NoError(t, os.Unsetenv(key))
			return
		}

		require.NoError(t, os.Setenv(key, previousValue))
	})
}

func TestAdminListCanvases(t *testing.T) {
	server, r, token := setupAdminTestServer(t)

	t.Run("returns canvases for existing org", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:     "GET",
			path:       "/admin/api/organizations/" + r.Organization.ID.String() + "/canvases",
			authCookie: token,
		})

		assert.Equal(t, http.StatusOK, response.Code)

		var page struct {
			Items []map[string]any `json:"items"`
			Total int64            `json:"total"`
		}
		err := json.Unmarshal(response.Body.Bytes(), &page)
		require.NoError(t, err)
		assert.NotNil(t, page.Items)
	})

	t.Run("returns 404 for non-existent org", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:     "GET",
			path:       "/admin/api/organizations/00000000-0000-0000-0000-000000000000/canvases",
			authCookie: token,
		})

		assert.Equal(t, http.StatusNotFound, response.Code)
	})
}

func TestAdminListOrgUsers(t *testing.T) {
	server, r, token := setupAdminTestServer(t)

	t.Run("returns users for existing org", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:     "GET",
			path:       "/admin/api/organizations/" + r.Organization.ID.String() + "/users",
			authCookie: token,
		})

		assert.Equal(t, http.StatusOK, response.Code)

		var page struct {
			Items []map[string]any `json:"items"`
			Total int64            `json:"total"`
		}
		err := json.Unmarshal(response.Body.Bytes(), &page)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(page.Items), 1)
	})

	t.Run("returns 404 for non-existent org", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:     "GET",
			path:       "/admin/api/organizations/00000000-0000-0000-0000-000000000000/users",
			authCookie: token,
		})

		assert.Equal(t, http.StatusNotFound, response.Code)
	})
}

func TestStartImpersonation(t *testing.T) {
	server, r, token := setupAdminTestServer(t)

	t.Run("starts impersonation with a different account", func(t *testing.T) {
		otherAccount, err := models.CreateAccount("Other User", "other@example.com")
		require.NoError(t, err)

		body, _ := json.Marshal(map[string]string{
			"account_id": otherAccount.ID.String(),
		})

		response := execRequest(server, requestParams{
			method:      "POST",
			path:        "/admin/api/impersonate/start",
			body:        body,
			authCookie:  token,
			contentType: "application/json",
		})

		assert.Equal(t, http.StatusOK, response.Code)

		var result map[string]string
		err = json.Unmarshal(response.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.Equal(t, "/", result["redirect_url"])

		cookies := response.Result().Cookies()
		found := false
		for _, c := range cookies {
			if c.Name == "impersonation_token" {
				found = true
				assert.NotEmpty(t, c.Value)
				assert.True(t, c.HttpOnly)
				break
			}
		}
		assert.True(t, found, "impersonation_token cookie should be set")
	})

	t.Run("rejects self-impersonation", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"account_id": r.Account.ID.String(),
		})

		response := execRequest(server, requestParams{
			method:      "POST",
			path:        "/admin/api/impersonate/start",
			body:        body,
			authCookie:  token,
			contentType: "application/json",
		})

		assert.Equal(t, http.StatusBadRequest, response.Code)
		assert.Contains(t, response.Body.String(), "Cannot impersonate yourself")
	})

	t.Run("rejects impersonation with missing account_id", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{})

		response := execRequest(server, requestParams{
			method:      "POST",
			path:        "/admin/api/impersonate/start",
			body:        body,
			authCookie:  token,
			contentType: "application/json",
		})

		assert.Equal(t, http.StatusBadRequest, response.Code)
	})

	t.Run("rejects impersonation with non-existent account", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"account_id": "00000000-0000-0000-0000-000000000000",
		})

		response := execRequest(server, requestParams{
			method:      "POST",
			path:        "/admin/api/impersonate/start",
			body:        body,
			authCookie:  token,
			contentType: "application/json",
		})

		assert.Equal(t, http.StatusNotFound, response.Code)
	})
}

func TestEndImpersonation(t *testing.T) {
	server, _, token := setupAdminTestServer(t)

	t.Run("ends impersonation and clears cookie", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:     "POST",
			path:       "/admin/api/impersonate/end",
			authCookie: token,
		})

		assert.Equal(t, http.StatusOK, response.Code)

		var result map[string]string
		err := json.Unmarshal(response.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.Equal(t, "/admin", result["redirect_url"])

		// Check impersonation cookie was cleared
		cookies := response.Result().Cookies()
		for _, c := range cookies {
			if c.Name == "impersonation_token" {
				assert.Equal(t, "", c.Value)
				assert.Equal(t, -1, c.MaxAge)
			}
		}
	})
}

func TestImpersonationStatus(t *testing.T) {
	server, r, token := setupAdminTestServer(t)

	t.Run("returns inactive when no impersonation cookie", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:     "GET",
			path:       "/admin/api/impersonate/status",
			authCookie: token,
		})

		assert.Equal(t, http.StatusOK, response.Code)

		var result map[string]any
		err := json.Unmarshal(response.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.Equal(t, false, result["active"])
	})

	t.Run("returns active when valid impersonation cookie exists", func(t *testing.T) {
		otherAccount, err := models.CreateAccount("Status Target", "status-target@example.com")
		require.NoError(t, err)

		signer := jwt.NewSigner("test-client-secret")
		impToken, err := signer.GenerateWithClaims(time.Hour, map[string]string{
			"type":                    "impersonation",
			"admin_account_id":        r.Account.ID.String(),
			"impersonated_account_id": otherAccount.ID.String(),
			"sub":                     r.Account.ID.String(),
		})
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/admin/api/impersonate/status", nil)
		req.AddCookie(&http.Cookie{Name: "account_token", Value: token})
		req.AddCookie(&http.Cookie{Name: "impersonation_token", Value: impToken})

		res := httptest.NewRecorder()
		server.Router.ServeHTTP(res, req)

		assert.Equal(t, http.StatusOK, res.Code)

		var result map[string]any
		err = json.Unmarshal(res.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.Equal(t, true, result["active"])
		assert.NotEmpty(t, result["user_name"])
	})

	t.Run("returns inactive and clears cookie when target is blocked", func(t *testing.T) {
		target, err := models.CreateAccount("Blocked Status Target", "blocked-status-target@example.com")
		require.NoError(t, err)

		signer := jwt.NewSigner("test-client-secret")
		impToken, err := signer.GenerateWithClaims(time.Hour, map[string]string{
			"type":                    "impersonation",
			"admin_account_id":        r.Account.ID.String(),
			"impersonated_account_id": target.ID.String(),
			"sub":                     r.Account.ID.String(),
		})
		require.NoError(t, err)
		require.NoError(t, target.Block(database.Conn(), time.Now()))

		req := httptest.NewRequest(http.MethodGet, "/admin/api/impersonate/status", nil)
		req.AddCookie(&http.Cookie{Name: "account_token", Value: token})
		req.AddCookie(&http.Cookie{Name: "impersonation_token", Value: impToken})

		res := httptest.NewRecorder()
		server.Router.ServeHTTP(res, req)

		assert.Equal(t, http.StatusOK, res.Code)
		var result map[string]any
		require.NoError(t, json.Unmarshal(res.Body.Bytes(), &result))
		assert.Equal(t, false, result["active"])

		for _, cookie := range res.Result().Cookies() {
			if cookie.Name == "impersonation_token" {
				assert.Equal(t, "", cookie.Value)
				assert.Equal(t, -1, cookie.MaxAge)
			}
		}
	})
}

func TestGetAccountIncludesInstallationAdmin(t *testing.T) {
	server, r, token := setupAdminTestServer(t)

	t.Run("account response includes installation_admin field", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:     "GET",
			path:       "/account",
			authCookie: token,
		})

		assert.Equal(t, http.StatusOK, response.Code)

		var result map[string]any
		err := json.Unmarshal(response.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.Equal(t, true, result["installation_admin"])
		assert.Equal(t, r.Account.ID.String(), result["id"])
	})

	t.Run("non-admin account has installation_admin false", func(t *testing.T) {
		require.NoError(t, database.TruncateTables())
		r2 := support.Setup(t)
		_, _, regularToken := setupTestServer(r2, t)

		response := execRequest(server, requestParams{
			method:     "GET",
			path:       "/account",
			authCookie: regularToken,
		})

		// May be 200 or redirect depending on state; if 200, check the field
		if response.Code == http.StatusOK {
			var result map[string]any
			err := json.Unmarshal(response.Body.Bytes(), &result)
			require.NoError(t, err)
			assert.Equal(t, false, result["installation_admin"])
		}
	})
}

// makeImpersonationRequest creates an HTTP request carrying both the admin's
// account_token and a valid impersonation_token for the given target account.
func makeImpersonationRequest(
	t *testing.T,
	method, path, adminToken string,
	adminAccountID, targetAccountID string,
) *http.Request {
	t.Helper()
	signer := jwt.NewSigner("test-client-secret")
	impToken, err := signer.GenerateWithClaims(time.Hour, map[string]string{
		"type":                    "impersonation",
		"admin_account_id":        adminAccountID,
		"impersonated_account_id": targetAccountID,
		"sub":                     adminAccountID,
	})
	require.NoError(t, err)

	req := httptest.NewRequest(method, path, nil)
	req.AddCookie(&http.Cookie{Name: "account_token", Value: adminToken})
	req.AddCookie(&http.Cookie{Name: "impersonation_token", Value: impToken})
	return req
}

func TestImpersonationEffectiveAccount(t *testing.T) {
	server, r, adminToken := setupAdminTestServer(t)

	otherAccount, err := models.CreateAccount("Target User", "target@example.com")
	require.NoError(t, err)
	// Create a user in the org so the target account has org membership
	_, err = models.CreateUser(r.Organization.ID, otherAccount.ID, otherAccount.Email, otherAccount.Name)
	require.NoError(t, err)

	t.Run("GET /account returns impersonated user data during impersonation", func(t *testing.T) {
		req := makeImpersonationRequest(t, http.MethodGet, "/account",
			adminToken, r.Account.ID.String(), otherAccount.ID.String())

		res := httptest.NewRecorder()
		server.Router.ServeHTTP(res, req)

		require.Equal(t, http.StatusOK, res.Code)

		var result map[string]any
		require.NoError(t, json.Unmarshal(res.Body.Bytes(), &result))

		assert.Equal(t, otherAccount.ID.String(), result["id"], "should return impersonated account ID")
		assert.Equal(t, otherAccount.Email, result["email"], "should return impersonated email")
		assert.Equal(t, false, result["installation_admin"], "impersonated user is not admin")
	})

	t.Run("GET /organizations returns impersonated user orgs during impersonation", func(t *testing.T) {
		req := makeImpersonationRequest(t, http.MethodGet, "/organizations",
			adminToken, r.Account.ID.String(), otherAccount.ID.String())

		res := httptest.NewRecorder()
		server.Router.ServeHTTP(res, req)

		require.Equal(t, http.StatusOK, res.Code)

		var orgs []map[string]any
		require.NoError(t, json.Unmarshal(res.Body.Bytes(), &orgs))

		for _, org := range orgs {
			assert.Equal(t, r.Organization.ID.String(), org["id"],
				"should only see the impersonated user's organizations")
		}
	})

	t.Run("admin endpoints still use real admin account during impersonation", func(t *testing.T) {
		req := makeImpersonationRequest(t, http.MethodGet, "/admin/api/organizations",
			adminToken, r.Account.ID.String(), otherAccount.ID.String())

		res := httptest.NewRecorder()
		server.Router.ServeHTTP(res, req)

		assert.Equal(t, http.StatusOK, res.Code, "admin endpoint should still work during impersonation")
	})
}

func TestImpersonationSecurityGuardrails(t *testing.T) {
	server, r, adminToken := setupAdminTestServer(t)

	otherAccount, err := models.CreateAccount("Target", "target-sec@example.com")
	require.NoError(t, err)

	signer := jwt.NewSigner("test-client-secret")

	t.Run("non-admin with impersonation cookie is ignored", func(t *testing.T) {
		regularToken, err := authentication.GenerateAccountToken(signer, otherAccount.ID.String(), time.Now(), time.Hour)
		require.NoError(t, err)

		impToken, err := signer.GenerateWithClaims(time.Hour, map[string]string{
			"type":                    "impersonation",
			"admin_account_id":        otherAccount.ID.String(),
			"impersonated_account_id": r.Account.ID.String(),
			"sub":                     otherAccount.ID.String(),
		})
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/account", nil)
		req.AddCookie(&http.Cookie{Name: "account_token", Value: regularToken})
		req.AddCookie(&http.Cookie{Name: "impersonation_token", Value: impToken})

		res := httptest.NewRecorder()
		server.Router.ServeHTTP(res, req)

		require.Equal(t, http.StatusOK, res.Code)

		var result map[string]any
		require.NoError(t, json.Unmarshal(res.Body.Bytes(), &result))

		assert.Equal(t, otherAccount.ID.String(), result["id"],
			"non-admin impersonation cookie must be ignored")
	})

	t.Run("impersonation cookie with mismatched admin ID is ignored", func(t *testing.T) {
		impToken, err := signer.GenerateWithClaims(time.Hour, map[string]string{
			"type":                    "impersonation",
			"admin_account_id":        "00000000-0000-0000-0000-000000000000",
			"impersonated_account_id": otherAccount.ID.String(),
			"sub":                     "00000000-0000-0000-0000-000000000000",
		})
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/account", nil)
		req.AddCookie(&http.Cookie{Name: "account_token", Value: adminToken})
		req.AddCookie(&http.Cookie{Name: "impersonation_token", Value: impToken})

		res := httptest.NewRecorder()
		server.Router.ServeHTTP(res, req)

		require.Equal(t, http.StatusOK, res.Code)

		var result map[string]any
		require.NoError(t, json.Unmarshal(res.Body.Bytes(), &result))

		assert.Equal(t, r.Account.ID.String(), result["id"],
			"mismatched admin ID should cause impersonation to be ignored")
	})

	t.Run("impersonation cookie signed with wrong secret is ignored", func(t *testing.T) {
		wrongSigner := jwt.NewSigner("wrong-secret")
		impToken, err := wrongSigner.GenerateWithClaims(time.Hour, map[string]string{
			"type":                    "impersonation",
			"admin_account_id":        r.Account.ID.String(),
			"impersonated_account_id": otherAccount.ID.String(),
			"sub":                     r.Account.ID.String(),
		})
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/account", nil)
		req.AddCookie(&http.Cookie{Name: "account_token", Value: adminToken})
		req.AddCookie(&http.Cookie{Name: "impersonation_token", Value: impToken})

		res := httptest.NewRecorder()
		server.Router.ServeHTTP(res, req)

		require.Equal(t, http.StatusOK, res.Code)

		var result map[string]any
		require.NoError(t, json.Unmarshal(res.Body.Bytes(), &result))

		assert.Equal(t, r.Account.ID.String(), result["id"],
			"forged token should be ignored")
	})

	t.Run("impersonation stops working after admin is demoted", func(t *testing.T) {
		req := makeImpersonationRequest(t, http.MethodGet, "/account",
			adminToken, r.Account.ID.String(), otherAccount.ID.String())

		res := httptest.NewRecorder()
		server.Router.ServeHTTP(res, req)
		require.Equal(t, http.StatusOK, res.Code)

		var before map[string]any
		require.NoError(t, json.Unmarshal(res.Body.Bytes(), &before))
		assert.Equal(t, otherAccount.ID.String(), before["id"], "impersonation should be active")

		require.NoError(t, models.DemoteFromInstallationAdmin(r.Account.ID.String()))

		req2 := makeImpersonationRequest(t, http.MethodGet, "/account",
			adminToken, r.Account.ID.String(), otherAccount.ID.String())

		res2 := httptest.NewRecorder()
		server.Router.ServeHTTP(res2, req2)
		require.Equal(t, http.StatusOK, res2.Code)

		var after map[string]any
		require.NoError(t, json.Unmarshal(res2.Body.Bytes(), &after))
		assert.Equal(t, r.Account.ID.String(), after["id"],
			"after demotion, should return admin's own account, not impersonated")

		require.NoError(t, models.PromoteToInstallationAdmin(r.Account.ID.String()))
	})
}

func TestAdminListOrgExperimentalFeatures(t *testing.T) {
	server, _, token := setupAdminTestServer(t)

	foreignOrg, err := models.CreateOrganization("admin-flags-foreign", "")
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = models.DisableExperimentalFeature(foreignOrg.ID, features.FeatureFactories)
	})

	t.Run("returns registry and enabled features without org membership", func(t *testing.T) {
		require.NoError(t, models.EnableExperimentalFeature(foreignOrg.ID, features.FeatureFactories))

		response := execRequest(server, requestParams{
			method:     "GET",
			path:       "/admin/api/organizations/" + foreignOrg.ID.String() + "/experimental-features",
			authCookie: token,
		})
		require.Equal(t, http.StatusOK, response.Code)

		var body struct {
			Features []map[string]any `json:"features"`
			Enabled  []string         `json:"enabled"`
		}
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
		require.NotEmpty(t, body.Features)
		assert.Contains(t, body.Enabled, features.FeatureFactories)

		ids := make([]string, 0, len(body.Features))
		for _, f := range body.Features {
			id, _ := f["id"].(string)
			ids = append(ids, id)
		}
		assert.Contains(t, ids, features.FeatureFactories)
	})

	t.Run("returns an empty enabled list when none are on", func(t *testing.T) {
		require.NoError(t, models.DisableExperimentalFeature(foreignOrg.ID, features.FeatureFactories))

		response := execRequest(server, requestParams{
			method:     "GET",
			path:       "/admin/api/organizations/" + foreignOrg.ID.String() + "/experimental-features",
			authCookie: token,
		})
		require.Equal(t, http.StatusOK, response.Code)

		var body struct {
			Enabled []string `json:"enabled"`
		}
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
		assert.Empty(t, body.Enabled)
		assert.NotNil(t, body.Enabled)
	})

	t.Run("returns 404 for non-existent org", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:     "GET",
			path:       "/admin/api/organizations/00000000-0000-0000-0000-000000000000/experimental-features",
			authCookie: token,
		})
		assert.Equal(t, http.StatusNotFound, response.Code)
	})
}

func TestAdminEnableOrgExperimentalFeature(t *testing.T) {
	server, r, token := setupAdminTestServer(t)

	t.Cleanup(func() {
		_ = models.DisableExperimentalFeature(r.Organization.ID, features.FeatureClaudeManagedAgents)
	})

	t.Run("enables a known feature", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:     "POST",
			path:       "/admin/api/organizations/" + r.Organization.ID.String() + "/experimental-features/" + features.FeatureClaudeManagedAgents,
			authCookie: token,
		})
		assert.Equal(t, http.StatusOK, response.Code)

		reloaded, err := models.FindOrganizationByID(r.Organization.ID.String())
		require.NoError(t, err)
		assert.Contains(t, []string(reloaded.EnabledExperimentalFeatures), features.FeatureClaudeManagedAgents)
	})

	t.Run("rejects unknown feature ids", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:     "POST",
			path:       "/admin/api/organizations/" + r.Organization.ID.String() + "/experimental-features/does-not-exist",
			authCookie: token,
		})
		assert.Equal(t, http.StatusBadRequest, response.Code)
	})

	t.Run("returns 404 for non-existent org", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:     "POST",
			path:       "/admin/api/organizations/00000000-0000-0000-0000-000000000000/experimental-features/" + features.FeatureClaudeManagedAgents,
			authCookie: token,
		})
		assert.Equal(t, http.StatusNotFound, response.Code)
	})
}

func TestAdminDisableOrgExperimentalFeature(t *testing.T) {
	server, r, token := setupAdminTestServer(t)

	t.Run("disables a previously enabled feature", func(t *testing.T) {
		require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureClaudeManagedAgents))

		response := execRequest(server, requestParams{
			method:     "DELETE",
			path:       "/admin/api/organizations/" + r.Organization.ID.String() + "/experimental-features/" + features.FeatureClaudeManagedAgents,
			authCookie: token,
		})
		assert.Equal(t, http.StatusOK, response.Code)

		reloaded, err := models.FindOrganizationByID(r.Organization.ID.String())
		require.NoError(t, err)
		assert.NotContains(t, []string(reloaded.EnabledExperimentalFeatures), features.FeatureClaudeManagedAgents)
	})

	t.Run("is idempotent for ids not currently enabled", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:     "DELETE",
			path:       "/admin/api/organizations/" + r.Organization.ID.String() + "/experimental-features/does-not-exist",
			authCookie: token,
		})
		assert.Equal(t, http.StatusOK, response.Code)
	})

	t.Run("returns 404 for non-existent org", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:     "DELETE",
			path:       "/admin/api/organizations/00000000-0000-0000-0000-000000000000/experimental-features/" + features.FeatureClaudeManagedAgents,
			authCookie: token,
		})
		assert.Equal(t, http.StatusNotFound, response.Code)
	})
}

func TestAdminListRunnerTasks(t *testing.T) {
	server, _, token := setupAdminTestServer(t)

	t.Run("broker not configured", func(t *testing.T) {
		t.Setenv("TASK_BROKER_BASE_URL", "")
		t.Setenv("TASK_BROKER_FLEET_ID", "")
		t.Setenv("TASK_BROKER_AUTH_TOKEN", "")

		response := execRequest(server, requestParams{
			method:     "GET",
			path:       "/admin/api/runner/tasks",
			authCookie: token,
		})
		assert.Equal(t, http.StatusOK, response.Code)

		var body map[string]any
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
		assert.Equal(t, false, body["configured"])
		assert.Equal(t, []any{}, body["tasks"])
	})

	t.Run("returns active tasks from broker", func(t *testing.T) {
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "/v1/tasks", r.URL.Path)
			assert.Equal(t, "Bearer broker-token", r.Header.Get("Authorization"))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"tasks":[{"id":"active-1","status":"queued","fleet_id":"fleet-1","created_at":"2026-05-24T12:00:00Z"}]}`))
		}))
		defer upstream.Close()

		t.Setenv("TASK_BROKER_BASE_URL", upstream.URL)
		t.Setenv("TASK_BROKER_FLEET_ID", "fleet-1")
		t.Setenv("TASK_BROKER_AUTH_TOKEN", "broker-token")

		response := execRequest(server, requestParams{
			method:     "GET",
			path:       "/admin/api/runner/tasks",
			authCookie: token,
		})
		assert.Equal(t, http.StatusOK, response.Code)

		var body struct {
			Configured bool `json:"configured"`
			Tasks      []struct {
				ID     string `json:"id"`
				Status string `json:"status"`
			} `json:"tasks"`
		}
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
		assert.True(t, body.Configured)
		require.Len(t, body.Tasks, 1)
		assert.Equal(t, "active-1", body.Tasks[0].ID)
		assert.Equal(t, "queued", body.Tasks[0].Status)
	})
}

func TestAdminBlockAndUnblockAccount(t *testing.T) {
	server, r, token := setupAdminTestServer(t)

	target, err := models.CreateAccount("Block Target", "block-target@example.com")
	require.NoError(t, err)

	t.Run("blocks an account", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:     "POST",
			path:       "/admin/api/accounts/" + target.ID.String() + "/block",
			authCookie: token,
		})
		assert.Equal(t, http.StatusOK, response.Code)

		blocked, err := models.FindAccountByID(target.ID.String())
		require.NoError(t, err)
		assert.True(t, blocked.IsBlocked())
	})

	t.Run("lists blocked flag", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:     "GET",
			path:       "/admin/api/accounts?search=block-target",
			authCookie: token,
		})
		assert.Equal(t, http.StatusOK, response.Code)

		var body struct {
			Items []struct {
				ID      string `json:"id"`
				Blocked bool   `json:"blocked"`
			} `json:"items"`
		}
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
		require.NotEmpty(t, body.Items)
		assert.True(t, body.Items[0].Blocked)
	})

	t.Run("rejects impersonating a blocked account", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"account_id": target.ID.String(),
		})
		response := execRequest(server, requestParams{
			method:      "POST",
			path:        "/admin/api/impersonate/start",
			body:        body,
			authCookie:  token,
			contentType: "application/json",
		})
		assert.Equal(t, http.StatusBadRequest, response.Code)
		assert.Contains(t, response.Body.String(), "Cannot impersonate a blocked account")
	})

	t.Run("unblocks an account", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:     "POST",
			path:       "/admin/api/accounts/" + target.ID.String() + "/unblock",
			authCookie: token,
		})
		assert.Equal(t, http.StatusOK, response.Code)

		unblocked, err := models.FindAccountByID(target.ID.String())
		require.NoError(t, err)
		assert.False(t, unblocked.IsBlocked())
	})

	t.Run("rejects self-block", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:     "POST",
			path:       "/admin/api/accounts/" + r.Account.ID.String() + "/block",
			authCookie: token,
		})
		assert.Equal(t, http.StatusBadRequest, response.Code)
		assert.Contains(t, response.Body.String(), "Cannot block yourself")
	})
}

func TestAdminDeleteAccount(t *testing.T) {
	server, r, token := setupAdminTestServer(t)

	t.Run("deletes the account, its organization, and its workspaces", func(t *testing.T) {
		target, err := models.CreateAccount("Delete Target", "delete-target@example.com")
		require.NoError(t, err)
		organization, err := models.CreateOrganization(support.RandomName("org"), "")
		require.NoError(t, err)
		require.NoError(t, models.SetOrganizationCreatedByAccount(database.Conn(), organization.ID, target.ID))
		owner, err := models.CreateUserInTransaction(database.Conn(), organization.ID, target.ID, target.Email, target.Name)
		require.NoError(t, err)
		require.NoError(t, models.SetUserIsOwner(database.Conn(), owner.ID, true))
		factory, err := models.CreateFactory(database.Conn(), organization.ID, "Workspace", "", "WSP")
		require.NoError(t, err)

		response := execRequest(server, requestParams{
			method:     "DELETE",
			path:       "/admin/api/accounts/" + target.ID.String(),
			authCookie: token,
		})
		require.Equal(t, http.StatusOK, response.Code)

		var body struct {
			DeletedOrganizationIDs []string `json:"deleted_organization_ids"`
		}
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
		assert.Equal(t, []string{organization.ID.String()}, body.DeletedOrganizationIDs)

		_, err = models.FindAccountByID(target.ID.String())
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
		_, err = models.FindAccountByEmail("delete-target@example.com")
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
		_, err = models.FindOrganizationByID(organization.ID.String())
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

		var remainingFactories int64
		require.NoError(t, database.Conn().Model(&models.Factory{}).Where("id = ?", factory.ID).Count(&remainingFactories).Error)
		assert.Zero(t, remainingFactories)
	})

	t.Run("keeps organizations that have another owner", func(t *testing.T) {
		target := support.CreateUser(t, r, r.Organization.ID)

		response := execRequest(server, requestParams{
			method:     "DELETE",
			path:       "/admin/api/accounts/" + target.AccountID.String(),
			authCookie: token,
		})
		require.Equal(t, http.StatusOK, response.Code)

		_, err := models.FindAccountByID(target.AccountID.String())
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
		_, err = models.FindOrganizationByID(r.Organization.ID.String())
		require.NoError(t, err)
	})

	t.Run("rejects self-delete", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:     "DELETE",
			path:       "/admin/api/accounts/" + r.Account.ID.String(),
			authCookie: token,
		})
		assert.Equal(t, http.StatusBadRequest, response.Code)

		_, err := models.FindAccountByID(r.Account.ID.String())
		require.NoError(t, err)
	})

	t.Run("returns not found for an unknown account", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:     "DELETE",
			path:       "/admin/api/accounts/" + uuid.NewString(),
			authCookie: token,
		})
		assert.Equal(t, http.StatusNotFound, response.Code)
	})
}
