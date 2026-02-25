package canvases

import (
	"context"
	"errors"
	"io"
	"io/fs"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/genproto/googleapis/api/httpbody"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func ListCanvasArtifacts(_ context.Context, organizationID string, canvasID string) (*pb.ListArtifactsResponse, error) {
	_, parsedCanvasID, err := parseOrganizationAndCanvas(organizationID, canvasID)
	if err != nil {
		return nil, err
	}

	names, err := registry.NewLocalArtifactStorage(registry.ArtifactResourceTypeCanvas, parsedCanvasID.String()).List()
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list artifacts")
	}

	return &pb.ListArtifactsResponse{Names: names}, nil
}

func GetCanvasArtifact(_ context.Context, organizationID string, canvasID string, name string) (*httpbody.HttpBody, error) {
	_, parsedCanvasID, err := parseOrganizationAndCanvas(organizationID, canvasID)
	if err != nil {
		return nil, err
	}

	return getArtifactContent(
		registry.NewLocalArtifactStorage(registry.ArtifactResourceTypeCanvas, parsedCanvasID.String()),
		name,
	)
}

func ListNodeArtifacts(_ context.Context, organizationID string, canvasID string, nodeID string) (*pb.ListArtifactsResponse, error) {
	_, parsedCanvasID, err := parseOrganizationAndCanvas(organizationID, canvasID)
	if err != nil {
		return nil, err
	}

	if _, err = models.FindCanvasNode(database.Conn(), parsedCanvasID, nodeID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "node not found")
		}

		return nil, status.Error(codes.Internal, "failed to load node")
	}

	names, err := registry.NewLocalArtifactStorage(
		registry.ArtifactResourceTypeNode,
		registry.NodeArtifactResourceID(parsedCanvasID.String(), nodeID),
	).List()
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list artifacts")
	}

	return &pb.ListArtifactsResponse{Names: names}, nil
}

func GetNodeArtifact(_ context.Context, organizationID string, canvasID string, nodeID string, name string) (*httpbody.HttpBody, error) {
	_, parsedCanvasID, err := parseOrganizationAndCanvas(organizationID, canvasID)
	if err != nil {
		return nil, err
	}

	if _, err = models.FindCanvasNode(database.Conn(), parsedCanvasID, nodeID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "node not found")
		}

		return nil, status.Error(codes.Internal, "failed to load node")
	}

	return getArtifactContent(
		registry.NewLocalArtifactStorage(
			registry.ArtifactResourceTypeNode,
			registry.NodeArtifactResourceID(parsedCanvasID.String(), nodeID),
		),
		name,
	)
}

func ListExecutionArtifacts(_ context.Context, organizationID string, canvasID string, executionID string) (*pb.ListArtifactsResponse, error) {
	_, parsedCanvasID, err := parseOrganizationAndCanvas(organizationID, canvasID)
	if err != nil {
		return nil, err
	}

	parsedExecutionID, err := uuid.Parse(executionID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid execution_id")
	}

	if _, err = models.FindNodeExecution(parsedCanvasID, parsedExecutionID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "execution not found")
		}

		return nil, status.Error(codes.Internal, "failed to load execution")
	}

	names, err := registry.NewLocalArtifactStorage(
		registry.ArtifactResourceTypeExecution,
		parsedExecutionID.String(),
	).List()
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list artifacts")
	}

	return &pb.ListArtifactsResponse{Names: names}, nil
}

func GetExecutionArtifact(_ context.Context, organizationID string, canvasID string, executionID string, name string) (*httpbody.HttpBody, error) {
	_, parsedCanvasID, err := parseOrganizationAndCanvas(organizationID, canvasID)
	if err != nil {
		return nil, err
	}

	parsedExecutionID, err := uuid.Parse(executionID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid execution_id")
	}

	if _, err = models.FindNodeExecution(parsedCanvasID, parsedExecutionID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "execution not found")
		}

		return nil, status.Error(codes.Internal, "failed to load execution")
	}

	return getArtifactContent(
		registry.NewLocalArtifactStorage(
			registry.ArtifactResourceTypeExecution,
			parsedExecutionID.String(),
		),
		name,
	)
}

func parseOrganizationAndCanvas(organizationID string, canvasID string) (uuid.UUID, uuid.UUID, error) {
	orgID, err := uuid.Parse(organizationID)
	if err != nil {
		return uuid.UUID{}, uuid.UUID{}, status.Error(codes.InvalidArgument, "invalid organization_id")
	}

	parsedCanvasID, err := uuid.Parse(canvasID)
	if err != nil {
		return uuid.UUID{}, uuid.UUID{}, status.Error(codes.InvalidArgument, "invalid canvas_id")
	}

	if _, err = models.FindCanvas(orgID, parsedCanvasID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return uuid.UUID{}, uuid.UUID{}, status.Error(codes.NotFound, "canvas not found")
		}

		return uuid.UUID{}, uuid.UUID{}, status.Error(codes.Internal, "failed to load canvas")
	}

	return orgID, parsedCanvasID, nil
}

func getArtifactContent(storage *registry.LocalArtifactStorage, name string) (*httpbody.HttpBody, error) {
	if name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	artifact, err := storage.Get(name)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, status.Error(codes.NotFound, "artifact not found")
		}

		return nil, status.Error(codes.Internal, "failed to load artifact")
	}
	defer artifact.Close()

	data, err := io.ReadAll(artifact)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to read artifact")
	}

	return &httpbody.HttpBody{
		ContentType: "application/octet-stream",
		Data:        data,
	}, nil
}
