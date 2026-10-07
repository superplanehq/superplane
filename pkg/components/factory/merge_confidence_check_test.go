package factory

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	checkfactory "github.com/superplanehq/superplane/pkg/models/factory"
)

func TestMergeConfidenceCheckParams(t *testing.T) {
	enabled := []string{"risk", "performance", "security", "drift", "reversibility"}
	cases := []struct {
		name  string
		check string
		score float64
		key   string
		label string
		level string
	}{
		{name: "blast radius healthy", check: "risk", score: 1, key: "risk-review", label: "Blast radius", level: checkfactory.CheckLevelPositive},
		{name: "blast radius caution", check: "risk", score: 2, key: "risk-review", label: "Blast radius", level: checkfactory.CheckLevelCaution},
		{name: "blast radius critical", check: "risk", score: 3, key: "risk-review", label: "Blast radius", level: checkfactory.CheckLevelCritical},
		{name: "performance healthy", check: "performance", score: 3, key: "performance-review", label: "Performance", level: checkfactory.CheckLevelPositive},
		{name: "performance caution", check: "performance", score: 2, key: "performance-review", label: "Performance", level: checkfactory.CheckLevelCaution},
		{name: "performance critical", check: "performance", score: 1, key: "performance-review", label: "Performance", level: checkfactory.CheckLevelCritical},
		{name: "security caution", check: "security", score: 2, key: "security-review", label: "Security", level: checkfactory.CheckLevelCaution},
		{name: "drift critical", check: "drift", score: 3, key: "drift-review", label: "Drift from Specification", level: checkfactory.CheckLevelCritical},
		{name: "reversibility healthy", check: "reversibility", score: 3, key: "reversibility-review", label: "Reversibility", level: checkfactory.CheckLevelPositive},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			params, err := mergeConfidenceCheckParams(tc.check, tc.score, "One sentence.", enabled, nil, 3)
			require.NoError(t, err)
			assert.Equal(t, tc.key, params.Key)
			assert.Equal(t, tc.label, params.Name)
			assert.Equal(t, tc.level, params.Level)
			assert.Equal(t, checkfactory.CheckFormatFraction, params.Format)
			assert.Equal(t, float64(3), params.MaxScore)
			assert.Equal(t, tc.score, params.Score)
		})
	}
}

func TestMergeConfidenceCheckParamsRejectsDisabledAndInvalidScores(t *testing.T) {
	_, err := mergeConfidenceCheckParams("performance", 3, "One sentence.", []string{"risk"}, nil, 3)
	assert.ErrorIs(t, err, ErrMergeConfidenceDisabled)

	_, err = mergeConfidenceCheckParams("risk", 4, "One sentence.", []string{"risk"}, nil, 3)
	assert.ErrorIs(t, err, ErrMergeConfidenceInvalid)
	assert.ErrorContains(t, err, "score must be an integer from 1 through 3")

	_, err = mergeConfidenceCheckParams("risk", 1.5, "One sentence.", []string{"risk"}, nil, 3)
	assert.ErrorIs(t, err, ErrMergeConfidenceInvalid)

	_, err = mergeConfidenceCheckParams("not a check", 1, "One sentence.", []string{"not a check"}, nil, 3)
	assert.ErrorIs(t, err, ErrMergeConfidenceInvalid)

	_, err = mergeConfidenceCheckParams("risk", 1, "  ", []string{"risk"}, nil, 3)
	assert.ErrorIs(t, err, ErrMergeConfidenceInvalid)
}

func TestMergeConfidenceCheckParamsKeepsTheLegacyScale(t *testing.T) {
	enabled := []string{"risk", "performance"}

	healthy, err := mergeConfidenceCheckParams("performance", 5, "No performance practice applies.", enabled, nil, 5)
	require.NoError(t, err)
	assert.Equal(t, checkfactory.CheckLevelPositive, healthy.Level)
	assert.Equal(t, float64(5), healthy.MaxScore)

	critical, err := mergeConfidenceCheckParams("performance", 2, "The change breaks a practice that applies.", enabled, nil, 5)
	require.NoError(t, err)
	assert.Equal(t, checkfactory.CheckLevelCritical, critical.Level)

	caution, err := mergeConfidenceCheckParams("risk", 3, "The change matches a medium category.", enabled, nil, 5)
	require.NoError(t, err)
	assert.Equal(t, checkfactory.CheckLevelCaution, caution.Level)

	_, err = mergeConfidenceCheckParams("risk", 6, "One sentence.", enabled, nil, 5)
	assert.ErrorIs(t, err, ErrMergeConfidenceInvalid)
	assert.ErrorContains(t, err, "score must be an integer from 1 through 5")
}

func TestMergeConfidenceCustomCheckUsesTheStepName(t *testing.T) {
	params, err := mergeConfidenceCheckParams("api-latency", 3, "The new query scans the whole table.", []string{"api-latency"}, map[string]string{
		"api-latency": "API latency",
	}, 3)
	require.NoError(t, err)
	assert.Equal(t, "api-latency-review", params.Key)
	assert.Equal(t, "API latency", params.Name)
	assert.Equal(t, checkfactory.CheckLevelCritical, params.Level)
	assert.Equal(t, float64(3), params.MaxScore)

	caution, err := mergeConfidenceCheckParams("api-latency", 2, "The new query scans one extra index.", []string{"api-latency"}, nil, 3)
	require.NoError(t, err)
	assert.Equal(t, checkfactory.CheckLevelCaution, caution.Level)

	_, err = mergeConfidenceCheckParams("api-latency", 3, "The new query scans the whole table.", []string{"risk"}, nil, 3)
	assert.ErrorIs(t, err, ErrMergeConfidenceDisabled)
}
