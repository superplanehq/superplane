package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	AppInstallationStatePending = "pending"
	AppInstallationStateReady   = "ready"
	AppInstallationStateError   = "error"
)

type AppInstallation struct {
	ID               uuid.UUID `gorm:"primary_key;default:uuid_generate_v4()"`
	OrganizationID   uuid.UUID
	AppName          string
	InstallationName string
	State            string
	StateDescription string
	Configuration    datatypes.JSONType[map[string]any]
	Metadata         datatypes.JSONType[map[string]any]
	BrowserAction    *datatypes.JSONType[BrowserAction]
	CreatedAt        *time.Time
	UpdatedAt        *time.Time
	DeletedAt        gorm.DeletedAt `gorm:"index"`
}

type AppInstallationSecret struct {
	ID             uuid.UUID `gorm:"primary_key;default:uuid_generate_v4()"`
	OrganizationID uuid.UUID
	InstallationID uuid.UUID
	Name           string
	Value          []byte
	CreatedAt      *time.Time
	UpdatedAt      *time.Time
}

type BrowserAction struct {
	URL         string
	Method      string
	FormFields  map[string]string
	Description string
}

func CreateAppInstallation(id, orgID uuid.UUID, appName string, installationName string, config map[string]any) (*AppInstallation, error) {
	now := time.Now()
	appInstallation := AppInstallation{
		ID:               id,
		OrganizationID:   orgID,
		AppName:          appName,
		InstallationName: installationName,
		State:            AppInstallationStatePending,
		Configuration:    datatypes.NewJSONType(config),
		CreatedAt:        &now,
		UpdatedAt:        &now,
	}

	err := database.Conn().Create(&appInstallation).Error
	if err != nil {
		return nil, err
	}

	return &appInstallation, nil
}

func ListAppInstallations(orgID uuid.UUID) ([]AppInstallation, error) {
	var appInstallations []AppInstallation
	err := database.Conn().Where("organization_id = ?", orgID).Find(&appInstallations).Error
	if err != nil {
		return nil, err
	}
	return appInstallations, nil
}

func ListAppInstallationWebhooks(tx *gorm.DB, installationID uuid.UUID) ([]Webhook, error) {
	var webhooks []Webhook
	err := tx.
		Where("app_installation_id = ?", installationID).
		Find(&webhooks).
		Error

	if err != nil {
		return nil, err
	}
	return webhooks, nil
}

func ListUnscopedAppInstallationWebhooks(tx *gorm.DB, installationID uuid.UUID) ([]Webhook, error) {
	var webhooks []Webhook
	err := tx.Unscoped().
		Where("app_installation_id = ?", installationID).
		Find(&webhooks).
		Error

	if err != nil {
		return nil, err
	}
	return webhooks, nil
}

type WorkflowNodeReference struct {
	WorkflowID   uuid.UUID
	WorkflowName string
	NodeID       string
	NodeName     string
}

func ListAppInstallationNodeReferences(installationID uuid.UUID) ([]WorkflowNodeReference, error) {
	var nodeReferences []WorkflowNodeReference
	err := database.Conn().
		Table("workflow_nodes AS wn").
		Joins("JOIN workflows AS w ON w.id = wn.workflow_id").
		Select("w.id as workflow_id, w.name as workflow_name, wn.node_id as node_id, wn.name as node_name").
		Where("wn.app_installation_id = ?", installationID).
		Where("wn.deleted_at IS NULL").
		Find(&nodeReferences).
		Error

	if err != nil {
		return nil, err
	}
	return nodeReferences, nil
}

func FindUnscopedAppInstallation(installationID uuid.UUID) (*AppInstallation, error) {
	return FindUnscopedAppInstallationInTransaction(database.Conn(), installationID)
}

func FindUnscopedAppInstallationInTransaction(tx *gorm.DB, installationID uuid.UUID) (*AppInstallation, error) {
	var appInstallation AppInstallation
	err := tx.
		Where("id = ?", installationID).
		First(&appInstallation).
		Error

	if err != nil {
		return nil, err
	}

	return &appInstallation, nil
}

func FindMaybeDeletedInstallationInTransaction(tx *gorm.DB, installationID uuid.UUID) (*AppInstallation, error) {
	var appInstallation AppInstallation
	err := tx.Unscoped().
		Where("id = ?", installationID).
		First(&appInstallation).
		Error

	if err != nil {
		return nil, err
	}

	return &appInstallation, nil
}

func FindAppInstallation(orgID, installationID uuid.UUID) (*AppInstallation, error) {
	var appInstallation AppInstallation
	err := database.Conn().
		Where("id = ?", installationID).
		Where("organization_id = ?", orgID).
		First(&appInstallation).
		Error

	if err != nil {
		return nil, err
	}

	return &appInstallation, nil
}

func FindAppInstallationByName(orgID uuid.UUID, installationName string) (*AppInstallation, error) {
	var appInstallation AppInstallation
	err := database.Conn().
		Where("organization_id = ?", orgID).
		Where("installation_name = ?", installationName).
		First(&appInstallation).
		Error

	if err != nil {
		return nil, err
	}

	return &appInstallation, nil
}

func ListDeletedAppInstallations() ([]AppInstallation, error) {
	var installations []AppInstallation
	err := database.Conn().Unscoped().
		Where("deleted_at IS NOT NULL").
		Find(&installations).
		Error

	if err != nil {
		return nil, err
	}

	return installations, nil
}

func LockAppInstallation(tx *gorm.DB, ID uuid.UUID) (*AppInstallation, error) {
	var installation AppInstallation

	err := tx.Unscoped().
		Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where("id = ?", ID).
		First(&installation).
		Error

	if err != nil {
		return nil, err
	}

	return &installation, nil
}

func (a *AppInstallation) SoftDelete() error {
	return a.SoftDeleteInTransaction(database.Conn())
}

func (a *AppInstallation) SoftDeleteInTransaction(tx *gorm.DB) error {
	now := time.Now()
	timestamp := now.Unix()

	newName := fmt.Sprintf("%s (deleted-%d)", a.InstallationName, timestamp)
	return tx.Model(a).Updates(map[string]interface{}{
		"deleted_at":        now,
		"installation_name": newName,
	}).Error
}

func (a *AppInstallation) GetRequest(ID string) (*AppInstallationRequest, error) {
	var request AppInstallationRequest

	err := database.Conn().
		Where("id = ?", ID).
		Where("app_installation_id = ?", a.ID).
		First(&request).
		Error

	if err != nil {
		return nil, err
	}

	return &request, nil
}

func (a *AppInstallation) ListRequests(reqType string) ([]AppInstallationRequest, error) {
	requests := []AppInstallationRequest{}

	err := database.Conn().
		Where("app_installation_id = ?", a.ID).
		Find(&requests).
		Error

	if err != nil {
		return nil, err
	}

	return requests, nil
}

func (a *AppInstallation) CreateSyncRequest(tx *gorm.DB, runAt *time.Time) error {
	now := time.Now()
	return tx.Create(&AppInstallationRequest{
		ID:                uuid.New(),
		AppInstallationID: a.ID,
		State:             AppInstallationRequestStatePending,
		Type:              AppInstallationRequestTypeSync,
		RunAt:             *runAt,
		CreatedAt:         now,
		UpdatedAt:         now,
	}).Error
}
