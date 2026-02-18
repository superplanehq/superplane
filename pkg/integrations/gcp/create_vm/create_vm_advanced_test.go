package createvm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	compute "google.golang.org/api/compute/v1"
)

func Test_trimmedNonEmptyStrings(t *testing.T) {
	assert.Nil(t, trimmedNonEmptyStrings(nil))
	assert.Empty(t, trimmedNonEmptyStrings([]string{"", "  ", ""}))
	assert.Equal(t, []string{"a", "b"}, trimmedNonEmptyStrings([]string{" a ", "", " b "}))
}

func Test_normalizeNodeAffinityOperator(t *testing.T) {
	assert.Equal(t, NodeAffinityOperatorIn, normalizeNodeAffinityOperator(""))
	assert.Equal(t, NodeAffinityOperatorIn, normalizeNodeAffinityOperator("IN"))
	assert.Equal(t, NodeAffinityOperatorNotIn, normalizeNodeAffinityOperator("NOT_IN"))
	assert.Equal(t, NodeAffinityOperatorIn, normalizeNodeAffinityOperator("unknown"))
}

func Test_BuildGuestAccelerators(t *testing.T) {
	t.Run("empty config returns nil", func(t *testing.T) {
		out := BuildGuestAccelerators(AdvancedConfig{})
		assert.Nil(t, out)
	})
	t.Run("skips empty type or zero count", func(t *testing.T) {
		cfg := AdvancedConfig{
			GuestAccelerators: []GuestAcceleratorEntry{
				{AcceleratorType: "", AcceleratorCount: 1},
				{AcceleratorType: "nvidia-tesla-t4", AcceleratorCount: 0},
				{AcceleratorType: "  nvidia-tesla-t4  ", AcceleratorCount: 2},
			},
		}
		out := BuildGuestAccelerators(cfg)
		require.Len(t, out, 1)
		assert.Equal(t, "nvidia-tesla-t4", out[0].AcceleratorType)
		assert.Equal(t, int64(2), out[0].AcceleratorCount)
	})
	t.Run("builds multiple accelerators", func(t *testing.T) {
		cfg := AdvancedConfig{
			GuestAccelerators: []GuestAcceleratorEntry{
				{AcceleratorType: "nvidia-tesla-t4", AcceleratorCount: 1},
				{AcceleratorType: "nvidia-tesla-v100", AcceleratorCount: 2},
			},
		}
		out := BuildGuestAccelerators(cfg)
		require.Len(t, out, 2)
		assert.Equal(t, "nvidia-tesla-t4", out[0].AcceleratorType)
		assert.Equal(t, int64(1), out[0].AcceleratorCount)
		assert.Equal(t, "nvidia-tesla-v100", out[1].AcceleratorType)
		assert.Equal(t, int64(2), out[1].AcceleratorCount)
	})
}

func Test_BuildNodeAffinities(t *testing.T) {
	t.Run("empty config returns nil", func(t *testing.T) {
		out := BuildNodeAffinities(AdvancedConfig{})
		assert.Nil(t, out)
	})
	t.Run("skips empty key or empty values", func(t *testing.T) {
		cfg := AdvancedConfig{
			NodeAffinities: []NodeAffinityEntry{
				{Key: "", Operator: NodeAffinityOperatorIn, Values: []string{"v1"}},
				{Key: "key", Operator: NodeAffinityOperatorIn, Values: nil},
				{Key: "  key  ", Operator: NodeAffinityOperatorNotIn, Values: []string{"  v1  "}},
			},
		}
		out := BuildNodeAffinities(cfg)
		require.Len(t, out, 1)
		assert.Equal(t, "key", out[0].Key)
		assert.Equal(t, NodeAffinityOperatorNotIn, out[0].Operator)
		assert.Equal(t, []string{"v1"}, out[0].Values)
	})
}

func Test_BuildInstanceResourcePolicies(t *testing.T) {
	assert.Nil(t, BuildInstanceResourcePolicies(AdvancedConfig{}))
	assert.Equal(t, []string{"policy-1"}, BuildInstanceResourcePolicies(AdvancedConfig{
		ResourcePolicies: []string{"  policy-1  "},
	}))
	assert.Equal(t, []string{"a", "b"}, BuildInstanceResourcePolicies(AdvancedConfig{
		ResourcePolicies: []string{" a ", "", " b "},
	}))
}

func Test_BuildLabels(t *testing.T) {
	assert.Nil(t, BuildLabels(AdvancedConfig{}))
	t.Run("skips empty key and dedupes by key", func(t *testing.T) {
		cfg := AdvancedConfig{
			Labels: []LabelEntry{
				{Key: "", Value: "v"},
				{Key: "env", Value: " prod "},
				{Key: "env", Value: "ignored"},
			},
		}
		out := BuildLabels(cfg)
		require.Len(t, out, 1)
		assert.Equal(t, "prod", out["env"])
	})
}

func Test_ApplyAdvancedScheduling(t *testing.T) {
	t.Run("nil scheduling is no-op", func(t *testing.T) {
		ApplyAdvancedScheduling(nil, AdvancedConfig{MinNodeCpus: 4})
	})
	t.Run("sets node affinities and min node cpus", func(t *testing.T) {
		s := &compute.Scheduling{}
		cfg := AdvancedConfig{
			MinNodeCpus: 4,
			NodeAffinities: []NodeAffinityEntry{
				{Key: "compute.soletenant.node", Operator: NodeAffinityOperatorIn, Values: []string{"node-1"}},
			},
		}
		ApplyAdvancedScheduling(s, cfg)
		require.NotNil(t, s.MinNodeCpus)
		assert.Equal(t, int64(4), s.MinNodeCpus)
		require.Len(t, s.NodeAffinities, 1)
		assert.Equal(t, "compute.soletenant.node", s.NodeAffinities[0].Key)
		assert.Equal(t, []string{"node-1"}, s.NodeAffinities[0].Values)
	})
}
