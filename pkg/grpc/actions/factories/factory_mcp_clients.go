package factories

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

// Matches mcpserver.LocalClientID and LocalClientName. This package cannot
// import mcpserver because mcpserver already imports factory actions.
const localMCPClientID = "superplane-local"
const localMCPClientName = "Cursor"

func ListFactoryMCPClients(
	ctx context.Context,
	organizationID string,
	req *pb.ListFactoryMCPClientsRequest,
) (*pb.ListFactoryMCPClientsResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list MCP clients")
	}
	if err := requireSuperPlaneMCPServer(orgID); err != nil {
		return nil, factoryErrorToStatus(err, "failed to list MCP clients")
	}
	db := database.DB(ctx)
	factory, err := findFactory(db, orgID, req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list MCP clients")
	}

	tokens, err := models.ListMCPOAuthRefreshTokensForFactory(db, orgID, factory.ID, time.Now())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list MCP clients")
	}

	clients, err := serializeFactoryMCPClients(db, orgID, tokens)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list MCP clients")
	}
	return &pb.ListFactoryMCPClientsResponse{Clients: clients}, nil
}

func RevokeFactoryMCPClient(
	ctx context.Context,
	organizationID string,
	req *pb.RevokeFactoryMCPClientRequest,
) (*pb.RevokeFactoryMCPClientResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to revoke MCP client")
	}
	if err := requireSuperPlaneMCPServer(orgID); err != nil {
		return nil, factoryErrorToStatus(err, "failed to revoke MCP client")
	}
	clientID, err := parseMCPClientID(req.GetClientId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to revoke MCP client")
	}
	db := database.DB(ctx)
	factory, err := findFactory(db, orgID, req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to revoke MCP client")
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		locked, err := models.FindMCPOAuthRefreshTokenForFactory(tx, orgID, factory.ID, clientID)
		if err != nil {
			return err
		}
		return models.DeleteMCPOAuthRefreshTokensForClient(tx, orgID, factory.ID, locked.UserID, locked.ClientID)
	})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to revoke MCP client")
	}
	return &pb.RevokeFactoryMCPClientResponse{}, nil
}

func requireSuperPlaneMCPServer(orgID uuid.UUID) error {
	enabled, err := models.HasExperimentalFeature(orgID, features.FeatureSuperPlaneMCPServer)
	if err != nil {
		return err
	}
	if enabled {
		return nil
	}
	return errSuperPlaneMCPServerDisabled
}

func parseMCPClientID(raw string) (uuid.UUID, error) {
	id, err := uuid.Parse(strings.TrimSpace(raw))
	if err != nil {
		return uuid.Nil, invalidArgument("invalid MCP client id")
	}
	return id, nil
}

func serializeFactoryMCPClients(tx *gorm.DB, orgID uuid.UUID, tokens []models.MCPOAuthRefreshToken) ([]*pb.FactoryMCPClient, error) {
	out := make([]*pb.FactoryMCPClient, 0, len(tokens))
	if len(tokens) == 0 {
		return out, nil
	}

	userIDs := make([]string, 0, len(tokens))
	clientIDs := make([]string, 0, len(tokens))
	seenUsers := map[string]struct{}{}
	seenClients := map[string]struct{}{}
	for _, token := range tokens {
		userID := token.UserID.String()
		if _, ok := seenUsers[userID]; !ok {
			seenUsers[userID] = struct{}{}
			userIDs = append(userIDs, userID)
		}
		if token.ClientID == "" {
			continue
		}
		if _, ok := seenClients[token.ClientID]; ok {
			continue
		}
		seenClients[token.ClientID] = struct{}{}
		clientIDs = append(clientIDs, token.ClientID)
	}

	users, err := models.FindUsersByIDsInOrganization(tx, orgID.String(), userIDs)
	if err != nil {
		return nil, err
	}
	usersByID := make(map[uuid.UUID]models.User, len(users))
	for _, user := range users {
		usersByID[user.ID] = user
	}

	oauthClients, err := models.ListMCPOAuthClientsByClientIDs(tx, clientIDs)
	if err != nil {
		return nil, err
	}
	clientsByID := make(map[string]models.MCPOAuthClient, len(oauthClients))
	for _, client := range oauthClients {
		clientsByID[client.ClientID] = client
	}

	for i := range tokens {
		token := tokens[i]
		client := &pb.FactoryMCPClient{
			Id:         token.ID.String(),
			ClientName: mcpClientName(token.ClientID, clientsByID),
			UserId:     token.UserID.String(),
			CreatedAt:  timestamppb.New(token.CreatedAt),
		}
		if user, ok := usersByID[token.UserID]; ok {
			client.UserName = user.Name
			client.UserEmail = user.GetEmail()
		}
		out = append(out, client)
	}
	return out, nil
}

func mcpClientName(clientID string, clients map[string]models.MCPOAuthClient) string {
	if clientID == localMCPClientID {
		return localMCPClientName
	}
	if client, ok := clients[clientID]; ok {
		name := strings.TrimSpace(client.ClientName)
		if name != "" {
			return name
		}
	}
	return strings.TrimSpace(clientID)
}
