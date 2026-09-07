package github

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

var generatedGitHubInstallationName = regexp.MustCompile(`^github(-[0-9]+)?$`)

const maxGitHubInstallationNameAttempts = 50

// IsGeneratedInstallationName reports whether the name is the placeholder
// created before GitHub bind (github, github-2, ...).
func IsGeneratedInstallationName(name string) bool {
	return generatedGitHubInstallationName.MatchString(strings.TrimSpace(name))
}

// OwnerInstallationName is the preferred display name after bind.
func OwnerInstallationName(owner string) string {
	return "github-" + strings.ToLower(strings.TrimSpace(owner))
}

// NextOwnerInstallationName returns github-<owner>, or github-<owner> (N)
// when that name is taken.
func NextOwnerInstallationName(owner string, taken func(string) bool) string {
	base := OwnerInstallationName(owner)
	if base == "github-" || taken == nil || !taken(base) {
		return base
	}

	for suffix := 1; suffix <= maxGitHubInstallationNameAttempts; suffix++ {
		candidate := fmt.Sprintf("%s (%d)", base, suffix)
		if !taken(candidate) {
			return candidate
		}
	}

	return fmt.Sprintf("%s (%d)", base, maxGitHubInstallationNameAttempts+1)
}

// RenameGeneratedInstallation names a GitHub connection after the bound
// owner when the current name is still the generated placeholder.
func RenameGeneratedInstallation(tx *gorm.DB, integration *models.Integration) error {
	if tx == nil || integration == nil || integration.AppName != "github" {
		return nil
	}
	if !IsGeneratedInstallationName(integration.InstallationName) {
		return nil
	}

	owner := strings.TrimSpace(githubOwnerFromMetadata(integration.Metadata.Data()))
	if owner == "" {
		return nil
	}

	if err := lockOrganizationInstallationNames(tx, integration.OrganizationID); err != nil {
		return err
	}

	name := NextOwnerInstallationName(owner, func(candidate string) bool {
		return installationNameTaken(tx, integration.OrganizationID, integration.ID, candidate)
	})
	integration.InstallationName = name
	return nil
}

func lockOrganizationInstallationNames(tx *gorm.DB, organizationID uuid.UUID) error {
	sum := sha256.Sum256(append([]byte("github-installation-name:"), organizationID[:]...))
	key := int64(binary.BigEndian.Uint64(sum[:8]))
	return tx.Exec("SELECT pg_advisory_xact_lock(?)", key).Error
}

func installationNameTaken(tx *gorm.DB, organizationID, currentID uuid.UUID, name string) bool {
	existing, err := models.FindIntegrationByName(tx, organizationID, name)
	if err != nil {
		return false
	}
	return existing.ID != currentID
}

func githubOwnerFromMetadata(metadata map[string]any) string {
	if metadata == nil {
		return ""
	}
	owner, _ := metadata["owner"].(string)
	return owner
}
