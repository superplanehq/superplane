package organizations

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func DeleteOrganization(ctx context.Context, authService authorization.Authorization, orgID string) (*pb.DeleteOrganizationResponse, error) {
	userID, userIsSet := authentication.GetUserIdFromMetadata(ctx)
	if !userIsSet {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	organization, err := models.FindOrganizationByID(orgID)
	if err != nil {
		return nil, status.Error(codes.NotFound, "organization not found")
	}

	tx := database.Conn().Begin()

	if err := models.SoftDeleteOrganizationInTransaction(tx, organization.ID.String()); err != nil {
		tx.Rollback()
		log.Errorf("Error soft-deleting organization %s: %v", orgID, err)
		return nil, status.Error(codes.Internal, "failed to delete organization")
	}

	if err := models.SoftDeleteOrganizationCanvasesInTransaction(tx, organization.ID.String()); err != nil {
		tx.Rollback()
		log.Errorf("Error soft-deleting canvases for organization %s: %v", orgID, err)
		return nil, status.Error(codes.Internal, "failed to delete organization canvases")
	}

	if err := models.SoftDeleteOrganizationIntegrationsInTransaction(tx, organization.ID.String()); err != nil {
		tx.Rollback()
		log.Errorf("Error soft-deleting integrations for organization %s: %v", orgID, err)
		return nil, status.Error(codes.Internal, "failed to delete organization integrations")
	}

	if err := models.SoftDeleteOrganizationUsersInTransaction(tx, organization.ID.String()); err != nil {
		tx.Rollback()
		log.Errorf("Error soft-deleting users for organization %s: %v", orgID, err)
		return nil, status.Error(codes.Internal, "failed to delete organization users")
	}

	if err := models.DeleteOrganizationBlueprintsInTransaction(tx, organization.ID.String()); err != nil {
		tx.Rollback()
		log.Errorf("Error deleting blueprints for organization %s: %v", orgID, err)
		return nil, status.Error(codes.Internal, "failed to delete organization blueprints")
	}

	if err := models.DeleteOrganizationInvitationsInTransaction(tx, organization.ID.String()); err != nil {
		tx.Rollback()
		log.Errorf("Error deleting invitations for organization %s: %v", orgID, err)
		return nil, status.Error(codes.Internal, "failed to delete organization invitations")
	}

	if err := models.DeleteOrganizationInviteLinksInTransaction(tx, organization.ID.String()); err != nil {
		tx.Rollback()
		log.Errorf("Error deleting invite links for organization %s: %v", orgID, err)
		return nil, status.Error(codes.Internal, "failed to delete organization invite links")
	}

	if err := models.DeleteOrganizationAgentSettingsInTransaction(tx, organization.ID.String()); err != nil {
		tx.Rollback()
		log.Errorf("Error deleting agent settings for organization %s: %v", orgID, err)
		return nil, status.Error(codes.Internal, "failed to delete organization agent settings")
	}

	if err := models.DeleteOrganizationSecretsInTransaction(tx, organization.ID); err != nil {
		tx.Rollback()
		log.Errorf("Error deleting secrets for organization %s: %v", orgID, err)
		return nil, status.Error(codes.Internal, "failed to delete organization secrets")
	}

	if err := models.DeleteOrganizationIntegrationSecretsInTransaction(tx, organization.ID.String()); err != nil {
		tx.Rollback()
		log.Errorf("Error deleting integration secrets for organization %s: %v", orgID, err)
		return nil, status.Error(codes.Internal, "failed to delete organization integration secrets")
	}

	if err := authService.DestroyOrganization(tx, organization.ID.String()); err != nil {
		tx.Rollback()
		log.Errorf("Error deleting organization roles for %s: %v", orgID, err)
		return nil, status.Error(codes.Internal, "failed to delete organization roles")
	}

	if err := tx.Commit().Error; err != nil {
		log.Errorf("Error committing transaction for organization %s (%s) deletion: %v", organization.Name, organization.ID.String(), err)
		return nil, status.Error(codes.Internal, "failed to commit organization deletion")
	}

	log.Infof(
		"Organization %s (%s) soft-deleted by user %s with cascade to all child resources",
		organization.Name,
		organization.ID.String(),
		userID,
	)

	return &pb.DeleteOrganizationResponse{}, nil
}
