package e2e

import (
	"bytes"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/beevik/etree"
	dsig "github.com/russellhaering/goxmldsig"
	saml2 "github.com/russellhaering/gosaml2"
	"github.com/russellhaering/gosaml2/types"
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

		steps.start()
		steps.login()
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

		steps.start()
		steps.login()
		steps.visitOktaSettingsPage()
		steps.fillSAMLSettings(samlIssuer, samlCertificate)
		steps.saveSettings()
		steps.assertOktaConfigPersisted(samlIssuer, samlCertificate, false)

		steps.rotateSCIMToken()
		steps.assertSCIMTokenPersisted()
	})

	t.Run("adding users via SCIM", func(t *testing.T) {
		const email = "okta-user@example.com"

		steps.start()
		steps.enableSCIM()
		steps.configureOkta()

		userID := steps.provisionUser(email)
		require.NotEmpty(t, userID)
		steps.assertUserPersisted(email)
		steps.assertUserFoundByFilter(email)
	})

	t.Run("deleting users via SCIM", func(t *testing.T) {
		const email = "okta-user-delete@example.com"

		steps.start()
		steps.enableSCIM()
		steps.configureOkta()

		userID := steps.provisionUser(email)
		require.NotEmpty(t, userID)

		steps.deactivateUser(userID, email)
		steps.assertUserDeactivated(email)
	})

	t.Run("provisioning groups and memberships", func(t *testing.T) {
		const email = "okta-group-user@example.com"
		const groupName = "okta-e2e-group"

		steps.start()
		steps.enableSCIM()
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

	t.Run("saml login via okta", func(t *testing.T) {
		steps.start()

		baseURL := os.Getenv("BASE_URL")
		require.NotEmpty(t, baseURL, "BASE_URL must be set for e2e tests")

		orgID := steps.session.OrgID.String()
		email := steps.session.Account.Email

		issuer := "http://example.okta.com/app/superplane/e2e"

		// Generate a key pair and self-signed certificate for signing.
		keyStore := dsig.RandomKeyStoreForTest()
		_, certDER, err := keyStore.GetKeyPair()
		require.NoError(t, err)

		cert, err := x509.ParseCertificate(certDER)
		require.NoError(t, err)

		pemCert := pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: cert.Raw,
		})

		oktaConfig := &models.OrganizationOktaConfig{
			OrganizationID: steps.session.OrgID,
			SamlIssuer:     issuer,
			SamlCertificate: string(pemCert),
		}

		err = models.SaveOrganizationOktaConfig(oktaConfig)
		require.NoError(t, err)

		acsURL := fmt.Sprintf("%s/orgs/%s/okta/auth", baseURL, orgID)

		samlResponse := buildTestSAMLResponse(t, acsURL, issuer, email, keyStore)

		client := &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				// Do not follow redirects so we can inspect the initial Set-Cookie + Location.
				return http.ErrUseLastResponse
			},
		}

		form := url.Values{}
		form.Set("SAMLResponse", samlResponse)

		resp, err := client.PostForm(acsURL, form)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusTemporaryRedirect, resp.StatusCode)

		location, err := resp.Location()
		require.NoError(t, err)
		require.Contains(t, location.Path, "/"+orgID)

		foundCookie := false
		for _, c := range resp.Cookies() {
			if c.Name == "account_token" && c.Value != "" {
				foundCookie = true
				break
			}
		}
		require.True(t, foundCookie, "expected account_token cookie to be set after SAML login")
	})
}

type oktaIntegrationSteps struct {
	t         *testing.T
	session   *session.TestSession
	scimToken string
}

func (s *oktaIntegrationSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
}

func (s *oktaIntegrationSteps) login() {
	s.session.Login()
}

func (s *oktaIntegrationSteps) enableSCIM() {
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

func buildTestSAMLResponse(t *testing.T, acsURL, issuer, email string, keyStore dsig.X509KeyStore) string {
	t.Helper()

	now := time.Now().UTC().Truncate(time.Second)

	notBefore := now.Add(-5 * time.Minute).Format(time.RFC3339)
	notOnOrAfter := now.Add(5 * time.Minute).Format(time.RFC3339)

	responseID := "_e2e-response"
	assertionID := "_e2e-assertion"

	resp := &types.Response{
		ID:           responseID,
		Destination:  acsURL,
		Version:      "2.0",
		IssueInstant: now,
		Status: &types.Status{
			StatusCode: &types.StatusCode{
				Value: saml2.StatusCodeSuccess,
			},
		},
		Issuer: &types.Issuer{
			Value: issuer,
		},
		Assertions: []types.Assertion{
			{
				Version:      "2.0",
				ID:           assertionID,
				IssueInstant: now,
				Issuer: &types.Issuer{
					Value: issuer,
				},
				Subject: &types.Subject{
					NameID: &types.NameID{
						Value: email,
					},
					SubjectConfirmation: &types.SubjectConfirmation{
						Method: saml2.SubjMethodBearer,
						SubjectConfirmationData: &types.SubjectConfirmationData{
							NotOnOrAfter: notOnOrAfter,
							Recipient:    acsURL,
						},
					},
				},
				Conditions: &types.Conditions{
					NotBefore:    notBefore,
					NotOnOrAfter: notOnOrAfter,
					AudienceRestrictions: []types.AudienceRestriction{
						{
							Audiences: []types.Audience{
								{Value: acsURL},
							},
						},
					},
				},
				AttributeStatement: &types.AttributeStatement{
					Attributes: []types.Attribute{
						{
							FriendlyName: "email",
							Name:         "email",
							NameFormat:   "urn:oasis:names:tc:SAML:2.0:attrname-format:basic",
							Values: []types.AttributeValue{
								{
									Type:  "xs:string",
									Value: email,
								},
							},
						},
					},
				},
			},
		},
	}

	rawXML, err := xml.Marshal(resp)
	require.NoError(t, err)

	doc := etree.NewDocument()
	err = doc.ReadFromBytes(rawXML)
	require.NoError(t, err)

	el := doc.Root()
	if el.SelectAttrValue("ID", "") == "" {
		el.CreateAttr("ID", responseID)
	}

	signingCtx := dsig.NewDefaultSigningContext(keyStore)
	signedEl, err := signingCtx.SignEnveloped(el)
	require.NoError(t, err)

	signedDoc := etree.NewDocument()
	signedDoc.SetRoot(signedEl)

	signedBytes, err := signedDoc.WriteToBytes()
	require.NoError(t, err)

	return base64.StdEncoding.EncodeToString(signedBytes)
}
