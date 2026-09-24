package runner

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/configuration"
)

func TestAgentModelFieldUsesOrganizationPicker(t *testing.T) {
	field := AgentModelField("anthropic")
	assert.Equal(t, "model", field.Name)
	assert.Equal(t, configuration.FieldTypeHostedModel, field.Type)
	assert.Equal(t, "Select a model from Organization LLM Models.", field.Description)
	assert.Equal(t, "Select a model", field.Placeholder)
	assert.Equal(t, "anthropic", field.TypeOptions.HostedModel.Provider)
	assert.False(t, field.Required)
}
