package integrations

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/integrations"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func DescribeIntegration(ctx context.Context, domainType, domainID, idOrName string) (*pb.DescribeIntegrationResponse, error) {
	err := actions.ValidateUUIDs(idOrName)
	var integration *models.Integration
	if err != nil {
		integration, err = models.FindIntegrationByName(domainType, uuid.MustParse(domainID), idOrName)
	} else {
		integration, err = models.FindDomainIntegration(domainType, uuid.MustParse(domainID), uuid.MustParse(idOrName))
	}

	if err != nil {
		return nil, status.Error(codes.NotFound, "integration not found")
	}

	response := &pb.DescribeIntegrationResponse{
		Integration: serializeIntegration(*integration),
	}

	return response, nil
}
