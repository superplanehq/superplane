package scripts

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/ai"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/scripts"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func GenerateScript(ctx context.Context, aiClient *ai.Client, organizationID string, scriptID string, message string) (*pb.GenerateScriptResponse, error) {
	if aiClient == nil {
		return nil, status.Error(codes.FailedPrecondition, "AI generation is not configured (OPENAI_API_KEY not set)")
	}

	if message == "" {
		return nil, status.Error(codes.InvalidArgument, "message is required")
	}

	if _, err := uuid.Parse(scriptID); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid script id: %v", err)
	}

	script, err := models.FindScript(organizationID, scriptID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "script not found")
	}

	response, source, err := aiClient.GenerateScript(ctx, message, script.Source)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "AI generation failed: %v", err)
	}

	return &pb.GenerateScriptResponse{
		Response: response,
		Source:   source,
	}, nil
}
