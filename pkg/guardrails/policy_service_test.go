package guardrails

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestValidatePolicyRequest_ValidModes(t *testing.T) {
	for _, mode := range []string{
		models.GuardrailEnforcementAuditOnly,
		models.GuardrailEnforcementWarnOnly,
		models.GuardrailEnforcementSoftBlock,
		models.GuardrailEnforcementHardBlock,
	} {
		req := UpsertOrgPolicyRequest{
			OrgID:                   uuid.New(),
			CallerID:                uuid.New(),
			EnforcementMode:         mode,
			SoftBlockScoreThreshold: 60,
			HardBlockScoreThreshold: 80,
			ClassifierSamplingRate:  0.5,
			ClassifierSensitivity:   models.GuardrailClassifierSensitivityBalanced,
			SoftBlockTimeoutSeconds: 3600,
		}
		assert.NoError(t, validatePolicyRequest(req), "mode=%s should be valid", mode)
	}
}

func TestValidatePolicyRequest_InvalidMode(t *testing.T) {
	req := UpsertOrgPolicyRequest{
		OrgID:                   uuid.New(),
		EnforcementMode:         "super_block",
		SoftBlockScoreThreshold: 60,
		HardBlockScoreThreshold: 80,
		ClassifierSamplingRate:  0.5,
	}
	err := validatePolicyRequest(req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid enforcement_mode")
}

func TestValidatePolicyRequest_HardBelowSoft(t *testing.T) {
	req := UpsertOrgPolicyRequest{
		OrgID:                   uuid.New(),
		EnforcementMode:         models.GuardrailEnforcementHardBlock,
		SoftBlockScoreThreshold: 80,
		HardBlockScoreThreshold: 60, // hard < soft — invalid
		ClassifierSamplingRate:  0.5,
	}
	err := validatePolicyRequest(req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "hard_block_score_threshold")
}

func TestValidatePolicyRequest_SamplingRateOutOfRange(t *testing.T) {
	req := UpsertOrgPolicyRequest{
		OrgID:                   uuid.New(),
		EnforcementMode:         models.GuardrailEnforcementAuditOnly,
		SoftBlockScoreThreshold: 60,
		HardBlockScoreThreshold: 80,
		ClassifierSamplingRate:  1.5, // > 1.0 — invalid
	}
	err := validatePolicyRequest(req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "classifier_sampling_rate")
}

func TestValidatePolicyRequest_InvalidSensitivity(t *testing.T) {
	req := UpsertOrgPolicyRequest{
		OrgID:                   uuid.New(),
		EnforcementMode:         models.GuardrailEnforcementAuditOnly,
		SoftBlockScoreThreshold: 60,
		HardBlockScoreThreshold: 80,
		ClassifierSamplingRate:  0.5,
		ClassifierSensitivity:   "ultra_strict",
	}
	err := validatePolicyRequest(req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid classifier_sensitivity")
}

func TestShouldSample_ZeroRate_NeverSamples(t *testing.T) {
	for i := 0; i < 100; i++ {
		assert.False(t, shouldSample(0))
	}
}

func TestShouldSample_FullRate_AlwaysSamples(t *testing.T) {
	for i := 0; i < 100; i++ {
		assert.True(t, shouldSample(1.0))
	}
}

func TestShouldSample_HalfRate_RoughlyHalf(t *testing.T) {
	hits := 0
	const trials = 10000
	for i := 0; i < trials; i++ {
		if shouldSample(0.5) {
			hits++
		}
	}
	// Expect roughly 50% ± 5% (very loose bound for a unit test)
	assert.Greater(t, hits, trials*40/100)
	assert.Less(t, hits, trials*60/100)
}
