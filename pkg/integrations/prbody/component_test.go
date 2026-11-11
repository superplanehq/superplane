package prbody_test

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/bitbucket"
	_ "github.com/superplanehq/superplane/pkg/integrations/github"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/test/support/contexts"
	githubmocks "github.com/superplanehq/superplane/test/support/mocks/github"
)

func response(body string) *http.Response {
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}
}

// These cases exercise both provider protocols through their public action hooks.
// A shared marker test cannot catch use of rendered Markdown or a full PR update.
func TestIntegrationBodySectionPublication(t *testing.T) {
	for _, provider := range []string{"github", "bitbucket"} {
		for _, scenario := range []string{"publish", "changed commit", "closed", "disabled", "invalid markers", "unchanged"} {
			t.Run(provider+"/"+scenario, func(t *testing.T) {
				reg, err := registry.NewRegistry(nil, registry.HTTPOptions{})
				require.NoError(t, err)
				action, err := reg.GetAction(provider + ".updatePullRequestBodySection")
				require.NoError(t, err)
				original := "User text\r\n<!-- superplane:notes:start -->old<!-- superplane:notes:end -->\nFooter"
				expected := "User text\r\n<!-- superplane:notes:start -->\nNew report\n<!-- superplane:notes:end -->\nFooter"
				if scenario == "invalid markers" {
					original = "User text<!-- superplane:notes:start -->"
				}
				if scenario == "unchanged" {
					original = expected
				}
				state, sha := "open", "abc"
				if scenario == "closed" {
					state = "closed"
				}
				if scenario == "changed commit" {
					sha = "new"
				}
				payload := map[string]any{"state": state, "head": map[string]any{"sha": sha}, "body": original, "body_html": "rendered HTML"}
				integration := githubmocks.IntegrationContextForNewSetupFlow()
				method, field, path := http.MethodPatch, "body", "/repos/acme/app/pulls/42"
				if provider == "bitbucket" {
					payload = map[string]any{"state": strings.ToUpper(state), "source": map[string]any{"commit": map[string]any{"hash": sha}}, "description": original}
					integration = &contexts.IntegrationContext{Configuration: map[string]any{"token": "workspace-token"}, Metadata: bitbucket.Metadata{AuthType: bitbucket.AuthTypeWorkspaceAccessToken, Workspace: &bitbucket.WorkspaceMetadata{Slug: "acme"}}}
					method, field, path = http.MethodPut, "description", "/2.0/repositories/acme/app/pullrequests/42"
				}
				encoded, err := json.Marshal(payload)
				require.NoError(t, err)
				httpCtx := &contexts.HTTPContext{Responses: []*http.Response{response(string(encoded)), response(`{}`)}}
				config := map[string]any{"repository": "acme/app", "pullNumber": "42", "sectionKey": "notes", "body": "New report", "expectedHeadSha": "abc", "enabled": scenario != "disabled"}
				execution := &contexts.ExecutionStateContext{}
				err = action.HandleHook(core.ActionHookContext{Name: "updatePullRequestBodySection", Configuration: config, HTTP: httpCtx, Integration: integration, ExecutionState: execution})
				require.NoError(t, err)
				require.Len(t, execution.Payloads, 1)
				encoded, err = json.Marshal(execution.Payloads[0].(map[string]any)["data"])
				require.NoError(t, err)
				var outcome map[string]any
				require.NoError(t, json.Unmarshal(encoded, &outcome))
				if scenario != "publish" {
					expectedStatus := "skipped"
					if scenario == "unchanged" {
						expectedStatus = "unchanged"
					}
					assert.Equal(t, expectedStatus, outcome["status"])
					expectedCalls := 1
					if scenario == "disabled" {
						expectedCalls = 0
					}
					assert.Len(t, httpCtx.Requests, expectedCalls)
					return
				}
				assert.Equal(t, "published", outcome["status"])
				require.Len(t, httpCtx.Requests, 2)
				assert.Equal(t, path, httpCtx.Requests[0].URL.Path)
				assert.Equal(t, method, httpCtx.Requests[1].Method)
				body, err := io.ReadAll(httpCtx.Requests[1].Body)
				require.NoError(t, err)
				var update map[string]any
				require.NoError(t, json.Unmarshal(body, &update))
				assert.Equal(t, map[string]any{field: expected}, update)
			})
		}
	}
}

func TestIntegrationBodySectionQueuesWithoutFactory(t *testing.T) {
	reg, err := registry.NewRegistry(nil, registry.HTTPOptions{})
	require.NoError(t, err)
	for _, provider := range []string{"github", "bitbucket"} {
		t.Run(provider, func(t *testing.T) {
			action, err := reg.GetAction(provider + ".updatePullRequestBodySection")
			require.NoError(t, err)
			requests := &contexts.RequestContext{}
			err = action.Execute(core.ExecutionContext{RunID: uuid.New(), Configuration: map[string]any{"repository": "acme/app", "pullNumber": "42", "sectionKey": "notes", "body": "Report"}, Metadata: &contexts.MetadataContext{}, Requests: requests})
			require.NoError(t, err)
			assert.Equal(t, "updatePullRequestBodySection", requests.Action)
		})
	}
}
