package canvases

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	git "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/git/repositoryurl"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func canvasRepositoryMetadata(
	ctx context.Context,
	canvas *models.Canvas,
	repository *models.Repository,
	gitProvider git.Provider,
	updatedAt time.Time,
) *pb.CanvasRepository_Metadata {
	repoID := gitProvider.GetRepositoryID(git.RepositoryOptions{
		OrganizationID: canvas.OrganizationID,
		CanvasID:       canvas.ID,
		Name:           canvas.Name,
	})
	providerName := gitProvider.Name()

	if repository != nil {
		repoID = repository.RepoID
		providerName = repository.Provider
	}

	repoURL, err := gitProvider.RepositoryURL(ctx, repoID, canvas.ID.String())
	if err != nil {
		logrus.WithError(err).Warnf("failed to build repository URL for canvas %s", canvas.ID)
		repoURL = ""
	}

	return &pb.CanvasRepository_Metadata{
		CanvasId:      canvas.ID.String(),
		RepoId:        repoID,
		Provider:      providerName,
		Url:           repoURL,
		DefaultBranch: repositoryurl.DefaultBranch(),
		UpdatedAt:     timestamppb.New(updatedAt),
	}
}

func canvasRepositoryMetadataForCanvas(ctx context.Context, canvas *models.Canvas, gitProvider git.Provider) *pb.CanvasRepository_Metadata {
	return canvasRepositoryMetadata(ctx, canvas, nil, gitProvider, time.Now())
}

func canvasRepositoryMetadataFromRepository(ctx context.Context, canvas *models.Canvas, repository *models.Repository, gitProvider git.Provider) *pb.CanvasRepository_Metadata {
	return canvasRepositoryMetadata(ctx, canvas, repository, gitProvider, repository.UpdatedAt)
}
