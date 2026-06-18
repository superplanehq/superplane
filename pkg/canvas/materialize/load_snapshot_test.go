package materialize_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/canvas/gitref"
	"github.com/superplanehq/superplane/pkg/canvas/materialize"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/git/inmemory"
	"github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
)

func TestLoadRepoSnapshotFromGitCommit(t *testing.T) {
	ctx := context.Background()
	gitProvider := inmemory.NewProvider()
	reg, err := registry.NewRegistry(&crypto.NoOpEncryptor{}, registry.HTTPOptions{})
	require.NoError(t, err)

	orgID := uuid.New()
	canvasID := uuid.New()
	repoID := gitProvider.GetRepositoryID(provider.RepositoryOptions{
		OrganizationID: orgID,
		CanvasID:       canvasID,
	})

	_, err = gitProvider.CreateRepository(ctx, repoID)
	require.NoError(t, err)

	canvasYAML, err := materialize.BuildCanvasYAMLFromCanvas(&materialize.CanvasYAML{
		APIVersion: "v1",
		Kind:       "Canvas",
		Metadata: materialize.CanvasYAMLMetadata{
			Name:        "Health Monitor",
			Description: "watches services",
		},
		Spec: materialize.CanvasYAMLSpec{
			Nodes: []models.Node{
				{
					ID:   "node-1",
					Name: "Check",
					Type: models.NodeTypeComponent,
					Ref: models.NodeRef{
						Component: &models.ComponentRef{Name: "noop"},
					},
				},
			},
		},
	})
	require.NoError(t, err)

	consoleYAML, err := materialize.BuildEmptyConsoleYAML(canvasID.String(), "Health Monitor")
	require.NoError(t, err)

	commitSHA, err := gitProvider.Commit(ctx, repoID, provider.CommitOptions{
		Branch:     models.CanvasGitBranchMain,
		BaseBranch: models.CanvasGitBranchMain,
		Message:    "seed",
		Operations: []provider.FileOperation{
			{Path: gitref.CanvasFileName, Content: bytes.NewReader(canvasYAML), SizeBytes: int64(len(canvasYAML))},
			{Path: gitref.ConsoleFileName, Content: bytes.NewReader(consoleYAML), SizeBytes: int64(len(consoleYAML))},
		},
	})
	require.NoError(t, err)
	require.NotEmpty(t, commitSHA)

	snapshot, err := materialize.LoadRepoSnapshot(ctx, gitProvider, reg, orgID, repoID, commitSHA)
	require.NoError(t, err)
	require.NotNil(t, snapshot)

	assert.Equal(t, "Health Monitor", snapshot.Name)
	assert.Equal(t, "watches services", snapshot.Description)
	require.Len(t, snapshot.Nodes, 1)
	assert.Equal(t, "node-1", snapshot.Nodes[0].ID)
}
