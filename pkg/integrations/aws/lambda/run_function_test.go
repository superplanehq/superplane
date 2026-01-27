package lambda

import (
	"encoding/base64"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__RunFunction__Setup(t *testing.T) {
	component := &RunFunction{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.AppInstallationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: "invalid",
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("missing function arn -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.AppInstallationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"functionArn": " "},
		})

		require.ErrorContains(t, err, "Function ARN is required")
	})

	t.Run("valid configuration -> stores metadata", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.AppInstallationContext{},
			Metadata:      metadata,
			Configuration: map[string]any{"functionArn": "  arn:aws:lambda:us-east-1:123:function:test  "},
		})

		require.NoError(t, err)
		stored, ok := metadata.Metadata.(RunFunctionMetadata)
		require.True(t, ok)
		assert.Equal(t, "arn:aws:lambda:us-east-1:123:function:test", stored.FunctionArn)
	})
}

func Test__RunFunction__Execute(t *testing.T) {
	component := &RunFunction{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Configuration:  "invalid",
			NodeMetadata:   &contexts.MetadataContext{},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Integration:    &contexts.AppInstallationContext{},
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("invalid metadata -> error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"payload": map[string]any{"hello": "world"}},
			NodeMetadata:   &contexts.MetadataContext{Metadata: "invalid"},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Integration:    &contexts.AppInstallationContext{},
		})

		require.ErrorContains(t, err, "failed to decode metadata")
	})

	t.Run("missing credentials -> error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"payload": map[string]any{"hello": "world"}},
			NodeMetadata:   &contexts.MetadataContext{Metadata: RunFunctionMetadata{FunctionArn: "arn:aws:lambda:us-east-1:123:function:test"}},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Integration: &contexts.AppInstallationContext{
				Configuration: map[string]any{"region": "us-east-1"},
				Secrets:       map[string]core.IntegrationSecret{},
			},
		})

		require.ErrorContains(t, err, "AWS session credentials are missing")
	})

	t.Run("region from arn -> emits payloadRaw", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("not-json")),
					Header:     http.Header{"X-Amzn-Requestid": []string{"req-123"}},
				},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"payload": map[string]any{"hello": "world"}},
			NodeMetadata:   &contexts.MetadataContext{Metadata: RunFunctionMetadata{FunctionArn: "arn:aws:lambda:us-west-2:123:function:test"}},
			ExecutionState: execState,
			HTTP:           httpContext,
			Integration: &contexts.AppInstallationContext{
				Configuration: map[string]any{"region": " "},
				Secrets: map[string]core.IntegrationSecret{
					"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
					"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
					"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
				},
			},
		})

		require.NoError(t, err)
		require.Len(t, execState.Payloads, 1)
		payload := execState.Payloads[0].(map[string]any)["data"].(map[string]any)
		assert.Equal(t, "req-123", payload["requestId"])
		assert.Equal(t, "not-json", payload["payloadRaw"])

		require.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "https://lambda.us-west-2.amazonaws.com")
	})

	t.Run("successful invocation -> emits payload and report", func(t *testing.T) {
		logText := strings.Join([]string{
			"START RequestId: req-123 Version: $LATEST",
			"REPORT RequestId: req-123\tDuration: 89.81 ms\tBilled Duration: 100 ms\tMemory Size: 128 MB\tMax Memory Used: 82 MB\tInit Duration: 160.97 ms",
		}, "\n")

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"message":"ok"}`)),
					Header: http.Header{
						"X-Amzn-Requestid":     []string{"req-123"},
						"X-Amz-Log-Result":     []string{base64.StdEncoding.EncodeToString([]byte(logText))},
						"X-Amz-Function-Error": []string{""},
					},
				},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"payload": map[string]any{"hello": "world"}},
			NodeMetadata:   &contexts.MetadataContext{Metadata: RunFunctionMetadata{FunctionArn: "arn:aws:lambda:us-east-1:123:function:test"}},
			ExecutionState: execState,
			HTTP:           httpContext,
			Integration: &contexts.AppInstallationContext{
				Configuration: map[string]any{"region": "us-east-1"},
				Secrets: map[string]core.IntegrationSecret{
					"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
					"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
					"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
				},
			},
		})

		require.NoError(t, err)
		require.Len(t, execState.Payloads, 1)
		payload := execState.Payloads[0].(map[string]any)["data"].(map[string]any)
		assert.Equal(t, "req-123", payload["requestId"])
		assert.Equal(t, map[string]any{"message": "ok"}, payload["payload"])

		report, ok := payload["report"].(*LambdaLogReport)
		require.True(t, ok)
		assert.Equal(t, "89.81 ms", report.Duration)
		assert.Equal(t, "100 ms", report.BilledDuration)
		assert.Equal(t, "128 MB", report.MemorySize)
		assert.Equal(t, "82 MB", report.MaxMemoryUsed)
		assert.Equal(t, "160.97 ms", report.InitDuration)
	})

	t.Run("function error -> returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"errorType":"Boom","errorMessage":"failed"}`)),
					Header:     http.Header{"X-Amz-Function-Error": []string{"Unhandled"}},
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"payload": map[string]any{"hello": "world"}},
			NodeMetadata:   &contexts.MetadataContext{Metadata: RunFunctionMetadata{FunctionArn: "arn:aws:lambda:us-east-1:123:function:test"}},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			HTTP:           httpContext,
			Integration: &contexts.AppInstallationContext{
				Configuration: map[string]any{"region": "us-east-1"},
				Secrets: map[string]core.IntegrationSecret{
					"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
					"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
					"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
				},
			},
		})

		require.ErrorContains(t, err, "Lambda function error: Boom: failed")
	})
}

func Test__ResolveLambdaRegion(t *testing.T) {
	t.Run("app region wins", func(t *testing.T) {
		region, err := resolveLambdaRegion("us-east-1", "arn:aws:lambda:us-west-2:123:function:test")
		require.NoError(t, err)
		assert.Equal(t, "us-east-1", region)
	})

	t.Run("region from arn", func(t *testing.T) {
		region, err := resolveLambdaRegion("", "arn:aws:lambda:us-west-2:123:function:test")
		require.NoError(t, err)
		assert.Equal(t, "us-west-2", region)
	})

	t.Run("missing region -> error", func(t *testing.T) {
		_, err := resolveLambdaRegion("", "")
		require.ErrorContains(t, err, "region is required")
	})
}

func Test__ParseLambdaLogReport(t *testing.T) {
	t.Run("empty log result -> error", func(t *testing.T) {
		_, err := parseLambdaLogReport("")
		require.ErrorContains(t, err, "log result is empty")
	})

	t.Run("invalid base64 -> error", func(t *testing.T) {
		_, err := parseLambdaLogReport("not-base64")
		require.ErrorContains(t, err, "failed to decode log result")
	})

	t.Run("missing report line -> error", func(t *testing.T) {
		logText := "START RequestId: req-123 Version: $LATEST"
		_, err := parseLambdaLogReport(base64.StdEncoding.EncodeToString([]byte(logText)))
		require.ErrorContains(t, err, "no report found in log result")
	})

	t.Run("valid report -> parsed", func(t *testing.T) {
		logText := "REPORT RequestId: req-123\tDuration: 3 ms\tBilled Duration: 4 ms\tMemory Size: 128 MB\tMax Memory Used: 64 MB\tInit Duration: 2 ms"
		report, err := parseLambdaLogReport(base64.StdEncoding.EncodeToString([]byte(logText)))
		require.NoError(t, err)
		assert.Equal(t, "3 ms", report.Duration)
		assert.Equal(t, "4 ms", report.BilledDuration)
		assert.Equal(t, "128 MB", report.MemorySize)
		assert.Equal(t, "64 MB", report.MaxMemoryUsed)
		assert.Equal(t, "2 ms", report.InitDuration)
	})
}
