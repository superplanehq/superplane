package mergeconfidence

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models/factory"
	"testing"
)

func TestAssessmentReportData(t *testing.T) {
	for _, tc := range []struct {
		name       string
		passed     bool
		enabled    []string
		scores     map[string]float64
		levels     map[string]string
		want       float64
		wantLevel  string
		incomplete bool
	}{
		{name: "invert risk and drift and take weakest", passed: true, enabled: []string{"risk", "performance", "security", "drift", "reversibility"}, scores: map[string]float64{"risk": 2, "performance": 5, "security": 4, "drift": 4, "reversibility": 5}, levels: map[string]string{"risk": "positive", "performance": "positive", "security": "positive", "drift": "critical", "reversibility": "positive"}, want: 2, wantLevel: "critical"},
		{name: "disabled checks and custom score", passed: true, enabled: []string{"performance", "custom"}, scores: map[string]float64{"performance": 4, "custom": 1, "risk": 5}, levels: map[string]string{"performance": "positive", "custom": "critical", "risk": "critical"}, want: 4, wantLevel: "positive"},
		{name: "unclassified check", passed: true, enabled: []string{"security"}, scores: map[string]float64{"security": 4}, want: 4, wantLevel: "neutral"},
		{name: "low blast radius", passed: true, enabled: []string{"risk"}, scores: map[string]float64{"risk": 1}, want: 5},
		{name: "high blast radius", passed: true, enabled: []string{"risk"}, scores: map[string]float64{"risk": 5}, want: 1},
		{name: "low drift", passed: true, enabled: []string{"drift"}, scores: map[string]float64{"drift": 1}, want: 5},
		{name: "high drift", passed: true, enabled: []string{"drift"}, scores: map[string]float64{"drift": 5}, want: 1},
		{name: "custom only", passed: true, enabled: []string{"custom"}, scores: map[string]float64{"custom": 2}},
		{name: "all disabled", passed: true},
		{name: "missing result", passed: true, enabled: []string{"risk", "security"}, scores: map[string]float64{"risk": 1}, incomplete: true},
		{name: "failed with all results", enabled: []string{"risk"}, scores: map[string]float64{"risk": 1}, incomplete: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			checks := map[string]factory.CheckRef{}
			for id, score := range tc.scores {
				checks[id+"-review"] = factory.CheckRef{Score: score, MaxScore: 5, Level: tc.levels[id], Summary: "available result"}
			}
			report := (Assessment{Passed: tc.passed, Enabled: tc.enabled, Checks: checks, ReviewedCommit: "abc123", TaskURL: "https://example.com/task"}).ReportData()
			assert.Equal(t, !tc.incomplete, report.Complete)
			if tc.want > 0 {
				require.NotNil(t, report.OverallScore)
				assert.Equal(t, tc.want, *report.OverallScore)
			} else {
				assert.Nil(t, report.OverallScore)
				assert.Empty(t, report.OverallLevel)
			}
			if tc.wantLevel != "" {
				assert.Equal(t, tc.wantLevel, report.OverallLevel)
			}
			require.Len(t, report.Checks, len(tc.enabled))
			for i, id := range tc.enabled {
				assert.Equal(t, id, report.Checks[i].ID)
				_, present := tc.scores[id]
				assert.Equal(t, !present, report.Checks[i].Missing)
			}
			assert.Equal(t, "abc123", report.ReviewedCommit)
			assert.Equal(t, "https://example.com/task", report.TaskURL)
		})
	}
}
