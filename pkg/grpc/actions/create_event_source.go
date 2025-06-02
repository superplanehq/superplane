package actions

import (
	"context"
	"errors"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/encryptor"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/superplane"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func CreateEventSource(ctx context.Context, encryptor encryptor.Encryptor, req *pb.CreateEventSourceRequest) (*pb.CreateEventSourceResponse, error) {
	err := ValidateUUIDs(req.CanvasIdOrName)
	var canvas *models.Canvas
	if err != nil {
		canvas, err = models.FindCanvasByName(req.CanvasIdOrName)
	} else {
		canvas, err = models.FindCanvasByID(req.CanvasIdOrName)
	}

	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "canvas not found")
	}

	logger := logging.ForCanvas(canvas)
	plainKey, encryptedKey, err := genNewEventSourceKey(ctx, encryptor, req.Name)
	if err != nil {
		logger.Errorf("Error generating event source key. Request: %v. Error: %v", req, err)
		return nil, status.Error(codes.Internal, "error generating key")
	}

	// TODO: Store key in secrethub secret and create a webhook notification
	// using Notifications API for semaphore event sources. This webhook should point
	// to the created secret, as designed in the API.

	eventSource, err := canvas.CreateEventSource(req.Name, encryptedKey)
	if err != nil {
		if errors.Is(err, models.ErrNameAlreadyUsed) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		log.Errorf("Error creating event source. Request: %v. Error: %v", req, err)
		return nil, err
	}

	response := &pb.CreateEventSourceResponse{
		EventSource: serializeEventSource(*eventSource),
		Key:         string(plainKey),
	}

	logger.Infof("Created event source. Request: %v", req)

	err = messages.NewEventSourceCreatedMessage(eventSource).Publish()

	if err != nil {
		logger.Errorf("failed to publish event source created message: %v", err)
	}

	return response, nil
}

func serializeEventSource(eventSource models.EventSource) *pb.EventSource {
	return &pb.EventSource{
		Id:        eventSource.ID.String(),
		Name:      eventSource.Name,
		CanvasId:  eventSource.CanvasID.String(),
		CreatedAt: timestamppb.New(*eventSource.CreatedAt),
	}
}

func genNewEventSourceKey(ctx context.Context, encryptor encryptor.Encryptor, name string) (string, []byte, error) {
	plainKey, _ := crypto.Base64String(32)
	encrypted, err := encryptor.Encrypt(ctx, []byte(plainKey), []byte(name))
	if err != nil {
		return "", nil, err
	}

	return plainKey, encrypted, nil
}
