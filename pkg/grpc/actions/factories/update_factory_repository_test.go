package factories

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestReplaceConfigurationValues(t *testing.T) {
	configuration := map[string]any{
		"repository": "acme/old",
		"environment": []any{
			map[string]any{"name": "REPO", "value": "acme/old"},
			map[string]any{"name": "BASE", "value": "main"},
		},
		"command": "git clone --branch ${BASE:-main} https://github.com/acme/main-service.git",
	}

	replaced, changed := replaceConfigurationValues(configuration, []configurationReplacement{
		{from: "acme/old", to: "{{ order().repository }}"},
		{from: "main", to: "{{ order().default_branch }}"},
	})

	assert.True(t, changed)
	assert.Equal(t, "acme/old", configuration["repository"])
	assert.Equal(t, "{{ order().repository }}", replaced.(map[string]any)["repository"])
	assert.Equal(t, "{{ order().default_branch }}", replaced.(map[string]any)["environment"].([]any)[1].(map[string]any)["value"])
	assert.Equal(t, configuration["command"], replaced.(map[string]any)["command"])
}

func TestReplaceTriggerRepository(t *testing.T) {
	nodes := []models.Node{
		{Ref: models.NodeRef{Trigger: &models.TriggerRef{Name: "github.onIssue"}}, Configuration: map[string]any{"repository": "acme/old"}},
		{Ref: models.NodeRef{Trigger: &models.TriggerRef{Name: "github.onIssue"}}, Configuration: map[string]any{"repository": "acme/custom"}},
	}

	changed := replaceTriggerRepository(nodes, "github.onIssue", "acme/old", "acme/new")

	assert.True(t, changed)
	assert.Equal(t, "acme/new", nodes[0].Configuration["repository"])
	assert.Equal(t, "acme/custom", nodes[1].Configuration["repository"])
}
