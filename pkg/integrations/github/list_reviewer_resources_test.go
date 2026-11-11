package github

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestListReviewerResources(t *testing.T) {
	t.Run("lists eligible humans from every page for a personal repository", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"owner":{"login":"mytmlt","type":"User"}}`)),
			},
			{
				StatusCode: http.StatusOK,
				Header: http.Header{"Link": []string{
					`<https://api.github.com/repos/mytmlt/app/collaborators?page=2&per_page=100>; rel="next"`,
				}},
				Body: io.NopCloser(strings.NewReader(`[
					{"login":"reader","type":"User","permissions":{"pull":true}},
					{"login":"ci[bot]","type":"Bot","permissions":{"push":true}},
					{"login":"service[bot]","type":"User","permissions":{"push":true}},
					null
				]`)),
			},
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`[
					{"login":"Alex","type":"User","permissions":{"push":true}},
					{"login":"admin","type":"User","permissions":{"admin":true}},
					{"login":"maintainer","type":"User","permissions":{"maintain":true}},
					{"login":"ALEX","type":"User","permissions":{"push":true}}
				]`)),
			},
		}}

		resources, err := (&GitHub{}).ListResources("reviewer", core.ListResourcesContext{
			Integration: reviewerResourceIntegration(),
			HTTP:        httpCtx,
			Parameters:  map[string]string{"repository": "mytmlt/app"},
		})

		require.NoError(t, err)
		assert.Equal(t, []core.IntegrationResource{
			{Type: "reviewer", ID: "admin", Name: "admin"},
			{Type: "reviewer", ID: "Alex", Name: "Alex"},
			{Type: "reviewer", ID: "maintainer", Name: "maintainer"},
			{Type: "reviewer", ID: "mytmlt", Name: "mytmlt"},
		}, resources)
		require.Len(t, httpCtx.Requests, 3)
		assert.Equal(t, "/repos/mytmlt/app/collaborators", httpCtx.Requests[1].URL.Path)
		assert.Equal(t, "2", httpCtx.Requests[2].URL.Query().Get("page"))
	})

	t.Run("returns the GitHub error instead of a partial reviewer list", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"owner":{"login":"mytmlt","type":"User"}}`)),
			},
			{
				StatusCode: http.StatusOK,
				Header: http.Header{"Link": []string{
					`<https://api.github.com/repos/mytmlt/app/collaborators?page=2&per_page=100>; rel="next"`,
				}},
				Body: io.NopCloser(strings.NewReader(`[
					{"login":"mytmlt","type":"User","permissions":{"admin":true}}
				]`)),
			},
			{
				StatusCode: http.StatusForbidden,
				Body:       io.NopCloser(strings.NewReader(`{"message":"Resource not accessible by integration"}`)),
			},
		}}

		resources, err := (&GitHub{}).ListResources("reviewer", core.ListResourcesContext{
			Integration: reviewerResourceIntegration(),
			HTTP:        httpCtx,
			Parameters:  map[string]string{"repository": "mytmlt/app"},
		})

		require.Error(t, err)
		assert.Empty(t, resources)
		assert.Contains(t, err.Error(), "Resource not accessible by integration")
	})

	t.Run("needs a repository before contacting GitHub", func(t *testing.T) {
		resources, err := (&GitHub{}).ListResources("reviewer", core.ListResourcesContext{})

		require.NoError(t, err)
		assert.Empty(t, resources)
	})
}

func reviewerResourceIntegration() *contexts.IntegrationContext {
	return &contexts.IntegrationContext{
		NewSetupFlow: true,
		CurrentProperties: map[string]any{
			common.PropertyAuthMethod: common.AuthMethodPAT,
			common.PropertyOwner:      "mytmlt",
			common.PropertyOwnerType:  common.OwnerTypeUser,
		},
		CurrentSecrets: map[string]core.IntegrationSecret{
			common.SecretPAT: {Name: common.SecretPAT, Value: []byte("test-token")},
		},
	}
}
