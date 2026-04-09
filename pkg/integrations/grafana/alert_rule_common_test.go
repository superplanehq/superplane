package grafana

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
)

func Test__mergeAlertRulePayload__clearsLabelsWhenUpdateSendsEmptyList(t *testing.T) {
	existing := map[string]any{
		"uid":       "rule-1",
		"title":     "Old",
		"folderUID": "f1",
		"ruleGroup": "g1",
		"labels":    map[string]any{"team": "ops", "env": "prod"},
		"annotations": map[string]any{
			"summary": "keep me",
		},
		"data": []any{
			map[string]any{
				"refId":         "A",
				"datasourceUid": "ds1",
				"model":         map[string]any{"expr": "up"},
			},
		},
	}

	emptyLabels := []AlertRuleKeyValuePair{}
	spec := UpdateAlertRuleSpec{
		AlertRuleUID: "rule-1",
		Title:        strPtr("New title"),
		Labels:       &emptyLabels,
	}

	merged, err := mergeAlertRulePayload(existing, spec)
	require.NoError(t, err)

	labels, ok := merged["labels"].(map[string]string)
	require.True(t, ok)
	assert.Empty(t, labels)
}

func strPtr(s string) *string {
	return &s
}

func Test__decodeUpdateAlertRuleSpec__whitespaceOnlyReducerConditionBecomeNil__notificationClearsContact(t *testing.T) {
	ws := "   "
	spec, err := decodeUpdateAlertRuleSpec(map[string]any{
		"alertRuleUid":         "rule-1",
		"title":                "Updated",
		"reducer":              ws,
		"conditionType":        ws,
		"notificationReceiver": ws,
	})
	require.NoError(t, err)
	assert.Nil(t, spec.Reducer)
	assert.Nil(t, spec.ConditionType)
	require.NotNil(t, spec.NotificationReceiver)
	assert.Equal(t, "", *spec.NotificationReceiver)
	assert.NotNil(t, spec.Title)
}

func Test__mergeAlertRulePayload__preservesConditionWhenOnlyQueryFieldsUpdate(t *testing.T) {
	existing := map[string]any{
		"uid":       "rule-1",
		"condition": "A",
		"data": []any{
			map[string]any{
				"refId":         "A",
				"datasourceUid": "ds1",
				"model":         map[string]any{"expr": "up"},
			},
		},
	}

	spec := UpdateAlertRuleSpec{
		AlertRuleUID: "rule-1",
		Query:        strPtr("sum(rate(http_requests_total[5m]))"),
	}

	merged, err := mergeAlertRulePayload(existing, spec)
	require.NoError(t, err)
	assert.Equal(t, "A", merged["condition"])
}

func Test__mergeAlertRulePayload__setsConditionCWhenConditionFieldsUpdate(t *testing.T) {
	existing := map[string]any{
		"uid":       "rule-1",
		"condition": "A",
		"data": []any{
			map[string]any{
				"refId":         "A",
				"datasourceUid": "ds1",
				"model":         map[string]any{"expr": "up"},
			},
			map[string]any{
				"refId": "B",
				"model": map[string]any{"type": "reduce", "reducer": "last", "expression": "A"},
			},
			map[string]any{
				"refId": "C",
				"model": map[string]any{
					"type": "threshold",
					"conditions": []any{
						map[string]any{
							"evaluator": map[string]any{"type": "gt", "params": []any{float64(0.5)}},
						},
					},
				},
			},
		},
	}

	spec := UpdateAlertRuleSpec{
		AlertRuleUID: "rule-1",
		Threshold:    float64Ptr(99),
	}

	merged, err := mergeAlertRulePayload(existing, spec)
	require.NoError(t, err)
	assert.Equal(t, alertRuleConditionRefID, merged["condition"])
}

func Test__mergeAlertRulePayload__clearsNotificationSettingsWhenReceiverEmpty(t *testing.T) {
	existing := map[string]any{
		"uid":   "rule-1",
		"title": "Old",
		"notification_settings": map[string]any{
			"receiver": "slack-alerts",
		},
		"data": []any{
			map[string]any{"refId": "A", "datasourceUid": "ds1", "model": map[string]any{"expr": "up"}},
		},
	}

	spec, err := decodeUpdateAlertRuleSpec(map[string]any{
		"alertRuleUid":         "rule-1",
		"notificationReceiver": "",
	})
	require.NoError(t, err)

	merged, err := mergeAlertRulePayload(existing, spec)
	require.NoError(t, err)
	_, has := merged["notification_settings"]
	assert.False(t, has)
}

func Test__validateUpdateAlertRuleSpec__rejectsInvalidReducerAndConditionType(t *testing.T) {
	err := validateUpdateAlertRuleSpec(UpdateAlertRuleSpec{
		AlertRuleUID:    "rule-1",
		Reducer:         strPtr("foobar"),
		ConditionType:   strPtr("gt"),
		Threshold:       float64Ptr(1),
		LookbackSeconds: intPtr(60),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid reducer")

	err = validateUpdateAlertRuleSpec(UpdateAlertRuleSpec{
		AlertRuleUID:    "rule-1",
		Reducer:         strPtr("last"),
		ConditionType:   strPtr("not_a_condition"),
		Threshold:       float64Ptr(1),
		LookbackSeconds: intPtr(60),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid conditionType")
}

func Test__mergeAlertRulePayload__rejectsRangeConditionWithoutThreshold2(t *testing.T) {
	existing := map[string]any{
		"uid":       "rule-1",
		"condition": "C",
		"data": []any{
			map[string]any{
				"refId":         "A",
				"datasourceUid": "ds1",
				"model":         map[string]any{"expr": "up"},
			},
			map[string]any{
				"refId": "B",
				"model": map[string]any{"type": "reduce", "reducer": "last", "expression": "A"},
			},
			map[string]any{
				"refId": "C",
				"model": map[string]any{
					"type": "threshold",
					"conditions": []any{
						map[string]any{
							"evaluator": map[string]any{"type": "gt", "params": []any{float64(1)}},
						},
					},
				},
			},
		},
	}

	spec := UpdateAlertRuleSpec{
		AlertRuleUID:  "rule-1",
		ConditionType: strPtr("within_range"),
	}

	_, err := mergeAlertRulePayload(existing, spec)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "threshold2 is required for range conditions")
}

func float64Ptr(f float64) *float64 {
	return &f
}

func intPtr(i int) *int {
	return &i
}

func Test__alertRuleFieldConfiguration__noDataAndExecErrStateSelectOptions(t *testing.T) {
	fields := alertRuleFieldConfiguration(false, false)
	noDataVals := alertRuleSelectOptionValues(t, fields, "noDataState")
	execErrVals := alertRuleSelectOptionValues(t, fields, "execErrState")

	assert.Contains(t, noDataVals, "NoData")
	assert.NotContains(t, noDataVals, "Error")

	assert.Contains(t, execErrVals, "Error")
	assert.NotContains(t, execErrVals, "NoData")
}

func Test__validateCreateAlertRuleSpec__rejectsEmptyAndInvalidForNoDataExecErr(t *testing.T) {
	base := CreateAlertRuleSpec{
		Title:           "test",
		FolderUID:       "f1",
		RuleGroup:       "g1",
		DataSourceUID:   "ds1",
		Query:           "up",
		LookbackSeconds: 60,
		Reducer:         "last",
		ConditionType:   "gt",
		Threshold:       float64Ptr(1),
		For:             "5m",
		NoDataState:     "NoData",
		ExecErrState:    "Error",
	}

	// Valid spec passes
	require.NoError(t, validateCreateAlertRuleSpec(base))

	// Empty for
	s := base
	s.For = ""
	err := validateCreateAlertRuleSpec(s)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "for is required")

	// Invalid for
	s = base
	s.For = "abc"
	err = validateCreateAlertRuleSpec(s)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid for")

	// Empty noDataState
	s = base
	s.NoDataState = ""
	err = validateCreateAlertRuleSpec(s)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "noDataState is required")

	// Invalid noDataState
	s = base
	s.NoDataState = "BadValue"
	err = validateCreateAlertRuleSpec(s)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid noDataState")

	// Empty execErrState
	s = base
	s.ExecErrState = ""
	err = validateCreateAlertRuleSpec(s)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "execErrState is required")

	// Invalid execErrState
	s = base
	s.ExecErrState = "BadValue"
	err = validateCreateAlertRuleSpec(s)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid execErrState")
}

func Test__validateUpdateAlertRuleSpec__rejectsInvalidForNoDataExecErr(t *testing.T) {
	err := validateUpdateAlertRuleSpec(UpdateAlertRuleSpec{
		AlertRuleUID: "rule-1",
		For:          strPtr("not-a-duration"),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid for")

	err = validateUpdateAlertRuleSpec(UpdateAlertRuleSpec{
		AlertRuleUID: "rule-1",
		NoDataState:  strPtr("Invalid"),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid noDataState")

	err = validateUpdateAlertRuleSpec(UpdateAlertRuleSpec{
		AlertRuleUID: "rule-1",
		ExecErrState: strPtr("Invalid"),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid execErrState")

	// Valid values pass
	err = validateUpdateAlertRuleSpec(UpdateAlertRuleSpec{
		AlertRuleUID: "rule-1",
		For:          strPtr("10m"),
		NoDataState:  strPtr("OK"),
		ExecErrState: strPtr("KeepLast"),
	})
	require.NoError(t, err)
}

func Test__isValidDuration(t *testing.T) {
	valid := []string{"0s", "5m", "1h", "1h30m", "2d", "30s"}
	for _, d := range valid {
		assert.True(t, isValidDuration(d), "expected %q to be valid", d)
	}

	invalid := []string{"", "abc", "5", "5x", "m5", " 5m"}
	for _, d := range invalid {
		assert.False(t, isValidDuration(d), "expected %q to be invalid", d)
	}
}

func alertRuleSelectOptionValues(t *testing.T, fields []configuration.Field, name string) []string {
	t.Helper()
	for _, f := range fields {
		if f.Name != name {
			continue
		}
		require.NotNil(t, f.TypeOptions)
		require.NotNil(t, f.TypeOptions.Select)
		out := make([]string, len(f.TypeOptions.Select.Options))
		for i, o := range f.TypeOptions.Select.Options {
			out[i] = o.Value
		}
		return out
	}
	t.Fatalf("field %q not found", name)
	return nil
}
