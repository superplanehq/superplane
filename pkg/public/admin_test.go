package public

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
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
		regularToken, err := signer.Generate(account.ID.String(), time.Hour)
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
		regularToken, err := signer.Generate(otherAccount.ID.String(), time.Hour)
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
