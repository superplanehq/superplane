package sso

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	jwtLib "github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/enterprise"
	"github.com/superplanehq/superplane/pkg/licensing"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestLogoutRedirectAndBackchannel(t *testing.T) {
	f := newFlow(t)
	f.setOwnerEmail("owner@example.com")
	f.verifyExample()
	require.NoError(t, models.ApplyInstallationMetadata(database.Conn(), func(metadata *models.InstallationMetadata) {
		metadata.AllowPrivateNetworkAccess = true
	}))
	fake, _ := newFakeOIDC(t)
	input := connectionInput(true)
	input.Issuer = fake.issuer
	input.IDPLogoutEnabled = true
	conn := f.put(input)
	f.markTested(conn)
	_, err := f.service.ActivateConnection(f.ctx(), f.r.Organization.ID)
	require.NoError(t, err)
	conn, err = models.FindSSOConnectionByOrganization(database.Conn(), f.r.Organization.ID)
	require.NoError(t, err)

	response := f.callback(t, conn, fake, callbackOptions{})
	require.Equal(t, http.StatusFound, response.Code)

	session := enterprise.SSOAccessSession{
		AccountID: f.r.Account.ID,
		SSO: authentication.FormatSSOSessions([]authentication.SSOSession{{
			ConnectionID: conn.ID,
			AuthUnix:     time.Now().Unix(),
		}}),
	}
	target, ok := f.service.LogoutRedirect(f.ctx(), f.r.Account.ID, session)
	require.True(t, ok)
	require.Contains(t, target, fake.issuer+"/logout")
	require.Contains(t, target, "id_token_hint=")
	require.Contains(t, target, url.QueryEscape(f.service.BaseURL+"/login"))

	conn.IDPLogoutEnabled = false
	require.NoError(t, conn.Save(database.Conn()))
	_, ok = f.service.LogoutRedirect(f.ctx(), f.r.Account.ID, session)
	require.False(t, ok)

	conn.IDPLogoutEnabled = true
	require.NoError(t, conn.Save(database.Conn()))
	logoutToken := signLogoutToken(t, fake, jwtLib.MapClaims{
		"events": map[string]any{"http://schemas.openid.net/event/backchannel-logout": map[string]any{}},
	})
	recorder := postLogout(f, conn.ID, logoutToken)
	require.Equal(t, http.StatusOK, recorder.Code)
	identity, err := models.FindSSOIdentityBySubject(database.Conn(), conn.ID, fake.subject)
	require.NoError(t, err)
	require.NotNil(t, identity.SessionsRevokedAt)
	require.Equal(t, http.StatusOK, postLogout(f, conn.ID, logoutToken).Code)

	require.Equal(t, http.StatusBadRequest, postLogout(f, conn.ID, "not-a-token").Code)
}

func TestPendingSCIMIdentityReceivesSubject(t *testing.T) {
	f := newFlow(t)
	f.setOwnerEmail("owner@example.com")
	f.verifyExample()
	conn := f.put(connectionInput(true))
	f.markTested(conn)
	_, err := f.service.ActivateConnection(f.ctx(), f.r.Organization.ID)
	require.NoError(t, err)
	conn, err = models.FindSSOConnectionByOrganization(database.Conn(), f.r.Organization.ID)
	require.NoError(t, err)

	pending := &models.SSOIdentity{
		ID: uuid.New(), ConnectionID: conn.ID, OrganizationID: f.r.Organization.ID,
		AccountID: f.r.Account.ID, Email: "owner@example.com", Name: "Owner",
		Active: true, ProvisionedBy: models.SSOProvisionedBySCIM,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, database.Conn().Create(pending).Error)

	require.NoError(t, f.signIn(conn, "okta-subject", "owner@example.com"))
	found, err := models.FindSSOIdentityBySubject(database.Conn(), conn.ID, "okta-subject")
	require.NoError(t, err)
	require.Equal(t, pending.ID, found.ID)
	require.Equal(t, f.r.Account.ID, found.AccountID)
}

func TestExistingConnectionWorksWithoutTheGrant(t *testing.T) {
	f := newFlow(t)
	conn := f.put(connectionInput(true))
	f.service.Entitlements = licensing.Community

	view, err := f.service.GetConnection(f.ctx(), f.r.Organization.ID)
	require.NoError(t, err)
	require.Equal(t, conn.ID, view.ID)

	disabled, err := f.service.DisableConnection(f.ctx(), f.r.Organization.ID)
	require.NoError(t, err)
	require.Equal(t, models.SSOStatusDisabled, disabled.Status)

	_, err = f.service.PutConnection(f.ctx(), f.r.Organization.ID, f.r.Account.ID, connectionInput(true))
	require.Error(t, err)
	require.NoError(t, f.service.DeleteConnection(f.ctx(), f.r.Organization.ID))
}

func TestBackchannelLogoutKeepsAnInactiveIdentity(t *testing.T) {
	f, fake, conn := activeOIDCConnection(t)
	response := f.callback(t, conn, fake, callbackOptions{})
	require.Equal(t, http.StatusFound, response.Code)
	identity, err := models.FindSSOIdentityByIDPSubject(database.Conn(), conn.ID, fake.subject)
	require.NoError(t, err)
	require.NoError(t, database.Conn().Model(&models.SSOIdentity{}).Where("id = ?", identity.ID).Update("active", false).Error)

	logoutToken := signLogoutToken(t, fake, jwtLib.MapClaims{
		"events": map[string]any{"http://schemas.openid.net/event/backchannel-logout": map[string]any{}},
	})
	require.Equal(t, http.StatusOK, postLogout(f, conn.ID, logoutToken).Code)

	saved, err := models.FindSSOIdentityByID(database.Conn(), conn.ID, identity.ID)
	require.NoError(t, err)
	require.False(t, saved.Active)
	require.NotNil(t, saved.SessionsRevokedAt)
}

func TestBackchannelLogoutUsesTheIDTokenSubject(t *testing.T) {
	f := newFlow(t)
	f.setOwnerEmail("owner@example.com")
	f.verifyExample()
	require.NoError(t, models.ApplyInstallationMetadata(database.Conn(), func(metadata *models.InstallationMetadata) {
		metadata.AllowPrivateNetworkAccess = true
	}))
	fake, _ := newFakeOIDC(t)
	fake.subject = "s1"
	fake.extraClaims = map[string]any{"oid": "o1"}
	input := connectionInput(true)
	input.Issuer = fake.issuer
	input.IDPLogoutEnabled = true
	input.SubjectClaim = "oid"
	conn := f.put(input)
	f.markTested(conn)
	_, err := f.service.ActivateConnection(f.ctx(), f.r.Organization.ID)
	require.NoError(t, err)
	conn, err = models.FindSSOConnectionByOrganization(database.Conn(), f.r.Organization.ID)
	require.NoError(t, err)

	response := f.callback(t, conn, fake, callbackOptions{})
	require.Equal(t, http.StatusFound, response.Code)
	identity, err := models.FindSSOIdentityBySubject(database.Conn(), conn.ID, "o1")
	require.NoError(t, err)
	require.NotNil(t, identity.IDPSubject)
	require.Equal(t, "s1", *identity.IDPSubject)

	logoutToken := signLogoutToken(t, fake, jwtLib.MapClaims{
		"sid":    nil,
		"events": map[string]any{"http://schemas.openid.net/event/backchannel-logout": map[string]any{}},
	})
	require.Equal(t, http.StatusOK, postLogout(f, conn.ID, logoutToken).Code)
	saved, err := models.FindSSOIdentityByID(database.Conn(), conn.ID, identity.ID)
	require.NoError(t, err)
	require.NotNil(t, saved.SessionsRevokedAt)

	require.NoError(t, database.Conn().Model(&models.SSOIdentity{}).Where("id = ?", identity.ID).Update("sessions_revoked_at", nil).Error)
	bySession := signLogoutToken(t, fake, jwtLib.MapClaims{
		"sub":    "other",
		"events": map[string]any{"http://schemas.openid.net/event/backchannel-logout": map[string]any{}},
	})
	require.Equal(t, http.StatusOK, postLogout(f, conn.ID, bySession).Code)
	saved, err = models.FindSSOIdentityByID(database.Conn(), conn.ID, identity.ID)
	require.NoError(t, err)
	require.NotNil(t, saved.SessionsRevokedAt)
}

func activeOIDCConnection(t *testing.T) (*flow, *fakeOIDC, *models.SSOConnection) {
	t.Helper()
	f := newFlow(t)
	f.setOwnerEmail("owner@example.com")
	f.verifyExample()
	require.NoError(t, models.ApplyInstallationMetadata(database.Conn(), func(metadata *models.InstallationMetadata) {
		metadata.AllowPrivateNetworkAccess = true
	}))
	fake, _ := newFakeOIDC(t)
	input := connectionInput(true)
	input.Issuer = fake.issuer
	input.IDPLogoutEnabled = true
	conn := f.put(input)
	f.markTested(conn)
	_, err := f.service.ActivateConnection(f.ctx(), f.r.Organization.ID)
	require.NoError(t, err)
	conn, err = models.FindSSOConnectionByOrganization(database.Conn(), f.r.Organization.ID)
	require.NoError(t, err)
	return f, fake, conn
}

func signLogoutToken(t *testing.T, fake *fakeOIDC, extra jwtLib.MapClaims) string {
	t.Helper()
	claims := jwtLib.MapClaims{
		"iss": fake.issuer, "sub": fake.subject, "aud": "client-1", "sid": "sid-1",
		"exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Unix(),
	}
	for key, value := range extra {
		if value == nil {
			delete(claims, key)
			continue
		}
		claims[key] = value
	}
	token := jwtLib.NewWithClaims(jwtLib.SigningMethodRS256, claims)
	token.Header["kid"] = "test"
	signed, err := token.SignedString(fake.key)
	require.NoError(t, err)
	return signed
}

func postLogout(f *flow, connectionID uuid.UUID, token string) *httptest.ResponseRecorder {
	body := strings.NewReader("logout_token=" + url.QueryEscape(token))
	request := httptest.NewRequest(http.MethodPost, "/auth/sso/"+connectionID.String()+"/backchannel-logout", body)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()
	f.service.AuthHandler().ServeHTTP(recorder, request)
	return recorder
}
