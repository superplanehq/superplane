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
	CanvasGroupColorBlue800   = "blue-800"
	CanvasGroupColorGreen800  = "green-800"
	CanvasGroupColorSlate700  = "slate-700"
	CanvasGroupColorViolet800 = "violet-800"
	CanvasGroupColorYellow800 = "yellow-800"

	canvasGroupTitleUniqueConstraint = "canvas_groups_organization_id_title_key"
	canvasGroupTitleMaxLength        = 128
)

var (
	ErrCanvasGroupTitleAlreadyExists     = errors.New("canvas group title already exists")
	ErrCanvasGroupTitleRequired          = errors.New("canvas group title is required")
	ErrCanvasGroupTitleTooLong           = errors.New("canvas group title is too long")
	ErrCanvasGroupInvalidBackgroundColor = errors.New("invalid canvas group background color")
	ErrCanvasGroupInvalidMoveDirection   = errors.New("invalid canvas group move direction")
)

var CanvasGroupBackgroundColors = []string{
	CanvasGroupColorBlue800,
	CanvasGroupColorGreen800,
	CanvasGroupColorSlate700,
	CanvasGroupColorViolet800,
	CanvasGroupColorYellow800,
}

type CanvasGroup struct {
	ID              uuid.UUID
	OrganizationID  uuid.UUID
	Title           string
	BackgroundColor string
	SortOrder       int64
	CreatedAt       *time.Time
	UpdatedAt       *time.Time
}

func (g *CanvasGroup) TableName() string {
	return "canvas_groups"
}

func ListCanvasGroups(organizationID uuid.UUID) ([]CanvasGroup, error) {
	return ListCanvasGroupsInTransaction(database.Conn(), organizationID)
}

func ListCanvasGroupsInTransaction(tx *gorm.DB, organizationID uuid.UUID) ([]CanvasGroup, error) {
	var groups []CanvasGroup
	err := tx.
		Where("organization_id = ?", organizationID).
		Order("sort_order ASC").
		Order("created_at DESC").
		Order("id DESC").
		Find(&groups).
		Error
	if err != nil {
		return nil, err
	}

	return groups, nil
}

func FindCanvasGroup(organizationID, id uuid.UUID) (*CanvasGroup, error) {
	return FindCanvasGroupInTransaction(database.Conn(), organizationID, id)
}

func FindCanvasGroupInTransaction(tx *gorm.DB, organizationID, id uuid.UUID) (*CanvasGroup, error) {
	var group CanvasGroup
	err := tx.
		Where("organization_id = ?", organizationID).
		Where("id = ?", id).
		First(&group).
		Error
	if err != nil {
		return nil, err
	}

	return &group, nil
}

// CreateCanvasGroup wraps CreateCanvasGroupInTransaction in a real transaction
// so the FOR UPDATE lock taken when computing the next sort order is held until
// the INSERT commits. Without this, two concurrent creates can compute the same
// sort_order value (Bugbot M2).
func CreateCanvasGroup(organizationID uuid.UUID, title, backgroundColor string) (*CanvasGroup, error) {
	var created *CanvasGroup
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		group, err := CreateCanvasGroupInTransaction(tx, organizationID, title, backgroundColor)
		if err != nil {
			return err
		}
		created = group
		return nil
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

func CreateCanvasGroupInTransaction(tx *gorm.DB, organizationID uuid.UUID, title, backgroundColor string) (*CanvasGroup, error) {
	normalizedTitle, err := normalizeCanvasGroupTitle(title)
	if err != nil {
		return nil, err
	}

	normalizedColor, err := normalizeCanvasGroupBackgroundColor(backgroundColor)
	if err != nil {
		return nil, err
	}

	sortOrder, err := nextCanvasGroupSortOrderForCreate(tx, organizationID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	group := &CanvasGroup{
		ID:              uuid.New(),
		OrganizationID:  organizationID,
		Title:           normalizedTitle,
		BackgroundColor: normalizedColor,
		SortOrder:       sortOrder,
		CreatedAt:       &now,
		UpdatedAt:       &now,
	}

	if err := tx.Create(group).Error; err != nil {
		return nil, mapCanvasGroupTitleUniqueConstraintError(err)
	}

	return group, nil
}

func UpdateCanvasGroup(organizationID, id uuid.UUID, title, backgroundColor string) (*CanvasGroup, error) {
	return UpdateCanvasGroupInTransaction(database.Conn(), organizationID, id, title, backgroundColor)
}

func UpdateCanvasGroupInTransaction(tx *gorm.DB, organizationID, id uuid.UUID, title, backgroundColor string) (*CanvasGroup, error) {
	normalizedTitle, err := normalizeCanvasGroupTitle(title)
	if err != nil {
		return nil, err
	}

	normalizedColor, err := normalizeCanvasGroupBackgroundColor(backgroundColor)
	if err != nil {
		return nil, err
	}

	group, err := FindCanvasGroupInTransaction(tx, organizationID, id)
	if err != nil {
		return nil, err
	}

	if group.Title == normalizedTitle && group.BackgroundColor == normalizedColor {
		return group, nil
	}

	now := time.Now()
	group.Title = normalizedTitle
	group.BackgroundColor = normalizedColor
	group.UpdatedAt = &now
	if err := tx.Save(group).Error; err != nil {
		return nil, mapCanvasGroupTitleUniqueConstraintError(err)
	}

	return group, nil
}

func DeleteCanvasGroup(organizationID, id uuid.UUID) error {
	return DeleteCanvasGroupInTransaction(database.Conn(), organizationID, id)
}

func DeleteCanvasGroupInTransaction(tx *gorm.DB, organizationID, id uuid.UUID) error {
	result := tx.
		Where("organization_id = ?", organizationID).
		Where("id = ?", id).
		Delete(&CanvasGroup{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

// MoveCanvasGroup wraps MoveCanvasGroupInTransaction in a real transaction so
// that the FOR UPDATE lock and the two sort_order swaps are atomic. Without
// this, the lock is released after the SELECT, the two UPDATEs run in their
// own auto-committed statements, and a partial failure leaves sort_order data
// corrupted (Bugbot M1).
func MoveCanvasGroup(organizationID, id uuid.UUID, direction string) ([]CanvasGroup, error) {
	var groups []CanvasGroup
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		moved, err := MoveCanvasGroupInTransaction(tx, organizationID, id, direction)
		if err != nil {
			return err
		}
		groups = moved
		return nil
	})
	if err != nil {
		return nil, err
	}
	return groups, nil
}

func MoveCanvasGroupInTransaction(tx *gorm.DB, organizationID, id uuid.UUID, direction string) ([]CanvasGroup, error) {
	var groups []CanvasGroup
	if err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("organization_id = ?", organizationID).
		Order("sort_order ASC").
		Order("created_at DESC").
		Order("id DESC").
		Find(&groups).
		Error; err != nil {
		return nil, err
	}

	groupIndex := slices.IndexFunc(groups, func(group CanvasGroup) bool {
		return group.ID == id
	})
	if groupIndex == -1 {
		return nil, gorm.ErrRecordNotFound
	}

	targetIndex := groupIndex
	switch direction {
	case "DIRECTION_UP":
		targetIndex = groupIndex - 1
	case "DIRECTION_DOWN":
		targetIndex = groupIndex + 1
	default:
		return nil, ErrCanvasGroupInvalidMoveDirection
	}

	if targetIndex < 0 || targetIndex >= len(groups) {
		return groups, nil
	}

	now := time.Now()
	currentGroup := groups[groupIndex]
	targetGroup := groups[targetIndex]

	if err := tx.
		Model(&CanvasGroup{}).
		Where("id = ?", currentGroup.ID).
		Updates(map[string]any{
			"sort_order": targetGroup.SortOrder,
			"updated_at": now,
		}).
		Error; err != nil {
		return nil, err
	}

	if err := tx.
		Model(&CanvasGroup{}).
		Where("id = ?", targetGroup.ID).
		Updates(map[string]any{
			"sort_order": currentGroup.SortOrder,
			"updated_at": now,
		}).
		Error; err != nil {
		return nil, err
	}

	return ListCanvasGroupsInTransaction(tx, organizationID)
}

func UpdateCanvasGroupMembership(organizationID, canvasID uuid.UUID, groupID *uuid.UUID) (*Canvas, error) {
	var canvas *Canvas
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		updatedCanvas, err := UpdateCanvasGroupMembershipInTransaction(tx, organizationID, canvasID, groupID)
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

func UpdateCanvasGroupMembershipInTransaction(tx *gorm.DB, organizationID, canvasID uuid.UUID, groupID *uuid.UUID) (*Canvas, error) {
	var canvas Canvas
	if err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("organization_id = ?", organizationID).
		Where("id = ?", canvasID).
		First(&canvas).
		Error; err != nil {
		return nil, err
	}

	if groupID != nil {
		if _, err := FindCanvasGroupInTransaction(tx, organizationID, *groupID); err != nil {
			return nil, err
		}
	}

	var groupValue any
	if groupID != nil {
		groupValue = *groupID
	}

	if err := tx.
		Model(&Canvas{}).
		Where("organization_id = ?", organizationID).
		Where("id = ?", canvasID).
		Updates(map[string]any{
			"canvas_group_id": groupValue,
			"updated_at":      time.Now(),
		}).
		Error; err != nil {
		return nil, err
	}

	return FindCanvasInTransaction(tx, organizationID, canvasID)
}

func normalizeCanvasGroupTitle(title string) (string, error) {
	normalized := strings.TrimSpace(title)
	if normalized == "" {
		return "", ErrCanvasGroupTitleRequired
	}

	if len(normalized) > canvasGroupTitleMaxLength {
		return "", ErrCanvasGroupTitleTooLong
	}

	return normalized, nil
}

func normalizeCanvasGroupBackgroundColor(backgroundColor string) (string, error) {
	if backgroundColor == "" {
		return CanvasGroupColorBlue800, nil
	}

	if !slices.Contains(CanvasGroupBackgroundColors, backgroundColor) {
		return "", ErrCanvasGroupInvalidBackgroundColor
	}

	return backgroundColor, nil
}

func nextCanvasGroupSortOrderForCreate(tx *gorm.DB, organizationID uuid.UUID) (int64, error) {
	var firstGroup CanvasGroup
	err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("organization_id = ?", organizationID).
		Order("sort_order ASC").
		Order("created_at DESC").
		Order("id DESC").
		Limit(1).
		First(&firstGroup).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	return firstGroup.SortOrder - 1, nil
}

func mapCanvasGroupTitleUniqueConstraintError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.ConstraintName == canvasGroupTitleUniqueConstraint {
		return ErrCanvasGroupTitleAlreadyExists
	}

	return err
}
