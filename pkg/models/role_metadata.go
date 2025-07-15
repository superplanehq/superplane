package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
)

// RoleMetadata stores display names and descriptions for roles
type RoleMetadata struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	RoleName    string    `json:"role_name" gorm:"not null;index"`
	DomainType  string    `json:"domain_type" gorm:"not null;index"` // "organization" or "canvas"
	DomainID    string    `json:"domain_id" gorm:"not null;index"`
	DisplayName string    `json:"display_name" gorm:"not null"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// GroupMetadata stores display names and descriptions for groups
type GroupMetadata struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	GroupName   string    `json:"group_name" gorm:"not null;index"`
	DomainType  string    `json:"domain_type" gorm:"not null;index"` // "organization" or "canvas"
	DomainID    string    `json:"domain_id" gorm:"not null;index"`
	DisplayName string    `json:"display_name" gorm:"not null"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (rm *RoleMetadata) BeforeCreate(tx *gorm.DB) error {
	if rm.ID == uuid.Nil {
		rm.ID = uuid.New()
	}
	return nil
}

func (gm *GroupMetadata) BeforeCreate(tx *gorm.DB) error {
	if gm.ID == uuid.Nil {
		gm.ID = uuid.New()
	}
	return nil
}

func (rm *RoleMetadata) Create() error {
	return database.Conn().Create(rm).Error
}

func (rm *RoleMetadata) Update() error {
	return database.Conn().Save(rm).Error
}

func (gm *GroupMetadata) Create() error {
	return database.Conn().Create(gm).Error
}

func (gm *GroupMetadata) Update() error {
	return database.Conn().Save(gm).Error
}

// FindRoleMetadata finds role metadata by role name, domain type, and domain ID
func FindRoleMetadata(roleName, domainType, domainID string) (*RoleMetadata, error) {
	var metadata RoleMetadata
	err := database.Conn().Where("role_name = ? AND domain_type = ? AND domain_id = ?", roleName, domainType, domainID).First(&metadata).Error
	if err != nil {
		return nil, err
	}
	return &metadata, nil
}

// FindGroupMetadata finds group metadata by group name, domain type, and domain ID
func FindGroupMetadata(groupName, domainType, domainID string) (*GroupMetadata, error) {
	var metadata GroupMetadata
	err := database.Conn().Where("group_name = ? AND domain_type = ? AND domain_id = ?", groupName, domainType, domainID).First(&metadata).Error
	if err != nil {
		return nil, err
	}
	return &metadata, nil
}

// UpsertRoleMetadata creates or updates role metadata
func UpsertRoleMetadata(roleName, domainType, domainID, displayName, description string) error {
	var metadata RoleMetadata
	err := database.Conn().Where("role_name = ? AND domain_type = ? AND domain_id = ?", roleName, domainType, domainID).First(&metadata).Error
	
	if err == gorm.ErrRecordNotFound {
		// Create new metadata
		metadata = RoleMetadata{
			RoleName:    roleName,
			DomainType:  domainType,
			DomainID:    domainID,
			DisplayName: displayName,
			Description: description,
		}
		return metadata.Create()
	} else if err != nil {
		return err
	}
	
	// Update existing metadata
	metadata.DisplayName = displayName
	metadata.Description = description
	return metadata.Update()
}

// UpsertGroupMetadata creates or updates group metadata
func UpsertGroupMetadata(groupName, domainType, domainID, displayName, description string) error {
	var metadata GroupMetadata
	err := database.Conn().Where("group_name = ? AND domain_type = ? AND domain_id = ?", groupName, domainType, domainID).First(&metadata).Error
	
	if err == gorm.ErrRecordNotFound {
		// Create new metadata
		metadata = GroupMetadata{
			GroupName:   groupName,
			DomainType:  domainType,
			DomainID:    domainID,
			DisplayName: displayName,
			Description: description,
		}
		return metadata.Create()
	} else if err != nil {
		return err
	}
	
	// Update existing metadata
	metadata.DisplayName = displayName
	metadata.Description = description
	return metadata.Update()
}

// DeleteRoleMetadata deletes role metadata
func DeleteRoleMetadata(roleName, domainType, domainID string) error {
	return database.Conn().Where("role_name = ? AND domain_type = ? AND domain_id = ?", roleName, domainType, domainID).Delete(&RoleMetadata{}).Error
}

// DeleteGroupMetadata deletes group metadata
func DeleteGroupMetadata(groupName, domainType, domainID string) error {
	return database.Conn().Where("group_name = ? AND domain_type = ? AND domain_id = ?", groupName, domainType, domainID).Delete(&GroupMetadata{}).Error
}

// GetRoleDisplayName gets the display name for a role, fallback to role name if not found
func GetRoleDisplayName(roleName, domainType, domainID string) string {
	metadata, err := FindRoleMetadata(roleName, domainType, domainID)
	if err != nil {
		return roleName // Fallback to role name
	}
	return metadata.DisplayName
}

// GetGroupDisplayName gets the display name for a group, fallback to group name if not found
func GetGroupDisplayName(groupName, domainType, domainID string) string {
	metadata, err := FindGroupMetadata(groupName, domainType, domainID)
	if err != nil {
		return groupName // Fallback to group name
	}
	return metadata.DisplayName
}

// GetRoleDescription gets the description for a role
func GetRoleDescription(roleName, domainType, domainID string) string {
	metadata, err := FindRoleMetadata(roleName, domainType, domainID)
	if err != nil {
		return "" // No description available
	}
	return metadata.Description
}

// GetGroupDescription gets the description for a group
func GetGroupDescription(groupName, domainType, domainID string) string {
	metadata, err := FindGroupMetadata(groupName, domainType, domainID)
	if err != nil {
		return "" // No description available
	}
	return metadata.Description
}