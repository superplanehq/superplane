package canvases

import (
	"strings"
	"time"

	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"gorm.io/gorm"
)

const canvasNameImmutableMessage = "canvas name cannot be changed via canvas.yaml; use UpdateCanvas instead"

func EnsureCanvasMetadataUnchanged(canvas *models.Canvas, pbCanvas *pb.Canvas) error {
	if canvas == nil || pbCanvas == nil || pbCanvas.GetMetadata() == nil {
		return grpcerrors.InvalidArgument(nil, "canvas metadata is required")
	}

	nextName := strings.TrimSpace(pbCanvas.GetMetadata().GetName())
	if nextName == "" {
		return grpcerrors.InvalidArgument(nil, "canvas name is required")
	}

	if nextName != canvas.Name {
		return grpcerrors.InvalidArgument(nil, canvasNameImmutableMessage)
	}

	return nil
}

func applyCanvasMetadataUpdates(
	canvas *models.Canvas,
	name *string,
	description *string,
) (bool, error) {
	nameChanged, err := applyCanvasNameUpdate(canvas, name)
	if err != nil {
		return false, err
	}

	descriptionChanged := applyCanvasDescriptionUpdate(canvas, description)

	return nameChanged || descriptionChanged, nil
}

func applyCanvasNameUpdate(canvas *models.Canvas, name *string) (bool, error) {
	if name == nil {
		return false, nil
	}

	nextName := strings.TrimSpace(*name)
	if nextName == "" {
		return false, grpcerrors.InvalidArgument(nil, "canvas name is required")
	}

	if canvas.Name == nextName {
		return false, nil
	}

	canvas.Name = nextName
	return true, nil
}

func applyCanvasDescriptionUpdate(canvas *models.Canvas, description *string) bool {
	if description == nil || canvas.Description == *description {
		return false
	}

	canvas.Description = *description
	return true
}

func saveCanvasMetadataUpdate(tx *gorm.DB, canvas *models.Canvas) error {
	now := time.Now()
	canvas.UpdatedAt = &now

	if err := tx.Save(canvas).Error; err != nil {
		return mapCanvasNameUniqueConstraintError(err)
	}

	return nil
}
