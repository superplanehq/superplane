package codeartifact

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestDeletePackageVersions_Setup(t *testing.T) {
	component := &DeletePackageVersions{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: "invalid",
		})
		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("missing versions -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: &contexts.IntegrationContext{},
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"region": "us-east-1", "domain": "d", "repository": "r", "format": "npm", "package": "pkg",
			},
		})
		require.ErrorContains(t, err, "at least one version is required")
	})

	t.Run("valid configuration -> success", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: &contexts.IntegrationContext{},
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"region": "us-east-1", "domain": "d", "repository": "r", "format": "npm", "package": "pkg",
				"versions": "1.0.0",
			},
		})
		require.NoError(t, err)
	})
}

func TestDeletePackageVersions_Execute(t *testing.T) {
	component := &DeletePackageVersions{}

	t.Run("missing credentials -> error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region": "us-east-1", "domain": "d", "repository": "r", "format": "npm", "package": "pkg",
				"versions": "1.0.0",
			},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Integration:    &contexts.IntegrationContext{Secrets: map[string]core.IntegrationSecret{}},
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "credentials")
	})

	t.Run("success -> emits result", func(t *testing.T) {
		resp := map[string]any{
			"successfulVersions": map[string]any{"1.0.0": map[string]any{"revision": "rev1", "status": "Deleted"}},
			"failedVersions":     map[string]any{},
		}
		body, _ := json.Marshal(resp)
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(string(body))),
			}},
		}
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region": "us-east-1", "domain": "d", "repository": "r", "format": "npm", "package": "pkg",
				"versions": "1.0.0",
			},
			ExecutionState: execState,
			HTTP:           httpContext,
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
		require.Equal(t, "aws.codeartifact.packageVersions", execState.Type)
	})
}
