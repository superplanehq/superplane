package grafana

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeSyntheticCheckConfigMap_liftsLegacyFlatKeys(t *testing.T) {
	m := map[string]any{
		"job":       "API",
		"target":    "https://example.com",
		"method":    "GET",
		"frequency": float64(60000),
		"timeout":   float64(3000),
		"probes":    []any{"1", "2"},
	}
	normalizeSyntheticCheckConfigMap(m)
	require.Contains(t, m, "request")
	require.Contains(t, m, "schedule")
	req := m["request"].(map[string]any)
	require.Equal(t, "https://example.com", req["target"])
	sch := m["schedule"].(map[string]any)
	require.Equal(t, int64(60), sch["frequency"])
	require.Equal(t, int64(60000), sch["frequencyMilliseconds"])
}

func TestFlattenSyntheticCheckConfigMap_roundTrip(t *testing.T) {
	m := map[string]any{
		"job": "API",
		"request": map[string]any{
			"target": "https://example.com",
			"method": "POST",
		},
		"schedule": map[string]any{
			"frequency": float64(120),
			"probes":    []any{"3"},
		},
	}
	flat := flattenSyntheticCheckConfigMap(m)
	require.Equal(t, "https://example.com", flat["target"])
	require.Equal(t, "POST", flat["method"])
	require.Equal(t, float64(120), flat["frequency"])
}

func TestNormalizeSyntheticCheckConfigMap_marksLegacyFlatMillisecondFrequency(t *testing.T) {
	m := map[string]any{
		"job":       "API",
		"frequency": float64(60000),
	}

	normalizeSyntheticCheckConfigMap(m)

	sch := m["schedule"].(map[string]any)
	require.Equal(t, int64(60), sch["frequency"])
	require.Equal(t, int64(60000), sch["frequencyMilliseconds"])
}

func TestNormalizeSyntheticCheckConfigMap_keepsNestedFrequencyInSeconds(t *testing.T) {
	m := map[string]any{
		"job": "API",
		"schedule": map[string]any{
			"frequency": float64(1000),
		},
	}

	normalizeSyntheticCheckConfigMap(m)

	sch := m["schedule"].(map[string]any)
	require.Equal(t, float64(1000), sch["frequency"])
	require.NotContains(t, sch, "frequencyMilliseconds")
}
