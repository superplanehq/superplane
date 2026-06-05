package monitoring

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__UpdateAlertingPolicy__Setup(t *testing.T) {
	u := &UpdateAlertingPolicy{}
	policy := "projects/my-project/alertPolicies/123"

	t.Run("no updates -> error", func(t *testing.T) {
		require.ErrorContains(t, u.Setup(core.SetupContext{
			Configuration: map[string]any{"alertPolicy": policy},
			Metadata:      &contexts.MetadataContext{},
		}), "at least one field")
	})

	t.Run("metric without comparison/duration -> error", func(t *testing.T) {
		require.Error(t, u.Setup(core.SetupContext{
			Configuration: map[string]any{"alertPolicy": policy, "metricType": "compute.googleapis.com/instance/cpu/utilization"},
			Metadata:      &contexts.MetadataContext{},
		}))
	})

	t.Run("invalid enabled -> error", func(t *testing.T) {
		require.ErrorContains(t, u.Setup(core.SetupContext{
			Configuration: map[string]any{"alertPolicy": policy, "enabled": "maybe"},
			Metadata:      &contexts.MetadataContext{},
		}), "enabled")
	})

	t.Run("displayName only -> valid", func(t *testing.T) {
		require.NoError(t, u.Setup(core.SetupContext{
			Configuration: map[string]any{"alertPolicy": policy, "displayName": "New name"},
			Metadata:      &contexts.MetadataContext{},
		}))
	})
}

func Test__UpdateAlertingPolicy__Execute(t *testing.T) {
	u := &UpdateAlertingPolicy{}
	policy := "projects/my-project/alertPolicies/123"

	t.Run("updates displayName and enabled via updateMask", func(t *testing.T) {
		var patchURL string
		var patchBody map[string]any
		mc := &mockClient{
			projectID: "my-project",
			patchFunc: func(ctx context.Context, url string, body any) ([]byte, error) {
				patchURL = url
				patchBody, _ = body.(map[string]any)
				return alertPolicyJSON(policy, "New name", false, comparisonGT, 0.8, "300s"), nil
			},
		}
		withFactory(mc)

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := u.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"alertPolicy": policy, "displayName": "New name", "enabled": "false"},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.True(t, state.Passed)
		assert.Equal(t, "gcp.monitoring.alertingPolicy.updated", state.Type)
		assert.Contains(t, patchURL, "updateMask=")
		assert.Contains(t, patchURL, "displayName")
		assert.Contains(t, patchURL, "enabled")
		assert.Equal(t, "New name", patchBody["displayName"])
		assert.Equal(t, false, patchBody["enabled"])
		_, hasConditions := patchBody["conditions"]
		assert.False(t, hasConditions, "conditions should not be sent when metric is unchanged")
	})

	t.Run("rebuilds condition when metric provided", func(t *testing.T) {
		var patchBody map[string]any
		mc := &mockClient{
			projectID: "my-project",
			patchFunc: func(ctx context.Context, url string, body any) ([]byte, error) {
				patchBody, _ = body.(map[string]any)
				assert.Contains(t, url, "conditions")
				return alertPolicyJSON(policy, "High CPU", true, comparisonGT, 0.9, "300s"), nil
			},
		}
		withFactory(mc)

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := u.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"alertPolicy": policy,
				"metricType":  "compute.googleapis.com/instance/cpu/utilization",
				"comparison":  comparisonGT,
				"threshold":   0.9,
				"duration":    "300s",
			},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.True(t, state.Passed)
		conds, ok := patchBody["conditions"].([]any)
		require.True(t, ok)
		require.Len(t, conds, 1)
	})

	t.Run("clears notification channels when provided empty", func(t *testing.T) {
		var patchURL string
		var patchBody map[string]any
		mc := &mockClient{
			projectID: "my-project",
			patchFunc: func(ctx context.Context, url string, body any) ([]byte, error) {
				patchURL = url
				patchBody, _ = body.(map[string]any)
				return alertPolicyJSON(policy, "High CPU", true, comparisonGT, 0.8, "300s"), nil
			},
		}
		withFactory(mc)

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := u.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"alertPolicy": policy, "notificationChannels": []any{}},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.True(t, state.Passed)
		assert.Contains(t, patchURL, "notificationChannels")
		channels, ok := patchBody["notificationChannels"].([]string)
		require.True(t, ok, "notificationChannels must be sent as a (possibly empty) slice")
		assert.Empty(t, channels)
	})

	t.Run("invalid enabled -> fails execution", func(t *testing.T) {
		mc := &mockClient{projectID: "my-project"}
		withFactory(mc)

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := u.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"alertPolicy": policy, "enabled": "maybe"},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "enabled")
	})

	t.Run("no updates -> fails execution", func(t *testing.T) {
		mc := &mockClient{projectID: "my-project"}
		withFactory(mc)

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := u.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"alertPolicy": policy},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "at least one field")
	})
}
