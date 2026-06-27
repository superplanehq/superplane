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

func TestDefaultAgentPromptGuidesRepositoryFileContextDiscovery(t *testing.T) {
	prompt := DefaultAgentPrompt()

	assert.Contains(t, prompt, "list_files")
	assert.Contains(t, prompt, "AGENTS.md")
	assert.Contains(t, prompt, "read_file")
	assert.Contains(t, prompt, "write_file")
	assert.Contains(t, prompt, "commit_files")
	assert.Contains(t, prompt, "Use `patch_draft` or `update_draft`, not `write_file`, for `canvas.yaml`")
	assert.Contains(t, prompt, "use `update_draft`, not `write_file`, for `console.yaml`")
}
