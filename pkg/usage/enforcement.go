package usage

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

func CheckOrgCreationLimit(accountEmail string) error {
	return CheckOrgCreationLimitInTransaction(database.Conn(), accountEmail)
}

func CheckOrgCreationLimitInTransaction(tx *gorm.DB, accountEmail string) error {
	if isEnforcementDisabled() {
		return nil
	}

	orgs, err := models.FindOrganizationsForAccountInTransaction(tx, accountEmail)
	if err != nil {
		return err
	}

	if len(orgs) == 0 {
		return nil
	}

	limits, err := ResolveEffectiveLimitsInTransaction(tx, orgs[0].ID)
	if err != nil {
		return err
	}

	if limits.IsUnlimited {
		return nil
	}

	if int64(len(orgs)) >= int64(limits.MaxOrgsPerAccount) {
		return fmt.Errorf("organization limit reached: maximum %d organizations per account", limits.MaxOrgsPerAccount)
	}

	return nil
}

func CheckCanvasCreationLimit(orgID uuid.UUID) error {
	return CheckCanvasCreationLimitInTransaction(database.Conn(), orgID)
}

func CheckCanvasCreationLimitInTransaction(tx *gorm.DB, orgID uuid.UUID) error {
	if isEnforcementDisabled() {
		return nil
	}

	limits, err := ResolveEffectiveLimitsInTransaction(tx, orgID)
	if err != nil {
		return err
	}

	if limits.IsUnlimited {
		return nil
	}

	var count int64
	err = tx.Model(&models.Canvas{}).
		Where("organization_id = ?", orgID).
		Where("deleted_at IS NULL").
		Count(&count).Error
	if err != nil {
		return err
	}

	if count >= int64(limits.MaxCanvasesPerOrg) {
		return fmt.Errorf("canvas limit reached: maximum %d canvases per organization", limits.MaxCanvasesPerOrg)
	}

	return nil
}

func CheckNodeLimit(orgID uuid.UUID, canvasID uuid.UUID, additionalNodes int) error {
	return CheckNodeLimitInTransaction(database.Conn(), orgID, canvasID, additionalNodes)
}

func CheckNodeLimitInTransaction(tx *gorm.DB, orgID uuid.UUID, canvasID uuid.UUID, additionalNodes int) error {
	if isEnforcementDisabled() {
		return nil
	}

	limits, err := ResolveEffectiveLimitsInTransaction(tx, orgID)
	if err != nil {
		return err
	}

	if limits.IsUnlimited {
		return nil
	}

	var count int64
	err = tx.Model(&models.CanvasNode{}).
		Where("workflow_id = ?", canvasID).
		Where("deleted_at IS NULL").
		Count(&count).Error
	if err != nil {
		return err
	}

	if count+int64(additionalNodes) > int64(limits.MaxNodesPerCanvas) {
		return fmt.Errorf("node limit reached: maximum %d nodes per canvas", limits.MaxNodesPerCanvas)
	}

	return nil
}

func CheckMemberLimit(orgID uuid.UUID) error {
	return CheckMemberLimitInTransaction(database.Conn(), orgID)
}

func CheckMemberLimitInTransaction(tx *gorm.DB, orgID uuid.UUID) error {
	if isEnforcementDisabled() {
		return nil
	}

	limits, err := ResolveEffectiveLimitsInTransaction(tx, orgID)
	if err != nil {
		return err
	}

	if limits.IsUnlimited {
		return nil
	}

	var count int64
	err = tx.Model(&models.User{}).
		Where("organization_id = ?", orgID).
		Where("deleted_at IS NULL").
		Count(&count).Error
	if err != nil {
		return err
	}

	if count >= int64(limits.MaxUsersPerOrg) {
		return fmt.Errorf("member limit reached: maximum %d users per organization", limits.MaxUsersPerOrg)
	}

	return nil
}

func CheckIntegrationLimit(orgID uuid.UUID) error {
	return CheckIntegrationLimitInTransaction(database.Conn(), orgID)
}

func CheckIntegrationLimitInTransaction(tx *gorm.DB, orgID uuid.UUID) error {
	if isEnforcementDisabled() {
		return nil
	}

	limits, err := ResolveEffectiveLimitsInTransaction(tx, orgID)
	if err != nil {
		return err
	}

	if limits.IsUnlimited {
		return nil
	}

	var count int64
	err = tx.Model(&models.Integration{}).
		Where("organization_id = ?", orgID).
		Where("deleted_at IS NULL").
		Count(&count).Error
	if err != nil {
		return err
	}

	if count >= int64(limits.MaxIntegrationsPerOrg) {
		return fmt.Errorf("integration limit reached: maximum %d integrations per organization", limits.MaxIntegrationsPerOrg)
	}

	return nil
}
