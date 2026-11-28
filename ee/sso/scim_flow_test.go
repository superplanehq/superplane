package sso

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/licensing"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

func TestSCIMUsers(t *testing.T) {
	f := newFlow(t)
	f.setOwnerEmail("owner@example.com")
	f.verifyExample()
	conn := f.put(connectionInput(true))
	created, err := f.service.CreateSCIMToken(f.ctx(), f.r.Organization.ID, f.r.Account.ID, "Okta")
	require.NoError(t, err)
	secret := created.Secret

	t.Run("lists only users", func(t *testing.T) {
		response := f.scim(http.MethodGet, conn, secret, "/ResourceTypes", "")
		require.Equal(t, http.StatusOK, response.Code)
		require.Contains(t, response.Body.String(), `"name":"User"`)
		require.NotContains(t, response.Body.String(), "Group")
	})

	t.Run("rejects groups", func(t *testing.T) {
		response := f.scim(http.MethodGet, conn, secret, "/Groups", "")
		require.Equal(t, http.StatusNotFound, response.Code)
	})

	t.Run("rejects another connection token", func(t *testing.T) {
		response := f.scim(http.MethodGet, conn, "sp_scim_wrong", "/Users", "")
		require.Equal(t, http.StatusUnauthorized, response.Code)
	})

	t.Run("rejects an unsupported filter", func(t *testing.T) {
		response := f.scim(http.MethodGet, conn, secret, "/Users?filter=title%20eq%20%22x%22", "")
		require.Equal(t, http.StatusBadRequest, response.Code)
		require.Contains(t, response.Body.String(), "invalidFilter")
	})

	var userID string
	t.Run("creates a member from an Okta filter then create", func(t *testing.T) {
		filter := f.scim(http.MethodGet, conn, secret, "/Users?filter=userName%20eq%20%22ada@example.com%22", "")
		require.Equal(t, http.StatusOK, filter.Code)
		require.Contains(t, filter.Body.String(), `"totalResults":0`)

		body := `{"userName":"ada@example.com","externalId":"okta-ada","displayName":"Ada Lovelace","active":true,"emails":[{"value":"ada@example.com","primary":true}]}`
		created := f.scim(http.MethodPost, conn, secret, "/Users", body)
		require.Equal(t, http.StatusCreated, created.Code, created.Body.String())
		userID = jsonString(t, created.Body.String(), "id")
		member, err := models.FindActiveUserByEmailInTransaction(database.Conn(), f.r.Organization.ID.String(), "ada@example.com")
		require.NoError(t, err)
		require.NotNil(t, member.ProvisionedBy)
		require.Equal(t, models.SSOProvisionedBySCIM, *member.ProvisionedBy)

		again := f.scim(http.MethodGet, conn, secret, "/Users?filter=userName%20eq%20%22ada@example.com%22", "")
		require.Equal(t, http.StatusOK, again.Code)
		require.Contains(t, again.Body.String(), `"totalResults":1`)
	})

	t.Run("rejects a duplicate user name", func(t *testing.T) {
		body := `{"userName":"ada@example.com","active":true,"emails":[{"value":"ada@example.com","primary":true}]}`
		response := f.scim(http.MethodPost, conn, secret, "/Users", body)
		require.Equal(t, http.StatusConflict, response.Code)
		require.Contains(t, response.Body.String(), "uniqueness")
	})

	t.Run("updates the profile with PUT and does not change the account email", func(t *testing.T) {
		body := `{"userName":"ada@example.com","displayName":"Ada","active":true,"emails":[{"value":"ada.renamed@example.com","primary":true}]}`
		response := f.scim(http.MethodPut, conn, secret, "/Users/"+userID, body)
		require.Equal(t, http.StatusOK, response.Code, response.Body.String())
		account, err := models.FindAccountByEmail("ada@example.com")
		require.NoError(t, err)
		require.Equal(t, "ada@example.com", account.Email)
	})

	t.Run("deactivates and reactivates with a string boolean", func(t *testing.T) {
		patch := `{"Operations":[{"op":"replace","value":{"active":"false"}}]}`
		response := f.scim(http.MethodPatch, conn, secret, "/Users/"+userID, patch)
		require.Equal(t, http.StatusOK, response.Code, response.Body.String())
		_, err := models.FindActiveUserByEmailInTransaction(database.Conn(), f.r.Organization.ID.String(), "ada@example.com")
		require.Error(t, err)

		patch = `{"Operations":[{"op":"replace","value":{"active":"TRUE"}}]}`
		response = f.scim(http.MethodPatch, conn, secret, "/Users/"+userID, patch)
		require.Equal(t, http.StatusOK, response.Code, response.Body.String())
		_, err = models.FindActiveUserByEmailInTransaction(database.Conn(), f.r.Organization.ID.String(), "ada@example.com")
		require.NoError(t, err)
	})

	t.Run("rolls back deactivation when membership removal fails", func(t *testing.T) {
		previous := f.service.Auth
		f.service.Auth = failingSCIMMembershipRemoval{Authorization: previous}
		t.Cleanup(func() { f.service.Auth = previous })

		patch := `{"Operations":[{"op":"replace","value":{"active":false}}]}`
		response := f.scim(http.MethodPatch, conn, secret, "/Users/"+userID, patch)
		require.NotEqual(t, http.StatusOK, response.Code)

		identityID, err := uuid.Parse(userID)
		require.NoError(t, err)
		identity, err := models.FindSSOIdentityByID(database.Conn(), conn.ID, identityID)
		require.NoError(t, err)
		require.True(t, identity.Active)
		_, err = models.FindActiveUserByEmailInTransaction(database.Conn(), f.r.Organization.ID.String(), "ada@example.com")
		require.NoError(t, err)
	})

	t.Run("keeps the last owner", func(t *testing.T) {
		body := `{"userName":"owner@example.com","active":true,"emails":[{"value":"owner@example.com","primary":true}]}`
		created := f.scim(http.MethodPost, conn, secret, "/Users", body)
		require.Equal(t, http.StatusCreated, created.Code, created.Body.String())
		ownerID := jsonString(t, created.Body.String(), "id")
		patch := `{"Operations":[{"op":"replace","path":"active","value":false}]}`
		response := f.scim(http.MethodPatch, conn, secret, "/Users/"+ownerID, patch)
		require.Equal(t, http.StatusOK, response.Code, response.Body.String())
		member, err := models.FindActiveUserByEmailInTransaction(database.Conn(), f.r.Organization.ID.String(), "owner@example.com")
		require.NoError(t, err)
		require.True(t, member.IsOwner)
	})

	t.Run("refuses a new user when the license is absent", func(t *testing.T) {
		f.service.Entitlements = licensing.Community
		body := `{"userName":"new@example.com","active":true,"emails":[{"value":"new@example.com","primary":true}]}`
		response := f.scim(http.MethodPost, conn, secret, "/Users", body)
		require.Equal(t, http.StatusForbidden, response.Code)
		require.Contains(t, response.Body.String(), "SSO is not licensed for this organization")
	})
}

type failingSCIMMembershipRemoval struct {
	authorization.Authorization
}

func (a failingSCIMMembershipRemoval) RemoveUserFromOrganizationInTransaction(tx *gorm.DB, userID, orgID string) error {
	if err := a.Authorization.RemoveUserFromOrganizationInTransaction(tx, userID, orgID); err != nil {
		return err
	}
	return errors.New("grant removal interrupted")
}

func (f *flow) scim(method string, conn *models.SSOConnection, secret, path, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, "/scim/v2/"+conn.ID.String()+path, strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+secret)
	request.Header.Set("Content-Type", "application/scim+json")
	response := httptest.NewRecorder()
	f.service.SCIMHandler().ServeHTTP(response, request)
	return response
}

func jsonString(t *testing.T, body, key string) string {
	t.Helper()
	var document map[string]any
	require.NoError(t, decodeJSON(body, &document))
	value, _ := document[key].(string)
	require.NotEmpty(t, value)
	return value
}

func decodeJSON(body string, dest any) error {
	return json.Unmarshal([]byte(body), dest)
}
