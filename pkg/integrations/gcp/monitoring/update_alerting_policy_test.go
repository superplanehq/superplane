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
	setup := func(cfg map[string]any) error {
		return u.Setup(core.SetupContext{Configuration: cfg, Metadata: &contexts.MetadataContext{}})
	}

	t.Run("no updates -> error", func(t *testing.T) {
		require.ErrorContains(t, setup(map[string]any{"alertPolicy": policy}), "at least one field")
	})

	t.Run("invalid enabled -> error", func(t *testing.T) {
		require.ErrorContains(t, setup(map[string]any{"alertPolicy": policy, "enabled": "maybe"}), "enabled")
	})

	t.Run("invalid combiner -> error", func(t *testing.T) {
		require.ErrorContains(t, setup(map[string]any{"alertPolicy": policy, "combiner": "XOR"}), "combiner")
	})

	t.Run("invalid conditions -> error", func(t *testing.T) {
		cond := cpuCondition()
		cond["duration"] = "42s"
		require.Error(t, setup(map[string]any{"alertPolicy": policy, "conditions": []any{cond}}))
	})

	t.Run("displayName only -> valid", func(t *testing.T) {
		require.NoError(t, setup(map[string]any{"alertPolicy": policy, "displayName": "New name"}))
	})

	t.Run("promql conditionKind without a query does not block other updates", func(t *testing.T) {
		// conditionKind may be persisted as "promql" from a prior toggle; with no
		// query it must not be treated as a condition replacement, so a
		// severity-only update still validates.
		require.NoError(t, setup(map[string]any{"alertPolicy": policy, "conditionKind": conditionKindPromQL, "severity": "CRITICAL"}))
	})

	t.Run("promql conditionKind with a query -> valid", func(t *testing.T) {
		require.NoError(t, setup(map[string]any{"alertPolicy": policy, "conditionKind": conditionKindPromQL, "promqlQuery": "vector(1) > 0"}))
	})
}

func Test__UpdateAlertingPolicy__Execute(t *testing.T) {
	u := &UpdateAlertingPolicy{}
	policy := "projects/my-project/alertPolicies/123"

	t.Run("updates displayName, enabled, severity via updateMask", func(t *testing.T) {
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
			Configuration:  map[string]any{"alertPolicy": policy, "displayName": "New name", "enabled": "false", "severity": "WARNING"},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.True(t, state.Passed)
		assert.Equal(t, "gcp.monitoring.alertingPolicy.updated", state.Type)
		assert.Contains(t, patchURL, "updateMask=")
		assert.Contains(t, patchURL, "displayName")
		assert.Contains(t, patchURL, "enabled")
		assert.Contains(t, patchURL, "severity")
		assert.Equal(t, "New name", patchBody["displayName"])
		assert.Equal(t, false, patchBody["enabled"])
		assert.Equal(t, "WARNING", patchBody["severity"])
		_, hasConditions := patchBody["conditions"]
		assert.False(t, hasConditions)
	})

	t.Run("promql conditionKind without query updates other fields only (no conditions patch)", func(t *testing.T) {
		var patchBody map[string]any
		mc := &mockClient{
			projectID: "my-project",
			patchFunc: func(ctx context.Context, url string, body any) ([]byte, error) {
				patchBody, _ = body.(map[string]any)
				return alertPolicyJSON(policy, "x", true, comparisonGT, 0.8, "300s"), nil
			},
		}
		withFactory(mc)

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := u.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"alertPolicy": policy, "conditionKind": conditionKindPromQL, "severity": "CRITICAL"},
			ExecutionState: state,
		})
		require.NoError(t, err)
		assert.True(t, state.Passed)
		assert.Equal(t, "CRITICAL", patchBody["severity"])
		_, hasConditions := patchBody["conditions"]
		assert.False(t, hasConditions)
	})

	t.Run("replaces conditions when provided", func(t *testing.T) {
		var patchBody map[string]any
		mc := &mockClient{
			projectID: "my-project",
			patchFunc: func(ctx context.Context, url string, body any) ([]byte, error) {
				patchBody, _ = body.(map[string]any)
				assert.Contains(t, url, "conditions")
				return alertPolicyJSON(policy, "High CPU", true, comparisonLT, 0.9, "300s"), nil
			},
		}
		withFactory(mc)

		cond := cpuCondition()
		cond["comparison"] = comparisonLT
		cond["threshold"] = 0.9
		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := u.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"alertPolicy": policy, "conditions": []any{cond}},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.True(t, state.Passed)
		require.Len(t, patchBody["conditions"].([]any), 1)
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
		require.True(t, ok)
		assert.Empty(t, channels)
	})

	t.Run("changing only auto-close masks just that field", func(t *testing.T) {
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
			Configuration:  map[string]any{"alertPolicy": policy, "autoClose": "3600s"},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.True(t, state.Passed)
		// Only the auto-close sub-field is masked, leaving the rate limit intact.
		assert.Contains(t, patchURL, "alertStrategy.autoClose")
		assert.NotContains(t, patchURL, "alertStrategy.notificationRateLimit")
		strategy := patchBody["alertStrategy"].(map[string]any)
		assert.Equal(t, "3600s", strategy["autoClose"])
		_, hasRateLimit := strategy["notificationRateLimit"]
		assert.False(t, hasRateLimit)
	})

	t.Run("changing only documentation subject masks just the subject", func(t *testing.T) {
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
			Configuration:  map[string]any{"alertPolicy": policy, "documentationSubject": "Runbook"},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.True(t, state.Passed)
		assert.Contains(t, patchURL, "documentation.subject")
		assert.NotContains(t, patchURL, "documentation.content")
		doc := patchBody["documentation"].(map[string]any)
		assert.Equal(t, "Runbook", doc["subject"])
		_, hasContent := doc["content"]
		assert.False(t, hasContent)
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
