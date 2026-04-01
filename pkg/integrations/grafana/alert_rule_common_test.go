package grafana

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
