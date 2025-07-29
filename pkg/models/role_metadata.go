package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
)

type RoleMetadata struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	RoleName    string    `json:"role_name" gorm:"not null;index"`
	DomainType  string    `json:"domain_type" gorm:"not null;index"`
	DomainID    string    `json:"domain_id" gorm:"not null;index"`
	DisplayName string    `json:"display_name" gorm:"not null"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type GroupMetadata struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	GroupName   string    `json:"group_name" gorm:"not null;index"`
	DomainType  string    `json:"domain_type" gorm:"not null;index"`
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
	return rm.CreateInTransaction(database.Conn())
}

func (rm *RoleMetadata) CreateInTransaction(tx *gorm.DB) error {
	return tx.Create(rm).Error
}

func (rm *RoleMetadata) Update() error {
	return rm.UpdateInTransaction(database.Conn())
}

func (rm *RoleMetadata) UpdateInTransaction(tx *gorm.DB) error {
	return tx.Save(rm).Error
}

func (gm *GroupMetadata) Create() error {
	return gm.CreateInTransaction(database.Conn())
}

func (gm *GroupMetadata) CreateInTransaction(tx *gorm.DB) error {
	return tx.Create(gm).Error
}

func (gm *GroupMetadata) Update() error {
	return gm.UpdateInTransaction(database.Conn())
}

func (gm *GroupMetadata) UpdateInTransaction(tx *gorm.DB) error {
	return tx.Save(gm).Error
}

func FindRoleMetadata(roleName, domainType, domainID string) (*RoleMetadata, error) {
	var metadata RoleMetadata
	err := database.Conn().Where("role_name = ? AND domain_type = ? AND domain_id = ?", roleName, domainType, domainID).First(&metadata).Error
	if err != nil {
		return nil, err
	}
	return &metadata, nil
}

func FindGroupMetadata(groupName, domainType, domainID string) (*GroupMetadata, error) {
	var metadata GroupMetadata
	err := database.Conn().Where("group_name = ? AND domain_type = ? AND domain_id = ?", groupName, domainType, domainID).First(&metadata).Error
	if err != nil {
		return nil, err
	}
	return &metadata, nil
}

func UpsertRoleMetadata(roleName, domainType, domainID, displayName, description string) error {
	return UpsertRoleMetadataInTransaction(database.Conn(), roleName, domainType, domainID, displayName, description)
}

func UpsertRoleMetadataInTransaction(tx *gorm.DB, roleName, domainType, domainID, displayName, description string) error {
	var metadata RoleMetadata
	err := tx.Where("role_name = ? AND domain_type = ? AND domain_id = ?", roleName, domainType, domainID).First(&metadata).Error

	if err == gorm.ErrRecordNotFound {
		metadata = RoleMetadata{
			RoleName:    roleName,
			DomainType:  domainType,
			DomainID:    domainID,
			DisplayName: displayName,
			Description: description,
		}
		return metadata.CreateInTransaction(tx)
	} else if err != nil {
		return err
	}

	metadata.DisplayName = displayName
	metadata.Description = description
	return metadata.UpdateInTransaction(tx)
}

func UpsertGroupMetadata(groupName, domainType, domainID, displayName, description string) error {
	return UpsertGroupMetadataInTransaction(database.Conn(), groupName, domainType, domainID, displayName, description)
}

func UpsertGroupMetadataInTransaction(tx *gorm.DB, groupName, domainType, domainID, displayName, description string) error {
	var metadata GroupMetadata
	err := tx.Where("group_name = ? AND domain_type = ? AND domain_id = ?", groupName, domainType, domainID).First(&metadata).Error

	if err == gorm.ErrRecordNotFound {
		metadata = GroupMetadata{
			GroupName:   groupName,
			DomainType:  domainType,
			DomainID:    domainID,
			DisplayName: displayName,
			Description: description,
		}
		return metadata.CreateInTransaction(tx)
	} else if err != nil {
		return err
	}

	metadata.DisplayName = displayName
	metadata.Description = description
	return metadata.UpdateInTransaction(tx)
}

func DeleteRoleMetadata(roleName, domainType, domainID string) error {
	return DeleteRoleMetadataInTransaction(database.Conn(), roleName, domainType, domainID)
}

func DeleteRoleMetadataInTransaction(tx *gorm.DB, roleName, domainType, domainID string) error {
	return tx.Where("role_name = ? AND domain_type = ? AND domain_id = ?", roleName, domainType, domainID).Delete(&RoleMetadata{}).Error
}

func DeleteGroupMetadata(groupName, domainType, domainID string) error {
	return DeleteGroupMetadataInTransaction(database.Conn(), groupName, domainType, domainID)
}

func DeleteGroupMetadataInTransaction(tx *gorm.DB, groupName, domainType, domainID string) error {
	return tx.Where("group_name = ? AND domain_type = ? AND domain_id = ?", groupName, domainType, domainID).Delete(&GroupMetadata{}).Error
}

func FindRoleMetadataByNames(roleNames []string, domainType, domainID string) (map[string]*RoleMetadata, error) {
	var metadata []RoleMetadata
	err := database.Conn().Where("role_name IN ? AND domain_type = ? AND domain_id = ?", roleNames, domainType, domainID).Find(&metadata).Error
	if err != nil {
		return nil, err
	}

	result := make(map[string]*RoleMetadata)
	for i := range metadata {
		result[metadata[i].RoleName] = &metadata[i]
	}
	return result, nil
}
