package authorization2

import (
	"fmt"

	"github.com/casbin/casbin/v2"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"github.com/superplanehq/superplane/pkg/models"
)

type orgverifier struct {
	enforcer *casbin.TransactionalEnforcer

	domain string
	user   string
}

func OrgVerifier(orgID string, userID string) (*orgverifier, error) {
	domain := fmt.Sprintf("%s:%s", models.DomainTypeOrganization, orgID)
	user := fmt.Sprintf("user:%s", userID)

	enforcer, err := enforcer()
	if err != nil {
		return nil, fmt.Errorf("failed to create enforcer: %w", err)
	}

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

	err = enforcer.LoadFilteredPolicy(filters)
	if err != nil {
		return nil, fmt.Errorf("failed to load filtered policies: %w", err)
	}

	return &orgverifier{
		enforcer: enforcer,
		domain:   domain,
		user:     user,
	}, nil
}

func (v *orgverifier) CanReadCanvas() (bool, error) {
	return v.enforcer.Enforce(v.user, v.domain, "canvas", "read")
}

func (v *orgverifier) CanCreateCanvas() (bool, error) {
	return v.enforcer.Enforce(v.user, v.domain, "canvas", "create")
}

func (v *orgverifier) CanUpdateCanvas() (bool, error) {
	return v.enforcer.Enforce(v.user, v.domain, "canvas", "update")
}

func (v *orgverifier) CanDeleteCanvas() (bool, error) {
	return v.enforcer.Enforce(v.user, v.domain, "canvas", "delete")
}

func (v *orgverifier) CanCreateMember() (bool, error) {
	return v.enforcer.Enforce(v.user, v.domain, "member", "create")
}

func (v *orgverifier) CanDeleteMember() (bool, error) {
	return v.enforcer.Enforce(v.user, v.domain, "member", "delete")
}

func (v *orgverifier) CanUpdateMember() (bool, error) {
	return v.enforcer.Enforce(v.user, v.domain, "member", "update")
}

func (v *orgverifier) CanReadMember() (bool, error) {
	return v.enforcer.Enforce(v.user, v.domain, "member", "read")
}

func (v *orgverifier) CanUpdateOrg() (bool, error) {
	return v.enforcer.Enforce(v.user, v.domain, "org", "update")
}

func (v *orgverifier) CanDeleteOrg() (bool, error) {
	return v.enforcer.Enforce(v.user, v.domain, "org", "delete")
}
