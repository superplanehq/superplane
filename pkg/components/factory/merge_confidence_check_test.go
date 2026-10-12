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
		check string
		score float64
		key   string
		name  string
		level string
	}{
		{check: "risk", score: 1, key: "risk-review", name: "Blast radius", level: checkfactory.CheckLevelPositive},
		{check: "risk", score: 3, key: "risk-review", name: "Blast radius", level: checkfactory.CheckLevelCaution},
		{check: "risk", score: 4, key: "risk-review", name: "Blast radius", level: checkfactory.CheckLevelCritical},
		{check: "performance", score: 5, key: "performance-review", name: "Performance", level: checkfactory.CheckLevelPositive},
		{check: "performance", score: 2, key: "performance-review", name: "Performance", level: checkfactory.CheckLevelCritical},
		{check: "security", score: 3, key: "security-review", name: "Security", level: checkfactory.CheckLevelCaution},
		{check: "drift", score: 4, key: "drift-review", name: "Drift from Specification", level: checkfactory.CheckLevelCritical},
		{check: "reversibility", score: 5, key: "reversibility-review", name: "Reversibility", level: checkfactory.CheckLevelPositive},
	}

	for _, tc := range cases {
		t.Run(tc.check, func(t *testing.T) {
			params, err := mergeConfidenceCheckParams(tc.check, tc.score, "One sentence.", enabled, nil)
			require.NoError(t, err)
			assert.Equal(t, tc.key, params.Key)
			assert.Equal(t, tc.name, params.Name)
			assert.Equal(t, tc.level, params.Level)
			assert.Equal(t, checkfactory.CheckFormatFraction, params.Format)
			assert.Equal(t, float64(5), params.MaxScore)
			assert.Equal(t, tc.score, params.Score)
		})
	}
}

func TestMergeConfidenceCheckParamsRejectsDisabledAndInvalidScores(t *testing.T) {
	_, err := mergeConfidenceCheckParams("performance", 5, "One sentence.", []string{"risk"}, nil)
	assert.ErrorIs(t, err, ErrMergeConfidenceDisabled)

	_, err = mergeConfidenceCheckParams("risk", 6, "One sentence.", []string{"risk"}, nil)
	assert.ErrorIs(t, err, ErrMergeConfidenceInvalid)

	_, err = mergeConfidenceCheckParams("risk", 1.5, "One sentence.", []string{"risk"}, nil)
	assert.ErrorIs(t, err, ErrMergeConfidenceInvalid)

	_, err = mergeConfidenceCheckParams("not a check", 1, "One sentence.", []string{"not a check"}, nil)
	assert.ErrorIs(t, err, ErrMergeConfidenceInvalid)

	_, err = mergeConfidenceCheckParams("risk", 1, "  ", []string{"risk"}, nil)
	assert.ErrorIs(t, err, ErrMergeConfidenceInvalid)
}

func TestMergeConfidenceCustomCheckUsesTheStepName(t *testing.T) {
	params, err := mergeConfidenceCheckParams("api-latency", 4, "The new query scans the whole table.", []string{"api-latency"}, map[string]string{
		"api-latency": "API latency",
	})
	require.NoError(t, err)
	assert.Equal(t, "api-latency-review", params.Key)
	assert.Equal(t, "API latency", params.Name)
	assert.Equal(t, checkfactory.CheckLevelCritical, params.Level)

	_, err = mergeConfidenceCheckParams("api-latency", 4, "The new query scans the whole table.", []string{"risk"}, nil)
	assert.ErrorIs(t, err, ErrMergeConfidenceDisabled)
}
