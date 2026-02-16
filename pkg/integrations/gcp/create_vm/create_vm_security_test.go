package createvm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_BuildShieldedInstanceConfig(t *testing.T) {
	t.Run("disabled returns nil", func(t *testing.T) {
		cfg := SecurityConfig{ShieldedVM: false}
		assert.Nil(t, BuildShieldedInstanceConfig(cfg))
	})
	t.Run("enabled with zero values for options", func(t *testing.T) {
		cfg := SecurityConfig{ShieldedVM: true}
		out := BuildShieldedInstanceConfig(cfg)
		require.NotNil(t, out)
		assert.False(t, out.EnableSecureBoot)
		assert.False(t, out.EnableVtpm)
		assert.False(t, out.EnableIntegrityMonitoring)
	})
	t.Run("secure boot can be enabled", func(t *testing.T) {
		cfg := SecurityConfig{
			ShieldedVM:                          true,
			ShieldedVMEnableSecureBoot:          true,
			ShieldedVMEnableVtpm:                false,
			ShieldedVMEnableIntegrityMonitoring: false,
		}
		out := BuildShieldedInstanceConfig(cfg)
		require.NotNil(t, out)
		assert.True(t, out.EnableSecureBoot)
		assert.False(t, out.EnableVtpm)
		assert.False(t, out.EnableIntegrityMonitoring)
	})
}

func Test_BuildConfidentialInstanceConfig(t *testing.T) {
	t.Run("disabled returns nil", func(t *testing.T) {
		cfg := SecurityConfig{ConfidentialVM: false}
		assert.Nil(t, BuildConfidentialInstanceConfig(cfg))
	})
	t.Run("enabled defaults to SEV", func(t *testing.T) {
		cfg := SecurityConfig{ConfidentialVM: true}
		out := BuildConfidentialInstanceConfig(cfg)
		require.NotNil(t, out)
		assert.True(t, out.EnableConfidentialCompute)
		assert.Equal(t, ConfidentialInstanceTypeSEV, out.ConfidentialInstanceType)
	})
	t.Run("explicit type", func(t *testing.T) {
		cfg := SecurityConfig{
			ConfidentialVM:     true,
			ConfidentialVMType: ConfidentialInstanceTypeTDX,
		}
		out := BuildConfidentialInstanceConfig(cfg)
		require.NotNil(t, out)
		assert.Equal(t, ConfidentialInstanceTypeTDX, out.ConfidentialInstanceType)
	})
}
