package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	IntegrationRequestTypeSync         = "sync"
	IntegrationRequestTypeInvokeAction = "invoke-action"

	IntegrationRequestStatePending   = "pending"
	IntegrationRequestStateCompleted = "completed"
)

type IntegrationRequest struct {
	ID                uuid.UUID
	AppInstallationID uuid.UUID
	State             string
	Type              string
	Spec              datatypes.JSONType[IntegrationRequestSpec]
	RunAt             time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (r *IntegrationRequest) TableName() string {
	return "app_installation_requests"
}

type IntegrationRequestSpec struct {
	InvokeAction *IntegrationInvokeAction `json:"invoke_action,omitempty"`
}

type IntegrationInvokeAction struct {
	ActionName string `json:"action_name"`
	Parameters any    `json:"parameters"`
}

// LeaseIntegrationRequest atomically leases a due pending request for processing.
// It locks the row with SKIP LOCKED, gated on the request being pending and due
// (run_at <= now), then pushes run_at past the work window. The poll loop only
// lists due pending requests, so the leased request drops out until either it is
// completed or the lease expires - at which point it becomes due again and is
// retried automatically, with no separate state or reset mechanism.
func LeaseIntegrationRequest(tx *gorm.DB, id uuid.UUID, lease time.Duration) (*IntegrationRequest, error) {
	var request IntegrationRequest

	now := time.Now()
	err := tx.
		Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where("id = ?", id).
		Where("state = ?", IntegrationRequestStatePending).
		Where("run_at <= ?", now).
		First(&request).
		Error
	if err != nil {
		return nil, err
	}

	leasedUntil := now.Add(lease)
	err = tx.Model(&request).
		Update("run_at", leasedUntil).
		Update("updated_at", now).
		Error
	if err != nil {
		return nil, err
	}

	request.RunAt = leasedUntil
	request.UpdatedAt = now
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
