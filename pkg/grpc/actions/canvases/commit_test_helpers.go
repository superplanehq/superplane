package canvases

import (
	"context"

	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/test/support"
)

func commitCanvasRepositoryFilesForTest(
	ctx context.Context,
	r *support.ResourceRegistry,
	organizationID string,
	canvasID string,
	versionID string,
	expectedHeadSha string,
	message string,
	operations []*pb.CanvasRepositoryFileOperation,
) (*pb.CommitCanvasRepositoryFilesResponse, error) {
	return CommitCanvasRepositoryFiles(
		ctx,
		r.GitProvider,
		nil,
		r.Encryptor,
		r.Registry,
		organizationID,
		canvasID,
		versionID,
		expectedHeadSha,
		message,
		operations,
		nil,
		"",
		r.AuthService,
	)
}
