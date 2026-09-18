package factories

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
)

// The upgrade recognises a default prompt by digest, so every default that
// ever shipped must be listed, the current one included. When you change
// analysis_user_prompt.md, add the new digest to defaultRefinePromptDigests.
func Test__DefaultRefinePromptDigestIsListed(t *testing.T) {
	assert.Truef(
		t,
		isDefaultRefinePrompt(intakeRefinementPrompt()),
		"add %q to defaultRefinePromptDigests",
		refinePromptDigest(intakeRefinementPrompt()),
	)
}

func Test__IsDefaultRefinePrompt(t *testing.T) {
	assert.True(t, isDefaultRefinePrompt(intakeRefinementPrompt()+"\n\n"), "trailing whitespace is not an edit")
	assert.False(t, isDefaultRefinePrompt("Use the team's custom scoring rules."))
	assert.False(t, isDefaultRefinePrompt(intakeRefinementPrompt()+"\nScore Confidence one step higher."))
}

func Test__RefreshBacklogRefinePrompt(t *testing.T) {
	// The Clarity-only default that shipped with the protocol split. A Backlog
	// seeded from it has no Confidence section for the user to edit.
	legacyPrompt := "## 3. Score\n\nScore Clarity from 1 through 5.\n\nTask:\n{{ root().data.workOrder }}"
	legacyDigest := refinePromptDigest(legacyPrompt)
	defaultRefinePromptDigests[legacyDigest] = struct{}{}
	t.Cleanup(func() { delete(defaultRefinePromptDigests, legacyDigest) })

	t.Run("replaces a stale default", func(t *testing.T) {
		nodes := backlogNodesWithRefinePrompt(legacyPrompt)

		assert.True(t, refreshBacklogRefinePrompt(nodes))
		assert.Equal(t, intakeRefinementPrompt(), refinePromptOf(t, nodes))
	})

	t.Run("leaves the current default alone", func(t *testing.T) {
		nodes := backlogNodesWithRefinePrompt(intakeRefinementPrompt())

		assert.False(t, refreshBacklogRefinePrompt(nodes))
	})

	t.Run("keeps a user edit", func(t *testing.T) {
		nodes := backlogNodesWithRefinePrompt("Use the team's custom scoring rules.")

		assert.False(t, refreshBacklogRefinePrompt(nodes))
		assert.Equal(t, "Use the team's custom scoring rules.", refinePromptOf(t, nodes))
	})

	t.Run("ignores a graph without a refine node", func(t *testing.T) {
		nodes := backlogNodesWithRefinePrompt(legacyPrompt)
		nodes[0].ID = intakeAnalysisNodeID

		assert.False(t, refreshBacklogRefinePrompt(nodes))
	})
}

func backlogNodesWithRefinePrompt(prompt string) []models.Node {
	return []models.Node{{
		ID: backlogRefinementNodeID,
		Configuration: map[string]any{
			"steps": []any{
				map[string]any{"name": "Clone repository", "type": "bash", "command": "git clone"},
				map[string]any{"name": backlogRefineStepName, "type": "prompt", "prompt": prompt},
			},
		},
	}}
}

func refinePromptOf(t *testing.T, nodes []models.Node) string {
	t.Helper()
	step := backlogRefineStep(nodes[0].Configuration)
	require.NotNil(t, step)
	prompt, _ := step["prompt"].(string)
	return prompt
}
