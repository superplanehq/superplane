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
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
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
	validRoles := map[string][]string{
		DomainOrg:    {RoleOrgViewer, RoleOrgAdmin, RoleOrgOwner},
		DomainCanvas: {RoleCanvasViewer, RoleCanvasAdmin, RoleCanvasOwner},
	}

	if roles, exists := validRoles[domainType]; exists {
		if !contains(roles, role) {
			return fmt.Errorf("invalid role %s for domain type %s", role, domainType)
		}
	}

	prefixedRole := fmt.Sprintf("role:%s", role)
	prefixedUserID := fmt.Sprintf("user:%s", userID)
	ruleAdded, err := a.enforcer.AddGroupingPolicy(prefixedUserID, prefixedRole, fmt.Sprintf("%s:%s", domainType, domainID))
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
	domain := fmt.Sprintf("%s:%s", domainType, domainID)

	if !a.roleExistsInDomain(roleName, domain) {
		return nil, fmt.Errorf("role %s not found in domain %s", roleName, domain)
	}

	roleDefinition := &RoleDefinition{
		Name:        roleName,
		DomainType:  domainType,
		Description: a.getRoleDescription(roleName),
		Permissions: a.getRolePermissions(roleName, domain, domainType),
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
	domain := fmt.Sprintf("%s:%s", domainType, domainID)

	if !a.roleExistsInDomain(roleName, domain) {
		return nil, fmt.Errorf("role %s not found in domain %s", roleName, domain)
	}

	return a.getRolePermissions(roleName, domain, domainType), nil
}

func (a *AuthService) GetRoleHierarchy(roleName string, domainType string, domainID string) ([]string, error) {
	domain := fmt.Sprintf("%s:%s", domainType, domainID)

	if !a.roleExistsInDomain(roleName, domain) {
		return nil, fmt.Errorf("role %s not found in domain %s", roleName, domain)
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

func (a *AuthService) SyncDefaultRoles() error {
	if err := a.syncOrganizationDefaultRoles(); err != nil {
		return fmt.Errorf("failed to sync organization default roles: %w", err)
	}

	if err := a.syncCanvasDefaultRoles(); err != nil {
		return fmt.Errorf("failed to sync canvas default roles: %w", err)
	}

	log.Info("Successfully synced default roles for all organizations and canvases")
	return nil
}

type CasbinRule struct {
	ID    uint   `gorm:"primaryKey;autoIncrement"`
	Ptype string `gorm:"size:100"`
	V0    string `gorm:"size:100"`
	V1    string `gorm:"size:100"`
	V2    string `gorm:"size:100"`
	V3    string `gorm:"size:100"`
	V4    string `gorm:"size:100"`
	V5    string `gorm:"size:100"`
}

func (CasbinRule) TableName() string {
	return "casbin_rule"
}

func (a *AuthService) DetectMissingPermissions() ([]string, []string, error) {
	missingOrgPerms, err := a.detectMissingOrganizationPermissions()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to detect missing organization permissions: %w", err)
	}

	missingCanvasPerms, err := a.detectMissingCanvasPermissions()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to detect missing canvas permissions: %w", err)
	}

	return missingOrgPerms, missingCanvasPerms, nil
}

func (a *AuthService) detectMissingOrganizationPermissions() ([]string, error) {
	orgIDs, err := a.getOrganizationsWithMissingPermissions()
	if err != nil {
		return nil, fmt.Errorf("failed to get organizations with missing permissions: %w", err)
	}

	var missingPerms []string
	for _, orgID := range orgIDs {
		domain := fmt.Sprintf("org:%s", orgID)

		missingInOrg, err := a.findMissingPermissionsInDomain(domain, a.orgPolicyTemplates)
		if err != nil {
			log.Errorf("Error checking permissions for org %s: %v", orgID, err)
			continue
		}

		if len(missingInOrg) > 0 {
			missingPerms = append(missingPerms, fmt.Sprintf("Organization %s: %d missing permissions", orgID, len(missingInOrg)))
		}
	}

	return missingPerms, nil
}

func (a *AuthService) detectMissingCanvasPermissions() ([]string, error) {
	canvasIDs, err := a.getCanvasesWithMissingPermissions()
	if err != nil {
		return nil, fmt.Errorf("failed to get canvases with missing permissions: %w", err)
	}

	var missingPerms []string
	for _, canvasID := range canvasIDs {
		domain := fmt.Sprintf("canvas:%s", canvasID)

		missingInCanvas, err := a.findMissingPermissionsInDomain(domain, a.canvasPolicyTemplates)
		if err != nil {
			log.Errorf("Error checking permissions for canvas %s: %v", canvasID, err)
			continue
		}

		if len(missingInCanvas) > 0 {
			missingPerms = append(missingPerms, fmt.Sprintf("Canvas %s: %d missing permissions", canvasID, len(missingInCanvas)))
		}
	}

	return missingPerms, nil
}

func (a *AuthService) getAllOrganizations() ([]models.Organization, error) {
	return models.ListOrganizations()
}

func (a *AuthService) getAllCanvases() ([]models.Canvas, error) {
	return models.ListCanvases()
}

// Optimized function to get only organization IDs that have missing permissions
func (a *AuthService) getOrganizationsWithMissingPermissions() ([]string, error) {
	// Get all organization IDs first (lightweight query)
	orgs, err := models.GetOrganizationIDs()
	if err != nil {
		return nil, fmt.Errorf("failed to get organization IDs: %w", err)
	}

	var missingOrgIDs []string
	for _, orgID := range orgs {
		domain := fmt.Sprintf("org:%s", orgID)
		hasMissing, err := a.domainHasMissingPermissions(domain, a.orgPolicyTemplates)
		if err != nil {
			log.Warnf("Error checking permissions for org %s: %v", orgID, err)
			continue
		}
		if hasMissing {
			missingOrgIDs = append(missingOrgIDs, orgID)
		}
	}

	return missingOrgIDs, nil
}

// Optimized function to get only canvas IDs that have missing permissions
func (a *AuthService) getCanvasesWithMissingPermissions() ([]string, error) {
	// Get all canvas IDs first (lightweight query)
	canvases, err := models.GetCanvasIDs()
	if err != nil {
		return nil, fmt.Errorf("failed to get canvas IDs: %w", err)
	}

	var missingCanvasIDs []string
	for _, canvasID := range canvases {
		domain := fmt.Sprintf("canvas:%s", canvasID)
		hasMissing, err := a.domainHasMissingPermissions(domain, a.canvasPolicyTemplates)
		if err != nil {
			log.Warnf("Error checking permissions for canvas %s: %v", canvasID, err)
			continue
		}
		if hasMissing {
			missingCanvasIDs = append(missingCanvasIDs, canvasID)
		}
	}

	return missingCanvasIDs, nil
}

// Helper function to check if a domain has missing permissions
func (a *AuthService) domainHasMissingPermissions(domain string, policyTemplates [][5]string) (bool, error) {
	for _, policy := range policyTemplates {
		if policy[0] == "g" {
			// Check grouping policy
			exists, err := a.enforcer.HasGroupingPolicy(policy[1], policy[2], domain)
			if err != nil {
				return false, err
			}
			if !exists {
				return true, nil // Found at least one missing permission
			}
		} else if policy[0] == "p" {
			// Check permission policy
			exists, err := a.enforcer.HasPolicy(policy[1], domain, policy[3], policy[4])
			if err != nil {
				return false, err
			}
			if !exists {
				return true, nil // Found at least one missing permission
			}
		}
	}
	return false, nil // No missing permissions found
}

func (a *AuthService) findMissingPermissionsInDomain(domain string, policyTemplates [][5]string) ([]string, error) {
	var missingPolicies []string

	for _, policy := range policyTemplates {
		if policy[0] == "g" {
			// Check grouping policy: g,lower_role,higher_role,domain
			exists, err := a.enforcer.HasGroupingPolicy(policy[1], policy[2], domain)
			if err != nil {
				return nil, fmt.Errorf("failed to check grouping policy: %w", err)
			}
			if !exists {
				missingPolicies = append(missingPolicies, fmt.Sprintf("Grouping: %s -> %s in %s", policy[1], policy[2], domain))
			}
		} else if policy[0] == "p" {
			// Check permission policy: p,role,domain,resource,action
			exists, err := a.enforcer.HasPolicy(policy[1], domain, policy[3], policy[4])
			if err != nil {
				return nil, fmt.Errorf("failed to check policy: %w", err)
			}
			if !exists {
				missingPolicies = append(missingPolicies, fmt.Sprintf("Policy: %s on %s.%s in %s", policy[1], policy[3], policy[4], domain))
			}
		}
	}

	return missingPolicies, nil
}

func (a *AuthService) syncOrganizationDefaultRoles() error {
	orgIDs, err := a.getOrganizationsWithMissingPermissions()
	if err != nil {
		return fmt.Errorf("failed to get organizations with missing permissions: %w", err)
	}

	if len(orgIDs) == 0 {
		log.Debug("No organizations with missing permissions found")
		return nil
	}

	log.Infof("Found %d organizations with missing permissions", len(orgIDs))

	for _, orgID := range orgIDs {
		if err := a.syncOrganizationRoles(orgID); err != nil {
			log.Errorf("Failed to sync roles for organization %s: %v", orgID, err)
			continue
		}
		log.Infof("Synced default roles for organization %s", orgID)
	}

	return nil
}

func (a *AuthService) syncCanvasDefaultRoles() error {
	canvasIDs, err := a.getCanvasesWithMissingPermissions()
	if err != nil {
		return fmt.Errorf("failed to get canvases with missing permissions: %w", err)
	}

	if len(canvasIDs) == 0 {
		log.Debug("No canvases with missing permissions found")
		return nil
	}

	log.Infof("Found %d canvases with missing permissions", len(canvasIDs))

	for _, canvasID := range canvasIDs {
		if err := a.syncCanvasRoles(canvasID); err != nil {
			log.Errorf("Failed to sync roles for canvas %s: %v", canvasID, err)
			continue
		}
		log.Infof("Synced default roles for canvas %s", canvasID)
	}

	return nil
}

func (a *AuthService) syncOrganizationRoles(orgID string) error {
	domain := fmt.Sprintf("org:%s", orgID)

	// First, apply default permissions from CSV templates
	err := a.applyDefaultPolicies(domain, a.orgPolicyTemplates)
	if err != nil {
		return fmt.Errorf("failed to apply default org policies: %w", err)
	}

	// Then, apply any permission overrides
	err = a.applyPermissionOverrides(&orgID, nil)
	if err != nil {
		log.Warnf("Failed to apply permission overrides for org %s: %v", orgID, err)
		// Don't fail sync if overrides fail - log warning and continue
	}

	return nil
}

func (a *AuthService) syncCanvasRoles(canvasID string) error {
	domain := fmt.Sprintf("canvas:%s", canvasID)

	// First, apply default permissions from CSV templates
	err := a.applyDefaultPolicies(domain, a.canvasPolicyTemplates)
	if err != nil {
		return fmt.Errorf("failed to apply default canvas policies: %w", err)
	}

	// Then, apply any permission overrides
	err = a.applyPermissionOverrides(nil, &canvasID)
	if err != nil {
		log.Warnf("Failed to apply permission overrides for canvas %s: %v", canvasID, err)
		// Don't fail sync if overrides fail - log warning and continue
	}

	return nil
}

func (a *AuthService) EnableCache(enable bool) {
	a.enforcer.EnableCache(enable)
}

// Permission Override Methods

// SetPermissionOverride creates or updates a permission override for a specific role
func (a *AuthService) SetPermissionOverride(organizationID, canvasID *string, roleName, resource, action string, isActive bool, createdBy string) error {
	var orgID, canvID *uuid.UUID
	var userID uuid.UUID
	var err error

	// Parse UUIDs
	if organizationID != nil {
		if orgUUID, parseErr := uuid.Parse(*organizationID); parseErr != nil {
			return fmt.Errorf("invalid organization ID: %w", parseErr)
		} else {
			orgID = &orgUUID
		}
	}

	if canvasID != nil {
		if canvUUID, parseErr := uuid.Parse(*canvasID); parseErr != nil {
			return fmt.Errorf("invalid canvas ID: %w", parseErr)
		} else {
			canvID = &canvUUID
		}
	}

	if userID, err = uuid.Parse(createdBy); err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	// Validate that exactly one domain is specified
	if (orgID == nil && canvID == nil) || (orgID != nil && canvID != nil) {
		return fmt.Errorf("exactly one of organizationID or canvasID must be specified")
	}

	// Check if override already exists
	existing, err := models.FindPermissionOverride(orgID, canvID, roleName, resource, action)
	if err == nil {
		// Update existing override
		return models.UpdatePermissionOverride(existing.ID, isActive)
	} else if err != gorm.ErrRecordNotFound {
		return fmt.Errorf("failed to check existing override: %w", err)
	}

	// Create new override
	_, err = models.CreatePermissionOverride(orgID, canvID, roleName, resource, action, isActive, userID)
	if err != nil {
		return fmt.Errorf("failed to create permission override: %w", err)
	}

	log.Infof("Created permission override: %s.%s.%s = %v", roleName, resource, action, isActive)
	return nil
}

// GetPermissionOverrides retrieves all permission overrides for a domain
func (a *AuthService) GetPermissionOverrides(organizationID, canvasID *string) ([]models.RolePermissionOverride, error) {
	var orgID, canvID *uuid.UUID

	// Parse UUIDs
	if organizationID != nil {
		if orgUUID, err := uuid.Parse(*organizationID); err != nil {
			return nil, fmt.Errorf("invalid organization ID: %w", err)
		} else {
			orgID = &orgUUID
		}
	}

	if canvasID != nil {
		if canvUUID, err := uuid.Parse(*canvasID); err != nil {
			return nil, fmt.Errorf("invalid canvas ID: %w", err)
		} else {
			canvID = &canvUUID
		}
	}

	// Validate that exactly one domain is specified
	if (orgID == nil && canvID == nil) || (orgID != nil && canvID != nil) {
		return nil, fmt.Errorf("exactly one of organizationID or canvasID must be specified")
	}

	return models.GetPermissionOverrides(orgID, canvID)
}

// GetAllPermissionOverrides retrieves all permission overrides (active and inactive) for a domain
func (a *AuthService) GetAllPermissionOverrides(organizationID, canvasID *string) ([]models.RolePermissionOverride, error) {
	var orgID, canvID *uuid.UUID

	// Parse UUIDs
	if organizationID != nil {
		if orgUUID, err := uuid.Parse(*organizationID); err != nil {
			return nil, fmt.Errorf("invalid organization ID: %w", err)
		} else {
			orgID = &orgUUID
		}
	}

	if canvasID != nil {
		if canvUUID, err := uuid.Parse(*canvasID); err != nil {
			return nil, fmt.Errorf("invalid canvas ID: %w", err)
		} else {
			canvID = &canvUUID
		}
	}

	// Validate that exactly one domain is specified
	if (orgID == nil && canvID == nil) || (orgID != nil && canvID != nil) {
		return nil, fmt.Errorf("exactly one of organizationID or canvasID must be specified")
	}

	return models.GetAllPermissionOverrides(orgID, canvID)
}

// GetAllHierarchyOverrides retrieves all hierarchy overrides (active and inactive) for a domain
func (a *AuthService) GetAllHierarchyOverrides(organizationID, canvasID *string) ([]models.RoleHierarchyOverride, error) {
	var orgID, canvID *uuid.UUID

	// Parse UUIDs
	if organizationID != nil {
		if orgUUID, err := uuid.Parse(*organizationID); err != nil {
			return nil, fmt.Errorf("invalid organization ID: %w", err)
		} else {
			orgID = &orgUUID
		}
	}

	if canvasID != nil {
		if canvUUID, err := uuid.Parse(*canvasID); err != nil {
			return nil, fmt.Errorf("invalid canvas ID: %w", err)
		} else {
			canvID = &canvUUID
		}
	}

	// Validate that exactly one domain is specified
	if (orgID == nil && canvID == nil) || (orgID != nil && canvID != nil) {
		return nil, fmt.Errorf("exactly one of organizationID or canvasID must be specified")
	}

	return models.GetAllHierarchyOverrides(orgID, canvID)
}

// SetHierarchyOverride creates or updates a role hierarchy override
func (a *AuthService) SetHierarchyOverride(organizationID, canvasID *string, childRole, parentRole string, isActive bool, createdBy string) error {
	var orgID, canvID *uuid.UUID
	var userID uuid.UUID
	var err error

	// Parse UUIDs
	if organizationID != nil {
		if orgUUID, parseErr := uuid.Parse(*organizationID); parseErr != nil {
			return fmt.Errorf("invalid organization ID: %w", parseErr)
		} else {
			orgID = &orgUUID
		}
	}

	if canvasID != nil {
		if canvUUID, parseErr := uuid.Parse(*canvasID); parseErr != nil {
			return fmt.Errorf("invalid canvas ID: %w", parseErr)
		} else {
			canvID = &canvUUID
		}
	}

	if userID, err = uuid.Parse(createdBy); err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	// Validate that exactly one domain is specified
	if (orgID == nil && canvID == nil) || (orgID != nil && canvID != nil) {
		return fmt.Errorf("exactly one of organizationID or canvasID must be specified")
	}

	// Check if override already exists
	existing, err := models.FindHierarchyOverride(orgID, canvID, childRole, parentRole)
	if err == nil {
		// Update existing override
		return models.UpdateHierarchyOverride(existing.ID, isActive)
	} else if err != gorm.ErrRecordNotFound {
		return fmt.Errorf("failed to check existing hierarchy override: %w", err)
	}

	// Create new override
	_, err = models.CreateHierarchyOverride(orgID, canvID, childRole, parentRole, isActive, userID)
	if err != nil {
		return fmt.Errorf("failed to create hierarchy override: %w", err)
	}

	log.Infof("Created hierarchy override: %s -> %s = %v", childRole, parentRole, isActive)
	return nil
}

// GetHierarchyOverrides retrieves all hierarchy overrides for a domain
func (a *AuthService) GetHierarchyOverrides(organizationID, canvasID *string) ([]models.RoleHierarchyOverride, error) {
	var orgID, canvID *uuid.UUID

	// Parse UUIDs
	if organizationID != nil {
		if orgUUID, err := uuid.Parse(*organizationID); err != nil {
			return nil, fmt.Errorf("invalid organization ID: %w", err)
		} else {
			orgID = &orgUUID
		}
	}

	if canvasID != nil {
		if canvUUID, err := uuid.Parse(*canvasID); err != nil {
			return nil, fmt.Errorf("invalid canvas ID: %w", err)
		} else {
			canvID = &canvUUID
		}
	}

	// Validate that exactly one domain is specified
	if (orgID == nil && canvID == nil) || (orgID != nil && canvID != nil) {
		return nil, fmt.Errorf("exactly one of organizationID or canvasID must be specified")
	}

	return models.GetHierarchyOverrides(orgID, canvID)
}

// Helper function to apply default policies from templates
func (a *AuthService) applyDefaultPolicies(domain string, policyTemplates [][5]string) error {
	for _, policy := range policyTemplates {
		if policy[0] == "g" {
			// Add grouping policy only if it doesn't exist
			exists, err := a.enforcer.HasGroupingPolicy(policy[1], policy[2], domain)
			if err != nil {
				return fmt.Errorf("failed to check grouping policy: %w", err)
			}
			if !exists {
				_, err := a.enforcer.AddGroupingPolicy(policy[1], policy[2], domain)
				if err != nil {
					return fmt.Errorf("failed to add grouping policy: %w", err)
				}
			}
		} else if policy[0] == "p" {
			// Add permission policy only if it doesn't exist
			exists, err := a.enforcer.HasPolicy(policy[1], domain, policy[3], policy[4])
			if err != nil {
				return fmt.Errorf("failed to check policy: %w", err)
			}
			if !exists {
				_, err := a.enforcer.AddPolicy(policy[1], domain, policy[3], policy[4])
				if err != nil {
					return fmt.Errorf("failed to add policy: %w", err)
				}
			}
		}
	}
	return nil
}

// Helper function to apply permission overrides
func (a *AuthService) applyPermissionOverrides(organizationID, canvasID *string) error {
	// Get all permission overrides (active and inactive)
	permOverrides, err := a.GetAllPermissionOverrides(organizationID, canvasID)
	if err != nil {
		return fmt.Errorf("failed to get permission overrides: %w", err)
	}

	// Get all hierarchy overrides (active and inactive)
	hierOverrides, err := a.GetAllHierarchyOverrides(organizationID, canvasID)
	if err != nil {
		return fmt.Errorf("failed to get hierarchy overrides: %w", err)
	}

	// Determine domain string
	var domain string
	if organizationID != nil {
		domain = fmt.Sprintf("org:%s", *organizationID)
	} else if canvasID != nil {
		domain = fmt.Sprintf("canvas:%s", *canvasID)
	} else {
		return fmt.Errorf("either organizationID or canvasID must be specified")
	}

	// Apply permission overrides
	for _, override := range permOverrides {
		roleWithPrefix := fmt.Sprintf("role:%s", override.RoleName)

		if override.IsActive {
			// Add the permission if it's active and doesn't exist
			exists, err := a.enforcer.HasPolicy(roleWithPrefix, domain, override.Resource, override.Action)
			if err != nil {
				log.Warnf("Failed to check policy for override %s.%s.%s: %v", override.RoleName, override.Resource, override.Action, err)
				continue
			}
			if !exists {
				_, err := a.enforcer.AddPolicy(roleWithPrefix, domain, override.Resource, override.Action)
				if err != nil {
					log.Warnf("Failed to add policy for override %s.%s.%s: %v", override.RoleName, override.Resource, override.Action, err)
				} else {
					log.Infof("Applied permission override: Added %s.%s.%s", override.RoleName, override.Resource, override.Action)
				}
			}
		} else {
			// Remove the permission if it's inactive and exists
			exists, err := a.enforcer.HasPolicy(roleWithPrefix, domain, override.Resource, override.Action)
			if err != nil {
				log.Warnf("Failed to check policy for override %s.%s.%s: %v", override.RoleName, override.Resource, override.Action, err)
				continue
			}
			if exists {
				_, err := a.enforcer.RemovePolicy(roleWithPrefix, domain, override.Resource, override.Action)
				if err != nil {
					log.Warnf("Failed to remove policy for override %s.%s.%s: %v", override.RoleName, override.Resource, override.Action, err)
				} else {
					log.Infof("Applied permission override: Removed %s.%s.%s", override.RoleName, override.Resource, override.Action)
				}
			}
		}
	}

	// Apply hierarchy overrides
	for _, override := range hierOverrides {
		childRoleWithPrefix := fmt.Sprintf("role:%s", override.ChildRole)
		parentRoleWithPrefix := fmt.Sprintf("role:%s", override.ParentRole)

		if override.IsActive {
			// Add the hierarchy if it's active and doesn't exist
			exists, err := a.enforcer.HasGroupingPolicy(childRoleWithPrefix, parentRoleWithPrefix, domain)
			if err != nil {
				log.Warnf("Failed to check grouping policy for override %s -> %s: %v", override.ChildRole, override.ParentRole, err)
				continue
			}
			if !exists {
				_, err := a.enforcer.AddGroupingPolicy(childRoleWithPrefix, parentRoleWithPrefix, domain)
				if err != nil {
					log.Warnf("Failed to add grouping policy for override %s -> %s: %v", override.ChildRole, override.ParentRole, err)
				} else {
					log.Infof("Applied hierarchy override: Added %s -> %s", override.ChildRole, override.ParentRole)
				}
			}
		} else {
			// Remove the hierarchy if it's inactive and exists
			exists, err := a.enforcer.HasGroupingPolicy(childRoleWithPrefix, parentRoleWithPrefix, domain)
			if err != nil {
				log.Warnf("Failed to check grouping policy for override %s -> %s: %v", override.ChildRole, override.ParentRole, err)
				continue
			}
			if exists {
				_, err := a.enforcer.RemoveGroupingPolicy(childRoleWithPrefix, parentRoleWithPrefix, domain)
				if err != nil {
					log.Warnf("Failed to remove grouping policy for override %s -> %s: %v", override.ChildRole, override.ParentRole, err)
				} else {
					log.Infof("Applied hierarchy override: Removed %s -> %s", override.ChildRole, override.ParentRole)
				}
			}
		}
	}

	return nil
}

// Example usage function for checking and syncing missing permissions
func (a *AuthService) CheckAndSyncMissingPermissions() error {
	// First, detect missing permissions
	missingOrgs, missingCanvases, err := a.DetectMissingPermissions()
	if err != nil {
		return fmt.Errorf("failed to detect missing permissions: %w", err)
	}

	if len(missingOrgs) > 0 {
		log.Infof("Found %d organizations with missing permissions:", len(missingOrgs))
		for _, org := range missingOrgs {
			log.Info(org)
		}
	}

	if len(missingCanvases) > 0 {
		log.Infof("Found %d canvases with missing permissions:", len(missingCanvases))
		for _, canvas := range missingCanvases {
			log.Info(canvas)
		}
	}

	// If there are missing permissions, sync them
	if len(missingOrgs) > 0 || len(missingCanvases) > 0 {
		log.Info("Syncing missing default roles...")
		if err := a.SyncDefaultRoles(); err != nil {
			return fmt.Errorf("failed to sync default roles: %w", err)
		}
		log.Info("Successfully synced all missing permissions")
	} else {
		log.Info("No missing permissions found - all organizations and canvases are up to date")
	}

	return nil
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

	// Check both sides of grouping policy due to inheritance definition
	leftRoles, err := a.enforcer.GetFilteredGroupingPolicy(0, prefixedRoleName, "", domain)

	if err != nil {
		return false
	}

	rightRoles, err := a.enforcer.GetFilteredGroupingPolicy(1, prefixedRoleName, domain)

	if err != nil {
		return false
	}

	for _, role := range append(leftRoles, rightRoles...) {
		if role[0] == prefixedRoleName || role[1] == prefixedRoleName {
			return true
		}
	}
	return false
}

func (a *AuthService) getRolePermissions(roleName, domain, domainType string) []*Permission {
	prefixedRoleName := fmt.Sprintf("role:%s", roleName)
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
