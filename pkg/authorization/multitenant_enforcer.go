package authorization

import (
	"context"
	"fmt"
	"os"

	"github.com/casbin/casbin/v2"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

func Verify(userID, orgID, resource, action string) (bool, error) {
	enforcer, err := newMultiTenantEnforcer()
	if err != nil {
		return false, fmt.Errorf("failed to create multi-tenant enforcer: %w", err)
	}

	if err := enforcer.LoadOrganizationPolicy(orgID); err != nil {
		return false, fmt.Errorf("failed to load organization policy: %w", err)
	}

	allowed, err := enforcer.EnforceOrganization(userID, orgID, resource, action)
	if err != nil {
		return false, fmt.Errorf("failed to enforce organization policy: %w", err)
	}

	return allowed, nil
}

//
// Provision sets up default roles and policies for a new organization.
//
// CLEAN ME UP!

func Provision(orgID string) error {
	if orgID == "" {
		return fmt.Errorf("orgID cannot be empty")
	}

	domain := prefixDomain(models.DomainTypeOrganization, orgID)

	tx := database.Conn().Begin()

	log.Infof("Setting up default organization role metadata for %s", orgID)

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
			log.Errorf("Error setting up default organization role metadata for %s: %v", orgID, err)
			tx.Rollback()
			return fmt.Errorf("failed to upsert role metadata for %s: %w", role.name, err)
		}
	}

	log.Infof("Role metadata added - adding policies for %s", orgID)

	orgPolicyPath := os.Getenv("RBAC_ORG_POLICY_PATH")
	orgPoliciesCsv, err := os.ReadFile(orgPolicyPath)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to read org policies: %w", err)
	}

	orgPolicyTemplates, err := parsePoliciesFromCsv(orgPoliciesCsv)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to parse org policies: %w", err)
	}

	enforcer, err := newMultiTenantEnforcer()
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to create multi-tenant enforcer: %w", err)
	}

	err = enforcer.WithOrganizationTransaction(context.Background(), orgID, func(casbinTx *casbin.Transaction) error {
		for _, policy := range orgPolicyTemplates {
			switch policy[0] {
			case "g":
				_, err := casbinTx.AddGroupingPolicy(policy[1], policy[2], domain)
				if err != nil {
					return fmt.Errorf("failed to add grouping policy: %w", err)
				}
			case "p":
				_, err := casbinTx.AddPolicy(policy[1], domain, policy[3], policy[4])
				if err != nil {
					return fmt.Errorf("failed to add policy: %w", err)
				}
			default:
				return fmt.Errorf("unknown policy type: %s", policy[0])
			}
		}

		return nil
	})

	if err != nil {
		log.Errorf("Error adding policies for %s: %v", orgID, err)
		tx.Rollback()
		return fmt.Errorf("failed to provision organization roles for %s: %w", orgID, err)
	}

	log.Infof("Policies added - loading policies for %s", orgID)

	if err := enforcer.enforcer.LoadPolicy(); err != nil {
		log.Errorf("Error loading policies after provisioning organization roles for %s: %v", orgID, err)
		tx.Rollback()
		return fmt.Errorf("failed to load policies after provisioning organization roles for %s: %w", orgID, err)
	}

	log.Infof("Policies loaded - committing transaction for %s", orgID)

	tx.Commit()

	log.Infof("Transaction committed for %s", orgID)

	return nil
}

func Update(orgID string, fn func(*casbin.Transaction) error) error {
	if fn == nil {
		return fmt.Errorf("update function cannot be nil")
	}

	enforcer, err := newMultiTenantEnforcer()
	if err != nil {
		return fmt.Errorf("failed to create multi-tenant enforcer: %w", err)
	}

	if err := enforcer.LoadOrganizationPolicy(orgID); err != nil {
		return fmt.Errorf("failed to load organization policy: %w", err)
	}

	if err := enforcer.WithOrganizationTransaction(context.Background(), orgID, fn); err != nil {
		return fmt.Errorf("failed to update organization policy: %w", err)
	}

	return nil
}

//
// Private methods and types for multi-tenant enforcer.
//

type multiTenantEnforcer struct {
	enforcer *casbin.TransactionalEnforcer
}

func newMultiTenantEnforcer() (*multiTenantEnforcer, error) {
	modelPath := os.Getenv("RBAC_MODEL_PATH")

	adapter, err := gormadapter.NewTransactionalAdapterByDB(database.Conn())
	if err != nil {
		return nil, fmt.Errorf("failed to create casbin adapter: %w", err)
	}

	enforcer, err := casbin.NewTransactionalEnforcer(modelPath, adapter)
	if err != nil {
		return nil, fmt.Errorf("failed to create casbin enforcer: %w", err)
	}

	enforcer.EnableAutoSave(true)

	return &multiTenantEnforcer{
		enforcer: enforcer,
	}, nil
}

func (m *multiTenantEnforcer) LoadOrganizationPolicy(orgID string) error {
	if orgID == "" {
		return fmt.Errorf("orgID cannot be empty")
	}

	domain := prefixDomain(models.DomainTypeOrganization, orgID)

	// Load both policy (p) and grouping (g) rules for this
	// organization's domain:
	// - p rules store domain in V1
	// - g rules store domain in V2 (user/role/group in V0/V1)
	filters := []gormadapter.Filter{
		{
			Ptype: []string{"p"},
			V1:    []string{domain},
		},
		{
			Ptype: []string{"g"},
			V2:    []string{domain},
		},
	}

	return m.enforcer.LoadFilteredPolicy(filters)
}

func (m *multiTenantEnforcer) EnforceOrganization(userID, orgID, resource, action string) (bool, error) {
	if userID == "" {
		return false, fmt.Errorf("userID cannot be empty")
	}
	if orgID == "" {
		return false, fmt.Errorf("orgID cannot be empty")
	}

	domain := prefixDomain(models.DomainTypeOrganization, orgID)
	prefixedUserID := prefixUserID(userID)

	return m.enforcer.Enforce(prefixedUserID, domain, resource, action)
}

func (m *multiTenantEnforcer) WithOrganizationTransaction(ctx context.Context, orgID string, fn func(*casbin.Transaction) error) error {
	if orgID == "" {
		return fmt.Errorf("orgID cannot be empty")
	}
	if fn == nil {
		return fmt.Errorf("transaction function cannot be nil")
	}

	return m.enforcer.WithTransaction(ctx, fn)
}
