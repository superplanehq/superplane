package models

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrCanvasNameAlreadyExists = errors.New("canvas name already exists")

const canvasNameUniqueConstraint = "workflows_organization_id_name_key"

type Canvas struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	LiveVersionID  *uuid.UUID
	CanvasFolderID *uuid.UUID `gorm:"column:folder_id"`
	Name           string
	// The `->` tag marks fields as read-only in GORM. These values are projected
	// from the live version via SELECT aliases; they are not stored on workflows.
	Description            string                    `gorm:"column:description;->"`
	Nodes                  datatypes.JSONSlice[Node] `gorm:"column:nodes;->"`
	Edges                  datatypes.JSONSlice[Edge] `gorm:"column:edges;->"`
	CreatedBy              *uuid.UUID
	NextDraftDisplayNumber int `gorm:"column:next_draft_display_number;not null;default:1"`
	CreatedAt              *time.Time
	UpdatedAt              *time.Time
	DeletedAt              gorm.DeletedAt `gorm:"index"`
}

func (c *Canvas) TableName() string {
	return "workflows"
}

func MapCanvasNameUniqueConstraintError(err error) error {
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.ConstraintName == canvasNameUniqueConstraint {
		return ErrCanvasNameAlreadyExists
	}

	return err
}

func queryCanvasWithLiveVersion(tx *gorm.DB) *gorm.DB {
	return tx.
		Model(&Canvas{}).
		Joins("JOIN workflow_versions live_version ON live_version.id = workflows.live_version_id").
		Select(
			"workflows.*",
			"live_version.description AS description",
			"live_version.nodes AS nodes",
			"live_version.edges AS edges",
		)
}

func withActiveCanvas(tx *gorm.DB, workflowIDColumn string) *gorm.DB {
	return tx.
		Joins(fmt.Sprintf("JOIN workflows ON %s = workflows.id", workflowIDColumn)).
		Joins("JOIN organizations ON workflows.organization_id = organizations.id").
		Where("workflows.deleted_at IS NULL").
		Where("organizations.deleted_at IS NULL")
}

func (c *Canvas) FindNode(id string) (*CanvasNode, error) {
	var node CanvasNode
	err := database.Conn().
		Where("workflow_id = ?", c.ID).
		Where("node_id = ?", id).
		First(&node).
		Error

	if err != nil {
		return nil, err
	}

	return &node, nil
}

func FindCanvasNodes(canvasID uuid.UUID) ([]CanvasNode, error) {
	return FindCanvasNodesInTransaction(database.Conn(), canvasID)
}

func FindCanvasNodesUnscoped(workflowID uuid.UUID) ([]CanvasNode, error) {
	return FindCanvasNodesUnscopedInTransaction(database.Conn(), workflowID)
}

func FindCanvasNodesInTransaction(tx *gorm.DB, workflowID uuid.UUID) ([]CanvasNode, error) {
	var nodes []CanvasNode
	err := tx.
		Where("workflow_id = ?", workflowID).
		Find(&nodes).
		Error

	if err != nil {
		return nil, err
	}

	return nodes, nil
}

func FindCanvasNodesUnscopedInTransaction(tx *gorm.DB, workflowID uuid.UUID) ([]CanvasNode, error) {
	var nodes []CanvasNode
	err := tx.
		Unscoped().
		Where("workflow_id = ?", workflowID).
		Find(&nodes).
		Error

	if err != nil {
		return nil, err
	}

	return nodes, nil
}

func (c *Canvas) SoftDelete() error {
	return c.SoftDeleteInTransaction(database.Conn())
}

func (c *Canvas) SoftDeleteInTransaction(tx *gorm.DB) error {
	now := time.Now()
	timestamp := now.Unix()

	newName := fmt.Sprintf("%s (deleted-%d)", c.Name, timestamp)
	return tx.Transaction(func(innerTx *gorm.DB) error {
		if err := innerTx.Model(c).Updates(map[string]any{
			"name":       newName,
			"deleted_at": now,
		}).Error; err != nil {
			return err
		}

		if c.LiveVersionID == nil {
			return nil
		}

		return innerTx.
			Model(&CanvasVersion{}).
			Where("id = ?", *c.LiveVersionID).
			Updates(map[string]any{
				"name":       newName,
				"updated_at": now,
			}).
			Error
	})
}

func FindCanvas(orgID, id uuid.UUID) (*Canvas, error) {
	return FindCanvasInTransaction(database.Conn(), orgID, id)
}

func FindCanvasByName(name string, organizationID uuid.UUID) (*Canvas, error) {
	return FindCanvasByNameInTransaction(database.Conn(), name, organizationID)
}

func FindCanvasByNameInTransaction(tx *gorm.DB, name string, organizationID uuid.UUID) (*Canvas, error) {
	var canvas Canvas
	err := queryCanvasWithLiveVersion(tx).
		Where("workflows.name = ? AND workflows.organization_id = ?", name, organizationID).
		First(&canvas).
		Error

	if err != nil {
		return nil, err
	}

	return &canvas, nil
}

func FindCanvasInTransaction(tx *gorm.DB, orgID, id uuid.UUID) (*Canvas, error) {
	var canvas Canvas
	err := queryCanvasWithLiveVersion(tx).
		Where("workflows.organization_id = ?", orgID).
		Where("workflows.id = ?", id).
		First(&canvas).
		Error

	if err != nil {
		return nil, err
	}

	return &canvas, nil
}

func CheckCanvasExistence(tx *gorm.DB, orgID, id uuid.UUID) (bool, error) {
	var count int64

	err := tx.Model(&Canvas{}).
		Where("organization_id = ?", orgID).
		Where("id = ?", id).
		Count(&count).Error

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func FindCanvasWithoutOrgScope(id uuid.UUID) (*Canvas, error) {
	return FindCanvasWithoutOrgScopeInTransaction(database.Conn(), id)
}

func FindCanvasWithoutOrgScopeInTransaction(tx *gorm.DB, id uuid.UUID) (*Canvas, error) {
	var canvas Canvas
	err := queryCanvasWithLiveVersion(tx).
		Where("workflows.id = ?", id).
		First(&canvas).
		Error

	if err != nil {
		return nil, err
	}

	return &canvas, nil
}

func FindUnscopedCanvas(id uuid.UUID) (*Canvas, error) {
	return FindUnscopedCanvasInTransaction(database.Conn(), id)
}

func FindUnscopedCanvasInTransaction(tx *gorm.DB, id uuid.UUID) (*Canvas, error) {
	var canvas Canvas
	err := queryCanvasWithLiveVersion(tx).
		Unscoped().
		Where("workflows.id = ?", id).
		First(&canvas).
		Error

	if err != nil {
		return nil, err
	}

	return &canvas, nil
}

func ListCanvasesPaginated(orgID, search string, limit, offset int) ([]Canvas, int64, error) {
	query := queryCanvasWithLiveVersion(database.Conn()).
		Where("workflows.organization_id = ?", orgID)

	if search != "" {
		query = query.Where("live_version.name ILIKE ?", "%"+search+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if limit > 0 {
		query = query.Limit(limit)
	}

	if offset > 0 {
		query = query.Offset(offset)
	}

	var canvases []Canvas
	if err := query.Order("live_version.name ASC").Find(&canvases).Error; err != nil {
		return nil, 0, err
	}

	return canvases, total, nil
}

func ListCanvases(orgID string) ([]Canvas, error) {
	var canvases []Canvas
	err := queryCanvasWithLiveVersion(database.Conn()).
		Where("workflows.organization_id = ?", orgID).
		Order("live_version.name ASC").
		Find(&canvases).
		Error

	if err != nil {
		return nil, err
	}

	return canvases, nil
}

func ListDeletedCanvases() ([]Canvas, error) {
	var canvases []Canvas
	err := database.Conn().
		Model(&Canvas{}).
		Unscoped().
		Joins("JOIN workflow_versions live_version ON live_version.id = workflows.live_version_id").
		Joins("JOIN organizations ON organizations.id = workflows.organization_id").
		Select(
			"workflows.id",
			"workflows.organization_id",
			"workflows.live_version_id",
			"workflows.folder_id",
			"workflows.name",
			"workflows.created_by",
			"workflows.created_at",
			"workflows.updated_at",
			"COALESCE(workflows.deleted_at, organizations.deleted_at) AS deleted_at",
			"live_version.description AS description",
		).
		Where("workflows.deleted_at IS NOT NULL OR organizations.deleted_at IS NOT NULL").
		Find(&canvases).
		Error

	if err != nil {
		return nil, err
	}

	return canvases, nil
}

func ListMaybeDeletedCanvasesByOrganizationInTransaction(tx *gorm.DB, orgID uuid.UUID) ([]Canvas, error) {
	var canvases []Canvas

	// Organization teardown must include every workflow for the org when deciding
	// whether cleanup can continue.
	err := queryCanvasWithLiveVersion(tx).
		Unscoped().
		Where("workflows.organization_id = ?", orgID).
		Find(&canvases).
		Error
	if err != nil {
		return nil, err
	}

	return canvases, nil
}

func LockCanvas(tx *gorm.DB, id uuid.UUID) (*Canvas, error) {
	var canvas Canvas

	err := tx.
		Unscoped().
		Model(&Canvas{}).
		Joins("JOIN organizations ON organizations.id = workflows.organization_id").
		Select(
			"workflows.id",
			"workflows.organization_id",
			"workflows.live_version_id",
			"workflows.folder_id",
			"workflows.name",
			"workflows.created_by",
			"workflows.created_at",
			"workflows.updated_at",
			"COALESCE(workflows.deleted_at, organizations.deleted_at) AS deleted_at",
		).
		Clauses(clause.Locking{
			Strength: "UPDATE",
			Table:    clause.Table{Name: "workflows"},
			Options:  "SKIP LOCKED",
		}).
		Where("workflows.id = ?", id).
		Where("workflows.deleted_at IS NOT NULL OR organizations.deleted_at IS NOT NULL").
		First(&canvas).
		Error

	if err != nil {
		return nil, err
	}

	if canvas.LiveVersionID != nil {
		liveVersion, err := FindCanvasVersionInTransaction(tx, canvas.ID, *canvas.LiveVersionID)
		if err != nil {
			return nil, err
		}

		canvas.Name = liveVersion.Name
		canvas.Description = liveVersion.Description
	}

	return &canvas, nil
}

func CountCanvasesByOrganizationIDs(orgIDs []string) (map[string]int64, error) {
	counts := make(map[string]int64)
	if len(orgIDs) == 0 {
		return counts, nil
	}

	type row struct {
		OrganizationID string
		Count          int64
	}

	var rows []row
	err := database.Conn().
		Table("workflows").
		Select("organization_id, COUNT(*) AS count").
		Where("deleted_at IS NULL").
		Where("organization_id IN ?", orgIDs).
		Group("organization_id").
		Scan(&rows).
		Error
	if err != nil {
		return nil, err
	}

	for _, r := range rows {
		counts[r.OrganizationID] = r.Count
	}

	return counts, nil
}

func CountCanvasesByOrganization(orgID string) (int64, error) {
	return CountCanvasesByOrganizationInTransaction(database.Conn(), orgID)
}

func CountCanvasesByOrganizationInTransaction(tx *gorm.DB, orgID string) (int64, error) {
	var count int64
	err := tx.Model(&Canvas{}).
		Where("organization_id = ?", orgID).
		Count(&count).
		Error
	if err != nil {
		return 0, err
	}

	return count, nil
}
