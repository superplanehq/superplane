package anthropic

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAgentPromptIncludesPatchStagingGuidance(t *testing.T) {
	prompt := DefaultAgentPrompt()

	assert.Contains(t, prompt, "Use `patch_staging` operations")
	assert.NotContains(t, prompt, "commit_files")
	assert.Contains(t, prompt, "Use `patch_staging`, not `write_file`, for `canvas.yaml` and `console.yaml`")
	assert.Contains(t, prompt, ":::staging-actions")
}
