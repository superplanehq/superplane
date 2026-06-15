package canvases

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/test/support"
)

func commitCanvasYAMLForTest(
	ctx context.Context,
	t *testing.T,
	r *support.ResourceRegistry,
	canvasID string,
	draftVersionID string,
	yamlText string,
) {
	t.Helper()

	version, err := models.FindCanvasVersion(uuid.MustParse(canvasID), uuid.MustParse(draftVersionID))
	require.NoError(t, err)

	_, err = commitCanvasRepositoryFilesForTest(
		ctx,
		r,
		r.Organization.ID.String(),
		canvasID,
		draftVersionID,
		version.CommitSHA,
		"Update canvas.yaml",
		[]*pb.CanvasRepositoryFileOperation{
			{Path: CanvasYAMLRepositoryPath, Content: []byte(yamlText)},
		},
	)
	require.NoError(t, err)
}

func TestCommitCanvasYAMLWithFilterExpressionDollar(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	canvas, draftVersionID := createGitCanvasWithDraft(ctx, t, r, "commit-yaml-filter-dollar")

	yamlText := `apiVersion: v1
kind: Canvas
metadata:
  name: ` + canvas.Name + `
spec:
  nodes:
    - id: s
      name: Start
      type: TYPE_TRIGGER
      component: start
    - id: f
      name: Filter
      type: TYPE_ACTION
      component: filter
      configuration:
        expression: $
  edges:
    - sourceId: s
      targetId: f
`

	commitCanvasYAMLForTest(ctx, t, r, canvas.ID.String(), draftVersionID, yamlText)

	exported, err := ReadRepositorySpecFile(ctx, r.Organization.ID.String(), canvas.ID.String(), draftVersionID, CanvasYAMLRepositoryPath)
	require.NoError(t, err)
	require.Contains(t, exported, "sourceId")
	require.Contains(t, exported, "targetId")
	require.Contains(t, exported, "channel: default")

	spec := canvasSpecFromVersionYAML(ctx, t, r.Organization.ID.String(), canvas.ID.String(), draftVersionID)
	require.Len(t, spec.Edges, 1)
	require.Equal(t, "s", spec.Edges[0].SourceId)
	require.Equal(t, "f", spec.Edges[0].TargetId)
}

func TestCommitCanvasYAMLRoundtripFilterExpressionUpdate(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	canvas, draftVersionID := createGitCanvasWithDraft(ctx, t, r, "commit-yaml-roundtrip")

	baseYAML := func(expression string) string {
		return `apiVersion: v1
kind: Canvas
metadata:
  name: ` + canvas.Name + `
spec:
  nodes:
    - id: start-start-abc
      name: Start
      type: TYPE_TRIGGER
      component: start
      configuration:
        templates:
          - name: run
            payload: {}
    - id: filter-filter-xyz
      name: Filter
      type: TYPE_ACTION
      component: filter
      configuration:
        expression: ` + expression + `
  edges:
    - sourceId: start-start-abc
      targetId: filter-filter-xyz
`
	}

	commitCanvasYAMLForTest(ctx, t, r, canvas.ID.String(), draftVersionID, baseYAML("true"))
	commitCanvasYAMLForTest(ctx, t, r, canvas.ID.String(), draftVersionID, baseYAML("$"))
}

func TestCommitCanvasYAMLWithoutEdgeChannel(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	canvas, draftVersionID := createGitCanvasWithDraft(ctx, t, r, "commit-yaml-no-channel")

	yamlText := `apiVersion: v1
kind: Canvas
metadata:
  name: ` + canvas.Name + `
spec:
  nodes:
    - id: start-start-abc
      name: Start
      type: TYPE_TRIGGER
      component: start
    - id: filter-filter-xyz
      name: Filter
      type: TYPE_ACTION
      component: filter
      configuration:
        expression: $
  edges:
    - sourceId: start-start-abc
      targetId: filter-filter-xyz
`

	commitCanvasYAMLForTest(ctx, t, r, canvas.ID.String(), draftVersionID, yamlText)
}

func TestCommitCanvasYAMLWithMetadataID(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	canvas, draftVersionID := createGitCanvasWithDraft(ctx, t, r, "commit-yaml-metadata-id")

	yamlText := `apiVersion: v1
kind: Canvas
metadata:
  id: ` + canvas.ID.String() + `
  name: ` + canvas.Name + `
  description: ""
spec:
  nodes:
    - id: start-start-abc
      name: Start
      type: TYPE_TRIGGER
      component: start
      position:
        x: 500
        y: 200
    - id: filter-filter-xyz
      name: Filter
      component: filter
      position:
        x: 700
        y: 200
      configuration:
        expression: $
  edges:
    - sourceId: start-start-abc
      targetId: filter-filter-xyz
      channel: default
`

	commitCanvasYAMLForTest(ctx, t, r, canvas.ID.String(), draftVersionID, yamlText)
}
