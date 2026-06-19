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

func Test__DeletePackage__Execute(t *testing.T) {
	component := &DeletePackage{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusNoContent,
				Body:       io.NopCloser(strings.NewReader("")),
			},
		},
	}
	executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

	err := component.Execute(cloudsmithPackageExecutionContext(httpContext, executionState))

	require.NoError(t, err)
	assert.Equal(t, http.MethodDelete, httpContext.Requests[0].Method)
	assert.Equal(t, "https://api.cloudsmith.io/v1/packages/acme/production/pkg_123/", httpContext.Requests[0].URL.String())
	assert.Equal(t, PackageDeletedPayloadType, executionState.Type)
	assert.True(t, executionState.Passed)
}
