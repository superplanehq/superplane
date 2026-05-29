package compute

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func timeSeriesJSON(values ...float64) []byte {
	points := make([]map[string]any, 0, len(values))
	for _, v := range values {
		points = append(points, map[string]any{
			"value": map[string]any{"doubleValue": v},
		})
	}
	b, _ := json.Marshal(map[string]any{
		"timeSeries": []map[string]any{{"points": points}},
	})
	return b
}

func Test__GetVMInstanceMetrics__Setup(t *testing.T) {
	component := &GetVMInstanceMetrics{}

	t.Run("missing instance returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"lookbackPeriod": "1h"},
			Metadata:      &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "instance is required")
	})

	t.Run("missing lookbackPeriod returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"instance": "zones/us-central1-a/instances/my-vm"},
			Metadata:      &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "lookbackPeriod is required")
	})

	t.Run("invalid lookbackPeriod returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"instance":       "zones/us-central1-a/instances/my-vm",
				"lookbackPeriod": "99y",
			},
			Metadata: &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "invalid lookbackPeriod")
	})
}

func Test__GetVMInstanceMetrics__Execute(t *testing.T) {
	component := &GetVMInstanceMetrics{}

	t.Run("aggregates CPU and network metrics", func(t *testing.T) {
		mc := &mockInstanceClient{
			projectID: "my-project",
			getFunc: func(ctx context.Context, path string) ([]byte, error) {
				return instanceGetJSON("123", "my-vm", "us-central1-a", "RUNNING", "e2-medium"), nil
			},
			getURLFunc: func(ctx context.Context, fullURL string) ([]byte, error) {
				switch {
				case strings.Contains(fullURL, "utilization"):
					return timeSeriesJSON(0.2, 0.3), nil // avg 0.25 -> 25%
				case strings.Contains(fullURL, "received_bytes_count"):
					return timeSeriesJSON(1000, 2000), nil // avg 1500
				case strings.Contains(fullURL, "sent_bytes_count"):
					return timeSeriesJSON(500, 500), nil // avg 500
				default:
					return timeSeriesJSON(), nil
				}
			},
		}

		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"instance":       "zones/us-central1-a/instances/my-vm",
				"lookbackPeriod": "1h",
			},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.True(t, state.Passed)
		assert.Equal(t, "gcp.compute.vmInstance.metrics", state.Type)

		data := state.Payloads[0].(map[string]any)["data"].(map[string]any)
		assert.Equal(t, "123", data["instanceId"])
		assert.Equal(t, 25.0, data["avgCpuUsagePercent"])
		assert.Equal(t, 1500.0, data["avgNetworkInboundBytesPerSec"])
		assert.Equal(t, 500.0, data["avgNetworkOutboundBytesPerSec"])
		assert.Equal(t, "1h", data["lookbackPeriod"])
	})
}
