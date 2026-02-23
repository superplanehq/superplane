package scripts

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/scripts"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"
)

func UpdateScript(ctx context.Context, organizationID string, id string, script *pb.Script) (*pb.UpdateScriptResponse, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid script id: %v", err)
	}

	existing, err := models.FindScript(organizationID, id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "script not found")
	}

	now := time.Now()
	existing.UpdatedAt = &now

	if script.Name != "" {
		existing.Name = script.Name
	}

	if script.Label != "" {
		existing.Label = script.Label
	}

	if script.Description != "" {
		existing.Description = script.Description
	}

	if script.Source != "" {
		existing.Source = script.Source
	}

	if script.ManifestJson != "" {
		existing.Manifest = datatypes.JSON([]byte(script.ManifestJson))
	}

	if script.Status != "" {
		existing.Status = script.Status
	}

	err = database.Conn().Save(existing).Error
	if err != nil {
		return nil, err
	}

	return &pb.UpdateScriptResponse{
		Script: SerializeScript(existing),
	}, nil
}
