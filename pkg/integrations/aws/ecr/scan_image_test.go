package ecr

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

func Test__ScanImage__Setup(t *testing.T) {
	component := &ScanImage{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: "invalid",
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("missing region -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":     " ",
				"repository": "backend",
				"imageTag":   "latest",
			},
		})

		require.ErrorContains(t, err, "region is required")
	})

	t.Run("missing repository -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":   "us-east-1",
				"imageTag": "latest",
			},
		})

		require.ErrorContains(t, err, "repository is required")
	})

	t.Run("missing image digest and tag -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":     "us-east-1",
				"repository": "backend",
			},
		})

		require.ErrorContains(t, err, "image digest or image tag is required")
	})

	t.Run("valid configuration -> ok", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":     "us-east-1",
				"repository": "backend",
				"imageTag":   "latest",
			},
		})

		require.NoError(t, err)
	})
}

func Test__ScanImage__Execute(t *testing.T) {
	component := &ScanImage{}

	t.Run("missing credentials -> error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":     "us-east-1",
				"repository": "backend",
				"imageTag":   "latest",
			},
			Integration:    &contexts.IntegrationContext{Secrets: map[string]core.IntegrationSecret{}},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		})

		require.ErrorContains(t, err, "AWS session credentials are missing")
	})

	t.Run("scan in progress -> schedules poll action", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"scanStatus": {"status": "IN_PROGRESS"},
							"imageId": {"imageDigest": "sha256:abc"},
							"repositoryName": "backend"
						}
					`)),
				},
			},
		}

		metadata := &contexts.MetadataContext{}
		requests := &contexts.RequestContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":     "us-east-1",
				"repository": "backend",
				"imageTag":   "latest",
			},
			HTTP:           httpContext,
			Metadata:       metadata,
			Requests:       requests,
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Integration: &contexts.IntegrationContext{
				Secrets: map[string]core.IntegrationSecret{
					"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
					"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
					"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
				},
			},
		})

		require.NoError(t, err)
		stored, ok := metadata.Metadata.(ScanImageMetadata)
		require.True(t, ok)
		assert.Equal(t, "us-east-1", stored.Region)
		assert.Equal(t, "backend", stored.Repository)
		assert.Equal(t, "", stored.ImageDigest)

		assert.Equal(t, "pollFindings", requests.Action)
		assert.Equal(t, time.Second*10, requests.Duration)
	})

	t.Run("scan complete -> emits findings", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"scanStatus": {"status": "COMPLETE"},
							"imageId": {"imageDigest": "sha256:abc"},
							"repositoryName": "backend"
						}
					`)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"imageScanStatus": {"status": "COMPLETE"},
							"imageScanFindings": {"findingSeverityCounts": {"HIGH": 1}}
						}
					`)),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":     "us-east-1",
				"repository": "backend",
				"imageTag":   "latest",
			},
			HTTP:           httpContext,
			ExecutionState: execState,
			Integration: &contexts.IntegrationContext{
				Secrets: map[string]core.IntegrationSecret{
					"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
					"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
					"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
				},
			},
		})

		require.NoError(t, err)
		require.Len(t, execState.Payloads, 1)
		payload := execState.Payloads[0].(map[string]any)["data"]
		findings, ok := payload.(*DescribeImageScanFindingsResponse)
		require.True(t, ok)
		assert.Equal(t, "COMPLETE", findings.ImageScanStatus.Status)

		require.Len(t, httpContext.Requests, 2)
		assert.Equal(t, "https://api.ecr.us-east-1.amazonaws.com/", httpContext.Requests[0].URL.String())
		assert.Equal(t, "https://api.ecr.us-east-1.amazonaws.com/", httpContext.Requests[1].URL.String())
	})
}

func Test__ScanImage__HandleAction(t *testing.T) {
	component := &ScanImage{}

	t.Run("unknown action -> error", func(t *testing.T) {
		err := component.HandleAction(core.ActionContext{
			Name: "unknown",
		})

		require.ErrorContains(t, err, "unknown action")
	})

	t.Run("scan still in progress -> schedules poll", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"imageScanStatus": {"status": "IN_PROGRESS"}
						}
					`)),
				},
			},
		}

		requests := &contexts.RequestContext{}
		err := component.HandleAction(core.ActionContext{
			Name:     "pollFindings",
			HTTP:     httpContext,
			Requests: requests,
			Metadata: &contexts.MetadataContext{
				Metadata: ScanImageMetadata{
					Region:      "us-east-1",
					Repository:  "backend",
					ImageDigest: "sha256:abc",
				},
			},
			Integration: &contexts.IntegrationContext{
				Secrets: map[string]core.IntegrationSecret{
					"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
					"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
					"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
				},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, "pollFindings", requests.Action)
		assert.Equal(t, time.Second*10, requests.Duration)
	})

	t.Run("scan complete -> emits findings", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"imageScanStatus": {"status": "COMPLETE"},
							"imageScanFindings": {"findingSeverityCounts": {"HIGH": 1}}
						}
					`)),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.HandleAction(core.ActionContext{
			Name:           "pollFindings",
			HTTP:           httpContext,
			ExecutionState: execState,
			Metadata: &contexts.MetadataContext{
				Metadata: ScanImageMetadata{
					Region:      "us-east-1",
					Repository:  "backend",
					ImageDigest: "sha256:abc",
				},
			},
			Integration: &contexts.IntegrationContext{
				Secrets: map[string]core.IntegrationSecret{
					"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
					"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
					"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
				},
			},
		})

		require.NoError(t, err)
		require.Len(t, execState.Payloads, 1)
		payload := execState.Payloads[0].(map[string]any)["data"]
		findings, ok := payload.(*DescribeImageScanFindingsResponse)
		require.True(t, ok)
		assert.Equal(t, "COMPLETE", findings.ImageScanStatus.Status)
	})
}
