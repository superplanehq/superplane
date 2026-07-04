package canvases

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	git "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

func GetCanvasRepository(ctx context.Context, gitProvider git.Provider, organizationID string, id string) (*pb.GetCanvasRepositoryResponse, error) {
	canvasID, err := uuid.Parse(id)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid canvas id")
	}

	canvas, err := models.FindCanvas(uuid.MustParse(organizationID), canvasID)
	if err != nil {
		return nil, grpcerrors.NotFound(err, "canvas not found")
	}

	orgID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid organization id")
	}

	repository, err := models.FindRepository(orgID, canvasID)
	if err != nil {
		return handleMissingRepository(gitProvider, canvas, err)
	}

	//
	// We only have a head SHA to look up when the repository has been
	// successfully provisioned. For pending/error states the repo does not
	// yet exist on the git provider side, so calling Head() would fail.
	//
	var headSha string
	if repository.Status == models.RepositoryStatusReady {
		headSha, err = gitProvider.Head(ctx, repository.RepoID, "")
		if err != nil {
			return nil, grpcerrors.Internal(err, "failed to get repository head sha")
		}
	}

	return &pb.GetCanvasRepositoryResponse{
		Repository: &pb.CanvasRepository{
			Metadata: &pb.CanvasRepository_Metadata{
				CanvasId:  canvas.ID.String(),
				UpdatedAt: timestamppb.New(repository.UpdatedAt),
			},
			Status: &pb.CanvasRepository_Status{
				State:   repositoryStateToProto(repository.Status),
				HeadSha: headSha,
			},
		},
	}, nil
}

func handleMissingRepository(gitProvider git.Provider, canvas *models.Canvas, err error) (*pb.GetCanvasRepositoryResponse, error) {
	//
	// If this is not a NotFound error, we return the error as is.
	//
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		logrus.Errorf("failed to find repository for canvas %s: %v", canvas.ID, err)
		return nil, grpcerrors.Internal(err, "failed to find repository for canvas")
	}

	//
	// If canvas exists, but repository does not, we create a pending repository,
	// and let the repository provisioner worker handle the rest.
	// This is a trick to provision repositories for existing canvases lazily.
	//
	_, err = canvas.CreatePendingRepository(gitProvider.Name(), gitProvider.GetRepositoryID(git.RepositoryOptions{
		OrganizationID: canvas.OrganizationID,
		CanvasID:       canvas.ID,
	}))

	//
	// If we fail to create it, we return NotFound still, and just log the error.
	//
	if err != nil {
		logrus.Errorf("failed to create pending repository for canvas %s: %v", canvas.ID, err)
		return nil, grpcerrors.NotFound(err, "repository not found")
	}

	return &pb.GetCanvasRepositoryResponse{
		Repository: &pb.CanvasRepository{
			Metadata: &pb.CanvasRepository_Metadata{
				CanvasId:  canvas.ID.String(),
				UpdatedAt: timestamppb.New(time.Now()),
			},
			Status: &pb.CanvasRepository_Status{
				State: pb.CanvasRepository_STATE_PENDING,
			},
		},
	}, nil
}

func repositoryStateToProto(state string) pb.CanvasRepository_State {
	switch state {
	case models.RepositoryStatusPending:
		return pb.CanvasRepository_STATE_PENDING
	case models.RepositoryStatusReady:
		return pb.CanvasRepository_STATE_READY
	case models.RepositoryStatusError:
		return pb.CanvasRepository_STATE_ERROR
	}

	return pb.CanvasRepository_STATE_UNSPECIFIED
}
