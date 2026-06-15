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

	IntegrationRequestStatePending    = "pending"
	IntegrationRequestStateProcessing = "processing"
	IntegrationRequestStateCompleted  = "completed"
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

// ClaimIntegrationRequest atomically claims a pending request for processing.
// It locks the row with SKIP LOCKED, gated on the pending state, and flips it to
// processing so the poll loop (which only lists pending requests) will not pick
// it up again while the external work runs outside this transaction.
func ClaimIntegrationRequest(tx *gorm.DB, id uuid.UUID) (*IntegrationRequest, error) {
	var request IntegrationRequest

	err := tx.
		Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where("id = ?", id).
		Where("state = ?", IntegrationRequestStatePending).
		First(&request).
		Error
	if err != nil {
		return nil, err
	}

	now := time.Now()
	err = tx.Model(&request).
		Update("state", IntegrationRequestStateProcessing).
		Update("updated_at", now).
		Error
	if err != nil {
		return nil, err
	}

	request.State = IntegrationRequestStateProcessing
	request.UpdatedAt = now
	return &request, nil
}

// ResetStuckProcessingIntegrationRequests returns requests stuck in the
// processing state (e.g. after a crash) back to pending so they get retried.
// The age cutoff avoids resetting requests still in flight on another replica
// during a rolling deploy.
func ResetStuckProcessingIntegrationRequests(olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)
	result := database.Conn().
		Model(&IntegrationRequest{}).
		Where("state = ?", IntegrationRequestStateProcessing).
		Where("updated_at < ?", cutoff).
		Updates(map[string]any{
			"state":      IntegrationRequestStatePending,
			"updated_at": time.Now(),
		})

	return result.RowsAffected, result.Error
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
