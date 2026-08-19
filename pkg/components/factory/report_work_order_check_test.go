package factory

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models/factory"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func float64Ptr(value float64) *float64 {
	return &value
}

func TestComputeCheckLevel(t *testing.T) {
	t.Run("no thresholds stays neutral", func(t *testing.T) {
		level, err := computeCheckLevel(5, CheckDirectionLowerIsBetter, nil, nil)
		require.NoError(t, err)
		assert.Equal(t, factory.CheckLevelNeutral, level)
	})

	t.Run("lower is better escalates as the score rises", func(t *testing.T) {
		cases := []struct {
			score float64
			want  string
		}{
			{3, factory.CheckLevelPositive},
			{5, factory.CheckLevelCaution},
			{6, factory.CheckLevelCaution},
			{8, factory.CheckLevelCritical},
			{9, factory.CheckLevelCritical},
		}
		for _, tc := range cases {
			level, err := computeCheckLevel(tc.score, CheckDirectionLowerIsBetter, float64Ptr(5), float64Ptr(8))
			require.NoError(t, err)
			assert.Equal(t, tc.want, level, "score %v", tc.score)
		}
	})

	t.Run("higher is better escalates as the score drops", func(t *testing.T) {
		cases := []struct {
			score float64
			want  string
		}{
			{90, factory.CheckLevelPositive},
			{70, factory.CheckLevelCaution},
			{40, factory.CheckLevelCritical},
		}
		for _, tc := range cases {
			level, err := computeCheckLevel(tc.score, CheckDirectionHigherIsBetter, float64Ptr(70), float64Ptr(40))
			require.NoError(t, err)
			assert.Equal(t, tc.want, level, "score %v", tc.score)
		}
	})

	t.Run("single threshold works alone", func(t *testing.T) {
		level, err := computeCheckLevel(9, CheckDirectionLowerIsBetter, nil, float64Ptr(8))
		require.NoError(t, err)
		assert.Equal(t, factory.CheckLevelCritical, level)

		level, err = computeCheckLevel(2, CheckDirectionLowerIsBetter, nil, float64Ptr(8))
		require.NoError(t, err)
		assert.Equal(t, factory.CheckLevelPositive, level)
	})

	t.Run("empty direction defaults to higher is better", func(t *testing.T) {
		level, err := computeCheckLevel(30, "", float64Ptr(50), nil)
		require.NoError(t, err)
		assert.Equal(t, factory.CheckLevelCaution, level)
	})

	t.Run("rejects unknown direction", func(t *testing.T) {
		_, err := computeCheckLevel(1, "sideways", nil, nil)
		require.Error(t, err)
	})

	t.Run("rejects unreachable threshold order", func(t *testing.T) {
		_, err := computeCheckLevel(1, CheckDirectionLowerIsBetter, float64Ptr(8), float64Ptr(5))
		require.Error(t, err)

		_, err = computeCheckLevel(1, CheckDirectionHigherIsBetter, float64Ptr(40), float64Ptr(70))
		require.Error(t, err)
	})
}

func TestReportWorkOrderCheck_Execute(t *testing.T) {
	component := &ReportWorkOrderCheck{}
	reported := &core.WorkOrderCheck{
		ID:          "chk-1",
		WorkOrderID: "wo-1",
		Key:         "risk-review",
		Name:        "Risk review",
		Score:       6,
		MaxScore:    10,
		Format:      "fraction",
		Level:       "caution",
	}

	t.Run("reports the check and emits workOrder.checkReported", func(t *testing.T) {
		factoryCtx := &fakeFactoryContext{reportCheckResult: reported}
		stateCtx := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"orderId":    "wo-1",
				"checkKey":   "risk-review",
				"name":       "Risk review",
				"score":      "6",
				"maxScore":   "10",
				"direction":  CheckDirectionLowerIsBetter,
				"cautionAt":  5,
				"criticalAt": 8,
				"summary":    "Migration touches shared tables.",
			},
			ExecutionState: stateCtx,
			Factory:        factoryCtx,
		})
		require.NoError(t, err)
		assert.Equal(t, 1, factoryCtx.reportCheckCalls)
		assert.Equal(t, "wo-1", factoryCtx.reportCheckParams.OrderID)
		assert.Equal(t, "risk-review", factoryCtx.reportCheckParams.CheckKey)
		assert.Equal(t, float64(6), factoryCtx.reportCheckParams.Score)
		assert.Equal(t, float64(10), factoryCtx.reportCheckParams.MaxScore)
		assert.Equal(t, factory.CheckLevelCaution, factoryCtx.reportCheckParams.Level)
		assert.Equal(t, core.DefaultOutputChannel.Name, stateCtx.Channel)
		assert.Equal(t, "workOrder.checkReported", stateCtx.Type)
		assert.Len(t, stateCtx.Payloads, 1)
	})

	t.Run("fails when the score expression did not resolve to a number", func(t *testing.T) {
		factoryCtx := &fakeFactoryContext{}
		stateCtx := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"orderId":  "wo-1",
				"checkKey": "risk-review",
				"name":     "Risk review",
				"score":    "not-a-number",
				"maxScore": "10",
			},
			ExecutionState: stateCtx,
			Factory:        factoryCtx,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "score must be a number")
		assert.Equal(t, 0, factoryCtx.reportCheckCalls)
	})

	t.Run("boolean pass reports 1/1 with level positive", func(t *testing.T) {
		factoryCtx := &fakeFactoryContext{reportCheckResult: reported}
		stateCtx := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"orderId":  "wo-1",
				"checkKey": "ci",
				"name":     "CI",
				"format":   factory.CheckFormatBoolean,
				"passed":   "true",
			},
			ExecutionState: stateCtx,
			Factory:        factoryCtx,
		})
		require.NoError(t, err)
		assert.Equal(t, float64(1), factoryCtx.reportCheckParams.Score)
		assert.Equal(t, float64(1), factoryCtx.reportCheckParams.MaxScore)
		assert.Equal(t, factory.CheckFormatBoolean, factoryCtx.reportCheckParams.Format)
		assert.Equal(t, factory.CheckLevelPositive, factoryCtx.reportCheckParams.Level)
	})

	t.Run("boolean fail reports 0/1 with level critical", func(t *testing.T) {
		factoryCtx := &fakeFactoryContext{reportCheckResult: reported}
		stateCtx := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"orderId":  "wo-1",
				"checkKey": "ci",
				"name":     "CI",
				"format":   factory.CheckFormatBoolean,
				"passed":   "false",
			},
			ExecutionState: stateCtx,
			Factory:        factoryCtx,
		})
		require.NoError(t, err)
		assert.Equal(t, float64(0), factoryCtx.reportCheckParams.Score)
		assert.Equal(t, float64(1), factoryCtx.reportCheckParams.MaxScore)
		assert.Equal(t, factory.CheckLevelCritical, factoryCtx.reportCheckParams.Level)
	})

	t.Run("boolean fails when passed did not resolve to a bool", func(t *testing.T) {
		factoryCtx := &fakeFactoryContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"orderId":  "wo-1",
				"checkKey": "ci",
				"name":     "CI",
				"format":   factory.CheckFormatBoolean,
				"passed":   "maybe",
			},
			ExecutionState: &contexts.ExecutionStateContext{},
			Factory:        factoryCtx,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "passed must be true or false")
		assert.Equal(t, 0, factoryCtx.reportCheckCalls)
	})

	t.Run("boolean rejects score thresholds", func(t *testing.T) {
		factoryCtx := &fakeFactoryContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"orderId":   "wo-1",
				"checkKey":  "ci",
				"name":      "CI",
				"format":    factory.CheckFormatBoolean,
				"passed":    "true",
				"cautionAt": 5,
			},
			ExecutionState: &contexts.ExecutionStateContext{},
			Factory:        factoryCtx,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "do not apply to the boolean format")
		assert.Equal(t, 0, factoryCtx.reportCheckCalls)
	})

	t.Run("numeric formats reject a passed value", func(t *testing.T) {
		factoryCtx := &fakeFactoryContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"orderId":  "wo-1",
				"checkKey": "risk-review",
				"name":     "Risk review",
				"score":    "6",
				"maxScore": "10",
				"passed":   "true",
			},
			ExecutionState: &contexts.ExecutionStateContext{},
			Factory:        factoryCtx,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "passed only applies to the boolean format")
		assert.Equal(t, 0, factoryCtx.reportCheckCalls)
	})

	t.Run("propagates errors from the factory context", func(t *testing.T) {
		factoryCtx := &fakeFactoryContext{reportCheckErr: errors.New("boom")}
		stateCtx := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"orderId":  "wo-1",
				"checkKey": "risk-review",
				"name":     "Risk review",
				"score":    "6",
				"maxScore": "10",
			},
			ExecutionState: stateCtx,
			Factory:        factoryCtx,
		})
		require.Error(t, err)
		assert.EqualError(t, err, "boom")
	})
}

func TestReportWorkOrderCheck_ValidatesConfiguration(t *testing.T) {
	c := &ReportWorkOrderCheck{}
	fields := c.Configuration()

	t.Run("requires orderId, checkKey, name, and score", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"maxScore": "10",
		})
		if err == nil {
			t.Fatal("expected error for missing required fields")
		}
	})

	t.Run("accepts a full configuration", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"orderId":    "{{ order().id }}",
			"checkKey":   "risk-review",
			"name":       "Risk review",
			"score":      "{{ previous().data.risk.score }}",
			"maxScore":   "10",
			"format":     "fraction",
			"direction":  CheckDirectionLowerIsBetter,
			"cautionAt":  5,
			"criticalAt": 8,
			"summary":    "One-line takeaway",
			"analysis":   "### Findings\n\nDetails here.",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("requires score for numeric formats", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"orderId":  "{{ order().id }}",
			"checkKey": "risk-review",
			"name":     "Risk review",
			"format":   "fraction",
			"maxScore": "10",
		})
		if err == nil {
			t.Fatal("expected error for missing score")
		}
	})

	t.Run("requires score when format is omitted", func(t *testing.T) {
		// Configs saved before the format field existed omit it and rely
		// on the fraction default — score must still be required at save
		// time, not only at run time.
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"orderId":  "{{ order().id }}",
			"checkKey": "risk-review",
			"name":     "Risk review",
			"maxScore": "10",
		})
		if err == nil {
			t.Fatal("expected error for missing score")
		}
	})

	t.Run("requires passed for the boolean format", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"orderId":  "{{ order().id }}",
			"checkKey": "ci",
			"name":     "CI",
			"format":   "boolean",
		})
		if err == nil {
			t.Fatal("expected error for missing passed")
		}
	})

	t.Run("accepts a boolean configuration", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"orderId":  "{{ order().id }}",
			"checkKey": "ci",
			"name":     "CI",
			"format":   "boolean",
			"passed":   "{{ previous().data.ci.passed }}",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("rejects unknown format", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"orderId":  "{{ order().id }}",
			"checkKey": "risk-review",
			"name":     "Risk review",
			"score":    "6",
			"maxScore": "10",
			"format":   "ratio",
		})
		if err == nil {
			t.Fatal("expected error for invalid format option")
		}
	})

	t.Run("rejects unknown direction", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"orderId":   "{{ order().id }}",
			"checkKey":  "risk-review",
			"name":      "Risk review",
			"score":     "6",
			"maxScore":  "10",
			"direction": "sideways",
		})
		if err == nil {
			t.Fatal("expected error for invalid direction option")
		}
	})
}
