package eventsources

import (
	"context"
	"errors"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/crypto"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func ResetEventSourceKey(ctx context.Context, encryptor crypto.Encryptor, canvasID string, idOrName string) (*pb.ResetEventSourceKeyResponse, error) {
	source, err := findEventSource(canvasID, idOrName)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "event source not found")
		}

		log.Errorf("Error describing event source %s in canvas %s: %v", idOrName, canvasID, err)
		return nil, err
	}

	plainKey, encryptedKey, err := crypto.NewRandomKey(ctx, encryptor, source.ID.String())
	if err != nil {
		log.Errorf("Error generating new key for event source %s in canvas %s: %v", idOrName, canvasID, err)
		return nil, status.Error(codes.Internal, "error generating key")
	}

	err = source.UpdateKey(encryptedKey)
	if err != nil {
		log.Errorf("Error updating key for event source %s in canvas %s: %v", idOrName, canvasID, err)
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
