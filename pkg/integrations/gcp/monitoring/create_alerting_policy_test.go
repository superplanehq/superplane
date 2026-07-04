package monitoring

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	gcpcommon "github.com/superplanehq/superplane/pkg/integrations/gcp/common"
	"github.com/superplanehq/superplane/test/support/contexts"
)

const cpuMetric = "compute.googleapis.com/instance/cpu/utilization"

func cpuCondition() map[string]any {
	return map[string]any{
		"metricType": cpuMetric,
		"comparison": comparisonGT,
		"threshold":  0.8,
		"duration":   "300s",
	}
}

func createConfig(conditions ...map[string]any) map[string]any {
	if len(conditions) == 0 {
		conditions = []map[string]any{cpuCondition()}
	}
	items := make([]any, len(conditions))
	for i, c := range conditions {
		items[i] = c
	}
	return map[string]any{"displayName": "High CPU", "conditions": items}
}

func Test__CreateAlertingPolicy__Setup(t *testing.T) {
	c := &CreateAlertingPolicy{}
	setup := func(cfg map[string]any) error {
		return c.Setup(core.SetupContext{Configuration: cfg, Metadata: &contexts.MetadataContext{}})
	}

	t.Run("valid passes", func(t *testing.T) {
		require.NoError(t, setup(createConfig()))
	})

	t.Run("missing displayName", func(t *testing.T) {
		cfg := createConfig()
		delete(cfg, "displayName")
		require.ErrorContains(t, setup(cfg), "displayName is required")
	})

	t.Run("no conditions", func(t *testing.T) {
		cfg := createConfig()
		cfg["conditions"] = []any{}
		require.ErrorContains(t, setup(cfg), "at least one condition")
	})

	t.Run("invalid metric", func(t *testing.T) {
		cond := cpuCondition()
		cond["metricType"] = "bogus"
		require.ErrorContains(t, setup(createConfig(cond)), "metricType")
	})

	t.Run("invalid comparison", func(t *testing.T) {
		cond := cpuCondition()
		cond["comparison"] = "NOPE"
		require.ErrorContains(t, setup(createConfig(cond)), "comparison")
	})

	t.Run("invalid duration", func(t *testing.T) {
		cond := cpuCondition()
		cond["duration"] = "42s"
		require.ErrorContains(t, setup(createConfig(cond)), "duration")
	})

	t.Run("missing threshold", func(t *testing.T) {
		cond := cpuCondition()
		delete(cond, "threshold")
		require.ErrorContains(t, setup(createConfig(cond)), "threshold is required")
	})

	t.Run("invalid combiner", func(t *testing.T) {
		cfg := createConfig()
		cfg["combiner"] = "XOR"
		require.ErrorContains(t, setup(cfg), "combiner")
	})
}

func Test__CreateAlertingPolicy__Execute(t *testing.T) {
	c := &CreateAlertingPolicy{}

	t.Run("creates policy -> emits created event", func(t *testing.T) {
		var postBody map[string]any
		mc := &mockClient{
			projectID: "my-project",
			postFunc: func(ctx context.Context, url string, body any) ([]byte, error) {
				assert.True(t, strings.HasSuffix(url, "/projects/my-project/alertPolicies"))
				postBody, _ = body.(map[string]any)
				return alertPolicyJSON("projects/my-project/alertPolicies/123", "High CPU", true, comparisonGT, 0.8, "300s"), nil
			},
		}
		withFactory(mc)

		cfg := createConfig()
		cfg["enabled"] = true
		cfg["severity"] = "CRITICAL"
		cfg["notificationChannels"] = []any{"projects/my-project/notificationChannels/9"}

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := c.Execute(core.ExecutionContext{Configuration: cfg, ExecutionState: state})

		require.NoError(t, err)
		assert.True(t, state.Passed)
		assert.Equal(t, "gcp.monitoring.alertingPolicy.created", state.Type)
		assert.Equal(t, "OR", postBody["combiner"])
		assert.Equal(t, true, postBody["enabled"])
		assert.Equal(t, "CRITICAL", postBody["severity"])
		require.Len(t, postBody["conditions"].([]any), 1)
		assert.Equal(t, []string{"projects/my-project/notificationChannels/9"}, postBody["notificationChannels"])
	})

	t.Run("PromQL condition -> builds conditionPrometheusQueryLanguage", func(t *testing.T) {
		var postBody map[string]any
		mc := &mockClient{
			projectID: "my-project",
			postFunc: func(ctx context.Context, url string, body any) ([]byte, error) {
				postBody, _ = body.(map[string]any)
				return alertPolicyJSON("projects/my-project/alertPolicies/777", "PromQL alert", true, comparisonGT, 0.8, "300s"), nil
			},
		}
		withFactory(mc)

		cfg := map[string]any{
			"displayName":              "PromQL alert",
			"conditionKind":            conditionKindPromQL,
			"promqlQuery":              "rate(http_requests_total[5m]) > 100",
			"promqlDuration":           "60s",
			"promqlEvaluationInterval": "30s",
		}
		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		require.NoError(t, c.Execute(core.ExecutionContext{Configuration: cfg, ExecutionState: state}))
		assert.True(t, state.Passed)

		conditions := postBody["conditions"].([]any)
		require.Len(t, conditions, 1)
		cond := conditions[0].(map[string]any)
		pql := cond["conditionPrometheusQueryLanguage"].(map[string]any)
		assert.Equal(t, "rate(http_requests_total[5m]) > 100", pql["query"])
		assert.Equal(t, "60s", pql["duration"])
		assert.Equal(t, "30s", pql["evaluationInterval"])
		// No threshold condition is built for a PromQL policy.
		_, hasThreshold := cond["conditionThreshold"]
		assert.False(t, hasThreshold)
	})

	t.Run("PromQL condition requires a query", func(t *testing.T) {
		withFactory(&mockClient{projectID: "my-project"})
		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		require.NoError(t, c.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"displayName": "x", "conditionKind": conditionKindPromQL},
			ExecutionState: state,
		}))
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "promqlQuery is required")
	})

	t.Run("multiple conditions + AND combiner + strategy", func(t *testing.T) {
		var postBody map[string]any
		mc := &mockClient{
			projectID: "my-project",
			postFunc: func(ctx context.Context, url string, body any) ([]byte, error) {
				postBody, _ = body.(map[string]any)
				return alertPolicyJSON("projects/my-project/alertPolicies/123", "Composite", true, comparisonGT, 0.8, "300s"), nil
			},
		}
		withFactory(mc)

		disk := map[string]any{"metricType": "compute.googleapis.com/instance/disk/read_bytes_count", "comparison": comparisonGT, "threshold": 1000, "duration": "60s"}
		cfg := createConfig(cpuCondition(), disk)
		cfg["combiner"] = "AND"
		cfg["autoClose"] = "3600s"
		cfg["notificationRateLimit"] = "300s"

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := c.Execute(core.ExecutionContext{Configuration: cfg, ExecutionState: state})

		require.NoError(t, err)
		assert.True(t, state.Passed)
		assert.Equal(t, "AND", postBody["combiner"])
		assert.Len(t, postBody["conditions"].([]any), 2)
		strategy := postBody["alertStrategy"].(map[string]any)
		assert.Equal(t, "3600s", strategy["autoClose"])
		assert.Equal(t, "300s", strategy["notificationRateLimit"].(map[string]any)["period"])
	})

	t.Run("omitted enabled defaults to true", func(t *testing.T) {
		var postBody map[string]any
		mc := &mockClient{
			projectID: "my-project",
			postFunc: func(ctx context.Context, url string, body any) ([]byte, error) {
				postBody, _ = body.(map[string]any)
				return alertPolicyJSON("projects/my-project/alertPolicies/123", "High CPU", true, comparisonGT, 0.8, "300s"), nil
			},
		}
		withFactory(mc)

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := c.Execute(core.ExecutionContext{Configuration: createConfig(), ExecutionState: state})
		require.NoError(t, err)
		assert.True(t, state.Passed)
		assert.Equal(t, true, postBody["enabled"], "enabled must default to true when omitted")
	})

	t.Run("explicit enabled=false is respected", func(t *testing.T) {
		var postBody map[string]any
		mc := &mockClient{
			projectID: "my-project",
			postFunc: func(ctx context.Context, url string, body any) ([]byte, error) {
				postBody, _ = body.(map[string]any)
				return alertPolicyJSON("projects/my-project/alertPolicies/123", "High CPU", false, comparisonGT, 0.8, "300s"), nil
			},
		}
		withFactory(mc)

		cfg := createConfig()
		cfg["enabled"] = false
		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := c.Execute(core.ExecutionContext{Configuration: cfg, ExecutionState: state})
		require.NoError(t, err)
		assert.Equal(t, false, postBody["enabled"])
	})

	t.Run("403 -> fails with IAM hint", func(t *testing.T) {
		mc := &mockClient{
			projectID: "my-project",
			postFunc: func(ctx context.Context, url string, body any) ([]byte, error) {
				return nil, &gcpcommon.GCPAPIError{StatusCode: http.StatusForbidden, Message: "Permission denied (or the resource may not exist)."}
			},
		}
		withFactory(mc)

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := c.Execute(core.ExecutionContext{Configuration: createConfig(), ExecutionState: state})
		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "failed to create alerting policy")
		assert.Contains(t, state.FailureMessage, "roles/monitoring.editor")
	})

	t.Run("API error -> fails execution", func(t *testing.T) {
		mc := &mockClient{
			projectID: "my-project",
			postFunc: func(ctx context.Context, url string, body any) ([]byte, error) {
				return nil, &gcpcommon.GCPAPIError{StatusCode: http.StatusBadRequest, Message: "invalid"}
			},
		}
		withFactory(mc)

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := c.Execute(core.ExecutionContext{Configuration: createConfig(), ExecutionState: state})
		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "failed to create alerting policy")
	})
}

func Test__CombinerVisibleWhenConditionKindUnset(t *testing.T) {
	found := false
	for _, f := range policyOptionFields() {
		if f.Name != "combiner" {
			continue
		}
		found = true
		require.Len(t, f.VisibilityConditions, 1)
		vals := f.VisibilityConditions[0].Values
		// Visible for threshold and when conditionKind is unset (Update's default,
		// where conditionKind is togglable and usually omitted); hidden for PromQL.
		assert.Contains(t, vals, conditionKindThreshold)
		assert.Contains(t, vals, "")
		assert.NotContains(t, vals, conditionKindPromQL)
	}
	require.True(t, found, "combiner field not found in policyOptionFields")
}
