package organizations

import (
	"context"
	"errors"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/pkg/telemetry"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

func DescribeOrganization(ctx context.Context, orgID string) (*pb.DescribeOrganizationResponse, error) {
	organization, err := loadOrganization(ctx, orgID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, grpcerrors.NotFound(err, "organization not found")
		}

		log.Errorf("Error describing organization %s: %v", orgID, err)
		return nil, err
	}

	response := &pb.DescribeOrganizationResponse{
		Organization: &pb.Organization{
			Metadata: &pb.Organization_Metadata{
				Id:          organization.ID.String(),
				Name:        organization.Name,
				Description: organization.Description,
				CreatedAt:   timestamppb.New(*organization.CreatedAt),
				UpdatedAt:   timestamppb.New(*organization.UpdatedAt),
			},
			Spec: &pb.Organization_Spec{
				EnabledExperimentalFeatures: []string(organization.EnabledExperimentalFeatures),
			},
		},
	}

	return response, nil
}

func loadOrganization(ctx context.Context, orgID string) (organization *models.Organization, err error) {
	ctx, done := telemetry.Span(ctx, "organizations.load")
	defer done(&err)

	return models.FindOrganizationByIDInTransaction(database.DB(ctx), orgID)
}
