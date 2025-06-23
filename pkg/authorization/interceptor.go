package authorization

import (
	"context"
	"fmt"

	"github.com/superplanehq/superplane/pkg/authentication"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
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
		"/Superplane.Superplane/CreateCanvas":        {Resource: "canvas", Action: "create", DomainType: "org"},
		"/Superplane.Superplane/DescribeCanvas":      {Resource: "canvas", Action: "read", DomainType: "org"},
		"/Superplane.Superplane/ListCanvases":        {Resource: "canvas", Action: "read", DomainType: "org"},
		"/Superplane.Superplane/CreateEventSource":   {Resource: "eventsource", Action: "create", DomainType: "canvas"},
		"/Superplane.Superplane/DescribeEventSource": {Resource: "eventsource", Action: "read", DomainType: "canvas"},
		"/Superplane.Superplane/ListEventSources":    {Resource: "eventsource", Action: "read", DomainType: "canvas"},
		"/Superplane.Superplane/CreateStage":         {Resource: "stage", Action: "create", DomainType: "canvas"},
		"/Superplane.Superplane/DescribeStage":       {Resource: "stage", Action: "read", DomainType: "canvas"},
		"/Superplane.Superplane/UpdateStage":         {Resource: "stage", Action: "update", DomainType: "canvas"},
		"/Superplane.Superplane/ListStages":          {Resource: "stage", Action: "read", DomainType: "canvas"},
		"/Superplane.Superplane/CreateSecret":        {Resource: "secret", Action: "create", DomainType: "canvas"},
		"/Superplane.Superplane/UpdateSecret":        {Resource: "secret", Action: "update", DomainType: "canvas"},
		"/Superplane.Superplane/DescribeSecret":      {Resource: "secret", Action: "read", DomainType: "canvas"},
		"/Superplane.Superplane/ListSecrets":         {Resource: "secret", Action: "read", DomainType: "canvas"},
		"/Superplane.Superplane/DeleteSecret":        {Resource: "secret", Action: "delete", DomainType: "canvas"},
		"/Superplane.Superplane/ApproveStageEvent":   {Resource: "stageevent", Action: "approve", DomainType: "canvas"},
		"/Superplane.Superplane/ListStageEvents":     {Resource: "stageevent", Action: "read", DomainType: "canvas"},

		// Organization rules
		"/Superplane.Organizations.Organizations/DescribeOrganization": {Resource: "org", Action: "read", DomainType: "org"},
		"/Superplane.Organizations.Organizations/UpdateOrganization":   {Resource: "org", Action: "update", DomainType: "org"},
		"/Superplane.Organizations.Organizations/DeleteOrganization":   {Resource: "org", Action: "delete", DomainType: "org"},

		// Authorization rules
		"/Superplane.Authorization.Authorization/ListUserPermissions":    {Resource: "user", Action: "read", DomainType: "org"},
		"/Superplane.Authorization.Authorization/AssignRole":             {Resource: "role", Action: "assign", DomainType: "org"},
		"/Superplane.Authorization.Authorization/RemoveRole":             {Resource: "role", Action: "remove", DomainType: "org"},
		"/Superplane.Authorization.Authorization/ListRoles":              {Resource: "role", Action: "read", DomainType: "org"},
		"/Superplane.Authorization.Authorization/DescribeRole":           {Resource: "role", Action: "read", DomainType: "org"},
		"/Superplane.Authorization.Authorization/GetUserRoles":           {Resource: "user", Action: "read", DomainType: "org"},
		"/Superplane.Authorization.Authorization/CreateGroup":            {Resource: "group", Action: "create", DomainType: "org"},
		"/Superplane.Authorization.Authorization/AddUserToGroup":         {Resource: "group", Action: "update", DomainType: "org"},
		"/Superplane.Authorization.Authorization/RemoveUserFromGroup":    {Resource: "group", Action: "update", DomainType: "org"},
		"/Superplane.Authorization.Authorization/ListOrganizationGroups": {Resource: "group", Action: "read", DomainType: "org"},
		"/Superplane.Authorization.Authorization/GetGroupUsers":          {Resource: "group", Action: "read", DomainType: "org"},
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

		user, userIsSet := authentication.GetUserFromContext(ctx)
		if !userIsSet {
			return nil, status.Error(codes.Unauthenticated, "user not authenticated")
		}

		domainID, err := a.extractDomainID(req, rule.DomainType)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid %s ID", rule.DomainType))
		}

		var allowed bool
		if rule.DomainType == "org" {
			allowed, err = a.authService.CheckOrganizationPermission(user.ID.String(), domainID, rule.Resource, rule.Action)
			if err != nil {
				return nil, err
			}
		}
		if rule.DomainType == "canvas" {
			allowed, err = a.authService.CheckCanvasPermission(user.ID.String(), domainID, rule.Resource, rule.Action)
			if err != nil {
				return nil, err
			}
		}

		if !allowed {
			return nil, status.Error(codes.PermissionDenied, "insufficient permissions")
		}

		return handler(ctx, req)
	}
}

func (a *AuthorizationInterceptor) extractDomainID(req interface{}, domainType string) (string, error) {
	if domainType == "org" {
		switch r := req.(type) {
		case interface{ GetOrganizationId() string }:
			return r.GetOrganizationId(), nil
		default:
			return "", status.Error(codes.Internal, "unable to extract organization ID")
		}
	}

	if domainType == "canvas" {
		switch r := req.(type) {
		case interface{ GetCanvasId() string }:
			return r.GetCanvasId(), nil
		case interface{ GetCanvasIdOrName() string }:
			return r.GetCanvasIdOrName(), nil
		default:
			return "", status.Error(codes.Internal, "unable to extract canvas ID")
		}
	}

	return "", status.Error(codes.Internal, "unsupported domain type")
}
