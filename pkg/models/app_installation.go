package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
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

// TODO: this is copied from pkg/applications here,
// because there is a circular dependency issue that was
// introduced by having pkg/components require pkg/models.
type BrowserAction struct {
	URL         string
	Method      string
	FormFields  map[string]string
	Description string
}

func CreateAppInstallation(orgID uuid.UUID, appName string, installationName string, config map[string]any) (*AppInstallation, error) {
	now := time.Now()
	appInstallation := AppInstallation{
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
