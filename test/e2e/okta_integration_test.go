package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
)

func TestOktaIntegration(t *testing.T) {
	steps := &oktaIntegrationSteps{t: t}

	t.Run("configuring okta", func(t *testing.T) {
		const samlIssuer = "https://e2e-okta.example.com/app/superplane/123"
		const samlCertificate = "-----BEGIN CERTIFICATE-----\nE2E-OKTA-CERT\n-----END CERTIFICATE-----"

		steps.startUI()
		steps.visitOktaSettingsPage()
		steps.fillSAMLSettings(samlIssuer, samlCertificate)
		steps.enableEnforceSSO()
		steps.saveSettings()
		steps.assertOktaConfigPersisted(samlIssuer, samlCertificate, true)
		steps.assertOktaURLsVisible()
	})

	t.Run("rotating SCIM tokens", func(t *testing.T) {
		const samlIssuer = "https://e2e-okta.example.com/app/superplane/123"
		const samlCertificate = "-----BEGIN CERTIFICATE-----\nE2E-OKTA-CERT\n-----END CERTIFICATE-----"

		steps.startUI()
		steps.visitOktaSettingsPage()
		steps.fillSAMLSettings(samlIssuer, samlCertificate)
		steps.saveSettings()
		steps.assertOktaConfigPersisted(samlIssuer, samlCertificate, false)

		steps.rotateSCIMToken()
		steps.assertSCIMTokenPersisted()
	})

	t.Run("adding users via SCIM", func(t *testing.T) {
		const email = "okta-user@example.com"

		steps.startSCIM()
		steps.configureOkta()

		userID := steps.provisionUser(email)
		require.NotEmpty(t, userID)
		steps.assertUserPersisted(email)
		steps.assertUserFoundByFilter(email)
	})

	t.Run("deleting users via SCIM", func(t *testing.T) {
		const email = "okta-user-delete@example.com"

		steps.startSCIM()
		steps.configureOkta()

		userID := steps.provisionUser(email)
		require.NotEmpty(t, userID)

		steps.deactivateUser(userID, email)
		steps.assertUserDeactivated(email)
	})

	t.Run("provisioning groups and memberships", func(t *testing.T) {
		const email = "okta-group-user@example.com"
		const groupName = "okta-e2e-group"

		steps.startSCIM()
		steps.configureOkta()

		userID := steps.provisionUser(email)
		steps.assertUserPersisted(email)

		steps.provisionGroup(groupName)
		steps.assertGroupPersisted(groupName)

		steps.addUserToGroup(groupName, userID)
		steps.assertGroupHasMember(groupName, userID)

		steps.removeUserFromGroup(groupName, userID)
		steps.assertGroupHasNoMembers(groupName)
	})
}

type oktaIntegrationSteps struct {
	t       *testing.T
	session *session.TestSession
	scimToken string
}

func (s *oktaIntegrationSteps) startUI() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
}

func (s *oktaIntegrationSteps) startSCIM() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.scimToken = "e2e-scim-token"
}

func (s *oktaIntegrationSteps) visitOktaSettingsPage() {
	s.session.Visit("/" + s.session.OrgID.String() + "/settings/okta")
	s.session.AssertText("Okta Integration")
}

func (s *oktaIntegrationSteps) fillSAMLSettings(issuer, certificate string) {
	issuerInput := q.Locator(`input[placeholder^="https://example.okta.com/app/"]`)
	certificateInput := q.Locator(`textarea[placeholder^="-----BEGIN CERTIFICATE-----"]`)

	s.session.FillIn(issuerInput, issuer)
	s.session.FillIn(certificateInput, certificate)
	s.session.Sleep(300)
}

func (s *oktaIntegrationSteps) enableEnforceSSO() {
	checkbox := q.Locator(`#enforce-sso`)
	s.session.Click(checkbox)
	s.session.Sleep(200)
}

func (s *oktaIntegrationSteps) saveSettings() {
	saveButton := q.Locator(`button:has-text("Save settings")`)
	s.session.Click(saveButton)
	s.session.Sleep(1000)
}

func (s *oktaIntegrationSteps) assertOktaConfigPersisted(issuer, certificate string, enforceSSO bool) {
	config, err := models.FindOrganizationOktaConfig(s.session.OrgID)
	require.NoError(s.t, err)

	require.Equal(s.t, issuer, config.SamlIssuer)
	require.Equal(s.t, certificate, config.SamlCertificate)
	require.Equal(s.t, enforceSSO, config.EnforceSSO)
}

func (s *oktaIntegrationSteps) assertOktaURLsVisible() {
	baseURL := os.Getenv("BASE_URL")
	require.NotEmpty(s.t, baseURL, "BASE_URL should be set in test environment")

	expectedSSO := fmt.Sprintf("%s/orgs/%s/okta/auth", baseURL, s.session.OrgID.String())
	expectedSCIM := fmt.Sprintf("%s/orgs/%s/okta/scim", baseURL, s.session.OrgID.String())

	// There are two read-only inputs on the page: SSO URL and SCIM base URL.
	ssoInput := s.session.Page().Locator(`input[readonly]`).Nth(0)
	scimInput := s.session.Page().Locator(`input[readonly]`).Nth(1)

	ssoValue, err := ssoInput.InputValue()
	require.NoError(s.t, err)
	scimValue, err := scimInput.InputValue()
	require.NoError(s.t, err)

	require.Equal(s.t, expectedSSO, ssoValue)
	require.Equal(s.t, expectedSCIM, scimValue)
}

func (s *oktaIntegrationSteps) rotateSCIMToken() {
	rotateButton := q.Locator(`button:has-text("Generate new token")`)
	s.session.Click(rotateButton)
	s.session.Sleep(1500)
	s.session.AssertText("New SCIM token")
}

func (s *oktaIntegrationSteps) assertSCIMTokenPersisted() {
	config, err := models.FindOrganizationOktaConfig(s.session.OrgID)
	require.NoError(s.t, err)
	require.NotEmpty(s.t, config.ScimTokenHash)
}

func (s *oktaIntegrationSteps) configureOkta() {
	config := &models.OrganizationOktaConfig{
		OrganizationID: s.session.OrgID,
		SamlIssuer:     "",
		// SCIM only needs the token hash; SAML fields can be empty here.
		ScimTokenHash: models.HashSCIMToken(s.scimToken),
	}

	err := models.SaveOrganizationOktaConfig(config)
	require.NoError(s.t, err)
}

func (s *oktaIntegrationSteps) scimBaseURL() string {
	baseURL := os.Getenv("BASE_URL")
	require.NotEmpty(s.t, baseURL, "BASE_URL should be set in test environment")

	return fmt.Sprintf("%s/orgs/%s/okta/scim", baseURL, s.session.OrgID.String())
}

func (s *oktaIntegrationSteps) scimRequest(method, path string, body any) (int, []byte) {
	urlStr := s.scimBaseURL() + path

	var buf io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		require.NoError(s.t, err)
		buf = bytes.NewReader(payload)
	}

	req, err := http.NewRequest(method, urlStr, buf)
	require.NoError(s.t, err)

	req.Header.Set("Authorization", "Bearer "+s.scimToken)
	req.Header.Set("Accept", "application/scim+json")
	if body != nil {
		req.Header.Set("Content-Type", "application/scim+json")
	}

	res, err := http.DefaultClient.Do(req)
	require.NoError(s.t, err)
	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	require.NoError(s.t, err)

	return res.StatusCode, data
}

func (s *oktaIntegrationSteps) provisionUser(email string) string {
	body := map[string]any{
		"schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
		"userName": email,
		"name": map[string]string{
			"givenName":  "Okta",
			"familyName": "User",
		},
		"active": true,
		"emails": []map[string]any{
			{
				"value":   email,
				"primary": true,
			},
		},
	}

	status, data := s.scimRequest(http.MethodPost, "/Users", body)
	require.Equal(s.t, http.StatusCreated, status, "unexpected status creating SCIM user: %s", string(data))

	var resp struct {
		ID       string `json:"id"`
		UserName string `json:"userName"`
		Active   bool   `json:"active"`
	}
	require.NoError(s.t, json.Unmarshal(data, &resp))
	require.Equal(s.t, email, resp.UserName)
	require.True(s.t, resp.Active)
	require.NotEmpty(s.t, resp.ID)

	return resp.ID
}

func (s *oktaIntegrationSteps) assertUserPersisted(email string) {
	account, err := models.FindAccountByEmail(email)
	require.NoError(s.t, err)

	user, err := models.FindMaybeDeletedUserByEmail(s.session.OrgID.String(), email)
	require.NoError(s.t, err)

	require.Equal(s.t, account.ID, user.AccountID)
	require.True(s.t, user.DeletedAt.Time.IsZero())
}

func (s *oktaIntegrationSteps) assertUserFoundByFilter(email string) {
	filter := fmt.Sprintf(`userName eq "%s"`, email)
	status, data := s.scimRequest(http.MethodGet, "/Users?filter="+url.QueryEscape(filter), nil)
	require.Equal(s.t, http.StatusOK, status, "unexpected status listing SCIM users: %s", string(data))

	var resp struct {
		TotalResults int `json:"totalResults"`
		Resources    []struct {
			ID       string `json:"id"`
			UserName string `json:"userName"`
		} `json:"Resources"`
	}
	require.NoError(s.t, json.Unmarshal(data, &resp))
	require.Equal(s.t, 1, resp.TotalResults)
	require.Len(s.t, resp.Resources, 1)
	require.Equal(s.t, email, resp.Resources[0].UserName)
}

func (s *oktaIntegrationSteps) deactivateUser(userID, email string) {
	body := map[string]any{
		"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
		"Operations": []map[string]any{
			{
				"op":    "replace",
				"path":  "active",
				"value": false,
			},
		},
	}

	status, data := s.scimRequest(http.MethodPatch, "/Users/"+userID, body)
	require.Equal(s.t, http.StatusNoContent, status, "unexpected status deactivating SCIM user: %s", string(data))

	user, err := models.FindMaybeDeletedUserByEmail(s.session.OrgID.String(), email)
	require.NoError(s.t, err)
	require.True(s.t, user.DeletedAt.Valid)
}

func (s *oktaIntegrationSteps) assertUserDeactivated(email string) {
	user, err := models.FindMaybeDeletedUserByEmail(s.session.OrgID.String(), email)
	require.NoError(s.t, err)
	require.True(s.t, user.DeletedAt.Valid)

	status, data := s.scimRequest(http.MethodGet, "/Users/"+user.ID.String(), nil)
	require.Equal(s.t, http.StatusOK, status, "unexpected status fetching SCIM user: %s", string(data))

	var resp struct {
		Active bool `json:"active"`
	}
	require.NoError(s.t, json.Unmarshal(data, &resp))
	require.False(s.t, resp.Active)
}

func (s *oktaIntegrationSteps) reactivateUser(userID, email string) {
	body := map[string]any{
		"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
		"Operations": []map[string]any{
			{
				"op":    "replace",
				"path":  "active",
				"value": true,
			},
		},
	}

	status, data := s.scimRequest(http.MethodPatch, "/Users/"+userID, body)
	require.Equal(s.t, http.StatusNoContent, status, "unexpected status reactivating SCIM user: %s", string(data))

	user, err := models.FindMaybeDeletedUserByEmail(s.session.OrgID.String(), email)
	require.NoError(s.t, err)
	require.False(s.t, user.DeletedAt.Valid)
}

func (s *oktaIntegrationSteps) assertUserReactivated(email string) {
	user, err := models.FindMaybeDeletedUserByEmail(s.session.OrgID.String(), email)
	require.NoError(s.t, err)
	require.False(s.t, user.DeletedAt.Valid)

	status, data := s.scimRequest(http.MethodGet, "/Users/"+user.ID.String(), nil)
	require.Equal(s.t, http.StatusOK, status, "unexpected status fetching SCIM user: %s", string(data))

	var resp struct {
		Active bool `json:"active"`
	}
	require.NoError(s.t, json.Unmarshal(data, &resp))
	require.True(s.t, resp.Active)
}

func (s *oktaIntegrationSteps) provisionGroup(groupName string) {
	body := map[string]any{
		"schemas":     []string{"urn:ietf:params:scim:schemas:core:2.0:Group"},
		"displayName": groupName,
	}

	status, data := s.scimRequest(http.MethodPost, "/Groups", body)
	require.Equal(s.t, http.StatusCreated, status, "unexpected status creating SCIM group: %s", string(data))

	var resp struct {
		ID          string `json:"id"`
		DisplayName string `json:"displayName"`
	}
	require.NoError(s.t, json.Unmarshal(data, &resp))
	require.Equal(s.t, groupName, resp.ID)
	require.Equal(s.t, groupName, resp.DisplayName)
}

func (s *oktaIntegrationSteps) assertGroupPersisted(groupName string) {
	metadata, err := models.FindGroupMetadata(groupName, models.DomainTypeOrganization, s.session.OrgID.String())
	require.NoError(s.t, err)
	require.Equal(s.t, groupName, metadata.GroupName)
	require.Equal(s.t, groupName, metadata.DisplayName)
}

func (s *oktaIntegrationSteps) addUserToGroup(groupName, userID string) {
	body := map[string]any{
		"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
		"Operations": []map[string]any{
			{
				"op": "add",
				"value": map[string]any{
					"members": []map[string]any{
						{"value": userID},
					},
				},
			},
		},
	}

	status, data := s.scimRequest(http.MethodPatch, "/Groups/"+groupName, body)
	require.Equal(s.t, http.StatusNoContent, status, "unexpected status adding user to SCIM group: %s", string(data))
}

func (s *oktaIntegrationSteps) removeUserFromGroup(groupName, userID string) {
	body := map[string]any{
		"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
		"Operations": []map[string]any{
			{
				"op": "remove",
				"value": map[string]any{
					"members": []map[string]any{
						{"value": userID},
					},
				},
			},
		},
	}

	status, data := s.scimRequest(http.MethodPatch, "/Groups/"+groupName, body)
	require.Equal(s.t, http.StatusNoContent, status, "unexpected status removing user from SCIM group: %s", string(data))
}

func (s *oktaIntegrationSteps) assertGroupHasMember(groupName, userID string) {
	status, data := s.scimRequest(http.MethodGet, "/Groups/"+groupName, nil)
	require.Equal(s.t, http.StatusOK, status, "unexpected status fetching SCIM group: %s", string(data))

	var resp struct {
		Members []struct {
			Value string `json:"value"`
		} `json:"members"`
	}
	require.NoError(s.t, json.Unmarshal(data, &resp))

	found := false
	for _, m := range resp.Members {
		if m.Value == userID {
			found = true
			break
		}
	}

	require.True(s.t, found, "expected user %s to be member of group %s", userID, groupName)
}

func (s *oktaIntegrationSteps) assertGroupHasNoMembers(groupName string) {
	status, data := s.scimRequest(http.MethodGet, "/Groups/"+groupName, nil)
	require.Equal(s.t, http.StatusOK, status, "unexpected status fetching SCIM group: %s", string(data))

	var resp struct {
		Members []struct {
			Value string `json:"value"`
		} `json:"members"`
	}
	require.NoError(s.t, json.Unmarshal(data, &resp))
	require.Len(s.t, resp.Members, 0)
}
