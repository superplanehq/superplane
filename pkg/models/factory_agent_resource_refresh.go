package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrFactoryAgentResourceRefreshBusy = errors.New("MCP OAuth refresh is already in progress")

// FactoryAgentResourceRefresh leases one rotating OAuth grant across gateways.
// Provider requests run after the transaction that claims the lease commits.
type FactoryAgentResourceRefresh struct {
	ResourceID uuid.UUID `gorm:"primaryKey"`
	AttemptID  uuid.UUID
	ExpiresAt  time.Time
	UpdatedAt  time.Time
}

func (FactoryAgentResourceRefresh) TableName() string {
	return "factory_agent_resource_refreshes"
}

func (r *FactoryAgentResource) ClaimOAuthRefresh(tx *gorm.DB, now, expiresAt time.Time) (*FactoryAgentResourceRefresh, error) {
	lease := &FactoryAgentResourceRefresh{ResourceID: r.ID, AttemptID: uuid.New(), ExpiresAt: expiresAt, UpdatedAt: now}
	result := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "resource_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"attempt_id", "expires_at", "updated_at"}),
		Where:     clause.Where{Exprs: []clause.Expression{clause.Lte{Column: clause.Column{Table: "factory_agent_resource_refreshes", Name: "expires_at"}, Value: now}}},
	}).Create(lease)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, ErrFactoryAgentResourceRefreshBusy
	}
	return lease, nil
}

// LockOAuthRefreshSnapshot uses the same factory-before-resource order as
// resource mutation. The caller must keep this transaction free of provider I/O.
func (r *FactoryAgentResource) LockOAuthRefreshSnapshot(tx *gorm.DB) (*FactoryAgentResource, []FactoryAgentResourceSecret, error) {
	factory := &Factory{ID: r.FactoryID, OrganizationID: r.OrganizationID}
	if err := factory.lockAgentResourceCapacity(tx); err != nil {
		return nil, nil, err
	}
	var fresh FactoryAgentResource
	err := tx.Session(&gorm.Session{NewDB: true}).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND organization_id = ? AND factory_id = ?", r.ID, r.OrganizationID, r.FactoryID).First(&fresh).Error
	if err != nil {
		return nil, nil, err
	}
	var secrets []FactoryAgentResourceSecret
	err = tx.Session(&gorm.Session{NewDB: true}).Clauses(clause.Locking{Strength: "UPDATE"}).Where("resource_id = ?", r.ID).Find(&secrets).Error
	return &fresh, secrets, err
}

func (l *FactoryAgentResourceRefresh) Lock(tx *gorm.DB, now time.Time) error {
	var current FactoryAgentResourceRefresh
	err := tx.Session(&gorm.Session{NewDB: true}).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("resource_id = ? AND attempt_id = ? AND expires_at > ?", l.ResourceID, l.AttemptID, now).First(&current).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrFactoryAgentResourceRefreshBusy
	}
	return err
}

func (l *FactoryAgentResourceRefresh) Release(tx *gorm.DB) error {
	return tx.Where("resource_id = ? AND attempt_id = ?", l.ResourceID, l.AttemptID).Delete(&FactoryAgentResourceRefresh{}).Error
}
