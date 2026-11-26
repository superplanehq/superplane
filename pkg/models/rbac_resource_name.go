package models

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"

	"gorm.io/gorm"
)

// LockRBACResourceName serializes creation across processes until tx ends.
func LockRBACResourceName(tx *gorm.DB, kind, domainType, domainID, name string) error {
	sum := sha256.Sum256([]byte(fmt.Sprintf("rbac-name:%q:%q:%q:%q", kind, domainType, domainID, name)))
	key := int64(binary.BigEndian.Uint64(sum[:8]))
	return tx.Exec("SELECT pg_advisory_xact_lock(?)", key).Error
}

func RBACResourceMetadataExists(tx *gorm.DB, kind, domainType, domainID, name string) (bool, error) {
	var count int64
	var err error
	switch kind {
	case "role":
		err = tx.Model(&RoleMetadata{}).Where("role_name = ? AND domain_type = ? AND domain_id = ?", name, domainType, domainID).Count(&count).Error
	case "group":
		err = tx.Model(&GroupMetadata{}).Where("group_name = ? AND domain_type = ? AND domain_id = ?", name, domainType, domainID).Count(&count).Error
	default:
		return false, fmt.Errorf("invalid RBAC resource kind: %s", kind)
	}
	return count > 0, err
}
