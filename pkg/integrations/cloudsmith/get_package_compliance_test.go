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

const gplPackageBody = `{
	"name": "sp-compliance-gpl",
	"version": "1.0.0",
	"slug_perm": "f3XvJCI9ufJa",
	"format": "npm",
	"license": "GPL-3.0-only",
	"spdx_license": "GPL-3.0-only",
	"osi_approved": true,
	"policy_violated": false,
	"is_quarantined": true,
	"status_str": "Quarantined",
	"stage_str": "Fully Synchronised",
	"tags": {"version": ["latest"]},
	"self_html_url": "https://cloudsmith.io/~weskk/repos/superplane-compliance/packages/detail/npm/sp-compliance-gpl/1.0.0/"
}`

func Test__GetPackageCompliance__Setup(t *testing.T) {
	component := &GetPackageCompliance{}

	t.Run("missing repository returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"package": "f3XvJCI9ufJa"},
			Metadata:      &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "repository is required")
	})

	t.Run("missing package returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"repository": "weskk/superplane-compliance"},
			Metadata:      &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "package is required")
	})

	t.Run("expression values are stored without API call", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"repository": "weskk/superplane-compliance",
				"package":    "{{ $.trigger.data.slug_perm }}",
			},
			Metadata: metadataCtx,
		})
		require.NoError(t, err)
		metadata, ok := metadataCtx.Metadata.(PackageComplianceNodeMetadata)
		require.True(t, ok)
		assert.Equal(t, "{{ $.trigger.data.slug_perm }}", metadata.PackageName)
	})

	t.Run("valid values resolve metadata", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"repository": "weskk/superplane-compliance",
				"package":    "f3XvJCI9ufJa",
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(gplPackageBody))},
				},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiKey": "test-key"},
			},
			Metadata: metadataCtx,
		})
		require.NoError(t, err)
		metadata, ok := metadataCtx.Metadata.(PackageComplianceNodeMetadata)
		require.True(t, ok)
		assert.Equal(t, "sp-compliance-gpl", metadata.PackageName)
		assert.Equal(t, "1.0.0", metadata.Version)
	})
}

func Test__GetPackageCompliance__Execute(t *testing.T) {
	component := &GetPackageCompliance{}

	t.Run("successful fetch emits compliance data", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(gplPackageBody))},
			},
		}
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"repository": "weskk/superplane-compliance",
				"package":    "f3XvJCI9ufJa",
			},
			HTTP:           httpContext,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "test-key"}},
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, "default", executionState.Channel)
		assert.Equal(t, "cloudsmith.package.complianceFetched", executionState.Type)
		require.Len(t, executionState.Payloads, 1)

		wrapped, ok := executionState.Payloads[0].(map[string]any)
		require.True(t, ok)
		compliance, ok := wrapped["data"].(PackageCompliance)
		require.True(t, ok)
		assert.Equal(t, "GPL-3.0-only", compliance.License)
		assert.True(t, compliance.IsQuarantined)
		assert.Equal(t, "Quarantined", compliance.Status)
	})

	t.Run("invalid repository format returns error", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"repository": "no-namespace", "package": "f3XvJCI9ufJa"},
			HTTP:           &contexts.HTTPContext{},
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "test-key"}},
			ExecutionState: executionState,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid repository")
		assert.False(t, executionState.Passed)
	})

	t.Run("package not found (404) returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader(`{"detail":"Not found."}`))},
			},
		}
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"repository": "weskk/superplane-compliance", "package": "missing"},
			HTTP:           httpContext,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "test-key"}},
			ExecutionState: executionState,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get package")
		assert.False(t, executionState.Passed)
	})
}
