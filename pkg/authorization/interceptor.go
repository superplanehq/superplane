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

const OrganizationContextKey contextKey = "organization"
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
		pbIntegrations.Integrations_UpdateIntegration_FullMethodName:   {Resource: "integration", Action: "update", DomainTypes: []string{models.DomainTypeOrganization, models.DomainTypeCanvas}},

		// Canvases rules
		pbSuperplane.Superplane_CreateCanvas_FullMethodName:                 {Resource: "canvas", Action: "create", DomainTypes: []string{models.DomainTypeOrganization}},
		pbSuperplane.Superplane_DeleteCanvas_FullMethodName:                 {Resource: "canvas", Action: "delete", DomainTypes: []string{models.DomainTypeOrganization}},
		pbSuperplane.Superplane_DescribeCanvas_FullMethodName:               {Resource: "canvas", Action: "read", DomainTypes: []string{models.DomainTypeOrganization}},
		pbSuperplane.Superplane_ListCanvases_FullMethodName:                 {Resource: "canvas", Action: "read", DomainTypes: []string{models.DomainTypeOrganization}},
		pbSuperplane.Superplane_AddUser_FullMethodName:                      {Resource: "member", Action: "create", DomainTypes: []string{models.DomainTypeCanvas}},
		pbSuperplane.Superplane_RemoveUser_FullMethodName:                   {Resource: "member", Action: "delete", DomainTypes: []string{models.DomainTypeCanvas}},
		pbSuperplane.Superplane_CreateEventSource_FullMethodName:            {Resource: "eventsource", Action: "create", DomainTypes: []string{models.DomainTypeCanvas}},
		pbSuperplane.Superplane_ResetEventSourceKey_FullMethodName:          {Resource: "eventsource", Action: "update", DomainTypes: []string{models.DomainTypeCanvas}},
		pbSuperplane.Superplane_UpdateEventSource_FullMethodName:            {Resource: "eventsource", Action: "update", DomainTypes: []string{models.DomainTypeCanvas}},
		pbSuperplane.Superplane_DeleteEventSource_FullMethodName:            {Resource: "eventsource", Action: "delete", DomainTypes: []string{models.DomainTypeCanvas}},
		pbSuperplane.Superplane_DescribeEventSource_FullMethodName:          {Resource: "eventsource", Action: "read", DomainTypes: []string{models.DomainTypeCanvas}},
		pbSuperplane.Superplane_ListEventSources_FullMethodName:             {Resource: "eventsource", Action: "read", DomainTypes: []string{models.DomainTypeCanvas}},
		pbSuperplane.Superplane_CreateStage_FullMethodName:                  {Resource: "stage", Action: "create", DomainTypes: []string{models.DomainTypeCanvas}},
		pbSuperplane.Superplane_DescribeStage_FullMethodName:                {Resource: "stage", Action: "read", DomainTypes: []string{models.DomainTypeCanvas}},
		pbSuperplane.Superplane_UpdateStage_FullMethodName:                  {Resource: "stage", Action: "update", DomainTypes: []string{models.DomainTypeCanvas}},
		pbSuperplane.Superplane_DeleteStage_FullMethodName:                  {Resource: "stage", Action: "delete", DomainTypes: []string{models.DomainTypeCanvas}},
		pbSuperplane.Superplane_ListStages_FullMethodName:                   {Resource: "stage", Action: "read", DomainTypes: []string{models.DomainTypeCanvas}},
		pbSuperplane.Superplane_ListStageExecutions_FullMethodName:          {Resource: "stageexecution", Action: "read", DomainTypes: []string{models.DomainTypeCanvas}},
		pbSuperplane.Superplane_CancelStageExecution_FullMethodName:         {Resource: "stageexecution", Action: "update", DomainTypes: []string{models.DomainTypeCanvas}},
		pbSuperplane.Superplane_CreateConnectionGroup_FullMethodName:        {Resource: "connectiongroup", Action: "create", DomainTypes: []string{models.DomainTypeCanvas}},
		pbSuperplane.Superplane_UpdateConnectionGroup_FullMethodName:        {Resource: "connectiongroup", Action: "update", DomainTypes: []string{models.DomainTypeCanvas}},
		pbSuperplane.Superplane_DeleteConnectionGroup_FullMethodName:        {Resource: "connectiongroup", Action: "delete", DomainTypes: []string{models.DomainTypeCanvas}},
		pbSuperplane.Superplane_DescribeConnectionGroup_FullMethodName:      {Resource: "connectiongroup", Action: "read", DomainTypes: []string{models.DomainTypeCanvas}},
		pbSuperplane.Superplane_ListConnectionGroups_FullMethodName:         {Resource: "connectiongroup", Action: "read", DomainTypes: []string{models.DomainTypeCanvas}},
		pbSuperplane.Superplane_ApproveStageEvent_FullMethodName:            {Resource: "stageevent", Action: "approve", DomainTypes: []string{models.DomainTypeCanvas}},
		pbSuperplane.Superplane_DiscardStageEvent_FullMethodName:            {Resource: "stageevent", Action: "discard", DomainTypes: []string{models.DomainTypeCanvas}},
		pbSuperplane.Superplane_ListStageEvents_FullMethodName:              {Resource: "stageevent", Action: "read", DomainTypes: []string{models.DomainTypeCanvas}},
		pbSuperplane.Superplane_ListEvents_FullMethodName:                   {Resource: "event", Action: "read", DomainTypes: []string{models.DomainTypeCanvas}},
		pbSuperplane.Superplane_CreateEvent_FullMethodName:                  {Resource: "event", Action: "create", DomainTypes: []string{models.DomainTypeCanvas}},
		pbSuperplane.Superplane_ListEventRejections_FullMethodName:          {Resource: "event", Action: "read", DomainTypes: []string{models.DomainTypeCanvas}},
		pbSuperplane.Superplane_ListConnectionGroupFieldSets_FullMethodName: {Resource: "connectiongroupfieldset", Action: "read", DomainTypes: []string{models.DomainTypeCanvas}},
		pbSuperplane.Superplane_ListAlerts_FullMethodName:                   {Resource: "alert", Action: "read", DomainTypes: []string{models.DomainTypeCanvas}},
		pbSuperplane.Superplane_AcknowledgeAlert_FullMethodName:             {Resource: "alert", Action: "acknowledge", DomainTypes: []string{models.DomainTypeCanvas}},

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
		pbUsers.Users_ListUserPermissions_FullMethodName: {Resource: "member", Action: "read", DomainTypes: []string{models.DomainTypeOrganization, models.DomainTypeCanvas}},
		pbUsers.Users_ListUserRoles_FullMethodName:       {Resource: "member", Action: "read", DomainTypes: []string{models.DomainTypeOrganization, models.DomainTypeCanvas}},
		pbUsers.Users_ListUsers_FullMethodName:           {Resource: "member", Action: "read", DomainTypes: []string{models.DomainTypeOrganization, models.DomainTypeCanvas}},

		// Roles rules
		pbRoles.Roles_AssignRole_FullMethodName:   {Resource: "member", Action: "update", DomainTypes: []string{models.DomainTypeOrganization, models.DomainTypeCanvas}},
		pbRoles.Roles_ListRoles_FullMethodName:    {Resource: "role", Action: "read", DomainTypes: []string{models.DomainTypeOrganization, models.DomainTypeCanvas}},
		pbRoles.Roles_DescribeRole_FullMethodName: {Resource: "role", Action: "read", DomainTypes: []string{models.DomainTypeOrganization, models.DomainTypeCanvas}},
		pbRoles.Roles_CreateRole_FullMethodName:   {Resource: "role", Action: "create", DomainTypes: []string{models.DomainTypeOrganization, models.DomainTypeCanvas}},
		pbRoles.Roles_UpdateRole_FullMethodName:   {Resource: "role", Action: "update", DomainTypes: []string{models.DomainTypeOrganization, models.DomainTypeCanvas}},
		pbRoles.Roles_DeleteRole_FullMethodName:   {Resource: "role", Action: "delete", DomainTypes: []string{models.DomainTypeOrganization, models.DomainTypeCanvas}},

		// Organization Rules
		pbOrganization.Organizations_DescribeOrganization_FullMethodName: {Resource: "org", Action: "read", DomainTypes: []string{models.DomainTypeOrganization}},
		pbOrganization.Organizations_ListInvitations_FullMethodName:      {Resource: "org", Action: "read", DomainTypes: []string{models.DomainTypeOrganization}},
		pbOrganization.Organizations_UpdateInvitation_FullMethodName:     {Resource: "member", Action: "update", DomainTypes: []string{models.DomainTypeOrganization}},
		pbOrganization.Organizations_RemoveInvitation_FullMethodName:     {Resource: "member", Action: "delete", DomainTypes: []string{models.DomainTypeOrganization}},
		pbOrganization.Organizations_UpdateOrganization_FullMethodName:   {Resource: "org", Action: "update", DomainTypes: []string{models.DomainTypeOrganization}},
		pbOrganization.Organizations_CreateInvitation_FullMethodName:     {Resource: "member", Action: "create", DomainTypes: []string{models.DomainTypeOrganization}},
		pbOrganization.Organizations_RemoveUser_FullMethodName:           {Resource: "member", Action: "delete", DomainTypes: []string{models.DomainTypeOrganization}},
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

		orgMeta, ok := md["x-organization-id"]
		if !ok || len(orgMeta) == 0 {
			log.Errorf("Organization not found in metadata, metadata %v", md)
			return nil, status.Error(codes.NotFound, "Not found")
		}

		userID := userMeta[0]
		organizationID := orgMeta[0]
		domainType, domainID, err := a.getDomainTypeAndId(req, rule.DomainTypes, organizationID)
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

		newContext := context.WithValue(ctx, OrganizationContextKey, organizationID)
		newContext = context.WithValue(newContext, DomainTypeContextKey, domainType)
		newContext = context.WithValue(newContext, DomainIdContextKey, domainID)
		return handler(newContext, req)
	}
}

func (a *AuthorizationInterceptor) getDomainTypeAndId(req interface{}, domainTypes []string, organizationID string) (string, string, error) {
	if len(domainTypes) == 1 && domainTypes[0] == models.DomainTypeOrganization {
		org, err := models.FindOrganizationByID(organizationID)
		if err != nil {
			return "", "", fmt.Errorf("organization %s not found", organizationID)
		}

		return models.DomainTypeOrganization, org.ID.String(), nil
	}

	if len(domainTypes) == 1 && domainTypes[0] == models.DomainTypeCanvas {
		return getCanvasIdFromRequest(req, organizationID)
	}

	// Handle mixed domain types (multiple domain types supported)
	return a.getDomainTypeAndIdFromRequest(req, organizationID)
}

func (a *AuthorizationInterceptor) getDomainTypeAndIdFromRequest(req interface{}, organizationID string) (string, string, error) {
	switch r := req.(type) {
	case interface {
		GetDomainType() pbAuth.DomainType
		GetDomainId() string
	}:
		return getDomainTypeAndId(r.GetDomainId(), r.GetDomainType(), organizationID)

	default:
		return "", "", fmt.Errorf("unable to extract domain information from request")
	}
}

func getDomainTypeAndId(domainID string, domainType pbAuth.DomainType, organizationID string) (string, string, error) {
	switch domainType {
	case pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION:
		_, err := uuid.Parse(domainID)
		if err != nil {
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
		orgID, err := uuid.Parse(organizationID)
		if err != nil {
			return "", "", fmt.Errorf("invalid organization ID: %s", organizationID)
		}

		_, err = uuid.Parse(domainID)
		if err != nil {
			// Try to find canvas by name if not a valid UUID
			canvas, err := models.FindCanvasByName(domainID, orgID)
			if err != nil {
				return "", "", fmt.Errorf("canvas %s not found in organization", domainID)
			}
			return models.DomainTypeCanvas, canvas.ID.String(), nil
		}

		canvas, err := models.FindCanvasByID(domainID, orgID)
		if err != nil {
			return "", "", fmt.Errorf("canvas %s not found in organization", domainID)
		}

		return models.DomainTypeCanvas, canvas.ID.String(), nil

	default:
		return "", "", fmt.Errorf("unknown domain type: %v", domainType)
	}
}

func getCanvasIdFromRequest(req interface{}, organizationID string) (string, string, error) {
	orgUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return "", "", fmt.Errorf("invalid organization ID: %s", organizationID)
	}

	switch r := req.(type) {
	case interface{ GetCanvasId() string }:
		_, err := models.FindCanvasByID(r.GetCanvasId(), orgUUID)
		if err != nil {
			return "", "", fmt.Errorf("canvas %s not found in organization", r.GetCanvasId())
		}

		return models.DomainTypeCanvas, r.GetCanvasId(), nil

	case interface{ GetCanvasIdOrName() string }:
		canvasIDOrName := r.GetCanvasIdOrName()
		_, err := uuid.Parse(canvasIDOrName)
		if err == nil {
			_, err := models.FindCanvasByID(canvasIDOrName, orgUUID)
			if err != nil {
				return "", "", fmt.Errorf("canvas %s not found in organization", canvasIDOrName)
			}

			return models.DomainTypeCanvas, canvasIDOrName, nil
		}

		canvas, err := models.FindCanvasByName(canvasIDOrName, orgUUID)
		if err != nil {
			return "", "", fmt.Errorf("canvas %s not found in organization", canvasIDOrName)
		}

		return models.DomainTypeCanvas, canvas.ID.String(), nil

	default:
		return "", "", fmt.Errorf("missing canvas ID")
	}
}
