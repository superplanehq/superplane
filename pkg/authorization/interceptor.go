package authorization

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	pbAuth "github.com/superplanehq/superplane/pkg/protos/authorization"
	pbOrganization "github.com/superplanehq/superplane/pkg/protos/organizations"
	pbSuperplane "github.com/superplanehq/superplane/pkg/protos/superplane"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type AuthorizationRule struct {
	Resource   string
	Action     string
	DomainType string
}

type AuthorizationInterceptor struct {
	authService Authorization
	rules       map[string]AuthorizationRule
}

func NewAuthorizationInterceptor(authService Authorization) *AuthorizationInterceptor {
	rules := map[string]AuthorizationRule{
		// Superplane rules
		pbSuperplane.Superplane_CreateCanvas_FullMethodName:                 {Resource: "canvas", Action: "create", DomainType: "org"},
		pbSuperplane.Superplane_DescribeCanvas_FullMethodName:               {Resource: "canvas", Action: "read", DomainType: "org"},
		pbSuperplane.Superplane_ListCanvases_FullMethodName:                 {Resource: "canvas", Action: "read", DomainType: "org"},
		pbSuperplane.Superplane_CreateEventSource_FullMethodName:            {Resource: "eventsource", Action: "create", DomainType: "canvas"},
		pbSuperplane.Superplane_DescribeEventSource_FullMethodName:          {Resource: "eventsource", Action: "read", DomainType: "canvas"},
		pbSuperplane.Superplane_ListEventSources_FullMethodName:             {Resource: "eventsource", Action: "read", DomainType: "canvas"},
		pbSuperplane.Superplane_CreateStage_FullMethodName:                  {Resource: "stage", Action: "create", DomainType: "canvas"},
		pbSuperplane.Superplane_DescribeStage_FullMethodName:                {Resource: "stage", Action: "read", DomainType: "canvas"},
		pbSuperplane.Superplane_UpdateStage_FullMethodName:                  {Resource: "stage", Action: "update", DomainType: "canvas"},
		pbSuperplane.Superplane_ListStages_FullMethodName:                   {Resource: "stage", Action: "read", DomainType: "canvas"},
		pbSuperplane.Superplane_CreateConnectionGroup_FullMethodName:        {Resource: "connectiongroup", Action: "create", DomainType: "canvas"},
		pbSuperplane.Superplane_DescribeConnectionGroup_FullMethodName:      {Resource: "connectiongroup", Action: "read", DomainType: "canvas"},
		pbSuperplane.Superplane_ListConnectionGroups_FullMethodName:         {Resource: "connectiongroup", Action: "read", DomainType: "canvas"},
		pbSuperplane.Superplane_CreateSecret_FullMethodName:                 {Resource: "secret", Action: "create", DomainType: "canvas"},
		pbSuperplane.Superplane_UpdateSecret_FullMethodName:                 {Resource: "secret", Action: "update", DomainType: "canvas"},
		pbSuperplane.Superplane_DescribeSecret_FullMethodName:               {Resource: "secret", Action: "read", DomainType: "canvas"},
		pbSuperplane.Superplane_ListSecrets_FullMethodName:                  {Resource: "secret", Action: "read", DomainType: "canvas"},
		pbSuperplane.Superplane_DeleteSecret_FullMethodName:                 {Resource: "secret", Action: "delete", DomainType: "canvas"},
		pbSuperplane.Superplane_ApproveStageEvent_FullMethodName:            {Resource: "stageevent", Action: "approve", DomainType: "canvas"},
		pbSuperplane.Superplane_ListStageEvents_FullMethodName:              {Resource: "stageevent", Action: "read", DomainType: "canvas"},
		pbSuperplane.Superplane_ListConnectionGroupFieldSets_FullMethodName: {Resource: "connectiongroupfieldset", Action: "read", DomainType: "canvas"},
		pbAuth.Authorization_CreateCanvasGroup_FullMethodName:               {Resource: "group", Action: "create", DomainType: "canvas"},
		pbAuth.Authorization_AddUserToCanvasGroup_FullMethodName:            {Resource: "group", Action: "update", DomainType: "canvas"},
		pbAuth.Authorization_RemoveUserFromCanvasGroup_FullMethodName:       {Resource: "group", Action: "update", DomainType: "canvas"},
		pbAuth.Authorization_ListCanvasGroups_FullMethodName:                {Resource: "group", Action: "read", DomainType: "canvas"},
		pbAuth.Authorization_GetCanvasGroupUsers_FullMethodName:             {Resource: "group", Action: "read", DomainType: "canvas"},
		pbAuth.Authorization_GetCanvasGroup_FullMethodName:                  {Resource: "group", Action: "read", DomainType: "canvas"},

		// Organization rules
		pbAuth.Authorization_ListUserPermissions_FullMethodName:             {Resource: "user", Action: "read", DomainType: "org"},
		pbAuth.Authorization_AssignRole_FullMethodName:                      {Resource: "role", Action: "assign", DomainType: "org"},
		pbAuth.Authorization_RemoveRole_FullMethodName:                      {Resource: "role", Action: "remove", DomainType: "org"},
		pbAuth.Authorization_ListRoles_FullMethodName:                       {Resource: "role", Action: "read", DomainType: "org"},
		pbAuth.Authorization_DescribeRole_FullMethodName:                    {Resource: "role", Action: "read", DomainType: "org"},
		pbAuth.Authorization_GetUserRoles_FullMethodName:                    {Resource: "user", Action: "read", DomainType: "org"},
		pbAuth.Authorization_CreateRole_FullMethodName:                      {Resource: "role", Action: "create", DomainType: "org"},
		pbAuth.Authorization_UpdateRole_FullMethodName:                      {Resource: "role", Action: "update", DomainType: "org"},
		pbAuth.Authorization_DeleteRole_FullMethodName:                      {Resource: "role", Action: "delete", DomainType: "org"},
		pbAuth.Authorization_CreateOrganizationGroup_FullMethodName:         {Resource: "group", Action: "create", DomainType: "org"},
		pbAuth.Authorization_AddUserToOrganizationGroup_FullMethodName:      {Resource: "group", Action: "update", DomainType: "org"},
		pbAuth.Authorization_RemoveUserFromOrganizationGroup_FullMethodName: {Resource: "group", Action: "update", DomainType: "org"},
		pbAuth.Authorization_ListOrganizationGroups_FullMethodName:          {Resource: "group", Action: "read", DomainType: "org"},
		pbAuth.Authorization_GetOrganizationGroupUsers_FullMethodName:       {Resource: "group", Action: "read", DomainType: "org"},
		pbAuth.Authorization_GetOrganizationGroup_FullMethodName:            {Resource: "group", Action: "read", DomainType: "org"},
		pbOrganization.Organizations_DescribeOrganization_FullMethodName:    {Resource: "org", Action: "read", DomainType: "org"},
		pbOrganization.Organizations_ListOrganizations_FullMethodName:       {Resource: "org", Action: "read", DomainType: "org"},
		pbOrganization.Organizations_UpdateOrganization_FullMethodName:      {Resource: "org", Action: "update", DomainType: "org"},
		pbOrganization.Organizations_DeleteOrganization_FullMethodName:      {Resource: "org", Action: "delete", DomainType: "org"},
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

		domainID, err := a.extractDomainID(req, rule.DomainType)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid %s ID", rule.DomainType))
		}

		var allowed bool
		if rule.DomainType == "org" {
			allowed, err = a.authService.CheckOrganizationPermission(userID, domainID, rule.Resource, rule.Action)
			if err != nil {
				return nil, err
			}
		}
		if rule.DomainType == "canvas" {
			allowed, err = a.authService.CheckCanvasPermission(userID, domainID, rule.Resource, rule.Action)
			if err != nil {
				return nil, err
			}
		}

		if !allowed {
			log.Warnf("User %s tried to %s %s in %s %s", userID, rule.Action, rule.Resource, rule.DomainType, domainID)
			return nil, status.Error(codes.NotFound, "Not found")
		}

		return handler(ctx, req)
	}
}

func (a *AuthorizationInterceptor) extractDomainID(req interface{}, domainType string) (string, error) {
	if domainType == "org" {
		return extractOrganizationID(req)
	}

	if domainType == "canvas" {
		return extractCanvasID(req)
	}

	return "", status.Error(codes.Internal, "unsupported domain type")
}

func extractOrganizationID(req interface{}) (string, error) {
	var domainID string
	switch r := req.(type) {
	case interface{ GetOrganizationId() string }:
		domainID = r.GetOrganizationId()
	default:
		return "", nil
	}

	if _, err := uuid.Parse(domainID); err != nil {
		org, err := models.FindOrganizationByName(domainID)
		if err != nil {
			return "", nil
		}
		domainID = org.ID.String()
	}

	return domainID, nil
}

func extractCanvasID(req interface{}) (string, error) {
	switch r := req.(type) {
	case interface{ GetCanvasId() string }:
		return r.GetCanvasId(), nil
	case interface{ GetCanvasIdOrName() string }:
		canvasIDOrName := r.GetCanvasIdOrName()
		if _, err := uuid.Parse(canvasIDOrName); err != nil {
			canvas, err := models.FindCanvasByName(canvasIDOrName)
			if err != nil {
				return "", nil
			}
			return canvas.ID.String(), nil
		}
		return canvasIDOrName, nil
	default:
		return "", nil
	}
}
