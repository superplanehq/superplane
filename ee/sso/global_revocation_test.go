package sso

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	jwtLib "github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/enterprise"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestGlobalTokenRevocationRejectsAnInvalidToken(t *testing.T) {
	f, fake, conn := signedInOIDCConnection(t)
	audience := revocationAudience(f, conn)
	body := issSubBody(t, fake.issuer+"/", fake.subject)

	otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	expired := globalClaims(fake.issuer, audience, "client-1")
	expired["exp"] = time.Now().Add(-3 * time.Minute).Unix()
	expired["iat"] = time.Now().Add(-4 * time.Minute).Unix()
	notYetValid := globalClaims(fake.issuer, audience, "client-1")
	notYetValid["iat"] = time.Now().Add(3 * time.Minute).Unix()
	notYetValid["nbf"] = time.Now().Add(3 * time.Minute).Unix()
	notYetValid["exp"] = time.Now().Add(10 * time.Minute).Unix()
	missingJTI := globalClaims(fake.issuer, audience, "client-1")
	delete(missingJTI, "jti")

	cases := []struct {
		name   string
		token  string
		raw    string
		detail string
	}{
		{name: "wrong audience", detail: "audience", token: signGlobalRevocation(t, fake.key, globalClaims(fake.issuer, "https://evil.example/logout", "client-1"), globalRevocationType)},
		{name: "wrong issuer", detail: "issuer", token: signGlobalRevocation(t, fake.key, globalClaims("https://evil.example", audience, "client-1"), globalRevocationType)},
		{name: "expired", detail: "expired", token: signGlobalRevocation(t, fake.key, expired, globalRevocationType)},
		{name: "not yet valid", detail: "not_yet_valid", token: signGlobalRevocation(t, fake.key, notYetValid, globalRevocationType)},
		{name: "bad signature", detail: "signature", token: signGlobalRevocation(t, otherKey, globalClaims(fake.issuer, audience, "client-1"), globalRevocationType)},
		{name: "refused type", detail: "type", token: signGlobalRevocation(t, fake.key, globalClaims(fake.issuer, audience, "client-1"), "logout+jwt")},
		{name: "missing jti", detail: "jti_missing", token: signGlobalRevocation(t, fake.key, missingJTI, globalRevocationType)},
		{name: "missing header", raw: ""},
		{name: "subject is not the client", detail: "subject", token: signGlobalRevocation(t, fake.key, globalClaims(fake.issuer, audience, "other-client"), globalRevocationType)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			header := tc.raw
			if tc.token != "" {
				header = "Bearer " + tc.token
			}
			recorder, fields := postGlobalRevocation(t, f, conn.ID, header, body)
			require.Equal(t, http.StatusUnauthorized, recorder.Code)
			require.Empty(t, recorder.Body.String())
			require.Equal(t, "auth", fields["sso_failure_reason"])
			if tc.detail == "" {
				_, logged := fields["sso_failure_detail"]
				require.False(t, logged)
			} else {
				require.Equal(t, tc.detail, fields["sso_failure_detail"])
			}
			requireNotRevoked(t, conn, fake.subject)
		})
	}
}

func TestGlobalTokenRevocationAcceptsCommonTokenTypes(t *testing.T) {
	f, fake, conn := signedInOIDCConnection(t)
	audience := revocationAudience(f, conn)
	body := issSubBody(t, fake.issuer+"/", fake.subject)
	for _, typ := range []string{"JWT", ""} {
		token := signGlobalRevocation(t, fake.key, globalClaims(fake.issuer, audience, "client-1"), typ)
		recorder, fields := postGlobalRevocation(t, f, conn.ID, "Bearer "+token, body)
		require.Equal(t, http.StatusNoContent, recorder.Code)
		require.Empty(t, recorder.Body.String())
		require.Equal(t, "success", fields["outcome"])
		_, logged := fields["sso_failure_detail"]
		require.False(t, logged)
	}
}

func TestGlobalTokenRevocationRejectsTheSubject(t *testing.T) {
	f, fake, conn := signedInOIDCConnection(t)
	token := signGlobalRevocation(t, fake.key, globalClaims(fake.issuer, revocationAudience(f, conn), "client-1"), globalRevocationType)

	t.Run("issuer in the body", func(t *testing.T) {
		recorder, fields := postGlobalRevocation(t, f, conn.ID, "Bearer "+token, issSubBody(t, "https://other.example", fake.subject))
		require.Equal(t, http.StatusBadRequest, recorder.Code)
		require.Empty(t, recorder.Body.String())
		require.Equal(t, "issuer", fields["sso_failure_reason"])
		requireNotRevoked(t, conn, fake.subject)
	})

	t.Run("unknown subject", func(t *testing.T) {
		recorder, fields := postGlobalRevocation(t, f, conn.ID, "Bearer "+token, issSubBody(t, fake.issuer, "missing-user"))
		require.Equal(t, http.StatusNotFound, recorder.Code)
		require.Empty(t, recorder.Body.String())
		require.Equal(t, "not_found", fields["outcome"])
		_, logged := fields["sso_failure_reason"]
		require.False(t, logged)
		require.Equal(t, false, fields["sso_logout_identity_found"])
		requireNotRevoked(t, conn, fake.subject)
	})

	t.Run("unknown format", func(t *testing.T) {
		recorder, fields := postGlobalRevocation(t, f, conn.ID, "Bearer "+token, `{"sub_id":{"format":"acct","id":"user-1"}}`)
		require.Equal(t, http.StatusBadRequest, recorder.Code)
		require.Equal(t, "subject_format", fields["sso_failure_reason"])
		requireNotRevoked(t, conn, fake.subject)
	})

	t.Run("body", func(t *testing.T) {
		recorder, fields := postGlobalRevocation(t, f, conn.ID, "Bearer "+token, "{")
		require.Equal(t, http.StatusBadRequest, recorder.Code)
		require.Equal(t, "body", fields["sso_failure_reason"])
		requireNotRevoked(t, conn, fake.subject)
	})
}

func TestGlobalTokenRevocationRevokesTheLinkedIdentity(t *testing.T) {
	f, fake, conn := signedInOIDCConnection(t)
	authAt := time.Now().Add(-2 * time.Hour)
	owner, err := models.FindActiveHumanUserByAccountAndOrganization(database.Conn(), conn.OrganizationID, f.r.Account.ID)
	require.NoError(t, err)
	cookie := enterprise.SSOAccessSession{
		AccountID: f.r.Account.ID,
		SSO:       conn.ID.String() + ":" + strconv.FormatInt(authAt.Unix(), 10),
	}
	open, err := f.service.CheckOrganizationAccess(f.ctx(), owner, cookie)
	require.NoError(t, err)
	require.Nil(t, open)

	token := signGlobalRevocation(t, fake.key, globalClaims(fake.issuer, revocationAudience(f, conn), "client-1"), globalRevocationType)
	recorder, fields := postGlobalRevocation(t, f, conn.ID, "Bearer "+token, issSubBody(t, fake.issuer+"/", fake.subject))
	require.Equal(t, http.StatusNoContent, recorder.Code)
	require.Empty(t, recorder.Body.String())
	require.Equal(t, "success", fields["outcome"])
	require.Equal(t, true, fields["sso_logout_identity_found"])
	require.NotContains(t, fieldsString(fields), token)

	identity, err := models.FindSSOIdentityByIDPSubject(database.Conn(), conn.ID, fake.subject)
	require.NoError(t, err)
	require.NotNil(t, identity.SessionsRevokedAt)
	revokedAt := *identity.SessionsRevokedAt

	required, err := f.service.CheckOrganizationAccess(f.ctx(), owner, cookie)
	require.NoError(t, err)
	require.NotNil(t, required)

	again, fields := postGlobalRevocation(t, f, conn.ID, "Bearer "+token, issSubBody(t, fake.issuer, fake.subject))
	require.Equal(t, http.StatusUnauthorized, again.Code)
	require.Equal(t, "replay", fields["sso_failure_reason"])
	saved, err := models.FindSSOIdentityByID(database.Conn(), conn.ID, identity.ID)
	require.NoError(t, err)
	require.True(t, saved.SessionsRevokedAt.Equal(revokedAt))
}

func TestGlobalTokenRevocationRevokesByEmail(t *testing.T) {
	f, fake, conn := signedInOIDCConnection(t)
	other, err := models.CreateAccount("Other", uuid.NewString()+"@example.com")
	require.NoError(t, err)
	second := &models.SSOIdentity{
		ID: uuid.New(), ConnectionID: conn.ID, OrganizationID: conn.OrganizationID, AccountID: other.ID,
		Email: "owner@example.com", Name: "Other", Active: true, ProvisionedBy: models.SSOProvisionedByJIT,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, database.Conn().Create(second).Error)

	token := signGlobalRevocation(t, fake.key, globalClaims(fake.issuer, revocationAudience(f, conn), "client-1"), globalRevocationType)
	body, err := json.Marshal(map[string]any{"sub_id": map[string]any{"format": "email", "email": "Owner@Example.com"}})
	require.NoError(t, err)
	recorder, _ := postGlobalRevocation(t, f, conn.ID, "Bearer "+token, string(body))
	require.Equal(t, http.StatusNoContent, recorder.Code)

	first, err := models.FindSSOIdentityByIDPSubject(database.Conn(), conn.ID, fake.subject)
	require.NoError(t, err)
	require.NotNil(t, first.SessionsRevokedAt)
	saved, err := models.FindSSOIdentityByID(database.Conn(), conn.ID, second.ID)
	require.NoError(t, err)
	require.NotNil(t, saved.SessionsRevokedAt)
	_, err = models.FindActiveUserByEmailInTransaction(database.Conn(), conn.OrganizationID.String(), "owner@example.com")
	require.NoError(t, err)
}

func signedInOIDCConnection(t *testing.T) (*flow, *fakeOIDC, *models.SSOConnection) {
	t.Helper()
	f, fake, conn := activeOIDCConnection(t)
	require.Equal(t, http.StatusFound, f.callback(t, conn, fake, callbackOptions{}).Code)
	return f, fake, conn
}

func revocationAudience(f *flow, conn *models.SSOConnection) string {
	return f.service.BaseURL + "/auth/sso/" + conn.ID.String() + "/global-token-revocation"
}

func globalClaims(issuer, audience, subject string) jwtLib.MapClaims {
	return jwtLib.MapClaims{
		"iss": issuer,
		"sub": subject,
		"aud": audience,
		"exp": time.Now().Add(5 * time.Minute).Unix(),
		"iat": time.Now().Unix(),
		"nbf": time.Now().Add(-5 * time.Minute).Unix(),
		"jti": uuid.NewString(),
	}
}

func signGlobalRevocation(t *testing.T, key *rsa.PrivateKey, claims jwtLib.MapClaims, typ string) string {
	t.Helper()
	token := jwtLib.NewWithClaims(jwtLib.SigningMethodRS256, claims)
	token.Header["kid"] = "test"
	if typ == "" {
		delete(token.Header, "typ")
	} else {
		token.Header["typ"] = typ
	}
	signed, err := token.SignedString(key)
	require.NoError(t, err)
	return signed
}

func issSubBody(t *testing.T, issuer, subject string) string {
	t.Helper()
	raw, err := json.Marshal(map[string]any{
		"sub_id": map[string]any{"format": "iss_sub", "iss": issuer, "sub": subject},
	})
	require.NoError(t, err)
	return string(raw)
}

func postGlobalRevocation(t *testing.T, f *flow, connectionID uuid.UUID, authorization, body string) (*httptest.ResponseRecorder, log.Fields) {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/auth/sso/"+connectionID.String()+"/global-token-revocation", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	if authorization != "" {
		request.Header.Set("Authorization", authorization)
	}
	fields := log.Fields{}
	request = request.WithContext(logging.WithRequestLogFields(request.Context(), fields))
	recorder := httptest.NewRecorder()
	f.service.AuthHandler().ServeHTTP(recorder, request)
	return recorder, fields
}

func requireNotRevoked(t *testing.T, conn *models.SSOConnection, subject string) {
	t.Helper()
	identity, err := models.FindSSOIdentityByIDPSubject(database.Conn(), conn.ID, subject)
	require.NoError(t, err)
	require.Nil(t, identity.SessionsRevokedAt)
}

func fieldsString(fields log.Fields) string {
	raw, err := json.Marshal(fields)
	if err != nil {
		return ""
	}
	return string(raw)
}

const globalRevocationType = "global-token-revocation+jwt"
