package cloudsmith

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__ResyncPackage__Execute(t *testing.T) {
	component := &ResyncPackage{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"name": "billing-api",
					"display_name": "billing-api",
					"slug_perm": "pkg_123",
					"version": "1.2.3",
					"format": "docker"
				}`)),
			},
		},
	}
	executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

	err := component.Execute(cloudsmithPackageExecutionContext(httpContext, executionState))

	require.NoError(t, err)
	assert.Equal(t, http.MethodPost, httpContext.Requests[0].Method)
	assert.Equal(t, "https://api.cloudsmith.io/v1/packages/acme/production/pkg_123/resync/", httpContext.Requests[0].URL.String())
	assert.Equal(t, PackageResyncedPayloadType, executionState.Type)
	assert.True(t, executionState.Passed)
}
