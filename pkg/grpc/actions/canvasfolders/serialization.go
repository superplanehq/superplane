package canvasfolders

import (
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvas_folders"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func SerializeCanvasFolders(folders []models.CanvasFolder) []*pb.CanvasFolder {
	protoFolders := make([]*pb.CanvasFolder, len(folders))
	for i, folder := range folders {
		protoFolders[i] = SerializeCanvasFolder(&folder)
	}

	return protoFolders
}

func SerializeCanvasFolder(folder *models.CanvasFolder) *pb.CanvasFolder {
	return &pb.CanvasFolder{
		Metadata: &pb.CanvasFolder_Metadata{
			Id:             folder.ID.String(),
			OrganizationId: folder.OrganizationID.String(),
			CreatedAt:      timestamppb.New(*folder.CreatedAt),
			UpdatedAt:      timestamppb.New(*folder.UpdatedAt),
		},
		Spec: &pb.CanvasFolder_Spec{
			Title:           folder.Title,
			BackgroundColor: folder.BackgroundColor,
			Canvases:        serializeCanvasRefs(folder.Canvases),
		},
	}
}

func serializeCanvasRefs(canvases []models.Canvas) []*pb.CanvasRef {
	refs := make([]*pb.CanvasRef, 0, len(canvases))

	for _, canvas := range canvases {
		refs = append(refs, &pb.CanvasRef{
			Id: canvas.ID.String(),
		})
	}

	return refs
}
