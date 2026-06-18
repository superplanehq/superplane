package cloudsmith

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__TagPackage__Setup(t *testing.T) {
	component := &TagPackage{}

	t.Run("requires tags unless action is Clear", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"repository": "acme/production",
				"package":    "pkg_123",
				"action":     TagActionAdd,
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "tags are required")
	})

	t.Run("allows Clear without tags", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"repository": "{{ $.event.repository }}",
				"package":    "pkg_123",
				"action":     TagActionClear,
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.NoError(t, err)
	})

	t.Run("rejects unknown action", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"repository": "acme/production",
				"package":    "pkg_123",
				"action":     "Append",
				"tags":       []string{"latest"},
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "action must be one of")
	})
}

func Test__TagPackage__Execute(t *testing.T) {
	component := &TagPackage{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"name": "billing-api",
					"slug_perm": "pkg_123",
					"version": "1.2.3",
					"tags": {"production": true}
				}`)),
			},
		},
	}
	executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

	err := component.Execute(cloudsmithPackageExecutionContextWithConfiguration(
		httpContext,
		executionState,
		map[string]any{
			"repository":  "acme/production",
			"package":     "pkg_123",
			"action":      TagActionReplace,
			"isImmutable": true,
			"tags":        []string{" production ", "stable"},
		},
	))

	require.NoError(t, err)
	assert.Equal(t, http.MethodPost, httpContext.Requests[0].Method)
	assert.Equal(t, "https://api.cloudsmith.io/v1/packages/acme/production/pkg_123/tag/", httpContext.Requests[0].URL.String())
	assert.Equal(t, PackageTaggedPayloadType, executionState.Type)

	body, err := io.ReadAll(httpContext.Requests[0].Body)
	require.NoError(t, err)

	var request PackageTagRequest
	require.NoError(t, json.Unmarshal(body, &request))
	assert.Equal(t, TagActionReplace, request.Action)
	assert.True(t, request.IsImmutable)
	assert.Equal(t, []string{"production", "stable"}, request.Tags)
}
