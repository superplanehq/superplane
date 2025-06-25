package models

import (
	"strings"
	"time"

	uuid "github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Organization struct {
	ID          uuid.UUID `gorm:"primary_key;default:uuid_generate_v4()"`
	Name        string    `gorm:"uniqueIndex"`
	DisplayName string
	CreatedAt   *time.Time
	CreatedBy   uuid.UUID
	UpdatedAt   *time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

func (Organization) TableName() string {
	return "organizations"
}

func ListOrganizations() ([]Organization, error) {
	var organizations []Organization

	err := database.Conn().
		Order("display_name ASC").
		Find(&organizations).
		Error

	if err != nil {
		return nil, err
	}

	return organizations, nil
}

func ListOrganizationsByIDs(ids []string) ([]Organization, error) {
	var organizations []Organization

	err := database.Conn().
		Where("id IN (?)", ids).
		Order("display_name ASC").
		Find(&organizations).
		Error

	if err != nil {
		return nil, err
	}

	return organizations, nil
}

func FindOrganizationByID(id string) (*Organization, error) {
	organization := Organization{}

	err := database.Conn().
		Where("id = ?", id).
		First(&organization).
		Error

	if err != nil {
		return nil, err
	}

	return &organization, nil
}

func FindOrganizationByName(name string) (*Organization, error) {
	organization := Organization{}

	err := database.Conn().
		Where("name = ?", name).
		First(&organization).
		Error

	if err != nil {
		return nil, err
	}

	return &organization, nil
}

func CreateOrganization(requesterID uuid.UUID, name, displayName string) (*Organization, error) {
	now := time.Now()
	organization := Organization{
		Name:        name,
		DisplayName: displayName,
		CreatedAt:   &now,
		CreatedBy:   requesterID,
		UpdatedAt:   &now,
	}

	err := database.Conn().
		Clauses(clause.Returning{}).
		Create(&organization).
		Error

	if err == nil {
		return &organization, nil
	}

	if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
		return nil, ErrNameAlreadyUsed
	}

	return nil, err
}

func SoftDeleteOrganization(id string) error {
	return database.Conn().
		Where("id = ?", id).
		Delete(&Organization{}).
		Error
}

func HardDeleteOrganization(id string) error {
	return database.Conn().
		Unscoped().
		Where("id = ?", id).
		Delete(&Organization{}).
		Error
}
