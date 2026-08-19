package canvases

import (
	"context"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/protobuf/types/known/structpb"
	"gorm.io/gorm"
)

func UpdateCanvasMemoryNamespace(ctx context.Context, db *gorm.DB, canvas *models.Canvas, namespace, newNamespace string, entries []*structpb.Value) (*pb.UpdateCanvasMemoryNamespaceResponse, error) {
	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		return nil, grpcerrors.InvalidArgument(nil, "namespace is required")
	}

	newNamespace = strings.TrimSpace(newNamespace)
	targetNamespace := newNamespace
	if targetNamespace == "" {
		targetNamespace = namespace
	}

	if len(entries) == 0 {
		return nil, grpcerrors.InvalidArgument(nil, "at least one entry is required")
	}

	decodedEntries := make([]any, 0, len(entries))
	for _, value := range entries {
		decodedEntries = append(decodedEntries, value.AsInterface())
	}

	var stored []models.CanvasMemory
	err := db.Transaction(func(tx *gorm.DB) error {
		existingSource, txErr := models.CanvasMemoryNamespaceSourceInTransaction(tx, canvas.ID, namespace)
		if txErr != nil {
			return txErr
		}

		if existingSource == "" {
			return grpcerrors.NotFound(nil, fmt.Sprintf("memory namespace %q not found", namespace))
		}

		if existingSource != models.CanvasMemorySourceManual {
			return grpcerrors.FailedPrecondition(nil, fmt.Sprintf("memory namespace %q is managed by nodes and cannot be edited", namespace))
		}

		if targetNamespace != namespace {
			conflict, txErr := models.CanvasMemoryNamespaceSourceInTransaction(tx, canvas.ID, targetNamespace)
			if txErr != nil {
				return txErr
			}

			if conflict != "" {
				return grpcerrors.InvalidArgument(nil, fmt.Sprintf("memory namespace %q already exists", targetNamespace))
			}

			if txErr := tx.
				Where("canvas_id = ? AND namespace = ? AND source = ?", canvas.ID, namespace, models.CanvasMemorySourceManual).
				Delete(&models.CanvasMemory{}).
				Error; txErr != nil {
				return txErr
			}
		}

		if txErr := models.ReplaceManualCanvasMemoryNamespaceInTransaction(tx, canvas.ID, targetNamespace, decodedEntries); txErr != nil {
			return txErr
		}

		records, txErr := models.ListCanvasMemoriesByNamespaceInTransaction(tx, canvas.ID, targetNamespace)
		if txErr != nil {
			return txErr
		}

		stored = records
		return nil
	})

	if err != nil {
		if _, _, ok := grpcerrors.HandlerStatus(err); ok {
			return nil, err
		}
		return nil, grpcerrors.Internal(err, "failed to update canvas memory namespace")
	}

	if err := messages.NewCanvasMemoryUpdatedMessage(canvas.ID.String()).PublishMemoryUpdated(); err != nil {
		log.Errorf("failed to publish canvas memory updated RabbitMQ message: %v", err)
	}

	items := make([]*pb.CanvasMemory, 0, len(stored))
	for _, record := range stored {
		item, err := canvasMemoryToProto(record)
		if err != nil {
			return nil, grpcerrors.Internal(err, "failed to serialize canvas memory")
		}
		items = append(items, item)
	}

	return &pb.UpdateCanvasMemoryNamespaceResponse{
		Namespace: targetNamespace,
		Items:     items,
	}, nil
}
