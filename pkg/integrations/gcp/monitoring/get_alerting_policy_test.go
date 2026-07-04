package monitoring

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	gcpcommon "github.com/superplanehq/superplane/pkg/integrations/gcp/common"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__GetAlertingPolicy__Setup(t *testing.T) {
	g := &GetAlertingPolicy{}

	t.Run("missing alertPolicy", func(t *testing.T) {
		require.ErrorContains(t, g.Setup(core.SetupContext{Configuration: map[string]any{}, Metadata: &contexts.MetadataContext{}}), "alertPolicy is required")
	})

	t.Run("expression accepted", func(t *testing.T) {
		require.NoError(t, g.Setup(core.SetupContext{
			Configuration: map[string]any{"alertPolicy": "{{ $.nodes.create.outputs.default[0].data.name }}"},
			Metadata:      &contexts.MetadataContext{},
		}))
	})

	t.Run("valid name accepted", func(t *testing.T) {
		require.NoError(t, g.Setup(core.SetupContext{
			Configuration: map[string]any{"alertPolicy": "projects/my-project/alertPolicies/123"},
			Metadata:      &contexts.MetadataContext{},
		}))
	})

	t.Run("invalid name rejected", func(t *testing.T) {
		require.Error(t, g.Setup(core.SetupContext{
			Configuration: map[string]any{"alertPolicy": "not-a-policy"},
			Metadata:      &contexts.MetadataContext{},
		}))
	})

	t.Run("resolves display name into node metadata", func(t *testing.T) {
		mc := &mockClient{
			projectID: "my-project",
			getFunc: func(ctx context.Context, url string) ([]byte, error) {
				return alertPolicyJSON("projects/my-project/alertPolicies/123", "High CPU", true, comparisonGT, 0.8, "300s"), nil
			},
		}
		withFactory(mc)

		meta := &contexts.MetadataContext{}
		err := g.Setup(core.SetupContext{
			Configuration: map[string]any{"alertPolicy": "projects/my-project/alertPolicies/123"},
			Integration:   &contexts.IntegrationContext{},
			Metadata:      meta,
		})
		require.NoError(t, err)

		var stored AlertPolicyNodeMetadata
		require.NoError(t, mapstructure.Decode(meta.Get(), &stored))
		assert.Equal(t, "High CPU", stored.DisplayName)
		assert.Equal(t, "123", stored.ID)
	})

	t.Run("falls back to ID without integration", func(t *testing.T) {
		meta := &contexts.MetadataContext{}
		err := g.Setup(core.SetupContext{
			Configuration: map[string]any{"alertPolicy": "projects/my-project/alertPolicies/123"},
			Metadata:      meta,
		})
		require.NoError(t, err)

		var stored AlertPolicyNodeMetadata
		require.NoError(t, mapstructure.Decode(meta.Get(), &stored))
		assert.Equal(t, "", stored.DisplayName)
		assert.Equal(t, "123", stored.ID)
	})
}

func Test__GetAlertingPolicy__Execute(t *testing.T) {
	g := &GetAlertingPolicy{}

	t.Run("fetches policy -> emits fetched event", func(t *testing.T) {
		mc := &mockClient{
			projectID: "my-project",
			getFunc: func(ctx context.Context, url string) ([]byte, error) {
				assert.True(t, strings.HasSuffix(url, "/projects/my-project/alertPolicies/123"))
				return alertPolicyJSON("projects/my-project/alertPolicies/123", "High CPU", true, comparisonGT, 0.8, "300s"), nil
			},
		}
		withFactory(mc)

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := g.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"alertPolicy": "projects/my-project/alertPolicies/123"},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.True(t, state.Passed)
		assert.Equal(t, "gcp.monitoring.alertingPolicy.fetched", state.Type)
		data := state.Payloads[0].(map[string]any)["data"].(map[string]any)
		assert.Equal(t, "High CPU", data["displayName"])
	})

	t.Run("cross-project -> fails before API call", func(t *testing.T) {
		var called bool
		mc := &mockClient{
			projectID: "my-project",
			getFunc: func(ctx context.Context, url string) ([]byte, error) {
				called = true
				return nil, nil
			},
		}
		withFactory(mc)

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := g.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"alertPolicy": "projects/other-project/alertPolicies/123"},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.False(t, called)
		assert.Contains(t, state.FailureMessage, "cross-project")
	})

	t.Run("not found -> fails execution", func(t *testing.T) {
		mc := &mockClient{
			projectID: "my-project",
			getFunc: func(ctx context.Context, url string) ([]byte, error) {
				return nil, &gcpcommon.GCPAPIError{StatusCode: http.StatusNotFound, Message: "not found"}
			},
		}
		withFactory(mc)

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := g.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"alertPolicy": "projects/my-project/alertPolicies/123"},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "failed to get alerting policy")
	})
}
