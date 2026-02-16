package authorization

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	pbBlueprints "github.com/superplanehq/superplane/pkg/protos/blueprints"
	pbCanvases "github.com/superplanehq/superplane/pkg/protos/canvases"
	pbGroups "github.com/superplanehq/superplane/pkg/protos/groups"
	pbOrganization "github.com/superplanehq/superplane/pkg/protos/organizations"
	pbRoles "github.com/superplanehq/superplane/pkg/protos/roles"
	pbSecrets "github.com/superplanehq/superplane/pkg/protos/secrets"
	pbServiceAccounts "github.com/superplanehq/superplane/pkg/protos/service_accounts"
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
		pbSecrets.Secrets_CreateSecret_FullMethodName:     {Resource: "secrets", Action: "create", DomainType: models.DomainTypeOrganization},
		pbSecrets.Secrets_UpdateSecret_FullMethodName:     {Resource: "secrets", Action: "update", DomainType: models.DomainTypeOrganization},
		pbSecrets.Secrets_DescribeSecret_FullMethodName:   {Resource: "secrets", Action: "read", DomainType: models.DomainTypeOrganization},
		pbSecrets.Secrets_ListSecrets_FullMethodName:      {Resource: "secrets", Action: "read", DomainType: models.DomainTypeOrganization},
		pbSecrets.Secrets_DeleteSecret_FullMethodName:     {Resource: "secrets", Action: "delete", DomainType: models.DomainTypeOrganization},
		pbSecrets.Secrets_SetSecretKey_FullMethodName:     {Resource: "secrets", Action: "update", DomainType: models.DomainTypeOrganization},
		pbSecrets.Secrets_DeleteSecretKey_FullMethodName:  {Resource: "secrets", Action: "update", DomainType: models.DomainTypeOrganization},
		pbSecrets.Secrets_UpdateSecretName_FullMethodName: {Resource: "secrets", Action: "update", DomainType: models.DomainTypeOrganization},

		// Groups rules
		pbGroups.Groups_CreateGroup_FullMethodName:         {Resource: "groups", Action: "create", DomainType: models.DomainTypeOrganization},
		pbGroups.Groups_AddUserToGroup_FullMethodName:      {Resource: "groups", Action: "update", DomainType: models.DomainTypeOrganization},
		pbGroups.Groups_RemoveUserFromGroup_FullMethodName: {Resource: "groups", Action: "update", DomainType: models.DomainTypeOrganization},
		pbGroups.Groups_UpdateGroup_FullMethodName:         {Resource: "groups", Action: "update", DomainType: models.DomainTypeOrganization},
		pbGroups.Groups_ListGroups_FullMethodName:          {Resource: "groups", Action: "read", DomainType: models.DomainTypeOrganization},
		pbGroups.Groups_ListGroupUsers_FullMethodName:      {Resource: "groups", Action: "read", DomainType: models.DomainTypeOrganization},
		pbGroups.Groups_DescribeGroup_FullMethodName:       {Resource: "groups", Action: "read", DomainType: models.DomainTypeOrganization},
		pbGroups.Groups_DeleteGroup_FullMethodName:         {Resource: "groups", Action: "delete", DomainType: models.DomainTypeOrganization},

		// Users rules
		pbUsers.Users_ListUserPermissions_FullMethodName: {Resource: "members", Action: "read", DomainType: models.DomainTypeOrganization},
		pbUsers.Users_ListUserRoles_FullMethodName:       {Resource: "members", Action: "read", DomainType: models.DomainTypeOrganization},
		pbUsers.Users_ListUsers_FullMethodName:           {Resource: "members", Action: "read", DomainType: models.DomainTypeOrganization},

		// Roles rules
		pbRoles.Roles_AssignRole_FullMethodName:   {Resource: "members", Action: "update", DomainType: models.DomainTypeOrganization},
		pbRoles.Roles_ListRoles_FullMethodName:    {Resource: "roles", Action: "read", DomainType: models.DomainTypeOrganization},
		pbRoles.Roles_DescribeRole_FullMethodName: {Resource: "roles", Action: "read", DomainType: models.DomainTypeOrganization},
		pbRoles.Roles_CreateRole_FullMethodName:   {Resource: "roles", Action: "create", DomainType: models.DomainTypeOrganization},
		pbRoles.Roles_UpdateRole_FullMethodName:   {Resource: "roles", Action: "update", DomainType: models.DomainTypeOrganization},
		pbRoles.Roles_DeleteRole_FullMethodName:   {Resource: "roles", Action: "delete", DomainType: models.DomainTypeOrganization},

		// Organization Rules
		pbOrganization.Organizations_DescribeOrganization_FullMethodName:     {Resource: "org", Action: "read", DomainType: models.DomainTypeOrganization},
		pbOrganization.Organizations_ListInvitations_FullMethodName:          {Resource: "members", Action: "read", DomainType: models.DomainTypeOrganization},
		pbOrganization.Organizations_RemoveInvitation_FullMethodName:         {Resource: "members", Action: "delete", DomainType: models.DomainTypeOrganization},
		pbOrganization.Organizations_UpdateOrganization_FullMethodName:       {Resource: "org", Action: "update", DomainType: models.DomainTypeOrganization},
		pbOrganization.Organizations_CreateInvitation_FullMethodName:         {Resource: "members", Action: "create", DomainType: models.DomainTypeOrganization},
		pbOrganization.Organizations_GetInviteLink_FullMethodName:            {Resource: "members", Action: "create", DomainType: models.DomainTypeOrganization},
		pbOrganization.Organizations_UpdateInviteLink_FullMethodName:         {Resource: "members", Action: "create", DomainType: models.DomainTypeOrganization},
		pbOrganization.Organizations_ResetInviteLink_FullMethodName:          {Resource: "members", Action: "create", DomainType: models.DomainTypeOrganization},
		pbOrganization.Organizations_RemoveUser_FullMethodName:               {Resource: "members", Action: "delete", DomainType: models.DomainTypeOrganization},
		pbOrganization.Organizations_DeleteOrganization_FullMethodName:       {Resource: "org", Action: "delete", DomainType: models.DomainTypeOrganization},
		pbOrganization.Organizations_CreateIntegration_FullMethodName:        {Resource: "integrations", Action: "create", DomainType: models.DomainTypeOrganization},
		pbOrganization.Organizations_UpdateIntegration_FullMethodName:        {Resource: "integrations", Action: "update", DomainType: models.DomainTypeOrganization},
		pbOrganization.Organizations_DeleteIntegration_FullMethodName:        {Resource: "integrations", Action: "delete", DomainType: models.DomainTypeOrganization},
		pbOrganization.Organizations_ListIntegrations_FullMethodName:         {Resource: "integrations", Action: "read", DomainType: models.DomainTypeOrganization},
		pbOrganization.Organizations_DescribeIntegration_FullMethodName:      {Resource: "integrations", Action: "read", DomainType: models.DomainTypeOrganization},
		pbOrganization.Organizations_ListIntegrationResources_FullMethodName: {Resource: "integrations", Action: "read", DomainType: models.DomainTypeOrganization},

		// Blueprints rules
		pbBlueprints.Blueprints_ListBlueprints_FullMethodName:    {Resource: "blueprints", Action: "read", DomainType: models.DomainTypeOrganization},
		pbBlueprints.Blueprints_DescribeBlueprint_FullMethodName: {Resource: "blueprints", Action: "read", DomainType: models.DomainTypeOrganization},
		pbBlueprints.Blueprints_CreateBlueprint_FullMethodName:   {Resource: "blueprints", Action: "create", DomainType: models.DomainTypeOrganization},
		pbBlueprints.Blueprints_UpdateBlueprint_FullMethodName:   {Resource: "blueprints", Action: "update", DomainType: models.DomainTypeOrganization},
		pbBlueprints.Blueprints_DeleteBlueprint_FullMethodName:   {Resource: "blueprints", Action: "delete", DomainType: models.DomainTypeOrganization},

		// Canvases rules
		pbCanvases.Canvases_ListCanvases_FullMethodName:              {Resource: "canvases", Action: "read", DomainType: models.DomainTypeOrganization},
		pbCanvases.Canvases_DescribeCanvas_FullMethodName:            {Resource: "canvases", Action: "read", DomainType: models.DomainTypeOrganization},
		pbCanvases.Canvases_CreateCanvas_FullMethodName:              {Resource: "canvases", Action: "create", DomainType: models.DomainTypeOrganization},
		pbCanvases.Canvases_UpdateCanvas_FullMethodName:              {Resource: "canvases", Action: "update", DomainType: models.DomainTypeOrganization},
		pbCanvases.Canvases_DeleteCanvas_FullMethodName:              {Resource: "canvases", Action: "delete", DomainType: models.DomainTypeOrganization},
		pbCanvases.Canvases_ListNodeExecutions_FullMethodName:        {Resource: "canvases", Action: "read", DomainType: models.DomainTypeOrganization},
		pbCanvases.Canvases_ListNodeQueueItems_FullMethodName:        {Resource: "canvases", Action: "read", DomainType: models.DomainTypeOrganization},
		pbCanvases.Canvases_DeleteNodeQueueItem_FullMethodName:       {Resource: "canvases", Action: "update", DomainType: models.DomainTypeOrganization},
		pbCanvases.Canvases_UpdateNodePause_FullMethodName:           {Resource: "canvases", Action: "update", DomainType: models.DomainTypeOrganization},
		pbCanvases.Canvases_ListCanvasEvents_FullMethodName:          {Resource: "canvases", Action: "read", DomainType: models.DomainTypeOrganization},
		pbCanvases.Canvases_ListEventExecutions_FullMethodName:       {Resource: "canvases", Action: "read", DomainType: models.DomainTypeOrganization},
		pbCanvases.Canvases_ListChildExecutions_FullMethodName:       {Resource: "canvases", Action: "read", DomainType: models.DomainTypeOrganization},
		pbCanvases.Canvases_CancelExecution_FullMethodName:           {Resource: "canvases", Action: "update", DomainType: models.DomainTypeOrganization},
		pbCanvases.Canvases_ResolveExecutionErrors_FullMethodName:    {Resource: "canvases", Action: "update", DomainType: models.DomainTypeOrganization},
		pbCanvases.Canvases_InvokeNodeExecutionAction_FullMethodName: {Resource: "canvases", Action: "update", DomainType: models.DomainTypeOrganization},
		pbCanvases.Canvases_InvokeNodeTriggerAction_FullMethodName:   {Resource: "canvases", Action: "update", DomainType: models.DomainTypeOrganization},
		pbCanvases.Canvases_ListNodeEvents_FullMethodName:            {Resource: "canvases", Action: "read", DomainType: models.DomainTypeOrganization},
		pbCanvases.Canvases_EmitNodeEvent_FullMethodName:             {Resource: "canvases", Action: "update", DomainType: models.DomainTypeOrganization},

		// Service Accounts rules
		pbServiceAccounts.ServiceAccounts_CreateServiceAccount_FullMethodName:          {Resource: "service_accounts", Action: "create", DomainType: models.DomainTypeOrganization},
		pbServiceAccounts.ServiceAccounts_ListServiceAccounts_FullMethodName:           {Resource: "service_accounts", Action: "read", DomainType: models.DomainTypeOrganization},
		pbServiceAccounts.ServiceAccounts_DescribeServiceAccount_FullMethodName:        {Resource: "service_accounts", Action: "read", DomainType: models.DomainTypeOrganization},
		pbServiceAccounts.ServiceAccounts_UpdateServiceAccount_FullMethodName:          {Resource: "service_accounts", Action: "update", DomainType: models.DomainTypeOrganization},
		pbServiceAccounts.ServiceAccounts_DeleteServiceAccount_FullMethodName:          {Resource: "service_accounts", Action: "delete", DomainType: models.DomainTypeOrganization},
		pbServiceAccounts.ServiceAccounts_RegenerateServiceAccountToken_FullMethodName: {Resource: "service_accounts", Action: "update", DomainType: models.DomainTypeOrganization},
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
