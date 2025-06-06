package authorization

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"os"

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

type AuthService struct {
	enforcer              *casbin.CachedEnforcer
	db                    *gorm.DB
	orgPolicyTemplates    [][5]string
	canvasPolicyTemplates [][5]string
}

func NewAuthService() (*AuthService, error) {
	adapter, err := gormadapter.NewAdapterByDB(database.Conn())
	if err != nil {
		return nil, fmt.Errorf("failed to create casbin adapter: %w", err)
	}

	enforcer, err := casbin.NewCachedEnforcer("../config/rbac_model.conf", adapter)
	if err != nil {
		return nil, fmt.Errorf("failed to create casbin enforcer: %w", err)
	}

	enforcer.EnableAutoSave(true)

	if err := enforcer.LoadPolicy(); err != nil {
		return nil, fmt.Errorf("failed to load policies: %w", err)
	}

	orgPoliciesCsv, err := os.ReadFile("../config/rbac_org_policy.csv")
	if err != nil {
		return nil, fmt.Errorf("failed to read org policies: %w", err)
	}
	canvasPoliciesCsv, err := os.ReadFile("../config/rbac_canvas_policy.csv")
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
	return a.enforcer.Enforce(userID, domain, resource, action)
}

func (a *AuthService) CreateGroup(orgID string, groupName string, role string) error {
	return nil
}

func (a *AuthService) AddUserToGroup(orgID string, userID string, group string) error {
	return nil
}

func (a *AuthService) RemoveUserFromGroup(orgID string, userID string, group string) error {
	return nil
}

func (a *AuthService) GetGroupUsers(orgID string, group string) ([]string, error) {
	return nil, nil
}

func (a *AuthService) GetGroups(orgID string) ([]string, error) {
	return nil, nil
}

func (a *AuthService) GetGroupRoles(orgID string, group string) ([]string, error) {
	return nil, nil
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

	ruleAdded, err := a.enforcer.AddGroupingPolicy(userID, role, fmt.Sprintf("%s:%s", domainType, domainID))
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
	ruleRemoved, err := a.enforcer.RemoveGroupingPolicy(userID, role, domain)
	if err != nil {
		return fmt.Errorf("failed to remove role: %w", err)
	}
	if !ruleRemoved {
		log.Infof("role %s not found for user %s", role, userID)
	}
	return nil
}

func (a *AuthService) GetOrgUsersForRole(role string, orgID string) ([]string, error) {
	orgDomain := fmt.Sprintf("org:%s", orgID)
	roles, err := a.enforcer.GetUsersForRole(role, orgDomain)
	if err != nil {
		return nil, err
	}
	return roles, nil
}

func (a *AuthService) GetCanvasUsersForRole(role string, canvasID string) ([]string, error) {
	canvasDomain := fmt.Sprintf("canvas:%s", canvasID)
	roles, err := a.enforcer.GetUsersForRole(role, canvasDomain)
	if err != nil {
		return nil, err
	}
	return roles, nil
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

func (a *AuthService) GetAccessibleOrgsForUser(userID string) ([]string, error) {
	orgs, err := a.enforcer.GetFilteredGroupingPolicy(0, userID, "", "org_viewer")
	if err != nil {
		return nil, err
	}

	orgIDs := make([]string, len(orgs))
	prefixLen := len("org:")
	for i, org := range orgs {
		orgIDs[i] = org[2][prefixLen:]
	}
	return orgIDs, nil
}

func (a *AuthService) GetAccessibleCanvasesForUser(userID string) ([]string, error) {
	canvases, err := a.enforcer.GetFilteredGroupingPolicy(0, userID, "", "canvas_viewer")
	if err != nil {
		return nil, err
	}

	canvasIDs := make([]string, len(canvases))
	prefixLen := len("canvas:")
	for i, canvas := range canvases {
		canvasIDs[i] = canvas[2][prefixLen:]
	}
	return canvasIDs, nil
}

func (a *AuthService) GetUserRolesForOrg(userID string, orgID string) ([]string, error) {
	orgDomain := fmt.Sprintf("org:%s", orgID)
	roles, err := a.enforcer.GetImplicitRolesForUser(userID, orgDomain)
	if err != nil {
		return nil, err
	}
	return roles, nil
}

func (a *AuthService) GetUserRolesForCanvas(userID string, canvasID string) ([]string, error) {
	canvasDomain := fmt.Sprintf("canvas:%s", canvasID)
	roles, err := a.enforcer.GetImplicitRolesForUser(userID, canvasDomain)
	if err != nil {
		return nil, err
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

func (a *AuthService) CreateOrganizationOwner(userID, orgID string) error {
	return a.AssignRole(userID, RoleOrgOwner, orgID, DomainOrg)
}

func (a *AuthService) Middleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// TODO: implement middleware once authentication is implemented
			next.ServeHTTP(w, r)
		})
	}
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

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
