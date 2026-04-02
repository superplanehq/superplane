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
