package organizations

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"gorm.io/gorm"
)

func RetryIntegrationWebhook(ctx context.Context, orgID, integrationID, webhookID string) (*pb.RetryIntegrationWebhookResponse, error) {
	org, err := resolveOrganizationID(ctx, orgID)
	if err != nil {
		return nil, err
	}
	integration, err := uuid.Parse(integrationID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(nil, "invalid integration ID")
	}
	id, err := uuid.Parse(webhookID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(nil, "invalid webhook ID")
	}
	db := database.DB(ctx)
	if _, err := models.FindIntegrationInTransaction(db, org, integration); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, grpcerrors.NotFound(err, "integration not found")
		}
		return nil, grpcerrors.Internal(err, "could not load integration")
	}
	hook, err := models.FindIntegrationWebhook(db, integration, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, grpcerrors.NotFound(err, "webhook not found")
		}
		return nil, grpcerrors.Internal(err, "could not load webhook")
	}
	if hook.State == models.WebhookStateReady {
		return nil, grpcerrors.FailedPrecondition(nil, "webhook setup has already completed")
	}
	if err := hook.RetrySetup(db); err != nil {
		return nil, grpcerrors.Internal(err, "could not retry webhook setup")
	}
	return &pb.RetryIntegrationWebhookResponse{}, nil
}
