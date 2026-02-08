package github

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/go-github/v74/github"
	"github.com/stretchr/testify/require"
)

func Test__GetWorkflowUsage__fetchBillingUsageSummaryWithFallback__UsesTokenOnIntegration403(t *testing.T) {
	t.Parallel()

	var appCalls int
	var tokenCalls int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/organizations/test-org/settings/billing/usage/summary" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		auth := r.Header.Get("Authorization")
		if auth == "" {
			appCalls++
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"message":"Resource not accessible by integration"}`))
			return
		}

		tokenCalls++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"usageItems":[{"product":"Actions","unitType":"minutes","sku":"actions_linux","netQuantity":12.5}]}`))
	}))
	t.Cleanup(srv.Close)

	baseURL, err := url.Parse(srv.URL + "/")
	require.NoError(t, err)

	appClient := github.NewClient(srv.Client())
	appClient.BaseURL = baseURL

	tokenClient := NewTokenClient("PAT123")
	tokenClient.BaseURL = baseURL

	summary, err := fetchBillingUsageSummaryWithFallback(appClient, tokenClient, "test-org", "", nil, nil, nil, "Actions", "")
	require.NoError(t, err)
	require.NotNil(t, summary)
	require.Len(t, summary.UsageItems, 1)
	require.Equal(t, 1, appCalls)
	require.Equal(t, 1, tokenCalls)
}

func Test__GetWorkflowUsage__wrapBillingUsageSummaryError__GuidesToAccessTokenWhenMissing(t *testing.T) {
	t.Parallel()

	err := wrapBillingUsageSummaryError(&github.ErrorResponse{
		Response: &http.Response{StatusCode: http.StatusForbidden},
		Message:  "Resource not accessible by integration",
	}, false)

	require.Error(t, err)
	require.ErrorContains(t, err, GitHubAccessToken)
	require.ErrorContains(t, err, "not accessible")
}
