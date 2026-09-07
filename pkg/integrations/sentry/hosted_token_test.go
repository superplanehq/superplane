package sentry

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__installationTokenNeedsRefresh(t *testing.T) {
	original := tokenNow
	tokenNow = func() time.Time { return time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC) }
	t.Cleanup(func() { tokenNow = original })

	missing := &contexts.IntegrationContext{}
	assert.True(t, installationTokenNeedsRefresh(missing))

	fresh := &contexts.IntegrationContext{
		CurrentSecrets: map[string]core.IntegrationSecret{
			secretTokenExpiresAt: {Name: secretTokenExpiresAt, Value: []byte("2026-09-07T20:00:00Z")},
		},
	}
	assert.False(t, installationTokenNeedsRefresh(fresh))

	soon := &contexts.IntegrationContext{
		CurrentSecrets: map[string]core.IntegrationSecret{
			secretTokenExpiresAt: {Name: secretTokenExpiresAt, Value: []byte("2026-09-07T12:04:00Z")},
		},
	}
	assert.True(t, installationTokenNeedsRefresh(soon))
}

func Test__RefreshSentryAppAuthorization(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(
					`{"token":"new-token","refreshToken":"new-refresh","expiresAt":"2026-09-08T04:00:00.000Z"}`,
				)),
				Header: make(http.Header),
			},
		},
	}

	authorization, err := RefreshSentryAppAuthorization(httpCtx, HostedApp{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		BaseURL:      "https://sentry.io",
	}, "install-1")
	require.NoError(t, err)
	assert.Equal(t, "new-token", authorization.Token)
	assert.Equal(t, "new-refresh", authorization.RefreshToken)
	require.Len(t, httpCtx.Requests, 1)
	assert.Equal(t, "https://sentry.io/api/0/sentry-app-installations/install-1/authorizations/", httpCtx.Requests[0].URL.String())
	assert.Equal(t, "Bearer ", httpCtx.Requests[0].Header.Get("Authorization")[:7])
}

func Test__ensureInstallationToken_refreshesNearExpiry(t *testing.T) {
	original := tokenNow
	tokenNow = func() time.Time { return time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC) }
	t.Cleanup(func() { tokenNow = original })

	integration := &contexts.IntegrationContext{
		CurrentSecrets: map[string]core.IntegrationSecret{
			secretInstallationToken: {Name: secretInstallationToken, Value: []byte("old-token")},
			secretTokenExpiresAt:    {Name: secretTokenExpiresAt, Value: []byte("2026-09-07T12:02:00Z")},
		},
	}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(
					`{"token":"rotated-token","refreshToken":"rotated-refresh","expiresAt":"2026-09-08T04:00:00.000Z"}`,
				)),
				Header: make(http.Header),
			},
		},
	}

	token, err := ensureInstallationToken(httpCtx, integration, HostedApp{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		BaseURL:      "https://sentry.io",
	}, "install-1")
	require.NoError(t, err)
	assert.Equal(t, "rotated-token", token)
	assert.Equal(t, "rotated-token", string(integration.CurrentSecrets[secretInstallationToken].Value))
}
