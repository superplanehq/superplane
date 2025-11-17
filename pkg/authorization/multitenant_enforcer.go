package authorization

import (
	"context"
	"fmt"
	"os"

	"github.com/casbin/casbin/v2"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

//
// Efficient multi-tenant enforcer using Casbin with Gorm adapter.
//

//
// How to enforce policies in the API:
//
// allowed, err := authorization.OrgEnforcer(userID, orgID, resource, action)
// if err != nil {
//     // handle error
// }
//
// if allowed {
// 	 proceed with the action
// } else {
// 	 deny access
// }
//

type MultiTenantEnforcer struct {
	enforcer *casbin.TransactionalEnforcer
}

func NewMultiTenantEnforcer() (*MultiTenantEnforcer, error) {
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

	return &MultiTenantEnforcer{
		enforcer: enforcer,
	}, nil
}

func (m *MultiTenantEnforcer) LoadOrganizationPolicy(orgID string) error {
	if orgID == "" {
		return fmt.Errorf("orgID cannot be empty")
	}

	domain := prefixDomain(models.DomainTypeOrganization, orgID)

	filter := gormadapter.Filter{
		V1: []string{domain},
	}

	return m.enforcer.LoadFilteredPolicy(filter)
}

func (m *MultiTenantEnforcer) EnforceOrganization(userID, orgID, resource, action string) (bool, error) {
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

func (m *MultiTenantEnforcer) WithOrganizationTransaction(ctx context.Context, orgID string, fn func(*casbin.Transaction) error) error {
	if orgID == "" {
		return fmt.Errorf("orgID cannot be empty")
	}
	if fn == nil {
		return fmt.Errorf("transaction function cannot be nil")
	}

	return m.enforcer.WithTransaction(ctx, fn)
}
