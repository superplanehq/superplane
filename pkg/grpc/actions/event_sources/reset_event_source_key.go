package eventsources

import (
	"context"
	"errors"

	uuid "github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func ResetEventSourceKey(ctx context.Context, encryptor crypto.Encryptor, req *pb.ResetEventSourceKeyRequest) (*pb.ResetEventSourceKeyResponse, error) {
	domainId := ctx.Value(authorization.DomainIdContextKey).(string)
	canvas, err := models.FindCanvasByID(domainId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "canvas not found")
	}

	logger := logging.ForCanvas(canvas)
	err = actions.ValidateUUIDs(req.IdOrName)
	var source *models.EventSource
	if err != nil {
		source, err = canvas.FindEventSourceByName(req.IdOrName)
	} else {
		source, err = canvas.FindEventSourceByID(uuid.MustParse(req.IdOrName))
	}

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "event source not found")
		}

		logger.Errorf("Error describing event source. Request: %v. Error: %v", req, err)
		return nil, err
	}

	plainKey, encryptedKey, err := crypto.NewRandomKey(ctx, encryptor, source.Name)
	if err != nil {
		logger.Errorf("Error generating event source key. Request: %v. Error: %v", req, err)
		return nil, status.Error(codes.Internal, "error generating key")
	}

	err = source.UpdateKey(encryptedKey)
	if err != nil {
		logger.Errorf("Error updating event source key. Request: %v. Error: %v", req, err)
		return nil, status.Error(codes.Internal, "error updating key")
	}

	protoSource, err := serializeEventSource(*source)
	if err != nil {
		return nil, err
	}

	response := &pb.ResetEventSourceKeyResponse{
		EventSource: protoSource,
		Key:         string(plainKey),
	}

	return response, nil
}
