package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
)

const (
	AppInstallationStatePending    = "pending"
	AppInstallationStateInProgress = "in-progress"
	AppInstallationStateReady      = "ready"
	AppInstallationStateError      = "error"
)

type AppInstallation struct {
	ID               uuid.UUID `gorm:"primary_key;default:uuid_generate_v4()"`
	OrganizationID   uuid.UUID
	AppName          string
	InstallationName string
	State            string
	Configuration    datatypes.JSONType[map[string]any]
	Metadata         datatypes.JSONType[map[string]any]
	BrowserAction    *datatypes.JSONType[BrowserAction]
	CreatedAt        *time.Time
	UpdatedAt        *time.Time
}

// TODO: this is copied from pkg/applications here,
// because there is a circular dependency issue that was
// introduced by having pkg/components require pkg/models.
type BrowserAction struct {
	URL        string
	Method     string
	FormFields map[string]string
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

func FindAppInstallation(installationID uuid.UUID) (*AppInstallation, error) {
	var appInstallation AppInstallation
	err := database.Conn().
		Where("id = ?", installationID).
		First(&appInstallation).
		Error

	if err != nil {
		return nil, err
	}

	return &appInstallation, nil
}
