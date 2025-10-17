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
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

const (
	OrgIDTemplate    = "{ORG_ID}"
	CanvasIDTemplate = "{CANVAS_ID}"
)

// implements Authorization
var _ Authorization = (*AuthService)(nil)

//
// NOTE: We need to use nested transaction to update Group and Role Metadata with casbin policies since
// Gorm Casbin Adapter GetDB() functions overrides the table name with casbin_rule. It is not possible
// to fix this issue with default gorm methods like Table() or Model()
//

type AuthService struct {
	enforcer              *casbin.CachedEnforcer
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
		orgPolicyTemplates:    orgPolicyTemplates,
		canvasPolicyTemplates: canvasPolicyTemplates,
	}

	return service, nil
}

func (a *AuthService) CheckCanvasPermission(userID, canvasID, resource, action string) (bool, error) {
	return a.checkPermission(userID, canvasID, models.DomainTypeCanvas, resource, action)
}

func (a *AuthService) CheckCanvasGlobalPermission(userID, orgID, resource, action string) (bool, error) {
	domain := fmt.Sprintf("canvas:*|org:%s", orgID)
	prefixedUserID := prefixUserID(userID)
	return a.enforcer.Enforce(prefixedUserID, domain, resource, action)
}

func (a *AuthService) CheckOrganizationPermission(userID, orgID, resource, action string) (bool, error) {
	return a.checkPermission(userID, orgID, models.DomainTypeOrganization, resource, action)
}

func (a *AuthService) checkPermission(userID, domainID, domainType, resource, action string) (bool, error) {
	domain := prefixDomain(domainType, domainID)
	prefixedUserID := prefixUserID(userID)
	return a.enforcer.Enforce(prefixedUserID, domain, resource, action)
}

func (a *AuthService) CreateGroup(domainID string, domainType string, groupName string, role string, displayName string, description string) error {
	err := a.CreateGroupWithNestedTransaction(domainID, domainType, groupName, role, displayName, description)
	if err != nil {
		return fmt.Errorf("failed to create group: %w", err)
	}

	return nil
}

func (a *AuthService) CreateGroupWithNestedTransaction(domainID string, domainType string, groupName string, role string, displayName string, description string) error {
	domain := prefixDomain(domainType, domainID)
	prefixedRole := prefixRoleName(role)

	if !a.roleExistsInDomain(role, domain) {
		return fmt.Errorf("invalid role %s for domain type %s", role, domainType)
	}

	prefixedGroupName := prefixGroupName(groupName)

	a.enforcer.EnableAutoSave(false)
	defer a.enforcer.EnableAutoSave(true)

	return a.enforcer.GetAdapter().(*gormadapter.Adapter).Transaction(a.enforcer, func(e casbin.IEnforcer) error {
		return database.Conn().Transaction(func(tx *gorm.DB) error {
			err := models.UpsertGroupMetadataInTransaction(tx, groupName, domainType, domainID, displayName, description)
			if err != nil {
				return fmt.Errorf("failed to create group metadata: %w", err)
			}

			ruleAdded, err := e.AddGroupingPolicy(prefixedGroupName, prefixedRole, domain)
			if err != nil {
				return fmt.Errorf("failed to create group: %w", err)
			}

			if !ruleAdded {
				return fmt.Errorf("group %s already exists with role %s in %s %s", groupName, role, domainType, domainID)
			}

			return e.SavePolicy()
		})
	})
}

func (a *AuthService) DeleteGroup(domainID string, domainType string, groupName string) error {
	err := a.deleteGroupDataWithNestedTransaction(domainID, domainType, groupName)

	if err != nil {
		return fmt.Errorf("failed to delete group: %w", err)
	}

	return nil
}

func (a *AuthService) deleteGroupDataWithNestedTransaction(domainID string, domainType string, groupName string) error {
	domain := prefixDomain(domainType, domainID)
	prefixedGroupName := prefixGroupName(groupName)

	a.enforcer.EnableAutoSave(false)
	defer a.enforcer.EnableAutoSave(true)

	return a.enforcer.GetAdapter().(*gormadapter.Adapter).Transaction(a.enforcer, func(e casbin.IEnforcer) error {
		return database.Conn().Transaction(func(tx *gorm.DB) error {
			err := models.DeleteGroupMetadataInTransaction(tx, groupName, domainType, domainID)
			if err != nil {
				return fmt.Errorf("failed to delete group metadata: %w", err)
			}

			_, err = e.RemoveFilteredGroupingPolicy(0, prefixedGroupName, "", domain)
			if err != nil {
				return fmt.Errorf("failed to remove group role assignment: %w", err)
			}

			_, err = e.RemoveFilteredGroupingPolicy(1, prefixedGroupName, domain)
			if err != nil {
				return fmt.Errorf("failed to remove users from group: %w", err)
			}

			return e.SavePolicy()
		})
	})
}

func (a *AuthService) UpdateGroup(domainID string, domainType string, groupName string, newRole string, displayName string, description string) error {
	if err := models.ValidateDomainType(domainType); err != nil {
		return err
	}

	err := a.updateGroupWithNestedTransaction(domainID, domainType, groupName, newRole, displayName, description)
	if err != nil {
		return fmt.Errorf("failed to update group: %w", err)
	}

	return nil
}

func (a *AuthService) updateGroupWithNestedTransaction(domainID string, domainType string, groupName string, newRole string, displayName string, description string) error {
	domain := prefixDomain(domainType, domainID)

	if !a.roleExistsInDomain(newRole, domain) {
		return fmt.Errorf("invalid role %s for domain type %s", newRole, domainType)
	}

	prefixedGroupName := prefixGroupName(groupName)

	currentRole, err := a.GetGroupRole(domainID, domainType, groupName)
	if err != nil {
		return fmt.Errorf("failed to get current group role: %w", err)
	}

	currentGroupMetadata, err := models.FindGroupMetadata(groupName, domainType, domainID)
	if err != nil {
		return fmt.Errorf("failed to get current group metadata: %w", err)
	}

	a.enforcer.EnableAutoSave(false)
	defer a.enforcer.EnableAutoSave(true)
	return a.enforcer.GetAdapter().(*gormadapter.Adapter).Transaction(a.enforcer, func(e casbin.IEnforcer) error {
		return database.Conn().Transaction(func(tx *gorm.DB) error {
			var updatedDisplayName, updatedDescription string
			if displayName != "" {
				updatedDisplayName = displayName
			} else {
				updatedDisplayName = currentGroupMetadata.DisplayName
			}
			if description != "" {
				updatedDescription = description
			} else {
				updatedDescription = currentGroupMetadata.Description
			}

			if displayName != "" || description != "" {
				err := models.UpsertGroupMetadataInTransaction(tx, groupName, domainType, domainID, updatedDisplayName, updatedDescription)
				if err != nil {
					return fmt.Errorf("failed to update group metadata: %w", err)
				}
			}

			prefixedOldRole := prefixRoleName(currentRole)
			ruleRemoved, err := e.RemoveGroupingPolicy(prefixedGroupName, prefixedOldRole, domain)
			if err != nil {
				return fmt.Errorf("failed to remove old group role: %w", err)
			}
			if !ruleRemoved {
				return fmt.Errorf("old group role assignment not found")
			}

			prefixedNewRole := prefixRoleName(newRole)
			ruleAdded, err := e.AddGroupingPolicy(prefixedGroupName, prefixedNewRole, domain)
			if err != nil {
				return fmt.Errorf("failed to add new group role: %w", err)
			}
			if !ruleAdded {
				return fmt.Errorf("failed to add new group role assignment")
			}

			return e.SavePolicy()
		})
	})
}

func (a *AuthService) AddUserToGroup(domainID string, domainType string, userID string, group string) error {
	domain := prefixDomain(domainType, domainID)
	prefixedGroupName := prefixGroupName(group)
	prefixedUserID := prefixUserID(userID)

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
		return fmt.Errorf("group %s does not exist in %s %s", group, domainType, domainID)
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

func (a *AuthService) RemoveUserFromGroup(domainID string, domainType string, userID string, group string) error {
	domain := prefixDomain(domainType, domainID)
	prefixedGroupName := prefixGroupName(group)
	prefixedUserID := prefixUserID(userID)

	ruleRemoved, err := a.enforcer.RemoveGroupingPolicy(prefixedUserID, prefixedGroupName, domain)
	if err != nil {
		return fmt.Errorf("failed to remove user from group: %w", err)
	}

	if !ruleRemoved {
		return fmt.Errorf("user %s is not a member of group %s", userID, group)
	}

	return nil
}

func (a *AuthService) GetGroupUsers(domainID string, domainType string, group string) ([]string, error) {
	domain := prefixDomain(domainType, domainID)
	prefixedGroupName := prefixGroupName(group)

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

func (a *AuthService) GetGroups(domainID string, domainType string) ([]string, error) {
	domain := prefixDomain(domainType, domainID)
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

func (a *AuthService) GetGroupRole(domainID string, domainType string, group string) (string, error) {
	domain := prefixDomain(domainType, domainID)
	prefixedGroupName := prefixGroupName(group)
	roles := a.enforcer.GetRolesForUserInDomain(prefixedGroupName, domain)
	unprefixedRoles := []string{}
	for _, role := range roles {
		if strings.HasPrefix(role, "role:") {
			unprefixedRoles = append(unprefixedRoles, strings.TrimPrefix(role, "role:"))
		}
	}
	if len(unprefixedRoles) == 0 {
		return "", fmt.Errorf("group %s not found in domain %s", group, domainID)
	}
	return unprefixedRoles[0], nil
}

func (a *AuthService) AssignRole(userID, role, domainID string, domainType string) error {
	domain := prefixDomain(domainType, domainID)
	prefixedRole := prefixRoleName(role)

	// Check if it's a default role
	validRoles := map[string][]string{
		models.DomainTypeOrganization: {models.RoleOrgViewer, models.RoleOrgAdmin, models.RoleOrgOwner},
		models.DomainTypeCanvas:       {models.RoleCanvasViewer, models.RoleCanvasAdmin, models.RoleCanvasOwner},
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

	prefixedUserID := prefixUserID(userID)

	existingRoles, err := a.enforcer.GetFilteredGroupingPolicy(0, prefixedUserID, "", domain)
	if err != nil {
		return fmt.Errorf("failed to get existing roles for user: %w", err)
	}

	for _, existingRole := range existingRoles {
		if strings.HasPrefix(existingRole[1], "role:") {
			_, err := a.enforcer.RemoveGroupingPolicy(prefixedUserID, existingRole[1], domain)
			if err != nil {
				log.Warnf("failed to remove existing role %s for user %s: %v", existingRole[1], userID, err)
			}
		}
	}

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
	domain := prefixDomain(domainType, domainID)
	prefixedRole := prefixRoleName(role)
	prefixedUserID := prefixUserID(userID)
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
	prefixedRole := prefixRoleName(role)
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
	prefixedRole := prefixRoleName(role)
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
	domain := prefixDomain(models.DomainTypeOrganization, orgID)

	a.enforcer.EnableAutoSave(false)
	defer a.enforcer.EnableAutoSave(true)
	err := a.enforcer.GetAdapter().(*gormadapter.Adapter).Transaction(a.enforcer, func(e casbin.IEnforcer) error {
		return database.Conn().Transaction(func(tx *gorm.DB) error {
			for _, policy := range a.orgPolicyTemplates {
				switch policy[0] {
				case "g":
					_, err := e.AddGroupingPolicy(policy[1], policy[2], domain)
					if err != nil {
						return fmt.Errorf("failed to add grouping policy: %w", err)
					}
				case "p":
					_, err := e.AddPolicy(policy[1], domain, policy[3], policy[4])
					if err != nil {
						return fmt.Errorf("failed to add policy: %w", err)
					}
				default:
					return fmt.Errorf("unknown policy type: %s", policy[0])
				}
			}

			log.Infof("Setting up default organization role metadata for %s", orgID)
			if err := a.setupDefaultOrganizationRoleMetadataInTransaction(tx, orgID); err != nil {
				log.Errorf("Error setting up default organization role metadata: %v", err)
				return err
			}

			return e.SavePolicy()
		})
	})

	if err != nil {
		return fmt.Errorf("failed to setup organization roles: %w", err)
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

func (a *AuthService) GetAccessibleCanvasesForUser(userID string) ([]string, error) {
	prefixedUserID := prefixUserID(userID)
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
	prefixedUserID := prefixUserID(userID)
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
		roleDef, err := a.GetRoleDefinition(roleName, models.DomainTypeOrganization, orgID)
		if err != nil {
			continue
		}
		roles = append(roles, roleDef)
	}

	return roles, nil
}

func (a *AuthService) GetUserRolesForCanvas(userID string, canvasID string) ([]*RoleDefinition, error) {
	canvasDomain := fmt.Sprintf("canvas:%s", canvasID)
	prefixedUserID := prefixUserID(userID)
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
		roleDef, err := a.GetRoleDefinition(roleName, models.DomainTypeCanvas, canvasID)
		if err != nil {
			continue
		}
		roles = append(roles, roleDef)
	}

	return roles, nil
}

func (a *AuthService) SetupCanvasRoles(canvasID string) error {
	domain := fmt.Sprintf("canvas:%s", canvasID)

	a.enforcer.EnableAutoSave(false)
	defer a.enforcer.EnableAutoSave(true)

	err := a.enforcer.GetAdapter().(*gormadapter.Adapter).Transaction(a.enforcer, func(e casbin.IEnforcer) error {
		return database.Conn().Transaction(func(tx *gorm.DB) error {
			for _, policy := range a.canvasPolicyTemplates {
				switch policy[0] {
				case "g":
					// g,lower_role,higher_role,canvas:{CANVAS_ID}
					_, err := e.AddGroupingPolicy(policy[1], policy[2], domain)
					if err != nil {
						return fmt.Errorf("failed to add grouping policy: %w", err)
					}
				case "p":
					// p,role,canvas:{CANVAS_ID},resource,action
					_, err := e.AddPolicy(policy[1], domain, policy[3], policy[4])
					if err != nil {
						return fmt.Errorf("failed to add policy: %w", err)
					}
				}
			}

			if err := a.setupDefaultCanvasRoleMetadataInTransaction(tx, canvasID); err != nil {
				log.Errorf("Error setting up default canvas role metadata: %v", err)
				return err
			}

			return e.SavePolicy()
		})
	})

	if err != nil {
		return fmt.Errorf("failed to setup canvas roles: %w", err)
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
	if err := models.ValidateDomainType(domainType); err != nil {
		return nil, err
	}

	domain := prefixDomain(domainType, domainID)
	prefixedRoleName := prefixRoleName(roleName)

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
	domain := prefixDomain(domainType, domainID)

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
	if err := models.ValidateDomainType(domainType); err != nil {
		return nil, err
	}

	domain := prefixDomain(domainType, domainID)
	prefixedRoleName := prefixRoleName(roleName)

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
	if err := models.ValidateDomainType(domainType); err != nil {
		return nil, err
	}

	domain := prefixDomain(domainType, domainID)
	prefixedRoleName := prefixRoleName(roleName)

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
	return a.AssignRole(userID, models.RoleOrgOwner, orgID, models.DomainTypeOrganization)
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

func (a *AuthService) DetectMissingPermissions() ([]string, []string, error) {
	orgs, err := a.getOrganizationsWithMissingPermissions()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to detect missing organization permissions: %w", err)
	}

	canvases, err := a.getCanvasesWithMissingPermissions()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to detect missing canvas permissions: %w", err)
	}

	return orgs, canvases, nil
}

func (a *AuthService) getOrganizationsWithMissingPermissions() ([]string, error) {
	orgs, err := models.GetActiveOrganizationIDs()
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

func (a *AuthService) getCanvasesWithMissingPermissions() ([]string, error) {
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

func (a *AuthService) domainHasMissingPermissions(domain string, policyTemplates [][5]string) (bool, error) {
	for _, policy := range policyTemplates {
		switch policy[0] {
		case "g":
			// Check grouping policy
			exists, err := a.enforcer.HasGroupingPolicy(policy[1], policy[2], domain)
			if err != nil {
				return false, err
			}
			if !exists {
				return true, nil // Found at least one missing permission
			}
		case "p":
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
		if err := a.SyncOrganizationRoles(orgID); err != nil {
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
		if err := a.SyncCanvasRoles(canvasID); err != nil {
			log.Errorf("Failed to sync roles for canvas %s: %v", canvasID, err)
			continue
		}
		log.Infof("Synced default roles for canvas %s", canvasID)
	}

	return nil
}

func (a *AuthService) SyncOrganizationRoles(orgID string) error {
	domain := fmt.Sprintf("org:%s", orgID)

	// First, apply default permissions from CSV templates
	err := a.applyDefaultPolicies(domain, a.orgPolicyTemplates)
	if err != nil {
		return fmt.Errorf("failed to apply default org policies: %w", err)
	}

	return nil
}

func (a *AuthService) SyncCanvasRoles(canvasID string) error {
	domain := fmt.Sprintf("canvas:%s", canvasID)

	// First, apply default permissions from CSV templates
	err := a.applyDefaultPolicies(domain, a.canvasPolicyTemplates)
	if err != nil {
		return fmt.Errorf("failed to apply default canvas policies: %w", err)
	}

	return nil
}

func (a *AuthService) EnableCache(enable bool) {
	a.enforcer.EnableCache(enable)
}

// Helper function to apply default policies from templates
func (a *AuthService) applyDefaultPolicies(domain string, policyTemplates [][5]string) error {
	for _, policy := range policyTemplates {
		switch policy[0] {
		case "g":
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
		case "p":
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

func (a *AuthService) CreateCustomRole(domainID string, roleDefinition *RoleDefinition) error {
	if a.IsDefaultRole(roleDefinition.Name, roleDefinition.DomainType) {
		return fmt.Errorf("cannot create custom role with default role name: %s", roleDefinition.Name)
	}

	err := a.createCustomRoleWithNestedTransaction(domainID, roleDefinition)

	if err != nil {
		return fmt.Errorf("failed to create custom role: %w", err)
	}

	return nil
}

func (a *AuthService) createCustomRoleWithNestedTransaction(domainID string, roleDefinition *RoleDefinition) error {
	domain := prefixDomain(roleDefinition.DomainType, domainID)
	prefixedRoleName := prefixRoleName(roleDefinition.Name)

	if roleDefinition.InheritsFrom != nil {
		if !a.IsDefaultRole(roleDefinition.InheritsFrom.Name, roleDefinition.DomainType) {
			prefixedInheritedRole := prefixRoleName(roleDefinition.InheritsFrom.Name)
			policies, _ := a.enforcer.GetFilteredPolicy(0, prefixedInheritedRole, domain)
			if len(policies) == 0 {
				return fmt.Errorf("inherited role not found: %s", roleDefinition.InheritsFrom.Name)
			}
		}
	}

	a.enforcer.EnableAutoSave(false)
	defer a.enforcer.EnableAutoSave(true)

	return a.enforcer.GetAdapter().(*gormadapter.Adapter).Transaction(a.enforcer, func(e casbin.IEnforcer) error {
		return database.Conn().Transaction(func(tx *gorm.DB) error {
			err := models.UpsertRoleMetadataInTransaction(tx, roleDefinition.Name, roleDefinition.DomainType, domainID, roleDefinition.DisplayName, roleDefinition.Description)

			if err != nil {
				return fmt.Errorf("failed to upsert role metadata for role %s: %w", roleDefinition.Name, err)
			}

			for _, permission := range roleDefinition.Permissions {
				_, err := e.AddPolicy(prefixedRoleName, domain, permission.Resource, permission.Action)
				if err != nil {
					return fmt.Errorf("failed to add policy for role %s: %w", roleDefinition.Name, err)
				}
			}

			if roleDefinition.InheritsFrom != nil {
				prefixedInheritedRole := prefixRoleName(roleDefinition.InheritsFrom.Name)
				_, err := e.AddGroupingPolicy(prefixedRoleName, prefixedInheritedRole, domain)
				if err != nil {
					return fmt.Errorf("failed to add inheritance for role %s: %w", roleDefinition.Name, err)
				}
			}

			return e.SavePolicy()
		})
	})
}

func (a *AuthService) UpdateCustomRole(domainID string, roleDefinition *RoleDefinition) error {
	if a.IsDefaultRole(roleDefinition.Name, roleDefinition.DomainType) {
		return fmt.Errorf("cannot update default role: %s", roleDefinition.Name)
	}

	err := a.updateCustomRoleWithNestedTransaction(domainID, roleDefinition)

	if err != nil {
		return fmt.Errorf("failed to update custom role: %w", err)
	}

	return nil
}

func (a *AuthService) updateCustomRoleWithNestedTransaction(domainID string, roleDefinition *RoleDefinition) error {
	domain := prefixDomain(roleDefinition.DomainType, domainID)
	prefixedRoleName := prefixRoleName(roleDefinition.Name)

	existingPolicies, _ := a.enforcer.GetFilteredPolicy(0, prefixedRoleName, domain)
	if len(existingPolicies) == 0 {
		return fmt.Errorf("role %s not found in domain %s", roleDefinition.Name, domain)
	}

	if roleDefinition.InheritsFrom != nil {
		if !a.IsDefaultRole(roleDefinition.InheritsFrom.Name, roleDefinition.DomainType) {
			prefixedInheritedRole := prefixRoleName(roleDefinition.InheritsFrom.Name)
			policies, _ := a.enforcer.GetFilteredPolicy(0, prefixedInheritedRole, domain)
			if len(policies) == 0 {
				return fmt.Errorf("inherited role not found: %s", roleDefinition.InheritsFrom.Name)
			}
		}
	}

	a.enforcer.EnableAutoSave(false)
	defer a.enforcer.EnableAutoSave(true)
	return a.enforcer.GetAdapter().(*gormadapter.Adapter).Transaction(a.enforcer, func(e casbin.IEnforcer) error {
		return database.Conn().Transaction(func(tx *gorm.DB) error {
			err := models.UpsertRoleMetadataInTransaction(tx, roleDefinition.Name, roleDefinition.DomainType, domainID, roleDefinition.DisplayName, roleDefinition.Description)

			if err != nil {
				return fmt.Errorf("failed to upsert role metadata for role %s: %w", roleDefinition.Name, err)
			}

			_, err = e.RemoveFilteredPolicy(0, prefixedRoleName, domain)
			if err != nil {
				return fmt.Errorf("failed to remove existing policies for role %s: %w", roleDefinition.Name, err)
			}

			_, err = e.RemoveFilteredGroupingPolicy(0, prefixedRoleName, "", domain)
			if err != nil {
				return fmt.Errorf("failed to remove existing inheritance for role %s: %w", roleDefinition.Name, err)
			}

			for _, permission := range roleDefinition.Permissions {
				_, err := e.AddPolicy(prefixedRoleName, domain, permission.Resource, permission.Action)
				if err != nil {
					return fmt.Errorf("failed to add policy for role %s: %w", roleDefinition.Name, err)
				}
			}

			if roleDefinition.InheritsFrom != nil {
				prefixedInheritedRole := prefixRoleName(roleDefinition.InheritsFrom.Name)
				_, err := e.AddGroupingPolicy(prefixedRoleName, prefixedInheritedRole, domain)
				if err != nil {
					return fmt.Errorf("failed to add inheritance for role %s: %w", roleDefinition.Name, err)
				}
			}

			return e.SavePolicy()
		})
	})
}

func (a *AuthService) DeleteCustomRole(domainID string, domainType string, roleName string) error {
	if a.IsDefaultRole(roleName, domainType) {
		return fmt.Errorf("cannot delete default role: %s", roleName)
	}

	err := a.deleteCustomRoleWithNestedTransaction(domainID, domainType, roleName)

	if err != nil {
		return fmt.Errorf("failed to delete custom role: %w", err)
	}

	return nil
}

func (a *AuthService) deleteCustomRoleWithNestedTransaction(domainID string, domainType string, roleName string) error {
	domain := prefixDomain(domainType, domainID)
	prefixedRoleName := prefixRoleName(roleName)

	existingPolicies, _ := a.enforcer.GetFilteredPolicy(0, prefixedRoleName, domain)
	if len(existingPolicies) == 0 {
		return fmt.Errorf("role %s not found in domain %s", roleName, domain)
	}

	a.enforcer.EnableAutoSave(false)
	defer a.enforcer.EnableAutoSave(true)
	return a.enforcer.GetAdapter().(*gormadapter.Adapter).Transaction(a.enforcer, func(e casbin.IEnforcer) error {
		return database.Conn().Transaction(func(tx *gorm.DB) error {
			err := models.DeleteRoleMetadataInTransaction(tx, roleName, domainType, domainID)
			if err != nil {
				return fmt.Errorf("failed to delete role metadata for role %s: %w", roleName, err)
			}

			_, err = e.RemoveFilteredPolicy(0, prefixedRoleName, domain)
			if err != nil {
				return fmt.Errorf("failed to remove policies for role %s: %w", roleName, err)
			}

			_, err = e.RemoveFilteredGroupingPolicy(0, prefixedRoleName, "", domain)
			if err != nil {
				return fmt.Errorf("failed to remove inheritance for role %s: %w", roleName, err)
			}

			_, err = e.RemoveFilteredGroupingPolicy(1, prefixedRoleName, domain)
			if err != nil {
				return fmt.Errorf("failed to remove users from role %s: %w", roleName, err)
			}

			return e.SavePolicy()
		})
	})
}

// IsDefaultRole checks if a role is a default system role
func (a *AuthService) IsDefaultRole(roleName string, domainType string) bool {
	defaultRoles := map[string][]string{
		models.DomainTypeOrganization: {models.RoleOrgOwner, models.RoleOrgAdmin, models.RoleOrgViewer},
		models.DomainTypeCanvas:       {models.RoleCanvasOwner, models.RoleCanvasAdmin, models.RoleCanvasViewer},
	}

	roles, exists := defaultRoles[domainType]
	if !exists {
		return false
	}

	return contains(roles, roleName)
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
	prefixedRoleName := prefixRoleName(roleName)

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
	prefixedRoleName := prefixRoleName(roleName)
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
	prefixedRoleName := prefixRoleName(roleName)
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
	prefixedRoleName := prefixRoleName(roleName)
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
		models.RoleOrgViewer:    models.DescOrgViewer,
		models.RoleOrgAdmin:     models.DescOrgAdmin,
		models.RoleOrgOwner:     models.DescOrgOwner,
		models.RoleCanvasViewer: models.DescCanvasViewer,
		models.RoleCanvasAdmin:  models.DescCanvasAdmin,
		models.RoleCanvasOwner:  models.DescCanvasOwner,
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
		return models.DomainTypeOrganization
	} else if strings.HasPrefix(domain, "canvas:") {
		return models.DomainTypeCanvas
	}
	return ""
}

func (a *AuthService) setupDefaultOrganizationRoleMetadataInTransaction(tx *gorm.DB, orgID string) error {
	defaultRoles := []struct {
		name        string
		displayName string
		description string
	}{
		{
			name:        models.RoleOrgOwner,
			displayName: models.DisplayNameOwner,
			description: models.MetaDescOrgOwner,
		},
		{
			name:        models.RoleOrgAdmin,
			displayName: models.DisplayNameAdmin,
			description: models.MetaDescOrgAdmin,
		},
		{
			name:        models.RoleOrgViewer,
			displayName: models.DisplayNameViewer,
			description: models.MetaDescOrgViewer,
		},
	}

	for _, role := range defaultRoles {
		if err := models.UpsertRoleMetadataInTransaction(tx, role.name, models.DomainTypeOrganization, orgID, role.displayName, role.description); err != nil {
			return fmt.Errorf("failed to upsert role metadata for %s: %w", role.name, err)
		}
	}

	return nil
}

func (a *AuthService) setupDefaultCanvasRoleMetadataInTransaction(tx *gorm.DB, canvasID string) error {
	defaultRoles := []struct {
		name        string
		displayName string
		description string
	}{
		{
			name:        models.RoleCanvasOwner,
			displayName: models.DisplayNameOwner,
			description: models.MetaDescCanvasOwner,
		},
		{
			name:        models.RoleCanvasAdmin,
			displayName: models.DisplayNameAdmin,
			description: models.MetaDescCanvasAdmin,
		},
		{
			name:        models.RoleCanvasViewer,
			displayName: models.DisplayNameViewer,
			description: models.MetaDescCanvasViewer,
		},
	}

	for _, role := range defaultRoles {
		if err := models.UpsertRoleMetadataInTransaction(tx, role.name, models.DomainTypeCanvas, canvasID, role.displayName, role.description); err != nil {
			return fmt.Errorf("failed to upsert role metadata for %s: %w", role.name, err)
		}
	}

	return nil
}

func prefixRoleName(roleName string) string {
	return fmt.Sprintf("role:%s", roleName)
}

func prefixGroupName(groupName string) string {
	return fmt.Sprintf("group:%s", groupName)
}

func prefixUserID(userID string) string {
	return fmt.Sprintf("user:%s", userID)
}

func prefixDomain(domainType string, domainID string) string {
	return fmt.Sprintf("%s:%s", domainType, domainID)
}
