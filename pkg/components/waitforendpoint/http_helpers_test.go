package waitforendpoint

import (
	"encoding/base64"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type secretsContext struct {
	values map[string][]byte
}

func (s *secretsContext) GetKey(secretName, keyName string) ([]byte, error) {
	value, ok := s.values[secretName+"/"+keyName]
	if !ok {
		return nil, core.ErrSecretKeyNotFound
	}
	return value, nil
}

func (s *secretsContext) GetSecretKeys(string) (map[string][]byte, error) {
	return nil, nil
}

func (s *secretsContext) GetIntegrationKeys(string) (map[string][]byte, error) {
	return nil, nil
}

func TestMatchesStatus(t *testing.T) {
	assert.True(t, MatchesStatus(http.StatusOK, "2xx"))
	assert.True(t, MatchesStatus(http.StatusNoContent, "200, 204"))
	assert.False(t, MatchesStatus(http.StatusNotFound, "2xx"))
}

func TestValidateStatusMatcher(t *testing.T) {
	require.NoError(t, ValidateStatusMatcher("2xx"))
	require.NoError(t, ValidateStatusMatcher("200,204"))
	assert.EqualError(t, ValidateStatusMatcher("20x"), "invalid HTTP status matcher: 20x")
	assert.EqualError(t, ValidateStatusMatcher("700"), "invalid HTTP status matcher: 700")
}

func TestValidateAuthorization(t *testing.T) {
	username := "deploy"
	password := configuration.SecretKeyRef{Secret: "credentials", Key: "password"}

	require.NoError(t, ValidateAuthorization(&AuthorizationSpec{
		Type:     AuthorizationTypeBasicAuth,
		Username: &username,
		Password: &password,
	}))
	assert.EqualError(
		t,
		ValidateAuthorization(&AuthorizationSpec{Type: AuthorizationTypeBasicAuth}),
		"authorization basic auth: username is required",
	)
	assert.EqualError(
		t,
		ValidateAuthorization(&AuthorizationSpec{Type: AuthorizationTypeCustomHeader}),
		"authorization custom header: header name is required",
	)
}

func TestApplyAuthorization(t *testing.T) {
	secrets := &secretsContext{
		values: map[string][]byte{
			"credentials/token":    []byte("token-value"),
			"credentials/password": []byte("password-value"),
		},
	}

	t.Run("bearer token", func(t *testing.T) {
		request, err := http.NewRequest(http.MethodGet, "https://example.com", nil)
		require.NoError(t, err)
		credential := configuration.SecretKeyRef{
			Secret: "credentials",
			Key:    "token",
		}

		header, err := ApplyAuthorization(secrets, &AuthorizationSpec{
			Type:       AuthorizationTypeBearer,
			Credential: &credential,
		}, request)
		require.NoError(t, err)
		assert.Equal(t, "Authorization", header)
		assert.Equal(t, "Bearer token-value", request.Header.Get("Authorization"))
	})

	t.Run("basic auth", func(t *testing.T) {
		request, err := http.NewRequest(http.MethodGet, "https://example.com", nil)
		require.NoError(t, err)
		username := "deploy"
		password := configuration.SecretKeyRef{
			Secret: "credentials",
			Key:    "password",
		}

		header, err := ApplyAuthorization(secrets, &AuthorizationSpec{
			Type:     AuthorizationTypeBasicAuth,
			Username: &username,
			Password: &password,
		}, request)
		require.NoError(t, err)
		assert.Equal(t, "Authorization", header)

		expected := "Basic " + base64.StdEncoding.EncodeToString([]byte("deploy:password-value"))
		assert.Equal(t, expected, request.Header.Get("Authorization"))
	})
}
