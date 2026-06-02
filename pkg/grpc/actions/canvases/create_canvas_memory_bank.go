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

func CreateCanvasMemoryBank(ctx context.Context, registry *registry.Registry, organizationID, canvasID, namespace string, entries []*structpb.Value) (*pb.CreateCanvasMemoryBankResponse, error) {
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

		if existingSource != "" {
			return status.Errorf(codes.InvalidArgument, "memory bank %q already exists", namespace)
		}

		if txErr := models.ReplaceManualCanvasMemoryBankInTransaction(tx, canvasUUID, namespace, decodedEntries); txErr != nil {
			return txErr
		}

		records, txErr := models.ListCanvasMemoriesByNamespaceInTransaction(tx, canvasUUID, namespace)
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
		return nil, status.Error(codes.Internal, "failed to create canvas memory bank")
	}

	items := make([]*pb.CanvasMemory, 0, len(stored))
	for _, record := range stored {
		item, err := canvasMemoryToProto(record)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to serialize canvas memory")
		}
		items = append(items, item)
	}

	return &pb.CreateCanvasMemoryBankResponse{
		Namespace: namespace,
		Items:     items,
	}, nil
}
