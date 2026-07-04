package anthropic

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultAgentPromptUsesPatchDraftForCanvasUpdates(t *testing.T) {
	prompt := DefaultAgentPrompt()

	assert.Contains(t, prompt, "Use `patch_draft` operations")
	assert.Contains(t, prompt, "instead of sending full Canvas YAML")
}

func TestDefaultAgentPromptGuidesRepositoryFileContextDiscovery(t *testing.T) {
	prompt := DefaultAgentPrompt()

	assert.Contains(t, prompt, "list_files")
	assert.Contains(t, prompt, "AGENTS.md")
	assert.Contains(t, prompt, "read_file")
	assert.Contains(t, prompt, "write_file")
	assert.Contains(t, prompt, "commit_files")
	assert.Contains(t, prompt, "Use `patch_draft`, not `write_file`, for `canvas.yaml` and `console.yaml`")
}
