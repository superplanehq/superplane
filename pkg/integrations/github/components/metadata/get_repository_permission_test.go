package metadata

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/go-github/v84/github"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__GetRepositoryPermission__Execute(t *testing.T) {
	component := GetRepositoryPermission{}

	t.Run("fails when configuration decode fails", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration:    &contexts.IntegrationContext{},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  "not a map",
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("fails when metadata decode fails", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Metadata: "not a valid metadata",
		}

		err := component.Execute(core.ExecutionContext{
			Integration:    integrationCtx,
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"repository": "hello",
				"username":   "octocat",
			},
		})

		require.ErrorContains(t, err, "failed to decode metadata")
	})
}

func TestGetRepositoryPermissionFallback(t *testing.T) {
	for _, tc := range []struct {
		name, fallback, author, actor string
		wantError                     bool
		wantPermission                string
		requests                      int
	}{
		{name: "single username", author: `{"permission":"read"}`, wantPermission: "read", requests: 1},
		{name: "actor approves", fallback: "maintainer", author: `{"permission":"none"}`, actor: `{"permission":"write"}`, wantPermission: "write", requests: 2},
		{name: "author sufficient", fallback: "maintainer", author: `{"permission":"read"}`, wantPermission: "read", requests: 1},
		{name: "author error actor approves", fallback: "maintainer", author: `invalid`, actor: `{"permission":"admin"}`, wantPermission: "admin", requests: 2},
		{name: "actor error without access", fallback: "maintainer", author: `{"permission":"none"}`, actor: `invalid`, wantError: true, requests: 2},
		{name: "neither has access", fallback: "maintainer", author: `{"permission":"none"}`, actor: `{"permission":"none"}`, wantPermission: "none", requests: 2},
		{name: "missing permission cannot approve", fallback: "maintainer", author: `{}`, actor: `{"permission":"none"}`, wantError: true, requests: 2},
		{name: "unknown permission cannot approve", author: `{"permission":"mystery"}`, wantError: true, requests: 1},
		{name: "normalized duplicate", fallback: " @OCTOCAT ", author: `{"permission":"none"}`, wantPermission: "none", requests: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			host := &contexts.HTTPContext{}
			for _, body := range []string{tc.author, tc.actor} {
				host.Responses = append(host.Responses, &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))})
			}
			state := &contexts.ExecutionStateContext{}
			err := (&GetRepositoryPermission{}).Execute(core.ExecutionContext{
				Configuration: map[string]any{"repository": "acme/app", "username": "octocat", "fallbackUsername": tc.fallback},
				Integration:   &contexts.IntegrationContext{NewSetupFlow: true, CurrentProperties: map[string]any{"authMethod": "pat", "owner": "acme", "ownerType": "Organization"}, CurrentSecrets: map[string]core.IntegrationSecret{"pat": {Name: "pat", Value: []byte("test-token")}}},
				HTTP:          host, ExecutionState: state,
			})
			if tc.wantError {
				require.Error(t, err)
				require.Empty(t, state.Payloads)
			} else {
				require.NoError(t, err)
				require.Len(t, state.Payloads, 1)
				permission := state.Payloads[0].(map[string]any)["data"].(*github.RepositoryPermissionLevel)
				require.Equal(t, tc.wantPermission, permission.GetPermission())
			}
			require.Len(t, host.Requests, tc.requests)
		})
	}
}
