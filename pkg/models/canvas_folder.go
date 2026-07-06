package models

import (
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	CanvasFolderColorBlue   = "blue"
	CanvasFolderColorGreen  = "green"
	CanvasFolderColorPurple = "purple"
	CanvasFolderColorSlate  = "slate"
	CanvasFolderColorOrange = "orange"

	canvasFolderTitleUniqueConstraint = "canvas_folders_organization_id_title_key"
	canvasFolderTitleMaxLength        = 128
)

var (
	ErrCanvasFolderTitleAlreadyExists     = errors.New("canvas folder title already exists")
	ErrCanvasFolderTitleRequired          = errors.New("canvas folder title is required")
	ErrCanvasFolderTitleTooLong           = errors.New("canvas folder title is too long")
	ErrCanvasFolderInvalidBackgroundColor = errors.New("invalid canvas folder background color")
	ErrCanvasFolderInvalidMoveDirection   = errors.New("invalid canvas folder move direction")
	ErrCanvasFolderCanvasNotFound         = errors.New("canvas not found")
)

var CanvasFolderBackgroundColors = []string{
	CanvasFolderColorBlue,
	CanvasFolderColorGreen,
	CanvasFolderColorPurple,
	CanvasFolderColorSlate,
	CanvasFolderColorOrange,
}

type CanvasFolder struct {
	ID              uuid.UUID
	OrganizationID  uuid.UUID
	Title           string
	BackgroundColor string
	SortOrder       int64
	CreatedAt       *time.Time
	UpdatedAt       *time.Time
	Canvases        []Canvas `gorm:"foreignKey:CanvasFolderID"`
}

func (f *CanvasFolder) TableName() string {
	return "canvas_folders"
}

func ListCanvasFolders(organizationID uuid.UUID) ([]CanvasFolder, error) {
	return ListCanvasFoldersInTransaction(database.Conn(), organizationID)
}

func ListCanvasFoldersInTransaction(tx *gorm.DB, organizationID uuid.UUID) ([]CanvasFolder, error) {
	var folders []CanvasFolder
	err := tx.
		Preload("Canvases", func(tx *gorm.DB) *gorm.DB {
			return tx.Order("name ASC").Order("id ASC")
		}).
		Where("organization_id = ?", organizationID).
		Order("sort_order ASC").
		Order("created_at DESC").
		Order("id DESC").
		Find(&folders).
		Error
	if err != nil {
		return nil, err
	}

	return folders, nil
}

func FindCanvasFolder(organizationID, id uuid.UUID) (*CanvasFolder, error) {
	return FindCanvasFolderInTransaction(database.Conn(), organizationID, id)
}

func FindCanvasFolderInTransaction(tx *gorm.DB, organizationID, id uuid.UUID) (*CanvasFolder, error) {
	var folder CanvasFolder
	err := tx.
		Where("organization_id = ?", organizationID).
		Where("id = ?", id).
		First(&folder).
		Error
	if err != nil {
		return nil, err
	}

	return &folder, nil
}

func CreateCanvasFolder(organizationID uuid.UUID, title, backgroundColor string) (*CanvasFolder, error) {
	var folder *CanvasFolder
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		createdFolder, err := CreateCanvasFolderInTransaction(tx, organizationID, title, backgroundColor)
		if err != nil {
			return err
		}

		folder = createdFolder
		return nil
	})
	if err != nil {
		return nil, err
	}

	return folder, nil
}

func CreateCanvasFolderInTransaction(tx *gorm.DB, organizationID uuid.UUID, title, backgroundColor string) (*CanvasFolder, error) {
	normalizedTitle, err := normalizeCanvasFolderTitle(title)
	if err != nil {
		return nil, err
	}

	normalizedColor, err := normalizeCanvasFolderBackgroundColor(backgroundColor)
	if err != nil {
		return nil, err
	}

	if err := lockOrganizationForCanvasFolderSort(tx, organizationID); err != nil {
		return nil, err
	}

	sortOrder, err := nextCanvasFolderSortOrderForCreate(tx, organizationID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	folder := &CanvasFolder{
		ID:              uuid.New(),
		OrganizationID:  organizationID,
		Title:           normalizedTitle,
		BackgroundColor: normalizedColor,
		SortOrder:       sortOrder,
		CreatedAt:       &now,
		UpdatedAt:       &now,
	}

	if err := tx.Create(folder).Error; err != nil {
		return nil, mapCanvasFolderTitleUniqueConstraintError(err)
	}

	return folder, nil
}

func UpdateCanvasFolder(organizationID, id uuid.UUID, title, backgroundColor string) (*CanvasFolder, error) {
	return UpdateCanvasFolderInTransaction(database.Conn(), organizationID, id, title, backgroundColor)
}

func UpdateCanvasFolderWithMembership(
	organizationID,
	id uuid.UUID,
	title,
	backgroundColor string,
	canvasIDs []uuid.UUID,
) (*CanvasFolder, []Canvas, error) {
	var folder *CanvasFolder
	var affectedCanvases []Canvas
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		updatedFolder, err := UpdateCanvasFolderInTransaction(tx, organizationID, id, title, backgroundColor)
		if err != nil {
			return err
		}

		canvases, affected, err := ReplaceCanvasFolderMembershipInTransaction(tx, organizationID, id, canvasIDs)
		if err != nil {
			return err
		}

		updatedFolder.Canvases = canvases
		folder = updatedFolder
		affectedCanvases = affected
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	return folder, affectedCanvases, nil
}

func UpdateCanvasFolderInTransaction(tx *gorm.DB, organizationID, id uuid.UUID, title, backgroundColor string) (*CanvasFolder, error) {
	normalizedTitle, err := normalizeCanvasFolderTitle(title)
	if err != nil {
		return nil, err
	}

	normalizedColor, err := normalizeCanvasFolderBackgroundColor(backgroundColor)
	if err != nil {
		return nil, err
	}

	folder, err := FindCanvasFolderInTransaction(tx, organizationID, id)
	if err != nil {
		return nil, err
	}

	if folder.Title == normalizedTitle && folder.BackgroundColor == normalizedColor {
		return folder, nil
	}

	now := time.Now()
	folder.Title = normalizedTitle
	folder.BackgroundColor = normalizedColor
	folder.UpdatedAt = &now
	if err := tx.Save(folder).Error; err != nil {
		return nil, mapCanvasFolderTitleUniqueConstraintError(err)
	}

	return folder, nil
}

func DeleteCanvasFolder(organizationID, id uuid.UUID) error {
	return DeleteCanvasFolderInTransaction(database.Conn(), organizationID, id)
}

func DeleteCanvasFolderInTransaction(tx *gorm.DB, organizationID, id uuid.UUID) error {
	result := tx.
		Where("organization_id = ?", organizationID).
		Where("id = ?", id).
		Delete(&CanvasFolder{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func MoveCanvasFolder(organizationID, id uuid.UUID, direction string) ([]CanvasFolder, error) {
	var folders []CanvasFolder
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		updatedFolders, err := MoveCanvasFolderInTransaction(tx, organizationID, id, direction)
		if err != nil {
			return err
		}

		folders = updatedFolders
		return nil
	})
	if err != nil {
		return nil, err
	}

	return folders, nil
}

func MoveCanvasFolderInTransaction(tx *gorm.DB, organizationID, id uuid.UUID, direction string) ([]CanvasFolder, error) {
	if err := lockOrganizationForCanvasFolderSort(tx, organizationID); err != nil {
		return nil, err
	}

	var folders []CanvasFolder
	if err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("organization_id = ?", organizationID).
		Order("sort_order ASC").
		Order("created_at DESC").
		Order("id DESC").
		Find(&folders).
		Error; err != nil {
		return nil, err
	}

	folderIndex := slices.IndexFunc(folders, func(folder CanvasFolder) bool {
		return folder.ID == id
	})
	if folderIndex == -1 {
		return nil, gorm.ErrRecordNotFound
	}

	targetIndex := folderIndex
	switch direction {
	case "DIRECTION_UP":
		targetIndex = folderIndex - 1
	case "DIRECTION_DOWN":
		targetIndex = folderIndex + 1
	default:
		return nil, ErrCanvasFolderInvalidMoveDirection
	}

	if targetIndex < 0 || targetIndex >= len(folders) {
		return folders, nil
	}

	now := time.Now()
	currentFolder := folders[folderIndex]
	targetFolder := folders[targetIndex]

	if err := tx.
		Model(&CanvasFolder{}).
		Where("id = ?", currentFolder.ID).
		Updates(map[string]any{
			"sort_order": targetFolder.SortOrder,
			"updated_at": now,
		}).
		Error; err != nil {
		return nil, err
	}

	if err := tx.
		Model(&CanvasFolder{}).
		Where("id = ?", targetFolder.ID).
		Updates(map[string]any{
			"sort_order": currentFolder.SortOrder,
			"updated_at": now,
		}).
		Error; err != nil {
		return nil, err
	}

	return ListCanvasFoldersInTransaction(tx, organizationID)
}

func lockOrganizationForCanvasFolderSort(tx *gorm.DB, organizationID uuid.UUID) error {
	return tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", organizationID).
		First(&Organization{}).
		Error
}

func UpdateCanvasFolderMembership(organizationID, canvasID uuid.UUID, folderID *uuid.UUID) (*Canvas, error) {
	var canvas *Canvas
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		updatedCanvas, err := UpdateCanvasFolderMembershipInTransaction(tx, organizationID, canvasID, folderID)
		if err != nil {
			return err
		}

		canvas = updatedCanvas
		return nil
	})
	if err != nil {
		return nil, err
	}

	return canvas, nil
}

func UpdateCanvasFolderMembershipInTransaction(tx *gorm.DB, organizationID, canvasID uuid.UUID, folderID *uuid.UUID) (*Canvas, error) {
	var canvas Canvas
	if err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("organization_id = ?", organizationID).
		Where("id = ?", canvasID).
		First(&canvas).
		Error; err != nil {
		return nil, err
	}

	if folderID != nil {
		if _, err := FindCanvasFolderInTransaction(tx, organizationID, *folderID); err != nil {
			return nil, err
		}
	}

	var folderValue any
	if folderID != nil {
		folderValue = *folderID
	}

	if err := tx.
		Model(&Canvas{}).
		Where("organization_id = ?", organizationID).
		Where("id = ?", canvasID).
		Updates(map[string]any{
			"folder_id":  folderValue,
			"updated_at": time.Now(),
		}).
		Error; err != nil {
		return nil, err
	}

	return FindCanvasInTransaction(tx, organizationID, canvasID)
}

func ReplaceCanvasFolderMembershipInTransaction(tx *gorm.DB, organizationID, folderID uuid.UUID, canvasIDs []uuid.UUID) ([]Canvas, []Canvas, error) {
	if _, err := FindCanvasFolderInTransaction(tx.Clauses(clause.Locking{Strength: "UPDATE"}), organizationID, folderID); err != nil {
		return nil, nil, err
	}

	uniqueCanvasIDs := make([]uuid.UUID, 0, len(canvasIDs))
	for _, id := range canvasIDs {
		if slices.Contains(uniqueCanvasIDs, id) {
			continue
		}

		uniqueCanvasIDs = append(uniqueCanvasIDs, id)
	}

	var existingCanvases []Canvas
	if err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("organization_id = ?", organizationID).
		Where("folder_id = ?", folderID).
		Find(&existingCanvases).
		Error; err != nil {
		return nil, nil, err
	}

	affectedCanvases := make([]Canvas, 0, len(existingCanvases)+len(uniqueCanvasIDs))
	affectedCanvasIDs := make([]uuid.UUID, 0, len(existingCanvases)+len(uniqueCanvasIDs))
	for _, canvas := range existingCanvases {
		affectedCanvases = append(affectedCanvases, canvas)
		affectedCanvasIDs = append(affectedCanvasIDs, canvas.ID)
	}

	if len(uniqueCanvasIDs) > 0 {
		var canvases []Canvas
		if err := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("organization_id = ?", organizationID).
			Where("id IN ?", uniqueCanvasIDs).
			Find(&canvases).
			Error; err != nil {
			return nil, nil, err
		}

		if len(canvases) != len(uniqueCanvasIDs) {
			return nil, nil, ErrCanvasFolderCanvasNotFound
		}

		for _, canvas := range canvases {
			if slices.Contains(affectedCanvasIDs, canvas.ID) {
				continue
			}

			affectedCanvases = append(affectedCanvases, canvas)
			affectedCanvasIDs = append(affectedCanvasIDs, canvas.ID)
		}
	}

	now := time.Now()
	clearQuery := tx.
		Model(&Canvas{}).
		Where("organization_id = ?", organizationID).
		Where("folder_id = ?", folderID)

	if len(uniqueCanvasIDs) > 0 {
		clearQuery = clearQuery.Where("id NOT IN ?", uniqueCanvasIDs)
	}

	if err := clearQuery.
		Updates(map[string]any{
			"folder_id":  nil,
			"updated_at": now,
		}).
		Error; err != nil {
		return nil, nil, err
	}

	if len(uniqueCanvasIDs) > 0 {
		if err := tx.
			Model(&Canvas{}).
			Where("organization_id = ?", organizationID).
			Where("id IN ?", uniqueCanvasIDs).
			Updates(map[string]any{
				"folder_id":  folderID,
				"updated_at": now,
			}).
			Error; err != nil {
			return nil, nil, err
		}
	}

	var canvases []Canvas
	if err := tx.
		Where("organization_id = ?", organizationID).
		Where("folder_id = ?", folderID).
		Order("name ASC").
		Order("id ASC").
		Find(&canvases).
		Error; err != nil {
		return nil, nil, err
	}

	return canvases, affectedCanvases, nil
}

func normalizeCanvasFolderTitle(title string) (string, error) {
	normalized := strings.TrimSpace(title)
	if normalized == "" {
		return "", ErrCanvasFolderTitleRequired
	}

	if len(normalized) > canvasFolderTitleMaxLength {
		return "", ErrCanvasFolderTitleTooLong
	}

	return normalized, nil
}

func normalizeCanvasFolderBackgroundColor(backgroundColor string) (string, error) {
	if backgroundColor == "" {
		return CanvasFolderColorBlue, nil
	}

	if !slices.Contains(CanvasFolderBackgroundColors, backgroundColor) {
		return "", ErrCanvasFolderInvalidBackgroundColor
	}

	return backgroundColor, nil
}

func nextCanvasFolderSortOrderForCreate(tx *gorm.DB, organizationID uuid.UUID) (int64, error) {
	var firstFolder CanvasFolder
	err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("organization_id = ?", organizationID).
		Order("sort_order ASC").
		Order("created_at DESC").
		Order("id DESC").
		Limit(1).
		First(&firstFolder).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	return firstFolder.SortOrder - 1, nil
}

func mapCanvasFolderTitleUniqueConstraintError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.ConstraintName == canvasFolderTitleUniqueConstraint {
		return ErrCanvasFolderTitleAlreadyExists
	}

	return err
}
