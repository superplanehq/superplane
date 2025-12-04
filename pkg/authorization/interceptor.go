package authorization

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	pbBlueprints "github.com/superplanehq/superplane/pkg/protos/blueprints"
	pbGroups "github.com/superplanehq/superplane/pkg/protos/groups"
	pbIntegrations "github.com/superplanehq/superplane/pkg/protos/integrations"
	pbOrganization "github.com/superplanehq/superplane/pkg/protos/organizations"
	pbRoles "github.com/superplanehq/superplane/pkg/protos/roles"
	pbSecrets "github.com/superplanehq/superplane/pkg/protos/secrets"
	pbUsers "github.com/superplanehq/superplane/pkg/protos/users"
	pbWorkflows "github.com/superplanehq/superplane/pkg/protos/workflows"
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
		// Secrets rules
		pbSecrets.Secrets_CreateSecret_FullMethodName:   {Resource: "secret", Action: "create", DomainType: models.DomainTypeOrganization},
		pbSecrets.Secrets_UpdateSecret_FullMethodName:   {Resource: "secret", Action: "update", DomainType: models.DomainTypeOrganization},
		pbSecrets.Secrets_DescribeSecret_FullMethodName: {Resource: "secret", Action: "read", DomainType: models.DomainTypeOrganization},
		pbSecrets.Secrets_ListSecrets_FullMethodName:    {Resource: "secret", Action: "read", DomainType: models.DomainTypeOrganization},
		pbSecrets.Secrets_DeleteSecret_FullMethodName:   {Resource: "secret", Action: "delete", DomainType: models.DomainTypeOrganization},

		// Integrations rules
		pbIntegrations.Integrations_DescribeIntegration_FullMethodName: {Resource: "integration", Action: "read", DomainType: models.DomainTypeOrganization},
		pbIntegrations.Integrations_ListIntegrations_FullMethodName:    {Resource: "integration", Action: "read", DomainType: models.DomainTypeOrganization},
		pbIntegrations.Integrations_ListResources_FullMethodName:       {Resource: "integration", Action: "read", DomainType: models.DomainTypeOrganization},
		pbIntegrations.Integrations_CreateIntegration_FullMethodName:   {Resource: "integration", Action: "create", DomainType: models.DomainTypeOrganization},
		pbIntegrations.Integrations_UpdateIntegration_FullMethodName:   {Resource: "integration", Action: "update", DomainType: models.DomainTypeOrganization},

		// Groups rules
		pbGroups.Groups_CreateGroup_FullMethodName:         {Resource: "group", Action: "create", DomainType: models.DomainTypeOrganization},
		pbGroups.Groups_AddUserToGroup_FullMethodName:      {Resource: "group", Action: "update", DomainType: models.DomainTypeOrganization},
		pbGroups.Groups_RemoveUserFromGroup_FullMethodName: {Resource: "group", Action: "update", DomainType: models.DomainTypeOrganization},
		pbGroups.Groups_UpdateGroup_FullMethodName:         {Resource: "group", Action: "update", DomainType: models.DomainTypeOrganization},
		pbGroups.Groups_ListGroups_FullMethodName:          {Resource: "group", Action: "read", DomainType: models.DomainTypeOrganization},
		pbGroups.Groups_ListGroupUsers_FullMethodName:      {Resource: "group", Action: "read", DomainType: models.DomainTypeOrganization},
		pbGroups.Groups_DescribeGroup_FullMethodName:       {Resource: "group", Action: "read", DomainType: models.DomainTypeOrganization},
		pbGroups.Groups_DeleteGroup_FullMethodName:         {Resource: "group", Action: "delete", DomainType: models.DomainTypeOrganization},

		// Users rules
		pbUsers.Users_ListUserPermissions_FullMethodName: {Resource: "member", Action: "read", DomainType: models.DomainTypeOrganization},
		pbUsers.Users_ListUserRoles_FullMethodName:       {Resource: "member", Action: "read", DomainType: models.DomainTypeOrganization},
		pbUsers.Users_ListUsers_FullMethodName:           {Resource: "member", Action: "read", DomainType: models.DomainTypeOrganization},

		// Roles rules
		pbRoles.Roles_AssignRole_FullMethodName:   {Resource: "member", Action: "update", DomainType: models.DomainTypeOrganization},
		pbRoles.Roles_ListRoles_FullMethodName:    {Resource: "role", Action: "read", DomainType: models.DomainTypeOrganization},
		pbRoles.Roles_DescribeRole_FullMethodName: {Resource: "role", Action: "read", DomainType: models.DomainTypeOrganization},
		pbRoles.Roles_CreateRole_FullMethodName:   {Resource: "role", Action: "create", DomainType: models.DomainTypeOrganization},
		pbRoles.Roles_UpdateRole_FullMethodName:   {Resource: "role", Action: "update", DomainType: models.DomainTypeOrganization},
		pbRoles.Roles_DeleteRole_FullMethodName:   {Resource: "role", Action: "delete", DomainType: models.DomainTypeOrganization},

		// Organization Rules
		pbOrganization.Organizations_DescribeOrganization_FullMethodName: {Resource: "org", Action: "read", DomainType: models.DomainTypeOrganization},
		pbOrganization.Organizations_ListInvitations_FullMethodName:      {Resource: "org", Action: "read", DomainType: models.DomainTypeOrganization},
		pbOrganization.Organizations_RemoveInvitation_FullMethodName:     {Resource: "member", Action: "delete", DomainType: models.DomainTypeOrganization},
		pbOrganization.Organizations_UpdateOrganization_FullMethodName:   {Resource: "org", Action: "update", DomainType: models.DomainTypeOrganization},
		pbOrganization.Organizations_CreateInvitation_FullMethodName:     {Resource: "member", Action: "create", DomainType: models.DomainTypeOrganization},
		pbOrganization.Organizations_RemoveUser_FullMethodName:           {Resource: "member", Action: "delete", DomainType: models.DomainTypeOrganization},
		pbOrganization.Organizations_DeleteOrganization_FullMethodName:   {Resource: "org", Action: "delete", DomainType: models.DomainTypeOrganization},

		// Blueprints rules
		pbBlueprints.Blueprints_ListBlueprints_FullMethodName:    {Resource: "blueprint", Action: "read", DomainType: models.DomainTypeOrganization},
		pbBlueprints.Blueprints_DescribeBlueprint_FullMethodName: {Resource: "blueprint", Action: "read", DomainType: models.DomainTypeOrganization},
		pbBlueprints.Blueprints_CreateBlueprint_FullMethodName:   {Resource: "blueprint", Action: "create", DomainType: models.DomainTypeOrganization},
		pbBlueprints.Blueprints_UpdateBlueprint_FullMethodName:   {Resource: "blueprint", Action: "update", DomainType: models.DomainTypeOrganization},
		pbBlueprints.Blueprints_DeleteBlueprint_FullMethodName:   {Resource: "blueprint", Action: "delete", DomainType: models.DomainTypeOrganization},

		// Workflows rules
		pbWorkflows.Workflows_ListWorkflows_FullMethodName:             {Resource: "workflow", Action: "read", DomainType: models.DomainTypeOrganization},
		pbWorkflows.Workflows_DescribeWorkflow_FullMethodName:          {Resource: "workflow", Action: "read", DomainType: models.DomainTypeOrganization},
		pbWorkflows.Workflows_CreateWorkflow_FullMethodName:            {Resource: "workflow", Action: "create", DomainType: models.DomainTypeOrganization},
		pbWorkflows.Workflows_UpdateWorkflow_FullMethodName:            {Resource: "workflow", Action: "update", DomainType: models.DomainTypeOrganization},
		pbWorkflows.Workflows_DeleteWorkflow_FullMethodName:            {Resource: "workflow", Action: "delete", DomainType: models.DomainTypeOrganization},
		pbWorkflows.Workflows_ListNodeExecutions_FullMethodName:        {Resource: "workflow", Action: "read", DomainType: models.DomainTypeOrganization},
		pbWorkflows.Workflows_ListNodeQueueItems_FullMethodName:        {Resource: "workflow", Action: "read", DomainType: models.DomainTypeOrganization},
		pbWorkflows.Workflows_DeleteNodeQueueItem_FullMethodName:       {Resource: "workflow", Action: "update", DomainType: models.DomainTypeOrganization},
		pbWorkflows.Workflows_ListWorkflowEvents_FullMethodName:        {Resource: "workflow", Action: "read", DomainType: models.DomainTypeOrganization},
		pbWorkflows.Workflows_ListEventExecutions_FullMethodName:       {Resource: "workflow", Action: "read", DomainType: models.DomainTypeOrganization},
		pbWorkflows.Workflows_ListChildExecutions_FullMethodName:       {Resource: "workflow", Action: "read", DomainType: models.DomainTypeOrganization},
		pbWorkflows.Workflows_CancelExecution_FullMethodName:           {Resource: "workflow", Action: "update", DomainType: models.DomainTypeOrganization},
		pbWorkflows.Workflows_InvokeNodeExecutionAction_FullMethodName: {Resource: "workflow", Action: "update", DomainType: models.DomainTypeOrganization},
		pbWorkflows.Workflows_ListNodeEvents_FullMethodName:            {Resource: "workflow", Action: "read", DomainType: models.DomainTypeOrganization},
		pbWorkflows.Workflows_EmitNodeEvent_FullMethodName:             {Resource: "workflow", Action: "update", DomainType: models.DomainTypeOrganization},
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
		org, err := models.FindOrganizationByID(organizationID)
		if err != nil {
			return nil, status.Error(codes.NotFound, "organization not found")
		}

		allowed, err := a.authService.CheckOrganizationPermission(userID, org.ID.String(), rule.Resource, rule.Action)
		if err != nil {
			return nil, err
		}

		if !allowed {
			log.Warnf("User %s tried to %s %s in organization %s", userID, rule.Action, rule.Resource, org.ID.String())
			return nil, status.Error(codes.NotFound, "Not found")
		}

		newContext := context.WithValue(ctx, OrganizationContextKey, organizationID)
		newContext = context.WithValue(newContext, DomainTypeContextKey, models.DomainTypeOrganization)
		newContext = context.WithValue(newContext, DomainIdContextKey, organizationID)
		return handler(newContext, req)
	}
}
