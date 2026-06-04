package integration_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/renderedtext/go-tackle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/canvas/materialize"
	"github.com/superplanehq/superplane/pkg/database"
	git "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	componentpb "github.com/superplanehq/superplane/pkg/protos/components"
	"github.com/superplanehq/superplane/pkg/workers"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const repositoryMaterializerWebhookBaseURL = "http://localhost:3000/api/v1"

func TestRepositoryMaterializerWorkerMainBranch(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	createCanvasResponse, err := canvases.CreateCanvas(
		ctx,
		r.Registry,
		r.Encryptor,
		r.AuthService,
		r.GitProvider,
		repositoryMaterializerWebhookBaseURL,
		r.Organization.ID,
		&pb.Canvas{
			Metadata: &pb.Canvas_Metadata{Name: "worker-main-materialize"},
			Spec: &pb.Canvas_Spec{
				Nodes: []*componentpb.Node{
					{Id: "node-1", Name: "Initial Name", Component: "noop"},
				},
			},
		},
		nil,
		nil,
	)
	require.NoError(t, err)

	canvasID := createCanvasResponse.Canvas.Metadata.Id
	canvasUUID := uuid.MustParse(canvasID)

	canvas, err := models.FindCanvas(r.Organization.ID, canvasUUID)
	require.NoError(t, err)
	require.NotNil(t, canvas.LiveVersionID)
	initialSHA := *canvas.LiveVersionID

	branchResp, err := canvases.CreateDraftBranch(
		ctx,
		r.GitProvider,
		r.Registry,
		r.Organization.ID.String(),
		canvasID,
		"Worker Draft",
	)
	require.NoError(t, err)
	branchName := branchResp.GetBranch().GetBranchName()

	liveVersion, err := models.FindLiveCanvasVersionInTransaction(database.Conn(), canvasUUID)
	require.NoError(t, err)

	nodes := append([]models.Node(nil), liveVersion.Nodes...)
	for i := range nodes {
		if nodes[i].ID == "node-1" {
			nodes[i].Name = "Worker Published Node"
		}
	}

	canvasYAML, err := materialize.BuildCanvasYAML(
		liveVersion.Name,
		liveVersion.Description,
		nodes,
		liveVersion.Edges,
		liveVersion.ChangeManagementEnabled,
		liveVersion.EffectiveChangeRequestApprovers(),
	)
	require.NoError(t, err)

	consoleYAML, err := materialize.BuildConsoleYAMLFromVersion(liveVersion)
	require.NoError(t, err)

	_, err = canvases.CommitCanvasRepositoryFiles(
		ctx,
		r.GitProvider,
		r.Registry,
		r.Organization.ID.String(),
		canvasID,
		branchName,
		branchResp.GetBranch().GetTipSha(),
		"Update draft",
		[]*pb.CanvasRepositoryFileOperation{
			{Path: materialize.CanvasFileName, Content: canvasYAML},
			{Path: materialize.ConsoleFileName, Content: consoleYAML},
		},
	)
	require.NoError(t, err)

	repository, err := models.FindRepository(r.Organization.ID, canvasUUID)
	require.NoError(t, err)

	mergeSHA, err := r.GitProvider.MergeBranch(
		context.Background(),
		repository.RepoID,
		branchName,
		models.CanvasGitBranchMain,
		"Publish via git",
		git.CommitAuthor{Name: "Test", Email: "test@example.com"},
	)
	require.NoError(t, err)
	require.NotEqual(t, initialSHA, mergeSHA)

	require.NoError(t, database.Conn().Model(&models.Canvas{}).
		Where("id = ?", canvasUUID).
		Update("live_version_id", initialSHA).Error)

	worker := workers.NewRepositoryMaterializerWorker(
		"",
		r.GitProvider,
		r.Registry,
		r.Encryptor,
		r.AuthService,
		repositoryMaterializerWebhookBaseURL,
	)

	message := &pb.RepositoryBranchUpdatedMessage{
		CanvasId:              canvasID,
		Branch:                models.CanvasGitBranchMain,
		HeadSha:               mergeSHA,
		MaterializationStatus: models.MaterializationStatusPending,
		Timestamp:             timestamppb.Now(),
	}
	body, err := proto.Marshal(message)
	require.NoError(t, err)

	require.NoError(t, worker.ConsumeRepositoryBranchUpdated(tackle.NewFakeDelivery(body)))

	updatedCanvas, err := models.FindCanvasWithoutOrgScope(canvasUUID)
	require.NoError(t, err)
	require.NotNil(t, updatedCanvas.LiveVersionID)
	assert.Equal(t, mergeSHA, *updatedCanvas.LiveVersionID)
}
