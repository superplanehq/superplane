package factories

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/yaml"
)

func TestReviewRequestMakesContributionHistoryAvailable(t *testing.T) {
	remote := bareRepository(t)
	commitRemoteBranch(t, remote, "main")
	setRemoteHead(t, remote, "main")
	work := t.TempDir()
	runGit(t, work, "clone", fileURL(remote), ".")
	runGit(t, work, "commit", "--allow-empty", "-m", "Recent change")
	runGit(t, work, "push", "origin", "main")

	checkout := runCloneScript(t, "checkout", fileURL(remote), "main", false)
	require.NoError(t, checkout.err, checkout.stderr)
	require.Equal(t, "1", revListCount(t, filepath.Join(checkout.dir, "repo"), "HEAD"), "the ordinary checkout hides earlier contributors")

	raw, err := factoryTemplateFiles.ReadFile("templates/github/review-request.canvas.yaml")
	require.NoError(t, err)
	doc, err := yaml.CanvasFromYAML(raw)
	require.NoError(t, err)
	steps := findYAMLNode(t, doc, reviewRequestAgentNodeID).Configuration["steps"].([]any)
	prepare := steps[0].(map[string]any)["command"].(string)
	// A retry must also work when the checkout already contains the history.
	result := runCloneScript(t, "checkout\n"+prepare+"\n"+prepare, fileURL(remote), "main", false)
	require.NoError(t, result.err, result.stderr)
	require.Equal(t, "2", revListCount(t, filepath.Join(result.dir, "repo"), "HEAD"))
}
