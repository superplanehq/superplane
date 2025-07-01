package authorization

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/casbin/casbin/v2"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
)

const (
	OrgIDTemplate    = "{ORG_ID}"
	CanvasIDTemplate = "{CANVAS_ID}"

	RoleOrgOwner  = "org_owner"
	RoleOrgAdmin  = "org_admin"
	RoleOrgViewer = "org_viewer"

	RoleCanvasOwner  = "canvas_owner"
	RoleCanvasAdmin  = "canvas_admin"
	RoleCanvasViewer = "canvas_viewer"

	DomainCanvas = "canvas"
	DomainOrg    = "org"
)

// implements Authorization
var _ Authorization = (*AuthService)(nil)

type AuthService struct {
	enforcer              *casbin.CachedEnforcer
	db                    *gorm.DB
	orgPolicyTemplates    [][5]string
	canvasPolicyTemplates [][5]string
}

func NewAuthService() (*AuthService, error) {
	modelPath := os.Getenv("RBAC_MODEL_PATH")
	orgPolicyPath := os.Getenv("RBAC_ORG_POLICY_PATH")
	canvasPolicyPath := os.Getenv("RBAC_CANVAS_POLICY_PATH")

	adapter, err := gormadapter.NewAdapterByDB(database.Conn())
	if err != nil {
		return nil, fmt.Errorf("failed to create casbin adapter: %w", err)
	}

	enforcer, err := casbin.NewCachedEnforcer(modelPath, adapter)
	if err != nil {
		return nil, fmt.Errorf("failed to create casbin enforcer: %w", err)
	}

	enforcer.EnableAutoSave(true)

	if err := enforcer.LoadPolicy(); err != nil {
		return nil, fmt.Errorf("failed to load policies: %w", err)
	}

	orgPoliciesCsv, err := os.ReadFile(orgPolicyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read org policies: %w", err)
	}
	canvasPoliciesCsv, err := os.ReadFile(canvasPolicyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read canvas policies: %w", err)
	}

	orgPolicyTemplates, err := parsePoliciesFromCsv(orgPoliciesCsv)
	if err != nil {
		return nil, fmt.Errorf("failed to parse org policies: %w", err)
	}

	canvasPolicyTemplates, err := parsePoliciesFromCsv(canvasPoliciesCsv)
	if err != nil {
		return nil, fmt.Errorf("failed to parse canvas policies: %w", err)
	}

	service := &AuthService{
		enforcer:              enforcer,
		db:                    database.Conn(),
		orgPolicyTemplates:    orgPolicyTemplates,
		canvasPolicyTemplates: canvasPolicyTemplates,
	}

	return service, nil
}

func (a *AuthService) CheckCanvasPermission(userID, canvasID, resource, action string) (bool, error) {
	return a.checkPermission(userID, canvasID, DomainCanvas, resource, action)
}

func (a *AuthService) CheckOrganizationPermission(userID, orgID, resource, action string) (bool, error) {
	return a.checkPermission(userID, orgID, DomainOrg, resource, action)
}

func (a *AuthService) checkPermission(userID, domainID, domainType, resource, action string) (bool, error) {
	domain := fmt.Sprintf("%s:%s", domainType, domainID)
	prefixedUserID := fmt.Sprintf("user:%s", userID)
	return a.enforcer.Enforce(prefixedUserID, domain, resource, action)
}

func (a *AuthService) CreateGroup(orgID string, groupName string, role string) error {
	validRoles := []string{RoleOrgViewer, RoleOrgAdmin, RoleOrgOwner}
	if !contains(validRoles, role) {
		return fmt.Errorf("invalid role %s for organization", role)
	}

	domain := fmt.Sprintf("org:%s", orgID)
	prefixedGroupName := fmt.Sprintf("group:%s", groupName)
	prefixedRole := fmt.Sprintf("role:%s", role)

	ruleAdded, err := a.enforcer.AddGroupingPolicy(prefixedGroupName, prefixedRole, domain)
	if err != nil {
		return fmt.Errorf("failed to create group: %w", err)
	}

	if !ruleAdded {
		return fmt.Errorf("group %s already exists with role %s in organization %s", groupName, role, orgID)
	}

	log.Infof("Created group %s with role %s in organization %s", groupName, role, orgID)
	return nil
}

func (a *AuthService) AddUserToGroup(orgID string, userID string, group string) error {
	domain := fmt.Sprintf("org:%s", orgID)
	prefixedGroupName := fmt.Sprintf("group:%s", group)
	prefixedUserID := fmt.Sprintf("user:%s", userID)

	groups, err := a.enforcer.GetFilteredGroupingPolicy(0, prefixedGroupName, "", domain)
	if err != nil {
		return fmt.Errorf("failed to check group existence: %w", err)
	}

	groupExists := false
	for _, g := range groups {
		if g[2] == domain {
			groupExists = true
			break
		}
	}

	if !groupExists {
		return fmt.Errorf("group %s does not exist in organization %s", group, orgID)
	}

	ruleAdded, err := a.enforcer.AddGroupingPolicy(prefixedUserID, prefixedGroupName, domain)
	if err != nil {
		return fmt.Errorf("failed to add user to group: %w", err)
	}

	if !ruleAdded {
		log.Infof("user %s is already a member of group %s", userID, group)
	}

	return nil
}

func (a *AuthService) RemoveUserFromGroup(orgID string, userID string, group string) error {
	domain := fmt.Sprintf("org:%s", orgID)
	prefixedGroupName := fmt.Sprintf("group:%s", group)
	prefixedUserID := fmt.Sprintf("user:%s", userID)

	ruleRemoved, err := a.enforcer.RemoveGroupingPolicy(prefixedUserID, prefixedGroupName, domain)
	if err != nil {
		return fmt.Errorf("failed to remove user from group: %w", err)
	}

	if !ruleRemoved {
		return fmt.Errorf("user %s is not a member of group %s", userID, group)
	}

	return nil
}

func (a *AuthService) GetGroupUsers(orgID string, group string) ([]string, error) {
	domain := fmt.Sprintf("org:%s", orgID)
	prefixedGroupName := fmt.Sprintf("group:%s", group)

	policies, err := a.enforcer.GetFilteredGroupingPolicy(1, prefixedGroupName, domain)
	if err != nil {
		return nil, fmt.Errorf("failed to get group users: %w", err)
	}

	var users []string
	for _, policy := range policies {
		unprefixedUserID := strings.TrimPrefix(policy[0], "user:")
		users = append(users, unprefixedUserID)
	}

	return users, nil
}

func (a *AuthService) GetGroups(orgID string) ([]string, error) {
	domain := fmt.Sprintf("org:%s", orgID)
	policies, err := a.enforcer.GetFilteredGroupingPolicy(2, domain)
	if err != nil {
		return nil, fmt.Errorf("failed to get groups: %w", err)
	}

	groupMap := make(map[string]bool)

	for _, policy := range policies {
		if strings.HasPrefix(policy[0], "group:") {
			groupName := policy[0][len("group:"):]
			groupMap[groupName] = true
		}
	}

	groups := make([]string, 0, len(groupMap))
	for group := range groupMap {
		groups = append(groups, group)
	}

	return groups, nil
}

func (a *AuthService) GetGroupRoles(orgID string, group string) ([]string, error) {
	domain := fmt.Sprintf("org:%s", orgID)
	prefixedGroupName := fmt.Sprintf("group:%s", group)
	roles := a.enforcer.GetRolesForUserInDomain(prefixedGroupName, domain)
	unprefixedRoles := []string{}
	for _, role := range roles {
		if strings.HasPrefix(role, "role:") {
			unprefixedRoles = append(unprefixedRoles, strings.TrimPrefix(role, "role:"))
		}
	}
	return unprefixedRoles, nil
}

func (a *AuthService) AssignRole(userID, role, domainID string, domainType string) error {
	domain := fmt.Sprintf("%s:%s", domainType, domainID)
	prefixedRole := fmt.Sprintf("role:%s", role)

	// Check if it's a default role
	validRoles := map[string][]string{
		DomainOrg:    {RoleOrgViewer, RoleOrgAdmin, RoleOrgOwner},
		DomainCanvas: {RoleCanvasViewer, RoleCanvasAdmin, RoleCanvasOwner},
	}

	isValidDefaultRole := false
	if roles, exists := validRoles[domainType]; exists {
		isValidDefaultRole = contains(roles, role)
	}

	// If not a default role, check if it's a custom role that exists
	if !isValidDefaultRole {
		policies, _ := a.enforcer.GetFilteredPolicy(0, prefixedRole, domain)
		if len(policies) == 0 {
			return fmt.Errorf("invalid role %s for domain type %s", role, domainType)
		}
	}

	prefixedUserID := fmt.Sprintf("user:%s", userID)
	ruleAdded, err := a.enforcer.AddGroupingPolicy(prefixedUserID, prefixedRole, domain)
	if err != nil {
		return fmt.Errorf("failed to add role: %w", err)
	}

	if !ruleAdded {
		log.Infof("role %s already exists for user %s", role, userID)
	}

	return nil
}

func (a *AuthService) RemoveRole(userID, role, domainID string, domainType string) error {
	domain := fmt.Sprintf("%s:%s", domainType, domainID)
	prefixedRole := fmt.Sprintf("role:%s", role)
	prefixedUserID := fmt.Sprintf("user:%s", userID)
	ruleRemoved, err := a.enforcer.RemoveGroupingPolicy(prefixedUserID, prefixedRole, domain)
	if err != nil {
		return fmt.Errorf("failed to remove role: %w", err)
	}
	if !ruleRemoved {
		log.Infof("role %s not found for user %s", role, userID)
	}
	return nil
}

func (a *AuthService) GetOrgUsersForRole(role string, orgID string) ([]string, error) {
	prefixedRole := fmt.Sprintf("role:%s", role)
	orgDomain := fmt.Sprintf("org:%s", orgID)
	users, err := a.enforcer.GetUsersForRole(prefixedRole, orgDomain)
	if err != nil {
		return nil, err
	}

	unprefixedUsers := []string{}
	for _, user := range users {
		if strings.HasPrefix(user, "user:") {
			unprefixedUsers = append(unprefixedUsers, strings.TrimPrefix(user, "user:"))
		}
	}
	return unprefixedUsers, nil
}

func (a *AuthService) GetCanvasUsersForRole(role string, canvasID string) ([]string, error) {
	prefixedRole := fmt.Sprintf("role:%s", role)
	canvasDomain := fmt.Sprintf("canvas:%s", canvasID)
	users, err := a.enforcer.GetUsersForRole(prefixedRole, canvasDomain)
	if err != nil {
		return nil, err
	}

	unprefixedUsers := []string{}
	for _, user := range users {
		if strings.HasPrefix(user, "user:") {
			unprefixedUsers = append(unprefixedUsers, strings.TrimPrefix(user, "user:"))
		}
	}
	return unprefixedUsers, nil
}

func (a *AuthService) SetupOrganizationRoles(orgID string) error {
	domain := fmt.Sprintf("org:%s", orgID)

	for _, policy := range a.orgPolicyTemplates {
		if policy[0] == "g" {
			// g,lower_role,higher_role,org:{ORG_ID}
			a.enforcer.AddGroupingPolicy(policy[1], policy[2], domain)
		} else if policy[0] == "p" {
			// p,role,org:{ORG_ID},resource,action
			a.enforcer.AddPolicy(policy[1], domain, policy[3], policy[4])
		} else {
			return fmt.Errorf("unknown policy type: %s", policy[0])
		}
	}

	return nil
}

func (a *AuthService) DestroyOrganizationRoles(orgID string) error {
	domain := fmt.Sprintf("org:%s", orgID)
	ok, err := a.enforcer.RemoveFilteredGroupingPolicy(2, domain)

	if err != nil {
		return fmt.Errorf("failed to remove organization roles: %w", err)
	}
	if !ok {
		return fmt.Errorf("organization roles not found for %s", orgID)
	}

	ok, err = a.enforcer.RemoveFilteredPolicy(1, domain)
	if err != nil {
		return fmt.Errorf("failed to remove organization policies: %w", err)
	}
	if !ok {
		return fmt.Errorf("organization policies not found for %s", orgID)
	}

	return nil
}

func (a *AuthService) GetAccessibleOrgsForUser(userID string) ([]string, error) {
	prefixedUserID := fmt.Sprintf("user:%s", userID)
	orgs, err := a.enforcer.GetFilteredGroupingPolicy(0, prefixedUserID)
	if err != nil {
		return nil, err
	}

	orgIDs := []string{}
	prefixLen := len("org:")
	for _, org := range orgs {
		if strings.HasPrefix(org[2], "org:") {
			orgIDs = append(orgIDs, org[2][prefixLen:])
		}
	}
	return orgIDs, nil
}

func (a *AuthService) GetAccessibleCanvasesForUser(userID string) ([]string, error) {
	prefixedUserID := fmt.Sprintf("user:%s", userID)
	canvases, err := a.enforcer.GetFilteredGroupingPolicy(0, prefixedUserID)
	if err != nil {
		return nil, err
	}

	canvasIDs := []string{}
	prefixLen := len("canvas:")
	for _, canvas := range canvases {
		if strings.HasPrefix(canvas[2], "canvas:") {
			canvasIDs = append(canvasIDs, canvas[2][prefixLen:])
		}
	}
	return canvasIDs, nil
}

func (a *AuthService) GetUserRolesForOrg(userID string, orgID string) ([]*RoleDefinition, error) {
	orgDomain := fmt.Sprintf("org:%s", orgID)
	prefixedUserID := fmt.Sprintf("user:%s", userID)
	roleNames, err := a.enforcer.GetImplicitRolesForUser(prefixedUserID, orgDomain)
	if err != nil {
		return nil, err
	}

	unprefixedRoleNames := []string{}
	for _, roleName := range roleNames {
		if strings.HasPrefix(roleName, "role:") {
			unprefixedRoleNames = append(unprefixedRoleNames, strings.TrimPrefix(roleName, "role:"))
		}
	}

	roles := []*RoleDefinition{}
	for _, roleName := range unprefixedRoleNames {
		roleDef, err := a.GetRoleDefinition(roleName, DomainOrg, orgID)
		if err != nil {
			continue
		}
		roles = append(roles, roleDef)
	}

	return roles, nil
}

func (a *AuthService) GetUserRolesForCanvas(userID string, canvasID string) ([]*RoleDefinition, error) {
	canvasDomain := fmt.Sprintf("canvas:%s", canvasID)
	prefixedUserID := fmt.Sprintf("user:%s", userID)
	roleNames, err := a.enforcer.GetImplicitRolesForUser(prefixedUserID, canvasDomain)
	if err != nil {
		return nil, err
	}

	unprefixedRoleNames := []string{}
	for _, roleName := range roleNames {
		if strings.HasPrefix(roleName, "role:") {
			unprefixedRoleNames = append(unprefixedRoleNames, strings.TrimPrefix(roleName, "role:"))
		}
	}

	roles := []*RoleDefinition{}
	for _, roleName := range unprefixedRoleNames {
		roleDef, err := a.GetRoleDefinition(roleName, DomainCanvas, canvasID)
		if err != nil {
			continue
		}
		roles = append(roles, roleDef)
	}

	return roles, nil
}

func (a *AuthService) SetupCanvasRoles(canvasID string) error {
	domain := fmt.Sprintf("canvas:%s", canvasID)

	for _, policy := range a.canvasPolicyTemplates {
		if policy[0] == "g" {
			// g,lower_role,higher_role,canvas:{CANVAS_ID}
			_, err := a.enforcer.AddGroupingPolicy(policy[1], policy[2], domain)
			if err != nil {
				return fmt.Errorf("failed to add grouping policy: %w", err)
			}
		} else if policy[0] == "p" {
			// p,role,canvas:{CANVAS_ID},resource,action
			_, err := a.enforcer.AddPolicy(policy[1], domain, policy[3], policy[4])
			if err != nil {
				return fmt.Errorf("failed to add policy: %w", err)
			}
		}
	}

	return nil
}

func (a *AuthService) DestroyCanvasRoles(canvasID string) error {
	domain := fmt.Sprintf("canvas:%s", canvasID)

	ok, err := a.enforcer.RemoveFilteredGroupingPolicy(2, domain)
	if err != nil {
		return fmt.Errorf("failed to remove canvas roles: %w", err)
	}
	if !ok {
		return fmt.Errorf("canvas roles not found for %s", canvasID)
	}

	ok, err = a.enforcer.RemoveFilteredPolicy(1, domain)
	if err != nil {
		return fmt.Errorf("failed to remove canvas policies: %w", err)
	}
	if !ok {
		return fmt.Errorf("canvas policies not found for %s", canvasID)
	}

	return nil
}

func (a *AuthService) GetRoleDefinition(roleName string, domainType string, domainID string) (*RoleDefinition, error) {
	// Validate domain type
	if domainType != DomainOrg && domainType != DomainCanvas {
		return nil, fmt.Errorf("invalid domain type: %s", domainType)
	}

	domain := fmt.Sprintf("%s:%s", domainType, domainID)
	prefixedRoleName := fmt.Sprintf("role:%s", roleName)

	// For default roles, check if the domain exists by looking for any policies in that domain
	if a.IsDefaultRole(roleName, domainType) {
		allPolicies, _ := a.enforcer.GetFilteredPolicy(1, domain)
		if len(allPolicies) == 0 {
			return nil, fmt.Errorf("role %s not found in domain %s", roleName, domain)
		}
	} else {
		// For custom roles, check if role exists by looking for policies
		policies, _ := a.enforcer.GetFilteredPolicy(0, prefixedRoleName, domain)
		groupingPolicies, _ := a.enforcer.GetFilteredGroupingPolicy(0, prefixedRoleName, "", domain)
		if len(policies) == 0 && len(groupingPolicies) == 0 {
			return nil, fmt.Errorf("role %s not found in domain %s", roleName, domain)
		}
	}

	// For default roles, show all permissions (including inherited)
	// For custom roles, show only direct permissions
	var permissions []*Permission
	if a.IsDefaultRole(roleName, domainType) {
		permissions = a.getRolePermissions(roleName, domain, domainType)
	} else {
		permissions = a.getDirectRolePermissions(roleName, domain, domainType)
	}

	roleDefinition := &RoleDefinition{
		Name:        roleName,
		DomainType:  domainType,
		Description: a.getRoleDescription(roleName),
		Permissions: permissions,
		Readonly:    true,
	}

	inheritedRole := a.getInheritedRole(roleName, domain, domainType)
	if inheritedRole != nil {
		roleDefinition.InheritsFrom = inheritedRole
	}

	return roleDefinition, nil
}

func (a *AuthService) GetAllRoleDefinitions(domainType string, domainID string) ([]*RoleDefinition, error) {
	domain := fmt.Sprintf("%s:%s", domainType, domainID)

	roles, err := a.getRolesFromPolicies(domain)
	if err != nil {
		return nil, fmt.Errorf("failed to get roles for domain %s: %w", domain, err)
	}

	roleDefinitions := []*RoleDefinition{}
	for _, roleName := range roles {
		roleDef, err := a.GetRoleDefinition(roleName, domainType, domainID)
		if err != nil {
			continue
		}
		roleDefinitions = append(roleDefinitions, roleDef)
	}

	return roleDefinitions, nil
}

func (a *AuthService) GetRolePermissions(roleName string, domainType string, domainID string) ([]*Permission, error) {
	// Validate domain type
	if domainType != DomainOrg && domainType != DomainCanvas {
		return nil, fmt.Errorf("invalid domain type: %s", domainType)
	}

	domain := fmt.Sprintf("%s:%s", domainType, domainID)
	prefixedRoleName := fmt.Sprintf("role:%s", roleName)

	// For default roles, check if the domain exists by looking for any policies in that domain
	if a.IsDefaultRole(roleName, domainType) {
		allPolicies, _ := a.enforcer.GetFilteredPolicy(1, domain)
		if len(allPolicies) == 0 {
			return nil, fmt.Errorf("role %s not found in domain %s", roleName, domain)
		}
	} else {
		// For custom roles, check if role exists by looking for policies
		policies, _ := a.enforcer.GetFilteredPolicy(0, prefixedRoleName, domain)
		groupingPolicies, _ := a.enforcer.GetFilteredGroupingPolicy(0, prefixedRoleName, "", domain)
		if len(policies) == 0 && len(groupingPolicies) == 0 {
			return nil, fmt.Errorf("role %s not found in domain %s", roleName, domain)
		}
	}

	return a.getRolePermissions(roleName, domain, domainType), nil
}

func (a *AuthService) GetRoleHierarchy(roleName string, domainType string, domainID string) ([]string, error) {
	// Validate domain type
	if domainType != DomainOrg && domainType != DomainCanvas {
		return nil, fmt.Errorf("invalid domain type: %s", domainType)
	}

	domain := fmt.Sprintf("%s:%s", domainType, domainID)
	prefixedRoleName := fmt.Sprintf("role:%s", roleName)

	// For default roles, check if the domain exists by looking for any policies in that domain
	if a.IsDefaultRole(roleName, domainType) {
		allPolicies, _ := a.enforcer.GetFilteredPolicy(1, domain)
		if len(allPolicies) == 0 {
			return nil, fmt.Errorf("role %s not found in domain %s", roleName, domain)
		}
	} else {
		// For custom roles, check if role exists by looking for policies
		policies, _ := a.enforcer.GetFilteredPolicy(0, prefixedRoleName, domain)
		groupingPolicies, _ := a.enforcer.GetFilteredGroupingPolicy(0, prefixedRoleName, "", domain)
		if len(policies) == 0 && len(groupingPolicies) == 0 {
			return nil, fmt.Errorf("role %s not found in domain %s", roleName, domain)
		}
	}

	implicitRoles, err := a.enforcer.GetImplicitRolesForUser(roleName, domain)
	if err != nil {
		return nil, fmt.Errorf("failed to get role hierarchy: %w", err)
	}

	hierarchy := []string{roleName}
	for _, role := range implicitRoles {
		if role != roleName {
			hierarchy = append(hierarchy, role)
		}
	}

	return hierarchy, nil
}

func (a *AuthService) CreateOrganizationOwner(userID, orgID string) error {
	return a.AssignRole(userID, RoleOrgOwner, orgID, DomainOrg)
}

func (a *AuthService) EnableCache(enable bool) {
	a.enforcer.EnableCache(enable)
}

// CreateCustomRole creates a new custom role for a domain
func (a *AuthService) CreateCustomRole(domainID string, roleDefinition *RoleDefinition) error {
	// Validate that the role name is not a default role
	if a.IsDefaultRole(roleDefinition.Name, roleDefinition.DomainType) {
		return fmt.Errorf("cannot create custom role with default role name: %s", roleDefinition.Name)
	}

	domain := fmt.Sprintf("%s:%s", roleDefinition.DomainType, domainID)
	prefixedRoleName := fmt.Sprintf("role:%s", roleDefinition.Name)

	// Validate inherited role exists if specified
	if roleDefinition.InheritsFrom != nil {
		// For inherited roles, check if it's a default role or if it has policies
		if !a.IsDefaultRole(roleDefinition.InheritsFrom.Name, roleDefinition.DomainType) {
			prefixedInheritedRole := fmt.Sprintf("role:%s", roleDefinition.InheritsFrom.Name)
			policies, _ := a.enforcer.GetFilteredPolicy(0, prefixedInheritedRole, domain)
			if len(policies) == 0 {
				return fmt.Errorf("inherited role not found: %s", roleDefinition.InheritsFrom.Name)
			}
		}
	}

	// Add policies for each permission
	for _, permission := range roleDefinition.Permissions {
		_, err := a.enforcer.AddPolicy(prefixedRoleName, domain, permission.Resource, permission.Action)
		if err != nil {
			return fmt.Errorf("failed to add policy for role %s: %w", roleDefinition.Name, err)
		}
	}

	// Add inheritance if specified
	if roleDefinition.InheritsFrom != nil {
		prefixedInheritedRole := fmt.Sprintf("role:%s", roleDefinition.InheritsFrom.Name)
		_, err := a.enforcer.AddGroupingPolicy(prefixedRoleName, prefixedInheritedRole, domain)
		if err != nil {
			return fmt.Errorf("failed to add inheritance for role %s: %w", roleDefinition.Name, err)
		}
	}

	return nil
}

// UpdateCustomRole updates an existing custom role
func (a *AuthService) UpdateCustomRole(domainID string, roleDefinition *RoleDefinition) error {
	// Validate that the role name is not a default role
	if a.IsDefaultRole(roleDefinition.Name, roleDefinition.DomainType) {
		return fmt.Errorf("cannot update default role: %s", roleDefinition.Name)
	}

	domain := fmt.Sprintf("%s:%s", roleDefinition.DomainType, domainID)
	prefixedRoleName := fmt.Sprintf("role:%s", roleDefinition.Name)

	// Check if role exists by looking for existing policies
	existingPolicies, _ := a.enforcer.GetFilteredPolicy(0, prefixedRoleName, domain)
	if len(existingPolicies) == 0 {
		return fmt.Errorf("role %s not found in domain %s", roleDefinition.Name, domain)
	}

	// Validate inherited role exists if specified
	if roleDefinition.InheritsFrom != nil {
		if !a.IsDefaultRole(roleDefinition.InheritsFrom.Name, roleDefinition.DomainType) {
			prefixedInheritedRole := fmt.Sprintf("role:%s", roleDefinition.InheritsFrom.Name)
			policies, _ := a.enforcer.GetFilteredPolicy(0, prefixedInheritedRole, domain)
			if len(policies) == 0 {
				return fmt.Errorf("inherited role not found: %s", roleDefinition.InheritsFrom.Name)
			}
		}
	}

	// Remove existing policies for this role in this domain
	_, err := a.enforcer.RemoveFilteredPolicy(0, prefixedRoleName, domain)
	if err != nil {
		return fmt.Errorf("failed to remove existing policies for role %s: %w", roleDefinition.Name, err)
	}

	// Remove existing role inheritance
	_, err = a.enforcer.RemoveFilteredGroupingPolicy(0, prefixedRoleName, "", domain)
	if err != nil {
		return fmt.Errorf("failed to remove existing inheritance for role %s: %w", roleDefinition.Name, err)
	}

	// Add new policies
	for _, permission := range roleDefinition.Permissions {
		_, err := a.enforcer.AddPolicy(prefixedRoleName, domain, permission.Resource, permission.Action)
		if err != nil {
			return fmt.Errorf("failed to add policy for role %s: %w", roleDefinition.Name, err)
		}
	}

	// Add inheritance if specified
	if roleDefinition.InheritsFrom != nil {
		prefixedInheritedRole := fmt.Sprintf("role:%s", roleDefinition.InheritsFrom.Name)
		_, err := a.enforcer.AddGroupingPolicy(prefixedRoleName, prefixedInheritedRole, domain)
		if err != nil {
			return fmt.Errorf("failed to add inheritance for role %s: %w", roleDefinition.Name, err)
		}
	}

	return nil
}

// DeleteCustomRole deletes a custom role
func (a *AuthService) DeleteCustomRole(domainID string, domainType string, roleName string) error {
	// Validate that the role name is not a default role
	if a.IsDefaultRole(roleName, domainType) {
		return fmt.Errorf("cannot delete default role: %s", roleName)
	}

	domain := fmt.Sprintf("%s:%s", domainType, domainID)
	prefixedRoleName := fmt.Sprintf("role:%s", roleName)

	// Check if role exists by looking for existing policies
	existingPolicies, _ := a.enforcer.GetFilteredPolicy(0, prefixedRoleName, domain)
	if len(existingPolicies) == 0 {
		return fmt.Errorf("role %s not found in domain %s", roleName, domain)
	}

	// Remove all policies for this role in this domain
	_, err := a.enforcer.RemoveFilteredPolicy(0, prefixedRoleName, domain)
	if err != nil {
		return fmt.Errorf("failed to remove policies for role %s: %w", roleName, err)
	}

	// Remove role inheritance for this domain
	_, err = a.enforcer.RemoveFilteredGroupingPolicy(0, prefixedRoleName, "", domain)
	if err != nil {
		return fmt.Errorf("failed to remove inheritance for role %s: %w", roleName, err)
	}

	// Remove users from this role in this domain
	_, err = a.enforcer.RemoveFilteredGroupingPolicy(1, prefixedRoleName, domain)
	if err != nil {
		return fmt.Errorf("failed to remove users from role %s: %w", roleName, err)
	}

	return nil
}

// IsDefaultRole checks if a role is a default system role
func (a *AuthService) IsDefaultRole(roleName string, domainType string) bool {
	defaultRoles := map[string][]string{
		DomainOrg:    {RoleOrgOwner, RoleOrgAdmin, RoleOrgViewer},
		DomainCanvas: {RoleCanvasOwner, RoleCanvasAdmin, RoleCanvasViewer},
	}

	roles, exists := defaultRoles[domainType]
	if !exists {
		return false
	}

	return contains(roles, roleName)
}

// CustomizeDefaultRole allows customizing permissions of default roles for a specific domain
func (a *AuthService) CustomizeDefaultRole(domainID string, domainType string, roleName string, permissions []*Permission) error {
	// Validate that the role is a default role
	if !a.IsDefaultRole(roleName, domainType) {
		return fmt.Errorf("role %s is not a default role", roleName)
	}

	domain := fmt.Sprintf("%s:%s", domainType, domainID)

	// Remove existing custom policies for this role in this domain (keep original template policies)
	existingPolicies, _ := a.enforcer.GetFilteredPolicy(0, roleName, domain)
	for _, policy := range existingPolicies {
		if len(policy) >= 4 {
			// Only remove if it's a custom policy (not from template)
			if !a.isPolicyFromTemplate(roleName, policy[2], policy[3], domainType) {
				_, err := a.enforcer.RemovePolicy(policy)
				if err != nil {
					return fmt.Errorf("failed to remove existing custom policy: %w", err)
				}
			}
		}
	}

	// Add new custom permissions
	for _, permission := range permissions {
		_, err := a.enforcer.AddPolicy(roleName, domain, permission.Resource, permission.Action)
		if err != nil {
			return fmt.Errorf("failed to add custom permission for role %s: %w", roleName, err)
		}
	}

	return nil
}

// isPolicyFromTemplate checks if a policy comes from the original template
func (a *AuthService) isPolicyFromTemplate(roleName, resource, action, domainType string) bool {
	var templates [][5]string
	if domainType == DomainOrg {
		templates = a.orgPolicyTemplates
	} else if domainType == DomainCanvas {
		templates = a.canvasPolicyTemplates
	}

	for _, template := range templates {
		if template[0] == roleName && template[2] == resource && template[3] == action {
			return true
		}
	}
	return false
}

func parsePoliciesFromCsv(content []byte) ([][5]string, error) {
	var policies [][5]string

	csvReader := csv.NewReader(bytes.NewReader(content))
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error reading CSV: %v", err)
		}
		if len(record) != 5 {
			return nil, fmt.Errorf("invalid CSV record: %v", record)
		}
		policies = append(policies, [5]string{record[0], record[1], record[2], record[3], record[4]})
	}

	return policies, nil
}

func (a *AuthService) roleExistsInDomain(roleName, domain string) bool {
	prefixedRoleName := fmt.Sprintf("role:%s", roleName)

	// Check if role has any policies in this domain
	policies, err := a.enforcer.GetFilteredPolicy(0, prefixedRoleName, domain)
	if err == nil && len(policies) > 0 {
		return true
	}

	// Check if role exists in grouping policies (inheritance)
	leftRoles, err := a.enforcer.GetFilteredGroupingPolicy(0, prefixedRoleName, "", domain)
	if err == nil && len(leftRoles) > 0 {
		return true
	}

	rightRoles, err := a.enforcer.GetFilteredGroupingPolicy(1, prefixedRoleName, domain)
	if err == nil && len(rightRoles) > 0 {
		return true
	}

	// Check if it's a default role (always exists)
	if a.IsDefaultRole(roleName, a.getDomainTypeFromDomain(domain)) {
		return true
	}

	return false
}

func (a *AuthService) getRolePermissions(roleName, domain, domainType string) []*Permission {
	prefixedRoleName := fmt.Sprintf("role:%s", roleName)
	// Get all permissions including inherited ones
	permissions, err := a.enforcer.GetImplicitPermissionsForUser(prefixedRoleName, domain)
	if err != nil {
		return []*Permission{}
	}

	rolePermissions := make([]*Permission, 0, len(permissions))
	for _, permission := range permissions {
		if len(permission) >= 4 {
			rolePermissions = append(rolePermissions, &Permission{
				Resource:   permission[2],
				Action:     permission[3],
				DomainType: domainType,
			})
		}
	}

	return rolePermissions
}

func (a *AuthService) getDirectRolePermissions(roleName, domain, domainType string) []*Permission {
	prefixedRoleName := fmt.Sprintf("role:%s", roleName)
	// Get only direct permissions for this role, not inherited ones
	permissions, err := a.enforcer.GetFilteredPolicy(0, prefixedRoleName, domain)
	if err != nil {
		return []*Permission{}
	}

	rolePermissions := make([]*Permission, 0, len(permissions))
	for _, permission := range permissions {
		if len(permission) >= 4 {
			rolePermissions = append(rolePermissions, &Permission{
				Resource:   permission[2],
				Action:     permission[3],
				DomainType: domainType,
			})
		}
	}

	return rolePermissions
}

func (a *AuthService) getInheritedRole(roleName, domain, domainType string) *RoleDefinition {
	prefixedRoleName := fmt.Sprintf("role:%s", roleName)
	implicitRoles, err := a.enforcer.GetImplicitRolesForUser(prefixedRoleName, domain)
	if err != nil || len(implicitRoles) == 0 {
		return nil
	}

	for _, inheritedRoleName := range implicitRoles {
		if inheritedRoleName != prefixedRoleName {
			unprefixedRoleName := strings.TrimPrefix(inheritedRoleName, "role:")
			return &RoleDefinition{
				Name:        unprefixedRoleName,
				DomainType:  domainType,
				Description: a.getRoleDescription(unprefixedRoleName),
				Permissions: a.getRolePermissions(unprefixedRoleName, domain, domainType),
				Readonly:    true,
			}
		}
	}

	return nil
}

func (a *AuthService) getRolesFromPolicies(domain string) ([]string, error) {
	roleSet := make(map[string]bool)

	// Get all policies where the domain matches (position 1 in policy)
	policies, err := a.enforcer.GetFilteredPolicy(1, domain)
	if err != nil {
		return nil, fmt.Errorf("failed to get filtered policy: %w", err)
	}
	for _, policy := range policies {
		if len(policy) >= 2 {
			// policy format: [role, domain, resource, action]
			roleName := policy[0]
			roleSet[roleName] = true
		}
	}

	// Also get roles from grouping policies (inheritance)
	groupingPolicies, err := a.enforcer.GetFilteredGroupingPolicy(2, domain)
	if err != nil {
		return nil, fmt.Errorf("failed to get filtered grouping policy: %w", err)
	}
	for _, policy := range groupingPolicies {
		if len(policy) >= 3 {
			// grouping policy format: [lower_role, higher_role, domain]
			lowerRole := policy[0]
			higherRole := policy[1]
			roleSet[lowerRole] = true
			roleSet[higherRole] = true
		}
	}

	roles := make([]string, 0, len(roleSet))
	for role := range roleSet {
		if strings.HasPrefix(role, "role:") {
			roles = append(roles, strings.TrimPrefix(role, "role:"))
		}
	}

	return roles, nil
}

func (a *AuthService) getRoleDescription(roleName string) string {
	descriptions := map[string]string{
		RoleOrgViewer:    "Read-only access to organization resources",
		RoleOrgAdmin:     "Full management access to organization resources including canvases and users",
		RoleOrgOwner:     "Complete control over the organization including settings and deletion",
		RoleCanvasViewer: "Read-only access to canvas resources",
		RoleCanvasAdmin:  "Full management access to canvas resources including stages and events",
		RoleCanvasOwner:  "Complete control over the canvas including member management",
	}

	if description, exists := descriptions[roleName]; exists {
		return description
	}
	return fmt.Sprintf("Role: %s", roleName)
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func (a *AuthService) getDomainTypeFromDomain(domain string) string {
	if strings.HasPrefix(domain, "org:") {
		return DomainOrg
	} else if strings.HasPrefix(domain, "canvas:") {
		return DomainCanvas
	}
	return ""
}
