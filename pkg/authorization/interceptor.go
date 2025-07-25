package authorization

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	pbAuth "github.com/superplanehq/superplane/pkg/protos/authorization"
	pbSuperplane "github.com/superplanehq/superplane/pkg/protos/canvases"
	pbGroups "github.com/superplanehq/superplane/pkg/protos/groups"
	pbIntegrations "github.com/superplanehq/superplane/pkg/protos/integrations"
	pbOrganization "github.com/superplanehq/superplane/pkg/protos/organizations"
	pbRoles "github.com/superplanehq/superplane/pkg/protos/roles"
	pbSecrets "github.com/superplanehq/superplane/pkg/protos/secrets"
	pbUsers "github.com/superplanehq/superplane/pkg/protos/users"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type contextKey string

const DomainTypeContextKey contextKey = "domainType"
const DomainIdContextKey contextKey = "domainId"

type AuthorizationRule struct {
	Resource    string
	Action      string
	DomainTypes []string
}

type AuthorizationInterceptor struct {
	authService Authorization
	rules       map[string]AuthorizationRule
}

func NewAuthorizationInterceptor(authService Authorization) *AuthorizationInterceptor {
	rules := map[string]AuthorizationRule{
		// Secrets rules
		pbSecrets.Secrets_CreateSecret_FullMethodName:   {Resource: "secret", Action: "create", DomainTypes: []string{models.DomainTypeOrganization, models.DomainTypeCanvas}},
		pbSecrets.Secrets_UpdateSecret_FullMethodName:   {Resource: "secret", Action: "update", DomainTypes: []string{models.DomainTypeOrganization, models.DomainTypeCanvas}},
		pbSecrets.Secrets_DescribeSecret_FullMethodName: {Resource: "secret", Action: "read", DomainTypes: []string{models.DomainTypeOrganization, models.DomainTypeCanvas}},
		pbSecrets.Secrets_ListSecrets_FullMethodName:    {Resource: "secret", Action: "read", DomainTypes: []string{models.DomainTypeOrganization, models.DomainTypeCanvas}},
		pbSecrets.Secrets_DeleteSecret_FullMethodName:   {Resource: "secret", Action: "delete", DomainTypes: []string{models.DomainTypeOrganization, models.DomainTypeCanvas}},

		// Integrations rules
		pbIntegrations.Integrations_DescribeIntegration_FullMethodName: {Resource: "integration", Action: "read", DomainTypes: []string{models.DomainTypeOrganization, models.DomainTypeCanvas}},
		pbIntegrations.Integrations_ListIntegrations_FullMethodName:    {Resource: "integration", Action: "read", DomainTypes: []string{models.DomainTypeOrganization, models.DomainTypeCanvas}},
		pbIntegrations.Integrations_CreateIntegration_FullMethodName:   {Resource: "integration", Action: "create", DomainTypes: []string{models.DomainTypeOrganization, models.DomainTypeCanvas}},

		// Canvases rules
		pbSuperplane.Superplane_CreateCanvas_FullMethodName:                 {Resource: "canvas", Action: "create", DomainTypes: []string{models.DomainTypeOrganization}},
		pbSuperplane.Superplane_DescribeCanvas_FullMethodName:               {Resource: "canvas", Action: "read", DomainTypes: []string{models.DomainTypeOrganization}},
		pbSuperplane.Superplane_ListCanvases_FullMethodName:                 {Resource: "canvas", Action: "read", DomainTypes: []string{models.DomainTypeOrganization}},
		pbSuperplane.Superplane_CreateEventSource_FullMethodName:            {Resource: "eventsource", Action: "create", DomainTypes: []string{models.DomainTypeCanvas}},
		pbSuperplane.Superplane_DescribeEventSource_FullMethodName:          {Resource: "eventsource", Action: "read", DomainTypes: []string{models.DomainTypeCanvas}},
		pbSuperplane.Superplane_ListEventSources_FullMethodName:             {Resource: "eventsource", Action: "read", DomainTypes: []string{models.DomainTypeCanvas}},
		pbSuperplane.Superplane_CreateStage_FullMethodName:                  {Resource: "stage", Action: "create", DomainTypes: []string{models.DomainTypeCanvas}},
		pbSuperplane.Superplane_DescribeStage_FullMethodName:                {Resource: "stage", Action: "read", DomainTypes: []string{models.DomainTypeCanvas}},
		pbSuperplane.Superplane_UpdateStage_FullMethodName:                  {Resource: "stage", Action: "update", DomainTypes: []string{models.DomainTypeCanvas}},
		pbSuperplane.Superplane_ListStages_FullMethodName:                   {Resource: "stage", Action: "read", DomainTypes: []string{models.DomainTypeCanvas}},
		pbSuperplane.Superplane_CreateConnectionGroup_FullMethodName:        {Resource: "connectiongroup", Action: "create", DomainTypes: []string{models.DomainTypeCanvas}},
		pbSuperplane.Superplane_DescribeConnectionGroup_FullMethodName:      {Resource: "connectiongroup", Action: "read", DomainTypes: []string{models.DomainTypeCanvas}},
		pbSuperplane.Superplane_ListConnectionGroups_FullMethodName:         {Resource: "connectiongroup", Action: "read", DomainTypes: []string{models.DomainTypeCanvas}},
		pbSuperplane.Superplane_ApproveStageEvent_FullMethodName:            {Resource: "stageevent", Action: "approve", DomainTypes: []string{models.DomainTypeCanvas}},
		pbSuperplane.Superplane_ListStageEvents_FullMethodName:              {Resource: "stageevent", Action: "read", DomainTypes: []string{models.DomainTypeCanvas}},
		pbSuperplane.Superplane_ListConnectionGroupFieldSets_FullMethodName: {Resource: "connectiongroupfieldset", Action: "read", DomainTypes: []string{models.DomainTypeCanvas}},

		// Groups rules
		pbGroups.Groups_CreateGroup_FullMethodName:         {Resource: "group", Action: "create", DomainTypes: []string{models.DomainTypeOrganization, models.DomainTypeCanvas}},
		pbGroups.Groups_AddUserToGroup_FullMethodName:      {Resource: "group", Action: "update", DomainTypes: []string{models.DomainTypeOrganization, models.DomainTypeCanvas}},
		pbGroups.Groups_RemoveUserFromGroup_FullMethodName: {Resource: "group", Action: "update", DomainTypes: []string{models.DomainTypeOrganization, models.DomainTypeCanvas}},
		pbGroups.Groups_UpdateGroup_FullMethodName:         {Resource: "group", Action: "update", DomainTypes: []string{models.DomainTypeOrganization, models.DomainTypeCanvas}},
		pbGroups.Groups_ListGroups_FullMethodName:          {Resource: "group", Action: "read", DomainTypes: []string{models.DomainTypeOrganization, models.DomainTypeCanvas}},
		pbGroups.Groups_ListGroupUsers_FullMethodName:      {Resource: "group", Action: "read", DomainTypes: []string{models.DomainTypeOrganization, models.DomainTypeCanvas}},
		pbGroups.Groups_DescribeGroup_FullMethodName:       {Resource: "group", Action: "read", DomainTypes: []string{models.DomainTypeOrganization, models.DomainTypeCanvas}},
		pbGroups.Groups_DeleteGroup_FullMethodName:         {Resource: "group", Action: "delete", DomainTypes: []string{models.DomainTypeOrganization, models.DomainTypeCanvas}},

		// Users rules
		pbUsers.Users_ListUserPermissions_FullMethodName: {Resource: "user", Action: "read", DomainTypes: []string{models.DomainTypeOrganization, models.DomainTypeCanvas}},
		pbUsers.Users_ListUserRoles_FullMethodName:       {Resource: "user", Action: "read", DomainTypes: []string{models.DomainTypeOrganization, models.DomainTypeCanvas}},
		pbUsers.Users_ListUsers_FullMethodName:           {Resource: "user", Action: "read", DomainTypes: []string{models.DomainTypeOrganization, models.DomainTypeCanvas}},

		// Roles rules
		pbRoles.Roles_AssignRole_FullMethodName:   {Resource: "role", Action: "assign", DomainTypes: []string{models.DomainTypeOrganization, models.DomainTypeCanvas}},
		pbRoles.Roles_RemoveRole_FullMethodName:   {Resource: "role", Action: "remove", DomainTypes: []string{models.DomainTypeOrganization, models.DomainTypeCanvas}},
		pbRoles.Roles_ListRoles_FullMethodName:    {Resource: "role", Action: "read", DomainTypes: []string{models.DomainTypeOrganization, models.DomainTypeCanvas}},
		pbRoles.Roles_DescribeRole_FullMethodName: {Resource: "role", Action: "read", DomainTypes: []string{models.DomainTypeOrganization, models.DomainTypeCanvas}},
		pbRoles.Roles_CreateRole_FullMethodName:   {Resource: "role", Action: "create", DomainTypes: []string{models.DomainTypeOrganization, models.DomainTypeCanvas}},
		pbRoles.Roles_UpdateRole_FullMethodName:   {Resource: "role", Action: "update", DomainTypes: []string{models.DomainTypeOrganization, models.DomainTypeCanvas}},
		pbRoles.Roles_DeleteRole_FullMethodName:   {Resource: "role", Action: "delete", DomainTypes: []string{models.DomainTypeOrganization, models.DomainTypeCanvas}},

		// Organization Rules
		pbOrganization.Organizations_DescribeOrganization_FullMethodName: {Resource: "org", Action: "read", DomainTypes: []string{models.DomainTypeOrganization}},
		pbOrganization.Organizations_UpdateOrganization_FullMethodName:   {Resource: "org", Action: "update", DomainTypes: []string{models.DomainTypeOrganization}},
		pbOrganization.Organizations_DeleteOrganization_FullMethodName:   {Resource: "org", Action: "delete", DomainTypes: []string{models.DomainTypeOrganization}},
	}

	return &AuthorizationInterceptor{
		authService: authService,
		rules:       rules,
	}
}

func (a *AuthorizationInterceptor) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		rule, requiresAuth := a.rules[info.FullMethod]
		if !requiresAuth {
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			log.Errorf("Metadata not found in context")
			return nil, status.Error(codes.NotFound, "Not found")
		}

		userMeta, ok := md["x-user-id"]
		if !ok || len(userMeta) == 0 {
			log.Errorf("User not found in metadata, metadata %v", md)
			return nil, status.Error(codes.NotFound, "Not found")
		}

		userID := userMeta[0]
		domainType, domainID, err := a.getDomainTypeAndId(req, rule.DomainTypes)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		var allowed bool
		if domainType == models.DomainTypeOrganization {
			allowed, err = a.authService.CheckOrganizationPermission(userID, domainID, rule.Resource, rule.Action)
			if err != nil {
				return nil, err
			}
		}
		if domainType == models.DomainTypeCanvas {
			allowed, err = a.authService.CheckCanvasPermission(userID, domainID, rule.Resource, rule.Action)
			if err != nil {
				return nil, err
			}
		}

		if !allowed {
			log.Warnf("User %s tried to %s %s in %s %s", userID, rule.Action, rule.Resource, domainType, domainID)
			return nil, status.Error(codes.NotFound, "Not found")
		}

		newContext := context.WithValue(ctx, DomainTypeContextKey, domainType)
		newContext = context.WithValue(newContext, DomainIdContextKey, domainID)
		return handler(newContext, req)
	}
}

func (a *AuthorizationInterceptor) getDomainTypeAndId(req interface{}, domainTypes []string) (string, string, error) {
	if len(domainTypes) == 1 && domainTypes[0] == models.DomainTypeOrganization {
		return getOrganizationIdFromRequest(req)
	}

	if len(domainTypes) == 1 && domainTypes[0] == models.DomainTypeCanvas {
		return getCanvasIdFromRequest(req)
	}

	// Handle mixed domain types (multiple domain types supported)
	return a.getDomainTypeAndIdFromRequest(req)
}

func (a *AuthorizationInterceptor) getDomainTypeAndIdFromRequest(req interface{}) (string, string, error) {
	switch r := req.(type) {
	case interface {
		GetDomainType() pbAuth.DomainType
		GetDomainId() string
	}:
		return getDomainTypeAndId(r.GetDomainId(), r.GetDomainType())

	default:
		return "", "", fmt.Errorf("unable to extract domain information from request")
	}
}

func getDomainTypeAndId(domainID string, domainType pbAuth.DomainType) (string, string, error) {
	switch domainType {
	case pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION:
		_, err := uuid.Parse(domainID)
		if err != nil {
			// Try to find organization by name if not a valid UUID
			org, err := models.FindOrganizationByName(domainID)
			if err != nil {
				return "", "", fmt.Errorf("organization %s not found", domainID)
			}
			return models.DomainTypeOrganization, org.ID.String(), nil
		}

		org, err := models.FindOrganizationByID(domainID)
		if err != nil {
			return "", "", fmt.Errorf("organization %s not found", domainID)
		}

		return models.DomainTypeOrganization, org.ID.String(), nil

	case pbAuth.DomainType_DOMAIN_TYPE_CANVAS:
		_, err := uuid.Parse(domainID)
		if err != nil {
			// Try to find canvas by name if not a valid UUID
			canvas, err := models.FindCanvasByName(domainID)
			if err != nil {
				return "", "", fmt.Errorf("canvas %s not found", domainID)
			}
			return models.DomainTypeCanvas, canvas.ID.String(), nil
		}

		canvas, err := models.FindCanvasByName(domainID)
		if err != nil {
			return "", "", fmt.Errorf("canvas %s not found", domainID)
		}

		return models.DomainTypeCanvas, canvas.ID.String(), nil

	default:
		return "", "", fmt.Errorf("unknown domain type: %v", domainType)
	}
}

func getOrganizationIdFromRequest(req interface{}) (string, string, error) {
	var domainID string
	switch r := req.(type) {
	case interface{ GetOrganizationId() string }:
		domainID = r.GetOrganizationId()
	case interface{ GetIdOrName() string }:
		domainID = r.GetIdOrName()
	default:
		return "", "", fmt.Errorf("missing organization ID")
	}

	_, err := uuid.Parse(domainID)
	if err != nil {
		// Try to find organization by name if not a valid UUID
		org, err := models.FindOrganizationByName(domainID)
		if err != nil {
			return "", "", fmt.Errorf("organization %s not found", domainID)
		}
		return models.DomainTypeOrganization, org.ID.String(), nil
	}

	org, err := models.FindOrganizationByID(domainID)
	if err != nil {
		return "", "", fmt.Errorf("organization %s not found", domainID)
	}

	return models.DomainTypeOrganization, org.ID.String(), nil
}

func getCanvasIdFromRequest(req interface{}) (string, string, error) {
	switch r := req.(type) {
	case interface{ GetCanvasId() string }:
		_, err := models.FindCanvasByID(r.GetCanvasId())
		if err != nil {
			return "", "", fmt.Errorf("canvas %s not found", r.GetCanvasId())
		}

		return models.DomainTypeCanvas, r.GetCanvasId(), nil

	case interface{ GetCanvasIdOrName() string }:
		canvasIDOrName := r.GetCanvasIdOrName()
		_, err := uuid.Parse(canvasIDOrName)
		if err == nil {
			_, err := models.FindCanvasByID(canvasIDOrName)
			if err != nil {
				return "", "", fmt.Errorf("canvas %s not found", canvasIDOrName)
			}

			return models.DomainTypeCanvas, canvasIDOrName, nil
		}

		canvas, err := models.FindCanvasByName(canvasIDOrName)
		if err != nil {
			return "", "", fmt.Errorf("canvas %s not found", canvasIDOrName)
		}

		return models.DomainTypeCanvas, canvas.ID.String(), nil

	default:
		return "", "", fmt.Errorf("missing canvas ID")
	}
}
