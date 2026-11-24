package models_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/test/support"
)

func TestCustomAutomationStageFeaturesMigration(t *testing.T) {
	r := support.Setup(t)
	migration, err := os.ReadFile("../../db/migrations/20261124164250_merge-custom-automation-and-stage-features.up.sql")
	require.NoError(t, err)
	tx := database.DB(t.Context()).Begin()
	require.NoError(t, tx.Error)
	defer tx.Rollback()

	for _, test := range []struct {
		name   string
		before string
		after  string
	}{
		{"neither", `[]`, `[]`},
		{"automation only", `["factory_custom_automations"]`, `["factory_custom_automations"]`},
		{"stage only", `["factory_custom_stages"]`, `["factory_custom_automations"]`},
		{"both", `["factory_custom_automations","factory_custom_stages"]`, `["factory_custom_automations"]`},
		{"unrelated only", `["workspace_skills","automations-as-code"]`, `["workspace_skills","automations-as-code"]`},
		{"stage with unrelated", `["workspace_skills","factory_custom_stages","automations-as-code"]`, `["workspace_skills","automations-as-code","factory_custom_automations"]`},
		{"both with unrelated", `["workspace_skills","factory_custom_automations","factory_custom_stages"]`, `["workspace_skills","factory_custom_automations"]`},
	} {
		t.Run(test.name, func(t *testing.T) {
			require.NoError(t, tx.Exec("UPDATE organizations SET enabled_experimental_features = ?::jsonb WHERE id = ?", test.before, r.Organization.ID).Error)
			for range 2 {
				require.NoError(t, tx.Exec(string(migration)).Error)
				var encoded string
				require.NoError(t, tx.Raw("SELECT enabled_experimental_features FROM organizations WHERE id = ?", r.Organization.ID).Scan(&encoded).Error)
				var actual, expected []string
				require.NoError(t, json.Unmarshal([]byte(encoded), &actual))
				require.NoError(t, json.Unmarshal([]byte(test.after), &expected))
				require.Equal(t, expected, actual)
			}
		})
	}
}
