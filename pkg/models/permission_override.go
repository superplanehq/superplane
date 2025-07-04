package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
)

type RolePermissionOverride struct {
	ID             uuid.UUID  `gorm:"primary_key;default:uuid_generate_v4()"`
	OrganizationID *uuid.UUID `gorm:"index"`
	CanvasID       *uuid.UUID `gorm:"index"`
	RoleName       string     `gorm:"index;size:100"`
	Resource       string     `gorm:"size:100"`
	Action         string     `gorm:"size:100"`
	IsActive       bool       `gorm:"index"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	CreatedBy      *uuid.UUID

	Organization *Organization `gorm:"foreignKey:OrganizationID;references:ID"`
	Canvas       *Canvas       `gorm:"foreignKey:CanvasID;references:ID"`
}

func (RolePermissionOverride) TableName() string {
	return "role_permission_overrides"
}

type RoleHierarchyOverride struct {
	ID             uuid.UUID  `gorm:"primary_key;default:uuid_generate_v4()"`
	OrganizationID *uuid.UUID `gorm:"index"`
	CanvasID       *uuid.UUID `gorm:"index"`
	ChildRole      string     `gorm:"index;size:100"`
	ParentRole     string     `gorm:"size:100"`
	IsActive       bool       `gorm:"index"`
	CreatedAt      time.Time
	CreatedBy      *uuid.UUID

	Organization *Organization `gorm:"foreignKey:OrganizationID;references:ID"`
	Canvas       *Canvas       `gorm:"foreignKey:CanvasID;references:ID"`
}

func (RoleHierarchyOverride) TableName() string {
	return "role_hierarchy_overrides"
}

// CreatePermissionOverride creates a new permission override
func CreatePermissionOverride(organizationID, canvasID *uuid.UUID, roleName, resource, action string, isActive bool, createdBy uuid.UUID) (*RolePermissionOverride, error) {
	override := &RolePermissionOverride{
		OrganizationID: organizationID,
		CanvasID:       canvasID,
		RoleName:       roleName,
		Resource:       resource,
		Action:         action,
		IsActive:       isActive,
		CreatedBy:      &createdBy,
	}

	err := database.Conn().Create(override).Error
	if err != nil {
		return nil, err
	}

	return override, nil
}

// GetPermissionOverrides gets active permission overrides for a specific domain
func GetPermissionOverrides(organizationID, canvasID *uuid.UUID) ([]RolePermissionOverride, error) {
	var overrides []RolePermissionOverride

	query := database.Conn().Where("is_active = ?", true)

	if organizationID != nil {
		query = query.Where("organization_id = ? AND canvas_id IS NULL", *organizationID)
	} else if canvasID != nil {
		query = query.Where("canvas_id = ? AND organization_id IS NULL", *canvasID)
	} else {
		return nil, nil // Must specify either org or canvas
	}

	err := query.Find(&overrides).Error
	return overrides, err
}

// GetAllPermissionOverrides gets all permission overrides (active and inactive) for a specific domain
func GetAllPermissionOverrides(organizationID, canvasID *uuid.UUID) ([]RolePermissionOverride, error) {
	var overrides []RolePermissionOverride

	query := database.Conn()

	if organizationID != nil {
		query = query.Where("organization_id = ? AND canvas_id IS NULL", *organizationID)
	} else if canvasID != nil {
		query = query.Where("canvas_id = ? AND organization_id IS NULL", *canvasID)
	} else {
		return nil, nil // Must specify either org or canvas
	}

	err := query.Find(&overrides).Error
	return overrides, err
}

// GetPermissionOverridesByRole gets permission overrides for a specific role in a domain
func GetPermissionOverridesByRole(organizationID, canvasID *uuid.UUID, roleName string) ([]RolePermissionOverride, error) {
	var overrides []RolePermissionOverride

	query := database.Conn().Where("is_active = ? AND role_name = ?", true, roleName)

	if organizationID != nil {
		query = query.Where("organization_id = ? AND canvas_id IS NULL", *organizationID)
	} else if canvasID != nil {
		query = query.Where("canvas_id = ? AND organization_id IS NULL", *canvasID)
	} else {
		return nil, nil // Must specify either org or canvas
	}

	err := query.Find(&overrides).Error
	return overrides, err
}

// UpdatePermissionOverride updates an existing permission override
func UpdatePermissionOverride(id uuid.UUID, isActive bool) error {
	result := database.Conn().Model(&RolePermissionOverride{}).
		Where("id = ?", id).
		Update("is_active", isActive)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("no rows affected when updating override %s", id)
	}

	return nil
}

// DeletePermissionOverride soft deletes a permission override by setting is_active to false
func DeletePermissionOverride(id uuid.UUID) error {
	return UpdatePermissionOverride(id, false)
}

// CreateHierarchyOverride creates a new role hierarchy override
func CreateHierarchyOverride(organizationID, canvasID *uuid.UUID, childRole, parentRole string, isActive bool, createdBy uuid.UUID) (*RoleHierarchyOverride, error) {
	override := &RoleHierarchyOverride{
		OrganizationID: organizationID,
		CanvasID:       canvasID,
		ChildRole:      childRole,
		ParentRole:     parentRole,
		IsActive:       isActive,
		CreatedBy:      &createdBy,
	}

	err := database.Conn().Create(override).Error
	if err != nil {
		return nil, err
	}

	return override, nil
}

// GetHierarchyOverrides gets active hierarchy overrides for a specific domain
func GetHierarchyOverrides(organizationID, canvasID *uuid.UUID) ([]RoleHierarchyOverride, error) {
	var overrides []RoleHierarchyOverride

	query := database.Conn().Where("is_active = ?", true)

	if organizationID != nil {
		query = query.Where("organization_id = ? AND canvas_id IS NULL", *organizationID)
	} else if canvasID != nil {
		query = query.Where("canvas_id = ? AND organization_id IS NULL", *canvasID)
	} else {
		return nil, nil // Must specify either org or canvas
	}

	err := query.Find(&overrides).Error
	return overrides, err
}

// GetAllHierarchyOverrides gets all hierarchy overrides (active and inactive) for a specific domain
func GetAllHierarchyOverrides(organizationID, canvasID *uuid.UUID) ([]RoleHierarchyOverride, error) {
	var overrides []RoleHierarchyOverride

	query := database.Conn()

	if organizationID != nil {
		query = query.Where("organization_id = ? AND canvas_id IS NULL", *organizationID)
	} else if canvasID != nil {
		query = query.Where("canvas_id = ? AND organization_id IS NULL", *canvasID)
	} else {
		return nil, nil // Must specify either org or canvas
	}

	err := query.Find(&overrides).Error
	return overrides, err
}

// UpdateHierarchyOverride updates an existing hierarchy override
func UpdateHierarchyOverride(id uuid.UUID, isActive bool) error {
	return database.Conn().Model(&RoleHierarchyOverride{}).
		Where("id = ?", id).
		Update("is_active", isActive).
		Error
}

// DeleteHierarchyOverride soft deletes a hierarchy override by setting is_active to false
func DeleteHierarchyOverride(id uuid.UUID) error {
	return UpdateHierarchyOverride(id, false)
}

// FindPermissionOverride finds a specific permission override
func FindPermissionOverride(organizationID, canvasID *uuid.UUID, roleName, resource, action string) (*RolePermissionOverride, error) {
	var override RolePermissionOverride

	query := database.Conn().Where("role_name = ? AND resource = ? AND action = ?",
		roleName, resource, action)

	if organizationID != nil {
		query = query.Where("organization_id = ? AND canvas_id IS NULL", *organizationID)
	} else if canvasID != nil {
		query = query.Where("canvas_id = ? AND organization_id IS NULL", *canvasID)
	} else {
		return nil, gorm.ErrRecordNotFound // Must specify either org or canvas
	}

	err := query.First(&override).Error
	if err != nil {
		return nil, err
	}

	return &override, nil
}

// FindHierarchyOverride finds a specific hierarchy override
func FindHierarchyOverride(organizationID, canvasID *uuid.UUID, childRole, parentRole string) (*RoleHierarchyOverride, error) {
	var override RoleHierarchyOverride

	query := database.Conn().Where("child_role = ? AND parent_role = ?",
		childRole, parentRole)

	if organizationID != nil {
		query = query.Where("organization_id = ? AND canvas_id IS NULL", *organizationID)
	} else if canvasID != nil {
		query = query.Where("canvas_id = ? AND organization_id IS NULL", *canvasID)
	} else {
		return nil, gorm.ErrRecordNotFound // Must specify either org or canvas
	}

	err := query.First(&override).Error
	if err != nil {
		return nil, err
	}

	return &override, nil
}
