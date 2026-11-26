package authorization

import (
	"errors"
	"fmt"

	"github.com/casbin/casbin/v2"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

var ErrResourceAlreadyExists = errors.New("resource already exists")

func (a *AuthService) withCreateEnforcer(kind, domainType, domainID, name string, create func(*gorm.DB, casbin.IEnforcer) error) error {
	if err := models.ValidateDomainType(domainType); err != nil {
		return err
	}
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		if err := models.LockRBACResourceName(tx, kind, domainType, domainID, name); err != nil {
			return err
		}
		exists, err := models.RBACResourceMetadataExists(tx, kind, domainType, domainID, name)
		if err != nil {
			return err
		}
		if exists || (kind == "role" && a.IsDefaultRole(name, domainType)) {
			return ErrResourceAlreadyExists
		}
		enforcer, err := a.newFilteredEnforcer(tx, policyFiltersForDomain(domainType, domainID))
		if err != nil {
			return err
		}
		identity := prefixGroupName(name)
		if kind == "role" {
			identity = prefixRoleName(name)
		}
		exists, err = policyIdentityExists(enforcer, identity, prefixDomain(domainType, domainID))
		if err != nil {
			return err
		}
		if exists {
			return ErrResourceAlreadyExists
		}
		// Only this transaction-local view may publish writes to the adapter.
		enforcer.EnableAutoSave(true)
		return create(tx, enforcer)
	})
	var conflict *pgconn.PgError
	if errors.As(err, &conflict) && conflict.Code == "23505" && conflict.ConstraintName == "uq_"+kind+"_metadata_key" {
		return fmt.Errorf("%w: %s", ErrResourceAlreadyExists, name)
	}
	return err
}

func policyIdentityExists(enforcer casbin.IEnforcer, identity, domain string) (bool, error) {
	permissions, err := enforcer.GetFilteredPolicy(0, identity, domain)
	if err != nil {
		return false, err
	}
	left, err := enforcer.GetFilteredGroupingPolicy(0, identity, "", domain)
	if err != nil {
		return false, err
	}
	right, err := enforcer.GetFilteredGroupingPolicy(1, identity, domain)
	if err != nil {
		return false, err
	}
	return len(permissions)+len(left)+len(right) > 0, nil
}

func (a *AuthService) roleExistsWithEnforcer(enforcer casbin.IEnforcer, name, domainType, domainID string) (bool, error) {
	if a.IsDefaultRole(name, domainType) {
		return true, nil
	}
	return policyIdentityExists(enforcer, prefixRoleName(name), prefixDomain(domainType, domainID))
}
