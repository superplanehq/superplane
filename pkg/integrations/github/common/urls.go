package common

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/core"
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

		return fmt.Sprintf("https://github.com/organizations/%s/settings/apps/%s/permissions", owner, appSlug), nil
	}

	return fmt.Sprintf("https://github.com/settings/apps/%s/permissions", appSlug), nil
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

		return fmt.Sprintf("https://github.com/organizations/%s/settings/installations/%s", owner, installationID), nil
	}

	return fmt.Sprintf("https://github.com/settings/installations/%s", installationID), nil
}
