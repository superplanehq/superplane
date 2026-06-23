package native

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultAgentPromptDocumentsStrictConsoleYAML(t *testing.T) {
	prompt := DefaultAgentPrompt()

	assert.Contains(t, prompt, "Strict Console YAML")
	assert.Contains(t, prompt, "get_skill")
	assert.Contains(t, prompt, `skill: "console_yaml"`)
	assert.Contains(t, prompt, "kind: Console")
	assert.Contains(t, prompt, "root-level name is INVALID")
	assert.Contains(t, prompt, "metadata.name")
	assert.Contains(t, prompt, `unknown field "name"`)
	assert.Contains(t, prompt, "type: nodes")
	assert.Contains(t, prompt, "content.nodes")
	assert.Contains(t, prompt, "Each entry must have `node`")
}
