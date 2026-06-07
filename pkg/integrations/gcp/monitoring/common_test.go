package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
)

// mockClient is a configurable monitoring.Client used by the component tests.
type mockClient struct {
	projectID  string
	getFunc    func(ctx context.Context, url string) ([]byte, error)
	postFunc   func(ctx context.Context, url string, body any) ([]byte, error)
	patchFunc  func(ctx context.Context, url string, body any) ([]byte, error)
	deleteFunc func(ctx context.Context, url string) ([]byte, error)
}

func (m *mockClient) GetURL(ctx context.Context, url string) ([]byte, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, url)
	}
	return nil, fmt.Errorf("unexpected GetURL(%s)", url)
}

func (m *mockClient) PostURL(ctx context.Context, url string, body any) ([]byte, error) {
	if m.postFunc != nil {
		return m.postFunc(ctx, url, body)
	}
	return nil, fmt.Errorf("unexpected PostURL(%s)", url)
}

func (m *mockClient) PatchURL(ctx context.Context, url string, body any) ([]byte, error) {
	if m.patchFunc != nil {
		return m.patchFunc(ctx, url, body)
	}
	return nil, fmt.Errorf("unexpected PatchURL(%s)", url)
}

func (m *mockClient) DeleteURL(ctx context.Context, url string) ([]byte, error) {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, url)
	}
	return nil, fmt.Errorf("unexpected DeleteURL(%s)", url)
}

func (m *mockClient) ProjectID() string {
	return m.projectID
}

// alertPolicyJSON serializes an AlertPolicy as the API would return it.
func alertPolicyJSON(name, displayName string, enabled bool, comparison string, threshold float64, duration string) []byte {
	b, _ := json.Marshal(map[string]any{
		"name":        name,
		"displayName": displayName,
		"combiner":    "OR",
		"enabled":     enabled,
		"conditions": []map[string]any{
			{
				"displayName": "cpu",
				"conditionThreshold": map[string]any{
					"filter":         instanceMetricFilter("compute.googleapis.com/instance/cpu/utilization"),
					"comparison":     comparison,
					"thresholdValue": threshold,
					"duration":       duration,
				},
			},
		},
	})
	return b
}

// withFactory installs a mock client for the duration of a component test.
func withFactory(mc *mockClient) {
	SetClientFactory(func(httpCtx core.HTTPContext, integration core.IntegrationContext) (Client, error) {
		return mc, nil
	})
}

func Test__ParsePolicyName(t *testing.T) {
	t.Run("relative name", func(t *testing.T) {
		project, name, err := parsePolicyName("projects/my-project/alertPolicies/123")
		require.NoError(t, err)
		assert.Equal(t, "my-project", project)
		assert.Equal(t, "projects/my-project/alertPolicies/123", name)
	})

	t.Run("full URL", func(t *testing.T) {
		project, name, err := parsePolicyName("https://monitoring.googleapis.com/v3/projects/elffie/alertPolicies/999")
		require.NoError(t, err)
		assert.Equal(t, "elffie", project)
		assert.Equal(t, "projects/elffie/alertPolicies/999", name)
	})

	t.Run("empty rejected", func(t *testing.T) {
		_, _, err := parsePolicyName("")
		require.Error(t, err)
	})

	t.Run("non-policy name rejected", func(t *testing.T) {
		_, _, err := parsePolicyName("projects/my-project/notificationChannels/1")
		require.Error(t, err)
	})

	t.Run("plain name rejected", func(t *testing.T) {
		_, _, err := parsePolicyName("just-a-name")
		require.Error(t, err)
	})
}

func ptrFloat(f float64) *float64 { return &f }

func Test__BuildConditions(t *testing.T) {
	t.Run("cpu defaults to ALIGN_MEAN + count trigger", func(t *testing.T) {
		out, err := buildConditions([]ConditionSpec{{
			MetricType: "compute.googleapis.com/instance/cpu/utilization",
			Comparison: comparisonGT, Threshold: ptrFloat(0.8), Duration: "300s",
		}})
		require.NoError(t, err)
		require.Len(t, out, 1)
		ct := out[0].(map[string]any)["conditionThreshold"].(map[string]any)
		assert.Equal(t, comparisonGT, ct["comparison"])
		assert.Equal(t, 0.8, ct["thresholdValue"])
		assert.Equal(t, "300s", ct["duration"])
		assert.Contains(t, ct["filter"], `resource.type="gce_instance"`)
		assert.Equal(t, 1, ct["trigger"].(map[string]any)["count"])
		agg := ct["aggregations"].([]any)[0].(map[string]any)
		assert.Equal(t, "ALIGN_MEAN", agg["perSeriesAligner"])
		assert.Equal(t, "60s", agg["alignmentPeriod"])
	})

	t.Run("honors aligner, window, reducer, group-by, percent trigger", func(t *testing.T) {
		out, err := buildConditions([]ConditionSpec{{
			MetricType: "compute.googleapis.com/instance/network/sent_bytes_count",
			Comparison: comparisonLT, Threshold: ptrFloat(1000), Duration: "60s",
			Aligner: "ALIGN_MAX", AlignmentPeriod: "300s",
			CrossSeriesReducer: "REDUCE_SUM", GroupByFields: []string{"resource.zone"},
			TriggerType: triggerPercent, TriggerValue: ptrFloat(50),
		}})
		require.NoError(t, err)
		ct := out[0].(map[string]any)["conditionThreshold"].(map[string]any)
		agg := ct["aggregations"].([]any)[0].(map[string]any)
		assert.Equal(t, "ALIGN_MAX", agg["perSeriesAligner"])
		assert.Equal(t, "300s", agg["alignmentPeriod"])
		assert.Equal(t, "REDUCE_SUM", agg["crossSeriesReducer"])
		assert.Equal(t, []string{"resource.zone"}, agg["groupByFields"])
		assert.Equal(t, 50.0, ct["trigger"].(map[string]any)["percent"])
	})

	t.Run("multiple conditions", func(t *testing.T) {
		out, err := buildConditions([]ConditionSpec{
			{MetricType: "compute.googleapis.com/instance/cpu/utilization", Comparison: comparisonGT, Threshold: ptrFloat(0.8), Duration: "300s"},
			{MetricType: "compute.googleapis.com/instance/disk/read_bytes_count", Comparison: comparisonGT, Threshold: ptrFloat(1000), Duration: "60s"},
		})
		require.NoError(t, err)
		assert.Len(t, out, 2)
	})

	t.Run("missing threshold errors", func(t *testing.T) {
		_, err := buildConditions([]ConditionSpec{{
			MetricType: "compute.googleapis.com/instance/cpu/utilization", Comparison: comparisonGT, Duration: "300s",
		}})
		require.ErrorContains(t, err, "threshold is required")
	})

	t.Run("empty conditions errors", func(t *testing.T) {
		_, err := buildConditions(nil)
		require.ErrorContains(t, err, "at least one condition")
	})

	t.Run("too many conditions errors", func(t *testing.T) {
		specs := make([]ConditionSpec, maxPolicyConditions+1)
		for i := range specs {
			specs[i] = ConditionSpec{
				MetricType: "compute.googleapis.com/instance/cpu/utilization",
				Comparison: comparisonGT, Threshold: ptrFloat(0.8), Duration: "300s",
			}
		}
		_, err := buildConditions(specs)
		require.ErrorContains(t, err, "at most")
	})

	t.Run("zero / fractional / negative count triggers error", func(t *testing.T) {
		for _, v := range []float64{0, -1, 0.5, 2.5} {
			_, err := buildConditions([]ConditionSpec{{
				MetricType: "compute.googleapis.com/instance/cpu/utilization",
				Comparison: comparisonGT, Threshold: ptrFloat(0.8), Duration: "300s",
				TriggerType: triggerCount, TriggerValue: ptrFloat(v),
			}})
			require.ErrorContains(t, err, "trigger count", "value %v should be rejected", v)
		}
	})

	t.Run("out-of-range percent trigger errors", func(t *testing.T) {
		for _, v := range []float64{0, -5, 150} {
			_, err := buildConditions([]ConditionSpec{{
				MetricType: "compute.googleapis.com/instance/cpu/utilization",
				Comparison: comparisonGT, Threshold: ptrFloat(0.8), Duration: "300s",
				TriggerType: triggerPercent, TriggerValue: ptrFloat(v),
			}})
			require.ErrorContains(t, err, "trigger percent", "value %v should be rejected", v)
		}
	})

	t.Run("valid count and percent triggers pass", func(t *testing.T) {
		_, err := buildConditions([]ConditionSpec{{
			MetricType: "compute.googleapis.com/instance/cpu/utilization",
			Comparison: comparisonGT, Threshold: ptrFloat(0.8), Duration: "300s",
			TriggerType: triggerCount, TriggerValue: ptrFloat(3),
		}})
		require.NoError(t, err)
		_, err = buildConditions([]ConditionSpec{{
			MetricType: "compute.googleapis.com/instance/cpu/utilization",
			Comparison: comparisonGT, Threshold: ptrFloat(0.8), Duration: "300s",
			TriggerType: triggerPercent, TriggerValue: ptrFloat(100),
		}})
		require.NoError(t, err)
	})
}

func Test__PolicyPayload(t *testing.T) {
	var p alertPolicy
	require.NoError(t, json.Unmarshal(
		alertPolicyJSON("projects/my-project/alertPolicies/123", "High CPU", true, comparisonGT, 0.8, "300s"),
		&p,
	))
	payload := policyPayload(&p)
	assert.Equal(t, "projects/my-project/alertPolicies/123", payload["name"])
	assert.Equal(t, "123", payload["id"])
	assert.Equal(t, "High CPU", payload["displayName"])
	assert.Equal(t, true, payload["enabled"])
	assert.Equal(t, 1, payload["conditionsCount"])
	assert.Equal(t, comparisonGT, payload["comparison"])
	assert.Equal(t, 0.8, payload["thresholdValue"])
	assert.Equal(t, "300s", payload["duration"])
}
