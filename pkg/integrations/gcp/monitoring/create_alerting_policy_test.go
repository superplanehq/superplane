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

func Test__CreateAlertingPolicy__Setup(t *testing.T) {
	c := &CreateAlertingPolicy{}

	base := func() map[string]any {
		return map[string]any{
			"displayName": "High CPU",
			"metricType":  "compute.googleapis.com/instance/cpu/utilization",
			"comparison":  comparisonGT,
			"threshold":   0.8,
			"duration":    "300s",
		}
	}

	t.Run("valid passes", func(t *testing.T) {
		require.NoError(t, c.Setup(core.SetupContext{Configuration: base(), Metadata: &contexts.MetadataContext{}}))
	})

	t.Run("missing displayName", func(t *testing.T) {
		cfg := base()
		delete(cfg, "displayName")
		require.ErrorContains(t, c.Setup(core.SetupContext{Configuration: cfg, Metadata: &contexts.MetadataContext{}}), "displayName is required")
	})

	t.Run("invalid metric", func(t *testing.T) {
		cfg := base()
		cfg["metricType"] = "bogus"
		require.ErrorContains(t, c.Setup(core.SetupContext{Configuration: cfg, Metadata: &contexts.MetadataContext{}}), "invalid metricType")
	})

	t.Run("invalid comparison", func(t *testing.T) {
		cfg := base()
		cfg["comparison"] = "NOPE"
		require.ErrorContains(t, c.Setup(core.SetupContext{Configuration: cfg, Metadata: &contexts.MetadataContext{}}), "comparison")
	})

	t.Run("invalid duration", func(t *testing.T) {
		cfg := base()
		cfg["duration"] = "42s"
		require.ErrorContains(t, c.Setup(core.SetupContext{Configuration: cfg, Metadata: &contexts.MetadataContext{}}), "duration")
	})

	t.Run("missing threshold", func(t *testing.T) {
		cfg := base()
		delete(cfg, "threshold")
		require.ErrorContains(t, c.Setup(core.SetupContext{Configuration: cfg, Metadata: &contexts.MetadataContext{}}), "threshold is required")
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

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := c.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"displayName":          "High CPU",
				"metricType":           "compute.googleapis.com/instance/cpu/utilization",
				"comparison":           comparisonGT,
				"threshold":            0.8,
				"duration":             "300s",
				"enabled":              true,
				"notificationChannels": []any{"projects/my-project/notificationChannels/9"},
			},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.True(t, state.Passed)
		assert.Equal(t, "gcp.monitoring.alertingPolicy.created", state.Type)
		data := state.Payloads[0].(map[string]any)["data"].(map[string]any)
		assert.Equal(t, "projects/my-project/alertPolicies/123", data["name"])
		assert.Equal(t, "123", data["id"])
		// request body assertions
		assert.Equal(t, "OR", postBody["combiner"])
		assert.Equal(t, true, postBody["enabled"])
		conds := postBody["conditions"].([]any)
		require.Len(t, conds, 1)
		assert.Equal(t, []string{"projects/my-project/notificationChannels/9"}, postBody["notificationChannels"])
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
		err := c.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"displayName": "High CPU",
				"metricType":  "compute.googleapis.com/instance/cpu/utilization",
				"comparison":  comparisonGT,
				"threshold":   0.8,
				"duration":    "300s",
				// enabled intentionally omitted
			},
			ExecutionState: state,
		})

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

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := c.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"displayName": "High CPU",
				"metricType":  "compute.googleapis.com/instance/cpu/utilization",
				"comparison":  comparisonGT,
				"threshold":   0.8,
				"duration":    "300s",
				"enabled":     false,
			},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.True(t, state.Passed)
		assert.Equal(t, false, postBody["enabled"], "explicit enabled=false must be sent")
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
		err := c.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"displayName": "High CPU",
				"metricType":  "compute.googleapis.com/instance/cpu/utilization",
				"comparison":  comparisonGT,
				"threshold":   0.8,
				"duration":    "300s",
			},
			ExecutionState: state,
		})

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
		err := c.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"displayName": "High CPU",
				"metricType":  "compute.googleapis.com/instance/cpu/utilization",
				"comparison":  comparisonGT,
				"threshold":   0.8,
				"duration":    "300s",
			},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "failed to create alerting policy")
	})
}
