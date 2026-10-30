package models_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test__ResolveBYOKRunnerModel__StandaloneKeepsConfiguredModel(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	configuration := map[string]any{
		"model":       "claude-sonnet-4-6",
		"credentials": map[string]any{"source": "secret"},
	}

	resolved, err := models.ResolveBYOKRunnerModel(db, r.Organization.ID, nil, "runnerClaudeCode", configuration)
	require.NoError(t, err)
	assert.Equal(t, "claude-sonnet-4-6", resolved)

	_, err = models.ResolveBYOKRunnerModel(db, r.Organization.ID, &factory.ID, "runnerClaudeCode", configuration)
	assert.ErrorIs(t, err, models.ErrSelectableLLMModelNotAllowed)
}
