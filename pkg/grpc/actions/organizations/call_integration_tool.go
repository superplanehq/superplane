package organizations

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"google.golang.org/protobuf/types/known/structpb"
	"gorm.io/gorm"
)

func CallIntegrationTool(ctx context.Context, reg *registry.Registry, orgID string, integrationID string, toolName string, parameters map[string]string) (*pb.CallIntegrationToolResponse, error) {
	org, err := uuid.Parse(orgID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(nil, "invalid organization ID")
	}

	ID, err := uuid.Parse(integrationID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(nil, "invalid installation ID")
	}

	instance, err := models.FindIntegration(org, ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, grpcerrors.NotFound(err, "integration not found")
		}

		return nil, grpcerrors.Internal(err, "failed to load integration")
	}

	logger := logging.ForIntegration(*instance)
	if instance.State != models.IntegrationStateReady {
		logger.WithError(err).Warn("integration is in error state - cannot call tool")
		return nil, grpcerrors.FailedPrecondition(nil, "integration is in error state")
	}

	integrationCtx := contexts.NewIntegrationContext(
		database.Conn(),
		nil,
		instance,
		reg.Encryptor,
		reg,
		nil,
	)

	action, err := reg.GetAction(toolName)
	if err != nil {
		return nil, grpcerrors.FailedPrecondition(err, "action not found")
	}

	if _, ok := registry.AsIntegrationTool(action); !ok {
		return nil, grpcerrors.FailedPrecondition(nil, "action is not a tool")
	}

	logger.Infof("Executing tool: %s", toolName)
	output, err := registry.CallActionTool(action, core.IntegrationToolContext{
		Logger:        logger,
		HTTP:          reg.HTTPContext(),
		Integration:   integrationCtx,
		Configuration: parameters,
	})

	if err != nil {
		logger.WithError(err).Error("failed to execute tool")
		return nil, grpcerrors.FailedPrecondition(err, "failed to execute tool")
	}

	outputData, err := json.Marshal(output)
	if err != nil {
		logger.WithError(err).Error("failed to marshal tool output")
		return nil, grpcerrors.FailedPrecondition(err, "failed to marshal tool output")
	}

	var outputMap map[string]any
	if err := json.Unmarshal(outputData, &outputMap); err != nil {
		logger.WithError(err).Error("failed to unmarshal tool output")
		return nil, grpcerrors.FailedPrecondition(err, "failed to unmarshal tool output")
	}

	structOutput, err := structpb.NewStruct(outputMap)
	if err != nil {
		logger.WithError(err).Error("failed to convert tool output to struct")
		return nil, grpcerrors.FailedPrecondition(err, "failed to convert tool output to struct")
	}

	return &pb.CallIntegrationToolResponse{
		Output: structOutput,
	}, nil
}
