package sso

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeDomain(t *testing.T) {
	domain, err := NormalizeDomain("  Example.COM. ")
	require.NoError(t, err)
	require.Equal(t, "example.com", domain)

	_, err = NormalizeDomain("gmail.com")
	require.Error(t, err)
	require.Contains(t, err.Error(), "public")

	_, err = NormalizeDomain("not a domain")
	require.Error(t, err)
}

func TestDiscoverDoesNotRevealUnknownUsers(t *testing.T) {
	service := New(Config{BaseURL: "http://localhost:8000"})
	request := httptest.NewRequest(http.MethodPost, "/auth/sso/discover", http.NoBody)
	request = request.WithContext(context.Background())
	response := httptest.NewRecorder()
	service.discover(response, request)
	require.Equal(t, http.StatusOK, response.Code)
	require.Contains(t, response.Body.String(), `"ssoAvailable":false`)
}
