package anthropic

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultAgentPromptWarnsAgainstTemplateFieldsInCanvasYAML(t *testing.T) {
	prompt := DefaultAgentPrompt()

	assert.Contains(t, prompt, "canonical live Canvas YAML")
	assert.Contains(t, prompt, "metadata.isTemplate")
	assert.Contains(t, prompt, "unknown field")
}
