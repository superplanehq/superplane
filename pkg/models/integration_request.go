package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	IntegrationRequestTypeSync = "sync"

	IntegrationRequestStatePending   = "pending"
	IntegrationRequestStateCompleted = "completed"
)

type IntegrationRequest struct {
	ID                uuid.UUID
	AppInstallationID uuid.UUID
	State             string
	Type              string
	RunAt             time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (r *IntegrationRequest) TableName() string {
	return "app_installation_requests"
}

func LockIntegrationRequest(tx *gorm.DB, id uuid.UUID) (*IntegrationRequest, error) {
	var request IntegrationRequest

	err := tx.
		Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where("id = ?", id).
		First(&request).
		Error
	if err != nil {
		return nil, err
	}

	return &request, nil
}

func ListIntegrationRequests() ([]IntegrationRequest, error) {
	var requests []IntegrationRequest

	now := time.Now()
	err := database.Conn().
		Joins("JOIN app_installations ON app_installation_requests.app_installation_id = app_installations.id").
		Where("app_installation_requests.state = ?", IntegrationRequestStatePending).
		Where("app_installation_requests.run_at <= ?", now).
		Where("app_installations.deleted_at IS NULL").
		Find(&requests).
		Error
	if err != nil {
		return nil, err
	}

	return requests, nil
}

func FindPendingRequestForIntegration(tx *gorm.DB, installationID uuid.UUID) (*IntegrationRequest, error) {
	var request IntegrationRequest

	err := tx.
		Where("app_installation_id = ?", installationID).
		Where("state = ?", IntegrationRequestStatePending).
		First(&request).
		Error
	if err != nil {
		return nil, err
	}

	return &request, nil
}

func (r *IntegrationRequest) Complete(tx *gorm.DB) error {
	return tx.Model(r).
		Update("state", IntegrationRequestStateCompleted).
		Update("updated_at", time.Now()).
		Error
}
