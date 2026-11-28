package sso

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestSCIMAuthorizationHeader(t *testing.T) {
	f := newFlow(t)
	conn := f.put(connectionInput(true))
	created, err := f.service.CreateSCIMToken(f.ctx(), f.r.Organization.ID, f.r.Account.ID, "Okta")
	require.NoError(t, err)
	secret := created.Secret

	org, err := models.CreateOrganization("Other "+uuid.NewString(), "Other")
	require.NoError(t, err)
	account, err := models.CreateAccount("Other", uuid.NewString()+"@example.com")
	require.NoError(t, err)
	_, err = f.service.PutConnection(f.ctx(), org.ID, account.ID, connectionInput(true))
	require.NoError(t, err)
	otherToken, err := f.service.CreateSCIMToken(f.ctx(), org.ID, account.ID, "Other")
	require.NoError(t, err)

	t.Run("accepts a bearer scheme in any case", func(t *testing.T) {
		response, fields := scimAuth(f, conn, "bEaReR "+secret, true)
		require.Equal(t, http.StatusOK, response.Code)
		require.NotContains(t, response.Body.String(), secret)
		require.NotContains(t, logText(fields), secret)
	})

	t.Run("accepts a token with no scheme", func(t *testing.T) {
		response, _ := scimAuth(f, conn, secret, true)
		require.Equal(t, http.StatusOK, response.Code)
	})

	t.Run("rejects a missing header", func(t *testing.T) {
		response, fields := scimAuth(f, conn, "", false)
		requireSCIMAuthFailure(t, response, fields, "missing", secret)
	})

	t.Run("rejects another scheme", func(t *testing.T) {
		response, fields := scimAuth(f, conn, "Basic "+secret, true)
		requireSCIMAuthFailure(t, response, fields, "scheme", secret)
	})

	t.Run("rejects an unknown token", func(t *testing.T) {
		response, fields := scimAuth(f, conn, "Bearer not-a-token", true)
		requireSCIMAuthFailure(t, response, fields, "unknown_token", secret)
	})

	t.Run("rejects a token for another connection", func(t *testing.T) {
		response, fields := scimAuth(f, conn, "Bearer "+otherToken.Secret, true)
		requireSCIMAuthFailure(t, response, fields, "connection_mismatch", otherToken.Secret)
	})

	t.Run("rejects a revoked token", func(t *testing.T) {
		require.NoError(t, f.service.RevokeSCIMToken(f.ctx(), f.r.Organization.ID, created.Token.ID))
		response, fields := scimAuth(f, conn, "Bearer "+secret, true)
		requireSCIMAuthFailure(t, response, fields, "revoked", secret)
	})
}

func scimAuth(f *flow, conn *models.SSOConnection, header string, include bool) (*httptest.ResponseRecorder, log.Fields) {
	request := httptest.NewRequest(http.MethodGet, "/scim/v2/"+conn.ID.String()+"/ResourceTypes", strings.NewReader(""))
	if include {
		request.Header.Set("Authorization", header)
	}
	fields := log.Fields{}
	request = request.WithContext(logging.WithRequestLogFields(request.Context(), fields))
	response := httptest.NewRecorder()
	f.service.SCIMHandler().ServeHTTP(response, request)
	return response, fields
}

func requireSCIMAuthFailure(t *testing.T, response *httptest.ResponseRecorder, fields log.Fields, reason, secret string) {
	t.Helper()
	require.Equal(t, http.StatusUnauthorized, response.Code)
	require.Contains(t, response.Body.String(), "The SCIM token is not valid.")
	require.Equal(t, reason, fields["scim_auth_failure"])
	require.NotContains(t, response.Body.String(), secret)
	require.NotContains(t, logText(fields), secret)
}

func logText(fields log.Fields) string {
	raw, err := json.Marshal(fields)
	if err != nil {
		return ""
	}
	return string(raw)
}
