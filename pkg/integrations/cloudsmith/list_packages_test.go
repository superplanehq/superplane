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

func Test__ListPackages__Setup(t *testing.T) {
	component := &ListPackages{}

	t.Run("missing repository returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{},
			Metadata:      &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "repository is required")
	})

	t.Run("empty repository returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"repository": ""},
			Metadata:      &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "repository is required")
	})

	t.Run("expression repository is stored without API call", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{}

		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"repository": "{{ $.trigger.data.repository }}",
			},
			Metadata: metadataCtx,
		})

		require.NoError(t, err)
		metadata, ok := metadataCtx.Metadata.(RepositoryNodeMetadata)
		require.True(t, ok)
		assert.Equal(t, "{{ $.trigger.data.repository }}", metadata.RepositoryName)
	})

	t.Run("valid repository resolves metadata", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{}

		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"repository": "acme/production",
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					okResponse(`{"name":"Production","slug":"production","namespace":"acme"}`),
				},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiKey": "test-key"},
			},
			Metadata: metadataCtx,
		})

		require.NoError(t, err)
		metadata, ok := metadataCtx.Metadata.(RepositoryNodeMetadata)
		require.True(t, ok)
		assert.Equal(t, "Production", metadata.RepositoryName)
	})
}

func Test__ListPackages__Execute(t *testing.T) {
	component := &ListPackages{}

	pkg1JSON := `{"slug":"my-package-1-0-0","slug_perm":"perm1","name":"my-package","version":"1.0.0","format":"docker","status":2,"status_str":"Available","stage":9,"stage_str":"Fully Synchronised","is_quarantined":false,"security_scan_status":"No Vulnerabilities Found","size":52428800,"size_str":"50.0 MB","uploaded_at":"2026-01-01T10:00:00Z"}`
	pkg2JSON := `{"slug":"my-package-1-1-0","slug_perm":"perm2","name":"my-package","version":"1.1.0","format":"docker","status":2,"status_str":"Available","stage":9,"stage_str":"Fully Synchronised","is_quarantined":false,"security_scan_status":"No Vulnerabilities Found","size":54525952,"size_str":"52.0 MB","uploaded_at":"2026-01-15T14:30:00Z"}`
	packagesJSON := "[" + pkg1JSON + "," + pkg2JSON + "]"

	t.Run("successful list emits all packages", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"repository": "acme/production",
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(packagesJSON)),
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
		assert.Equal(t, "cloudsmith.packages.listed", executionState.Type)
		require.Len(t, executionState.Payloads, 1)
		wrapped, ok := executionState.Payloads[0].(map[string]any)
		require.True(t, ok)
		result, ok := wrapped["data"].(ListPackagesResult)
		require.True(t, ok)
		assert.Len(t, result.Packages, 2)
	})

	t.Run("empty repository returns error", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"repository": "no-namespace"},
			HTTP:           &contexts.HTTPContext{},
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "test-key"}},
			ExecutionState: executionState,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid repository")
	})

	t.Run("repository not found returns error", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{"repository": "acme/missing"},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusNotFound,
						Body:       io.NopCloser(strings.NewReader(`{"detail":"Not found."}`)),
					},
				},
			},
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "test-key"}},
			ExecutionState: executionState,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to list packages")
	})

	t.Run("fully_synchronised filter builds correct query", func(t *testing.T) {
		spec := ListPackagesSpec{SyncStatus: "fully_synchronised"}
		assert.Equal(t, "is_sync_completed:true", buildPackageQuery(spec))
	})

	t.Run("quarantined filter builds correct query", func(t *testing.T) {
		spec := ListPackagesSpec{QuarantineStatus: "quarantined"}
		assert.Equal(t, "is_quarantined:true", buildPackageQuery(spec))
	})

	t.Run("combined filters build AND query", func(t *testing.T) {
		spec := ListPackagesSpec{
			SyncStatus:          "fully_synchronised",
			QuarantineStatus:    "not_quarantined",
			VulnerabilityStatus: "no_vulnerabilities",
		}
		query := buildPackageQuery(spec)
		assert.Contains(t, query, "is_sync_completed:true")
		assert.Contains(t, query, "is_quarantined:false")
		assert.Contains(t, query, "No Vulnerabilities Found")
		assert.Contains(t, query, " AND ")
	})

	t.Run("any filters produce empty query", func(t *testing.T) {
		spec := ListPackagesSpec{
			SyncStatus:          "any",
			QuarantineStatus:    "any",
			VulnerabilityStatus: "any",
		}
		assert.Equal(t, "", buildPackageQuery(spec))
	})
}

func Test__ListPackages__ExampleOutput(t *testing.T) {
	output := (&ListPackages{}).ExampleOutput()
	require.NotNil(t, output)
	assert.Equal(t, "cloudsmith.packages.listed", output["type"])
	assert.NotEmpty(t, output["timestamp"])
	assert.NotNil(t, output["data"])
}
