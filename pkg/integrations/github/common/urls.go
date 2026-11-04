package common

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
)

const (
	gitHubAccountTypeOrganization = "Organization"
	gitHubAccountTypeUser         = "User"
)

func AppURL(properties core.IntegrationPropertyStorageReader, appSlug string) (string, error) {
	ownerType, err := properties.GetString(PropertyOwnerType)
	if err != nil {
		return "", fmt.Errorf("error getting owner type: %v", err)
	}

	if ownerType == OwnerTypeOrganization {
		owner, err := properties.GetString(PropertyOwner)
		if err != nil {
			return "", fmt.Errorf("error getting owner: %v", err)
		}

		return HostFromProperties(properties).AppPermissionsURL(ownerType, owner, appSlug), nil
	}

	return HostFromProperties(properties).AppPermissionsURL(ownerType, "", appSlug), nil
}

func AppInstallationURL(properties core.IntegrationPropertyStorageReader, installationID string) (string, error) {
	ownerType, err := properties.GetString(PropertyOwnerType)
	if err != nil {
		return "", fmt.Errorf("error getting owner type: %v", err)
	}

	if ownerType == OwnerTypeOrganization {
		owner, err := properties.GetString(PropertyOwner)
		if err != nil {
			return "", fmt.Errorf("error getting owner: %v", err)
		}

		return HostFromProperties(properties).InstallationSettingsURL(ownerType, owner, installationID), nil
	}

	return HostFromProperties(properties).InstallationSettingsURL(ownerType, "", installationID), nil
}

// InstallationSettingsURL is the GitHub page where an admin manages this App
// installation. GitHub's html_url wins. Account login and type fill the same
// page when catalog sync has not stored that URL yet.
func InstallationSettingsURL(installation models.VCSProviderInstallation) string {
	if url := strings.TrimSpace(installation.HTMLURL); url != "" {
		return url
	}
	if installation.InstallationID <= 0 {
		return ""
	}

	installationID := strconv.FormatInt(installation.InstallationID, 10)
	login := strings.TrimSpace(installation.AccountLogin)
	switch installation.AccountType {
	case gitHubAccountTypeOrganization:
		if login == "" {
			return ""
		}
		return DotCom().InstallationSettingsURL(OwnerTypeOrganization, login, installationID)
	case gitHubAccountTypeUser:
		return DotCom().InstallationSettingsURL(OwnerTypeUser, login, installationID)
	default:
		return ""
	}
}
