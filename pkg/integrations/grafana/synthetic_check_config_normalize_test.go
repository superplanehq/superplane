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
		"frequency": float64(60),
		"timeout":   float64(3000),
		"probes":    []any{"1", "2"},
	}
	normalizeSyntheticCheckConfigMap(m)
	require.Contains(t, m, "request")
	require.Contains(t, m, "schedule")
	req := m["request"].(map[string]any)
	require.Equal(t, "https://example.com", req["target"])
	sch := m["schedule"].(map[string]any)
	require.Equal(t, float64(60), sch["frequency"])
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
