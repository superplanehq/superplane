package common

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
	mocks "github.com/superplanehq/superplane/test/support/mocks/github"
)

func TestVerifyCheckRunPermissions(t *testing.T) {
	key := testPrivateKeyPEM(t)
	for _, legacy := range []bool{true, false} {
		for _, tc := range []struct {
			name, permissions string
			status            int
			allowed           bool
		}{
			{"accepted", `{"checks":"write","pull_requests":"read"}`, 201, true},
			{"pull request write includes read", `{"checks":"write","pull_requests":"write"}`, 201, true},
			{"permission update not accepted", `{"checks":"read","pull_requests":"write"}`, 201, false},
			{"checks absent", `{"pull_requests":"write"}`, 201, false},
			{"pull requests absent", `{"checks":"write"}`, 201, false},
			{"GitHub unavailable", `{}`, 503, false},
		} {
			t.Run(fmt.Sprintf("legacy=%t/%s", legacy, tc.name), func(t *testing.T) {
				integration := mocks.IntegrationContextForLegacySetupFlow(key)
				if !legacy {
					integration = &contexts.IntegrationContext{
						NewSetupFlow:      true,
						CurrentProperties: map[string]any{PropertyAuthMethod: AuthMethodApp, PropertyOwner: "acme", PropertyOwnerType: OwnerTypeOrganization, PropertyAppID: "12345", PropertyAppInstallationID: "67890"},
						CurrentSecrets:    map[string]core.IntegrationSecret{SecretAppPEM: {Name: SecretAppPEM, Value: key}},
					}
				}
				httpCtx := &contexts.HTTPContext{Responses: []*http.Response{mocks.GitHubResponse(tc.status, fmt.Sprintf(`{"token":"test-token","expires_at":%q,"permissions":%s}`, time.Now().Add(time.Hour).Format(time.RFC3339), tc.permissions))}}
				err := VerifyCheckRunPermissions(t.Context(), integration, httpCtx)
				if tc.allowed {
					require.NoError(t, err)
				} else if tc.status == 201 {
					require.ErrorIs(t, err, ErrCheckRunPermissions)
				} else {
					require.Error(t, err)
					require.NotErrorIs(t, err, ErrCheckRunPermissions)
				}
				require.Len(t, httpCtx.Requests, 1, "verification must not publish a check")
				require.Equal(t, "/app/installations/67890/access_tokens", httpCtx.Requests[0].URL.Path)
			})
		}
	}
	t.Run("personal token cannot verify app permissions", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{}
		require.ErrorIs(t, VerifyCheckRunPermissions(t.Context(), mocks.IntegrationContextForNewSetupFlow(), httpCtx), ErrCheckRunAppRequired)
		require.Empty(t, httpCtx.Requests)
	})
}
