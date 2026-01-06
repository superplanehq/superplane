package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	AppInstallationRequestTypeSync = "sync"

	AppInstallationRequestStatePending   = "pending"
	AppInstallationRequestStateCompleted = "completed"
)

type AppInstallationRequest struct {
	ID                uuid.UUID
	AppInstallationID uuid.UUID
	State             string
	Type              string
	RunAt             time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func LockAppInstallationRequest(tx *gorm.DB, id uuid.UUID) (*AppInstallationRequest, error) {
	var request AppInstallationRequest

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

func ListAppInstallationRequests() ([]AppInstallationRequest, error) {
	var requests []AppInstallationRequest

	now := time.Now()
	err := database.Conn().
		Joins("JOIN app_installations ON app_installation_requests.app_installation_id = app_installations.id").
		Where("app_installation_requests.state = ?", AppInstallationRequestStatePending).
		Where("app_installation_requests.run_at <= ?", now).
		Where("app_installations.deleted_at IS NULL").
		Find(&requests).
		Error
	if err != nil {
		return nil, err
	}

	return requests, nil
}

func FindPendingRequestForAppInstallation(tx *gorm.DB, installationID uuid.UUID) (*AppInstallationRequest, error) {
	var request AppInstallationRequest

	err := tx.
		Where("app_installation_id = ?", installationID).
		Where("state = ?", AppInstallationRequestStatePending).
		First(&request).
		Error
	if err != nil {
		return nil, err
	}

	return &request, nil
}

func (r *AppInstallationRequest) Complete(tx *gorm.DB) error {
	return tx.Model(r).
		Update("state", AppInstallationRequestStateCompleted).
		Update("updated_at", time.Now()).
		Error
}
