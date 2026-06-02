package canvases

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"gorm.io/gorm"
)

func UpdateCanvasMemoryBank(ctx context.Context, registry *registry.Registry, organizationID, canvasID, namespace, newNamespace string, entries []*structpb.Value) (*pb.UpdateCanvasMemoryBankResponse, error) {
	orgUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid organization_id")
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid canvas_id")
	}

	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		return nil, status.Error(codes.InvalidArgument, "namespace is required")
	}

	newNamespace = strings.TrimSpace(newNamespace)
	targetNamespace := newNamespace
	if targetNamespace == "" {
		targetNamespace = namespace
	}

	if len(entries) == 0 {
		return nil, status.Error(codes.InvalidArgument, "at least one entry is required")
	}

	_, err = models.FindCanvas(orgUUID, canvasUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "canvas not found")
		}
		return nil, status.Error(codes.Internal, "failed to load canvas")
	}

	decodedEntries := make([]any, 0, len(entries))
	for _, value := range entries {
		decodedEntries = append(decodedEntries, value.AsInterface())
	}

	var stored []models.CanvasMemory
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		existingSource, txErr := models.CanvasMemoryNamespaceSourceInTransaction(tx, canvasUUID, namespace)
		if txErr != nil {
			return txErr
		}

		if existingSource == "" {
			return status.Errorf(codes.NotFound, "memory bank %q not found", namespace)
		}

		if existingSource != models.CanvasMemorySourceManual {
			return status.Errorf(codes.FailedPrecondition, "memory bank %q is managed by nodes and cannot be edited", namespace)
		}

		if targetNamespace != namespace {
			conflict, txErr := models.CanvasMemoryNamespaceSourceInTransaction(tx, canvasUUID, targetNamespace)
			if txErr != nil {
				return txErr
			}

			if conflict != "" {
				return status.Errorf(codes.InvalidArgument, "memory bank %q already exists", targetNamespace)
			}

			if txErr := tx.
				Where("canvas_id = ? AND namespace = ? AND source = ?", canvasUUID, namespace, models.CanvasMemorySourceManual).
				Delete(&models.CanvasMemory{}).
				Error; txErr != nil {
				return txErr
			}
		}

		if txErr := models.ReplaceManualCanvasMemoryBankInTransaction(tx, canvasUUID, targetNamespace, decodedEntries); txErr != nil {
			return txErr
		}

		records, txErr := models.ListCanvasMemoriesByNamespaceInTransaction(tx, canvasUUID, targetNamespace)
		if txErr != nil {
			return txErr
		}

		stored = records
		return nil
	})

	if err != nil {
		if statusErr, ok := status.FromError(err); ok {
			return nil, statusErr.Err()
		}
		return nil, status.Error(codes.Internal, "failed to update canvas memory bank")
	}

	items := make([]*pb.CanvasMemory, 0, len(stored))
	for _, record := range stored {
		item, err := canvasMemoryToProto(record)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to serialize canvas memory")
		}
		items = append(items, item)
	}

	return &pb.UpdateCanvasMemoryBankResponse{
		Namespace: targetNamespace,
		Items:     items,
	}, nil
}
