package admincli

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIClientUsesBearerToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"fleets":[]}`)
	}))
	defer server.Close()

	client := NewAPIClient(&ClientConfig{
		BaseURL:    server.URL,
		APIToken:   "test-token",
		HTTPClient: server.Client(),
	})
	_, _, err := client.RunnersAPI.RunnersListFleets(context.Background()).Execute()
	require.NoError(t, err)
}

func TestRedirectPolicyRejectsMethodChanges(t *testing.T) {
	policy := methodSafeRedirectPolicy()
	original := httptest.NewRequest(http.MethodPatch, "http://example.com/original", nil)
	redirect := httptest.NewRequest(http.MethodGet, "http://example.com/redirect", nil)

	err := policy(redirect, []*http.Request{original})
	require.EqualError(
		t,
		err,
		"refusing to follow redirect that changes method from PATCH to GET (original URL: http://example.com/original, redirect target: http://example.com/redirect)",
	)
}
