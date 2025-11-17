package org

import (
	"fmt"

	"github.com/casbin/casbin/v2"
	"github.com/superplanehq/superplane/pkg/authorization2/base"
	"github.com/superplanehq/superplane/pkg/models"

	gormadapter "github.com/casbin/gorm-adapter/v3"
)

//
// Verifier for organization-level permissions.
//

type Verifier struct {
	enforcer *casbin.TransactionalEnforcer

	domain string
	user   string
}

func NewVerifier(orgID string, userID string) (*Verifier, error) {
	domain := fmt.Sprintf("%s:%s", models.DomainTypeOrganization, orgID)
	user := fmt.Sprintf("user:%s", userID)

	enforcer, err := base.Enforcer()
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

	return &Verifier{
		enforcer: enforcer,
		domain:   domain,
		user:     user,
	}, nil
}

func (v *Verifier) CanReadCanvas() (bool, error) {
	return v.enforcer.Enforce(v.user, v.domain, "canvas", "read")
}

func (v *Verifier) CanCreateCanvas() (bool, error) {
	return v.enforcer.Enforce(v.user, v.domain, "canvas", "create")
}

func (v *Verifier) CanUpdateCanvas() (bool, error) {
	return v.enforcer.Enforce(v.user, v.domain, "canvas", "update")
}

func (v *Verifier) CanDeleteCanvas() (bool, error) {
	return v.enforcer.Enforce(v.user, v.domain, "canvas", "delete")
}

func (v *Verifier) CanCreateMember() (bool, error) {
	return v.enforcer.Enforce(v.user, v.domain, "member", "create")
}

func (v *Verifier) CanDeleteMember() (bool, error) {
	return v.enforcer.Enforce(v.user, v.domain, "member", "delete")
}

func (v *Verifier) CanUpdateMember() (bool, error) {
	return v.enforcer.Enforce(v.user, v.domain, "member", "update")
}

func (v *Verifier) CanReadMember() (bool, error) {
	return v.enforcer.Enforce(v.user, v.domain, "member", "read")
}

func (v *Verifier) CanUpdateOrg() (bool, error) {
	return v.enforcer.Enforce(v.user, v.domain, "org", "update")
}

func (v *Verifier) CanDeleteOrg() (bool, error) {
	return v.enforcer.Enforce(v.user, v.domain, "org", "delete")
}
