package cloudsmith

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__GetPackageVulnerabilities__Setup(t *testing.T) {
	component := &GetPackageVulnerabilities{}

	t.Run("missing repository returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{},
			Metadata:      &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "repository is required")
	})

	t.Run("missing package returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"repository": "acme/production",
			},
			Metadata: &contexts.MetadataContext{},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					okResponse(`{"name":"Production","slug":"production","namespace":"acme"}`),
				},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiKey": "test-key"},
			},
		})

		require.ErrorContains(t, err, "package is required")
	})

	t.Run("valid configuration resolves package metadata", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{}

		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"repository": "acme/production",
				"package":    "perm123",
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					okResponse(`{"name":"Production","slug":"production","namespace":"acme"}`),
					okResponse(`{"slug":"my-package-1-0-0","slug_perm":"perm123","name":"my-package","version":"1.0.0"}`),
				},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiKey": "test-key"},
			},
			Metadata: metadataCtx,
		})

		require.NoError(t, err)
		metadata, ok := metadataCtx.Metadata.(PackageNodeMetadata)
		require.True(t, ok)
		assert.Equal(t, "Production", metadata.RepositoryName)
		assert.Equal(t, "my-package 1.0.0", metadata.PackageName)
	})
}

func Test__GetPackageVulnerabilities__Execute(t *testing.T) {
	component := &GetPackageVulnerabilities{}

	scanResultsJSON := `[
		{
			"identifier": "1ceRAXarsZ93o5b7",
			"created_at": "2026-06-18T07:08:34.479287Z",
			"package": {
				"identifier": "YFf7Vw1SnOnK",
				"name": "hello-go-app",
				"version": "cd8e0196c8cfe78b87690ec03900b775c7823d32f09ec8f87f760411059de7e2",
				"url": "https://api.cloudsmith.io/v1/packages/acme/production/YFf7Vw1SnOnK/"
			},
			"scan_id": null,
			"has_vulnerabilities": true,
			"num_vulnerabilities": 27,
			"max_severity": "Critical"
		}
	]`

	t.Run("successful fetch returns most recent scan result", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"repository": "acme/production",
				"package":    "YFf7Vw1SnOnK",
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(scanResultsJSON)),
					},
				},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiKey": "test-key"},
			},
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, "default", executionState.Channel)
		assert.Equal(t, "cloudsmith.package.vulnerabilities", executionState.Type)
		require.Len(t, executionState.Payloads, 1)

		wrapped, ok := executionState.Payloads[0].(map[string]any)
		require.True(t, ok)
		result, ok := wrapped["data"].(VulnerabilityScanResult)
		require.True(t, ok)
		assert.Equal(t, "1ceRAXarsZ93o5b7", result.Identifier)
		assert.True(t, result.HasVulnerabilities)
		assert.Equal(t, 27, result.NumVulnerabilities)
		assert.Equal(t, "Critical", result.MaxSeverity)
		require.NotNil(t, result.Package)
		assert.Equal(t, "YFf7Vw1SnOnK", result.Package.Identifier)
		assert.Equal(t, "hello-go-app", result.Package.Name)
	})

	t.Run("no scan results emits empty result", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"repository": "acme/production",
				"package":    "YFf7Vw1SnOnK",
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(`[]`)),
					},
				},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiKey": "test-key"},
			},
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		require.Len(t, executionState.Payloads, 1)

		wrapped, ok := executionState.Payloads[0].(map[string]any)
		require.True(t, ok)
		result, ok := wrapped["data"].(VulnerabilityScanResult)
		require.True(t, ok)
		assert.Empty(t, result.Identifier)
		assert.False(t, result.HasVulnerabilities)
	})

	t.Run("invalid repository format returns error", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"repository": "no-namespace",
				"package":    "perm123",
			},
			HTTP:           &contexts.HTTPContext{},
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "test-key"}},
			ExecutionState: executionState,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid repository")
	})
}
