package canvases

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/test/support"
)

func TestCommitCanvasYAMLWithFilterExpressionDollar(t *testing.T) {
	r := support.Setup(t)
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

	draftVersion, err := models.CreateDraftBranchFromLiveInTransaction(database.Conn(), canvas.ID, r.User, "", nil, nil)
	require.NoError(t, err)

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

	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	err = ApplyRepositorySpecFileOperations(
		ctx,
		nil,
		r.Encryptor,
		r.Registry,
		r.Organization.ID.String(),
		canvas.ID.String(),
		draftVersion.ID.String(),
		"",
		r.AuthService,
		nil,
		false,
		[]*pb.CanvasRepositoryFileOperation{
			{Path: CanvasYAMLRepositoryPath, Content: []byte(yamlText)},
		},
	)
	require.NoError(t, err)

	exported, err := ReadRepositorySpecFile(ctx, r.Organization.ID.String(), canvas.ID.String(), draftVersion.ID.String(), CanvasYAMLRepositoryPath)
	require.NoError(t, err)
	require.Contains(t, exported, "sourceId")
	require.Contains(t, exported, "targetId")
	require.Contains(t, exported, "channel: default")

	spec := canvasSpecFromVersionYAML(ctx, t, r.Organization.ID.String(), canvas.ID.String(), draftVersion.ID.String())
	require.Len(t, spec.Edges, 1)
	require.Equal(t, "s", spec.Edges[0].SourceId)
	require.Equal(t, "f", spec.Edges[0].TargetId)
}

func TestCommitCanvasYAMLRoundtripFilterExpressionUpdate(t *testing.T) {
	r := support.Setup(t)
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

	draftVersion, err := models.CreateDraftBranchFromLiveInTransaction(database.Conn(), canvas.ID, r.User, "", nil, nil)
	require.NoError(t, err)

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

	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	err = ApplyRepositorySpecFileOperations(
		ctx,
		nil,
		r.Encryptor,
		r.Registry,
		r.Organization.ID.String(),
		canvas.ID.String(),
		draftVersion.ID.String(),
		"",
		r.AuthService,
		nil,
		false,
		[]*pb.CanvasRepositoryFileOperation{
			{Path: CanvasYAMLRepositoryPath, Content: []byte(baseYAML("true"))},
		},
	)
	require.NoError(t, err)

	err = ApplyRepositorySpecFileOperations(
		ctx,
		nil,
		r.Encryptor,
		r.Registry,
		r.Organization.ID.String(),
		canvas.ID.String(),
		draftVersion.ID.String(),
		"",
		r.AuthService,
		nil,
		false,
		[]*pb.CanvasRepositoryFileOperation{
			{Path: CanvasYAMLRepositoryPath, Content: []byte(baseYAML("$"))},
		},
	)
	require.NoError(t, err)
}

func TestCommitCanvasYAMLWithoutEdgeChannel(t *testing.T) {
	r := support.Setup(t)
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

	draftVersion, err := models.CreateDraftBranchFromLiveInTransaction(database.Conn(), canvas.ID, r.User, "", nil, nil)
	require.NoError(t, err)

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

	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	err = ApplyRepositorySpecFileOperations(
		ctx,
		nil,
		r.Encryptor,
		r.Registry,
		r.Organization.ID.String(),
		canvas.ID.String(),
		draftVersion.ID.String(),
		"",
		r.AuthService,
		nil,
		false,
		[]*pb.CanvasRepositoryFileOperation{
			{Path: CanvasYAMLRepositoryPath, Content: []byte(yamlText)},
		},
	)
	require.NoError(t, err)
}

func TestCommitCanvasYAMLWithMetadataID(t *testing.T) {
	r := support.Setup(t)
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

	draftVersion, err := models.CreateDraftBranchFromLiveInTransaction(database.Conn(), canvas.ID, r.User, "", nil, nil)
	require.NoError(t, err)

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

	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	err = ApplyRepositorySpecFileOperations(
		ctx,
		nil,
		r.Encryptor,
		r.Registry,
		r.Organization.ID.String(),
		canvas.ID.String(),
		draftVersion.ID.String(),
		"",
		r.AuthService,
		nil,
		false,
		[]*pb.CanvasRepositoryFileOperation{
			{Path: CanvasYAMLRepositoryPath, Content: []byte(yamlText)},
		},
	)
	require.NoError(t, err)
}
