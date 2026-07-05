package hetzner

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestListObjects_Execute_EmitsTruncatedFlag(t *testing.T) {
	xmlBody := `<?xml version="1.0" encoding="UTF-8"?>
<ListBucketResult>
  <IsTruncated>true</IsTruncated>
  <Contents>
    <Key>a.txt</Key>
    <Size>10</Size>
  </Contents>
</ListBucketResult>`

	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(xmlBody)),
			},
		},
	}
	integration := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"s3AccessKeyId":     "AKIAEXAMPLE",
			"s3SecretAccessKey": "secret",
			"s3Region":          "fsn1",
		},
	}
	executionState := &contexts.ExecutionStateContext{}

	component := &ListObjects{}
	err := component.Execute(core.ExecutionContext{
		Configuration:  map[string]any{"bucket": "my-bucket", "maxKeys": 1},
		HTTP:           httpCtx,
		Integration:    integration,
		ExecutionState: executionState,
	})
	require.NoError(t, err)
	require.Len(t, executionState.Payloads, 1)

	data := executionState.Payloads[0].(map[string]any)["data"].(map[string]any)
	require.Equal(t, true, data["truncated"])
}
