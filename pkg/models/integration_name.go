package models

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const maxUniqueInstallationNameAttempts = 50

// AssignUniqueInstallationName sets InstallationName to name, or name (1),
// name (2), when another integration in the organization already uses it.
// The advisory lock is held until the caller commits tx.
func (a *Integration) AssignUniqueInstallationName(tx *gorm.DB, name string) error {
	if tx == nil || a == nil {
		return nil
	}

	base := strings.TrimSpace(name)
	if base == "" {
		return nil
	}

	if err := lockOrganizationInstallationNames(tx, a.OrganizationID); err != nil {
		return err
	}

	a.InstallationName = nextUniqueInstallationName(base, func(candidate string) bool {
		return installationNameTaken(tx, a.OrganizationID, a.ID, candidate)
	})
	return nil
}

func lockOrganizationInstallationNames(tx *gorm.DB, organizationID uuid.UUID) error {
	sum := sha256.Sum256(append([]byte("installation-name:"), organizationID[:]...))
	key := int64(binary.BigEndian.Uint64(sum[:8]))
	return tx.Exec("SELECT pg_advisory_xact_lock(?)", key).Error
}

func nextUniqueInstallationName(base string, taken func(string) bool) string {
	if taken == nil || !taken(base) {
		return base
	}

	for suffix := 1; suffix <= maxUniqueInstallationNameAttempts; suffix++ {
		candidate := fmt.Sprintf("%s (%d)", base, suffix)
		if !taken(candidate) {
			return candidate
		}
	}

	return fmt.Sprintf("%s (%d)", base, maxUniqueInstallationNameAttempts+1)
}

func installationNameTaken(tx *gorm.DB, organizationID, currentID uuid.UUID, name string) bool {
	existing, err := FindIntegrationByName(tx, organizationID, name)
	if err != nil {
		return false
	}
	return existing.ID != currentID
}
