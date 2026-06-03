package ec2provision

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	fmmetrics "github.com/superplane/runner/fleet-manager/internal/metrics"
)

func TestObserveInstanceSpinupRecordsRunning(t *testing.T) {
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	pm, err := fmmetrics.New(provider.Meter("test"))
	require.NoError(t, err)

	requestedAt := time.Now().Add(-2 * time.Second)
	l := &Launcher{
		Config:  Config{RunnerFleetID: "fleet-a"},
		Metrics: pm,
		pending: map[string]time.Time{"i-1": requestedAt},
	}

	l.observeInstanceSpinup(context.Background(), []managedInstance{
		{id: "i-1", state: string(types.InstanceStateNameRunning)},
	})

	require.Empty(t, l.pending)

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &rm))
	found := false
	for _, sm := range rm.ScopeMetrics {
		for _, met := range sm.Metrics {
			if met.Name != "instance.spinup.duration" {
				continue
			}
			hist := met.Data.(metricdata.Histogram[float64])
			require.Len(t, hist.DataPoints, 1)
			require.InDelta(t, 2.0, hist.DataPoints[0].Sum, 0.5)
			found = true
		}
	}
	require.True(t, found)
}

func TestObserveInstanceSpinupDropsMissingInstances(t *testing.T) {
	l := &Launcher{
		pending: map[string]time.Time{"i-gone": time.Now()},
	}
	l.observeInstanceSpinup(context.Background(), nil)
	require.Empty(t, l.pending)
}
