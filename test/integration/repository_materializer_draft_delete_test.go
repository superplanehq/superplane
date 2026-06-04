package integration_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/renderedtext/go-tackle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	componentpb "github.com/superplanehq/superplane/pkg/protos/components"
	"github.com/superplanehq/superplane/pkg/workers"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestRepositoryMaterializerWorkerReconcilesDeletedDraftBranch(t *testing.T) {
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
			Metadata: &pb.Canvas_Metadata{Name: "worker-draft-delete-reconcile"},
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

	branchResp, err := canvases.CreateDraftBranch(
		ctx,
		r.GitProvider,
		r.Registry,
		r.Organization.ID.String(),
		canvasID,
		"Worker Delete Draft",
	)
	require.NoError(t, err)
	branchName := branchResp.GetBranch().GetBranchName()

	repository, err := models.FindRepository(r.Organization.ID, canvasUUID)
	require.NoError(t, err)

	_, err = models.FindDraftBranch(canvasUUID, branchName)
	require.NoError(t, err)

	require.NoError(t, r.GitProvider.DeleteBranch(context.Background(), repository.RepoID, branchName))

	mainHead, err := r.GitProvider.Head(context.Background(), repository.RepoID, models.CanvasGitBranchMain)
	require.NoError(t, err)

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
		HeadSha:               mainHead,
		MaterializationStatus: models.MaterializationStatusPending,
		Timestamp:             timestamppb.Now(),
	}
	body, err := proto.Marshal(message)
	require.NoError(t, err)

	require.NoError(t, worker.ConsumeRepositoryBranchUpdated(tackle.NewFakeDelivery(body)))

	_, err = models.FindDraftBranch(canvasUUID, branchName)
	require.Error(t, err)

	listResp, err := canvases.ListDraftBranches(
		ctx,
		r.Organization.ID.String(),
		canvasID,
	)
	require.NoError(t, err)
	assert.Empty(t, listResp.GetBranches())
}
