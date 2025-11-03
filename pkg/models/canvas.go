package models

import (
	"fmt"
	"strings"
	"time"

	uuid "github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrNameAlreadyUsed         = fmt.Errorf("name already used")
	ErrInvitationAlreadyExists = fmt.Errorf("invitation already exists")
)

type Canvas struct {
	ID             uuid.UUID `gorm:"primary_key;default:uuid_generate_v4()"`
	Name           string
	Description    string
	CreatedAt      *time.Time
	CreatedBy      uuid.UUID
	UpdatedAt      *time.Time
	DeletedAt      gorm.DeletedAt `gorm:"index"`
	OrganizationID uuid.UUID

	Organization *Organization `gorm:"foreignKey:OrganizationID;references:ID"`
}

func (Canvas) TableName() string {
	return "canvases"
}

func (c *Canvas) DeleteInTransaction(tx *gorm.DB) error {
	deletedName := fmt.Sprintf("%s-deleted-%d", c.Name, time.Now().Unix())

	return tx.Model(c).
		Where("id = ?", c.ID).
		Update("name", deletedName).
		Update("deleted_at", time.Now()).
		Error
}

func ListCanvases() ([]Canvas, error) {
	var canvases []Canvas

	err := database.Conn().
		Order("name ASC").
		Find(&canvases).
		Error

	if err != nil {
		return nil, err
	}

	return canvases, nil
}

func ListCanvasesByIDs(ids []string, organizationID string) ([]Canvas, error) {
	var canvases []Canvas

	if organizationID != "" {
		err := database.Conn().
			Where("organization_id = ?", organizationID).
			Where("id IN (?)", ids).
			Order("name ASC").
			Find(&canvases).
			Error

		if err != nil {
			return nil, err
		}

		return canvases, nil
	}

	err := database.Conn().
		Where("id IN (?)", ids).
		Order("name ASC").
		Find(&canvases).
		Error

	if err != nil {
		return nil, err
	}

	return canvases, nil
}

func FindUnscopedCanvasByID(id string) (*Canvas, error) {
	return FindUnscopedCanvasByIDInTransaction(database.Conn(), id)
}

func FindUnscopedCanvasByIDInTransaction(tx *gorm.DB, id string) (*Canvas, error) {
	canvas := Canvas{}

	err := tx.
		Where("id = ?", id).
		First(&canvas).
		Error

	if err != nil {
		return nil, err
	}

	return &canvas, nil
}

func FindUnscopedSoftDeletedCanvasByID(id string) (*Canvas, error) {
	canvas := Canvas{}

	err := database.Conn().
		Unscoped().
		Where("id = ?", id).
		First(&canvas).
		Error

	if err != nil {
		return nil, err
	}

	return &canvas, nil
}

func FindCanvasByID(id string, organizationID uuid.UUID) (*Canvas, error) {
	canvas := Canvas{}

	err := database.Conn().
		Where("id = ? AND organization_id = ?", id, organizationID).
		First(&canvas).
		Error

	if err != nil {
		return nil, err
	}

	return &canvas, nil
}

func FindCanvasByName(name string, organizationID uuid.UUID) (*Canvas, error) {
	canvas := Canvas{}

	err := database.Conn().
		Where("name = ? AND organization_id = ?", name, organizationID).
		First(&canvas).
		Error

	if err != nil {
		return nil, err
	}

	return &canvas, nil
}

func CreateCanvas(requesterID uuid.UUID, orgID uuid.UUID, name string, description string) (*Canvas, error) {
	now := time.Now()
	canvas := Canvas{
		Name:           name,
		Description:    description,
		OrganizationID: orgID,
		CreatedAt:      &now,
		CreatedBy:      requesterID,
		UpdatedAt:      &now,
	}

	err := database.Conn().
		Clauses(clause.Returning{}).
		Create(&canvas).
		Error

	if err == nil {
		return &canvas, nil
	}

	if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
		return nil, ErrNameAlreadyUsed
	}

	return nil, err
}

// GetCanvasIDs returns only the IDs of all canvases
func GetCanvasIDs() ([]string, error) {
	var canvasIDs []string
	err := database.Conn().Model(&Canvas{}).
		Select("id").
		Pluck("id", &canvasIDs).Error

	if err != nil {
		return nil, err
	}

	return canvasIDs, nil
}

func ExistManyCanvases(orgID uuid.UUID, ids []uuid.UUID) (bool, error) {
	var count int64

	err := database.Conn().
		Model(&Canvas{}).
		Where("organization_id = ? AND id IN (?)", orgID, ids).
		Count(&count).
		Error

	if err != nil {
		return false, err
	}

	return count == int64(len(ids)), nil
}
