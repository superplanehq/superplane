package organizations

import (
	"context"
	"errors"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func DescribeOrganization(ctx context.Context, orgID string) (*pb.DescribeOrganizationResponse, error) {
	organization, err := models.FindOrganizationByID(orgID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "organization not found")
		}

		log.Errorf("Error describing organization %s: %v", orgID, err)
		return nil, err
	}

	response := &pb.DescribeOrganizationResponse{
		Organization: organizationToProto(organization),
	}

	return response, nil
}
