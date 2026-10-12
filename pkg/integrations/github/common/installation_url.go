package common

import (
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

func HostedInstallationSettingsURL(tx *gorm.DB, integrationID uuid.UUID) (string, error) {
	binding, err := models.FindVCSProviderIntegrationBinding(tx, integrationID)
	if err != nil {
		return "", err
	}

	installation, err := models.FindVCSProviderInstallation(tx, binding.Provider, binding.InstallationID)
	if err != nil {
		return "", err
	}

	return InstallationSettingsURL(*installation), nil
}
