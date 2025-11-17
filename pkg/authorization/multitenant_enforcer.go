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

	filter := gormadapter.Filter{
		V1: []string{domain},
	}

	return m.enforcer.LoadFilteredPolicy(filter)
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
