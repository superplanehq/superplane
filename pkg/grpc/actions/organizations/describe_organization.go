package organizations

import (
	"context"
	"errors"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/pkg/telemetry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

func DescribeOrganization(ctx context.Context, orgID string) (*pb.DescribeOrganizationResponse, error) {
	var organization *models.Organization
	err := telemetry.RunSpan(ctx, "organizations.load", func(ctx context.Context) error {
		var loadErr error
		organization, loadErr = models.FindOrganizationByIDInTransaction(database.DB(ctx), orgID)
		return loadErr
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "organization not found")
		}

		log.WithError(err).
			WithField("organization_id", orgID).
			Error("failed to describe organization")
		return nil, status.Error(codes.Internal, "failed to describe organization")
	}

	response := &pb.DescribeOrganizationResponse{
		Organization: &pb.Organization{
			Metadata: &pb.Organization_Metadata{
				Id:          organization.ID.String(),
				Name:        organization.Name,
				Description: organization.Description,
				CreatedAt:   protoTime(organization.CreatedAt),
				UpdatedAt:   protoTime(organization.UpdatedAt),
			},
			Spec: &pb.Organization_Spec{
				EnabledExperimentalFeatures: []string(organization.EnabledExperimentalFeatures),
			},
		},
	}

	return response, nil
}

// protoTime converts a nullable time.Time pointer into a protobuf
// Timestamp without panicking when the pointer is nil. Organizations
// should always have CreatedAt/UpdatedAt populated, but historical or
// partially-migrated rows can have NULL values; without this guard the
// describe handler panics, which the gateway translates into an
// information-level HTTP 500 in Sentry with no underlying context.
func protoTime(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	}
	return timestamppb.New(*t)
}
