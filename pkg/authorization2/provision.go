package authorization2

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"os"

	"github.com/casbin/casbin/v2"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

//
// Provisions a new organization with default roles and policies.
//

func Provision(tx *gorm.DB, orgID string, ownerID string) error {
	var err error

	p := newProvisioner(tx, orgID, ownerID)

	err = p.initializeEnforcer()
	if err != nil {
		return err
	}

	err = p.createDefaultRoles()
	if err != nil {
		return err
	}

	err = p.loadOrgPolicies()
	if err != nil {
		return err
	}

	err = p.addPolicies()
	if err != nil {
		return err
	}

	err = p.addOwner()
	if err != nil {
		return err
	}

	return nil
}

//
// Implements the provisioning logic.
//

type provisioner struct {
	orgID   string
	tx      *gorm.DB
	ownerID string

	enforcer *casbin.TransactionalEnforcer
	policies [][5]string
}

func newProvisioner(tx *gorm.DB, orgID string, ownerID string) *provisioner {
	return &provisioner{
		tx:      tx,
		orgID:   orgID,
		ownerID: ownerID,
	}
}

func (p *provisioner) initializeEnforcer() error {
	var err error

	p.enforcer, err = enforcer()
	if err != nil {
		return fmt.Errorf("failed to create enforcer: %w", err)
	}

	return nil
}

func (p *provisioner) createDefaultRoles() error {
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
		if err := models.UpsertRoleMetadataInTransaction(p.tx, role.name, models.DomainTypeOrganization, p.orgID, role.displayName, role.description); err != nil {
			return fmt.Errorf("failed to upsert role metadata for %s: %w", role.name, err)
		}
	}

	return nil
}

func (p *provisioner) loadOrgPolicies() error {
	path := os.Getenv("RBAC_ORG_POLICY_PATH")

	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read org policies: %w", err)
	}

	var policies [][5]string

	csvReader := csv.NewReader(bytes.NewReader(content))

	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}

		if err != nil {
			return fmt.Errorf("error reading CSV: %v", err)
		}

		if len(record) != 5 {
			return fmt.Errorf("invalid CSV record: %v", record)
		}

		policies = append(policies, [5]string{record[0], record[1], record[2], record[3], record[4]})
	}

	p.policies = policies

	return nil
}

func (p *provisioner) addPolicies() error {
	domain := fmt.Sprintf("%s:%s", models.DomainTypeOrganization, p.orgID)

	for _, policy := range p.policies {
		err := p.addPolicy(policy, domain)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *provisioner) addPolicy(policy [5]string, domain string) error {
	switch policy[0] {
	case "g":
		_, err := p.enforcer.AddGroupingPolicy(policy[1], policy[2], domain)
		if err != nil {
			return fmt.Errorf("failed to add grouping policy: %w", err)
		}
	case "p":
		_, err := p.enforcer.AddPolicy(policy[1], domain, policy[3], policy[4])
		if err != nil {
			return fmt.Errorf("failed to add policy: %w", err)
		}
	default:
		return fmt.Errorf("unknown policy type: %s", policy[0])
	}

	return nil
}

func (p *provisioner) addOwner() error {
	domain := fmt.Sprintf("%s:%s", models.DomainTypeOrganization, p.orgID)
	role := fmt.Sprintf("role:%s", models.RoleOrgOwner)
	user := fmt.Sprintf("user:%s", p.ownerID)

	_, err := p.enforcer.AddGroupingPolicy(user, role, domain)
	if err != nil {
		return fmt.Errorf("failed to add role: %w", err)
	}

	return nil
}
