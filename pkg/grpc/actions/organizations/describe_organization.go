package organizations

import (
	"context"
	"errors"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

func DescribeOrganization(ctx context.Context, req *pb.DescribeOrganizationRequest) (*pb.DescribeOrganizationResponse, error) {
	if req.IdOrName == "" {
		return nil, status.Error(codes.InvalidArgument, "id_or_name is required")
	}

	var organization *models.Organization
	var err error

	err = actions.ValidateUUIDs(req.IdOrName)
	if err != nil {
		organization, err = models.FindOrganizationByName(req.IdOrName)
	} else {
		organization, err = models.FindOrganizationByID(req.IdOrName)
	}

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "organization not found")
		}

		log.Errorf("Error describing organization. Request: %v. Error: %v", req, err)
		return nil, err
	}

	response := &pb.DescribeOrganizationResponse{
		Organization: &pb.Organization{
			Metadata: &pb.Organization_Metadata{
				Id:          organization.ID.String(),
				Name:        organization.Name,
				DisplayName: organization.DisplayName,
				CreatedBy:   organization.CreatedBy.String(),
				CreatedAt:   timestamppb.New(*organization.CreatedAt),
				UpdatedAt:   timestamppb.New(*organization.UpdatedAt),
			},
		},
	}

	return response, nil
}
