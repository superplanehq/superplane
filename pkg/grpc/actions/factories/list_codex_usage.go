package factories

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ListFactoryCodexUsage(ctx context.Context, organizationID string, req *pb.ListFactoryCodexUsageRequest) (*pb.ListFactoryCodexUsageResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, err
	}
	db := database.DB(ctx)
	if _, err := findFactory(db, orgID, req.GetFactoryId()); err != nil {
		return nil, factoryErrorToStatus(err, "failed to load workspace")
	}
	connections, err := models.ListEligibleCodexConnections(db, orgID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to load connected ChatGPT members")
	}
	users, err := models.ListHumanUsersInOrganization(db, orgID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to load workspace members")
	}
	byOwner := make(map[uuid.UUID]models.FactoryCodexConnection, len(connections))
	for _, connection := range connections {
		byOwner[connection.OwnerUserID] = connection
	}
	response := &pb.ListFactoryCodexUsageResponse{}
	for _, user := range users {
		connection, connected := byOwner[user.ID]
		if !connected {
			continue
		}
		member := &pb.CodexMemberUsage{UserId: user.ID.String(), DisplayName: user.Name, Email: user.GetEmail()}
		if connection.UsageObservedAt != nil {
			member.ObservedAt = timestamppb.New(*connection.UsageObservedAt)
		}
		for _, limit := range connection.UsageLimits {
			member.Limits = append(member.Limits, &pb.CodexUsageLimit{
				LimitId: limit.LimitID, LimitName: limit.LimitName, PlanType: limit.PlanType,
				Primary: actions.SerializeCodexUsageWindow(limit.Primary), Secondary: actions.SerializeCodexUsageWindow(limit.Secondary),
			})
		}
		response.Members = append(response.Members, member)
	}
	return response, nil
}
