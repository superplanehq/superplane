package models

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/superplanehq/superplane/pkg/database"
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
	Description    string
	CreatedBy      *uuid.UUID
	CreatedAt      *time.Time
	UpdatedAt      *time.Time
	DeletedAt      gorm.DeletedAt `gorm:"index"`
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

func withActiveCanvas(tx *gorm.DB, workflowIDColumn string) *gorm.DB {
	return tx.
		Joins(fmt.Sprintf("JOIN workflows ON %s = workflows.id", workflowIDColumn)).
		Joins("JOIN organizations ON workflows.organization_id = organizations.id").
		Where("workflows.deleted_at IS NULL").
		Where("organizations.deleted_at IS NULL")
}

func LockCanvasForUpdate(tx *gorm.DB, organizationUUID, canvasID uuid.UUID) (*Canvas, error) {
	lockedCanvas := &Canvas{}

	err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("organization_id = ?", organizationUUID).
		Where("id = ?", canvasID).
		First(lockedCanvas).
		Error
	if err != nil {
		return nil, err
	}

	return lockedCanvas, nil
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
	return tx.Model(c).Updates(map[string]any{
		"name":       newName,
		"deleted_at": now,
	}).Error
}

func FindCanvas(orgID, id uuid.UUID) (*Canvas, error) {
	return FindCanvasInTransaction(database.Conn(), orgID, id)
}

func FindCanvasByName(name string, organizationID uuid.UUID) (*Canvas, error) {
	return FindCanvasByNameInTransaction(database.Conn(), name, organizationID)
}

func FindCanvasByNameInTransaction(tx *gorm.DB, name string, organizationID uuid.UUID) (*Canvas, error) {
	var canvas Canvas
	err := tx.
		Where("name = ? AND organization_id = ?", name, organizationID).
		First(&canvas).
		Error

	if err != nil {
		return nil, err
	}

	return &canvas, nil
}

func FindCanvasInTransaction(tx *gorm.DB, orgID, id uuid.UUID) (*Canvas, error) {
	var canvas Canvas
	err := tx.
		Where("organization_id = ?", orgID).
		Where("id = ?", id).
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
	err := tx.
		Where("id = ?", id).
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
	err := tx.
		Unscoped().
		Where("id = ?", id).
		First(&canvas).
		Error

	if err != nil {
		return nil, err
	}

	return &canvas, nil
}

func ListCanvasesPaginated(orgID, search string, limit, offset int) ([]Canvas, int64, error) {
	query := database.Conn().Model(&Canvas{}).Where("organization_id = ?", orgID)

	if search != "" {
		query = query.Where("name ILIKE ?", "%"+search+"%")
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
	if err := query.Order("name ASC").Find(&canvases).Error; err != nil {
		return nil, 0, err
	}

	return canvases, total, nil
}

func ListCanvases(orgID string) ([]Canvas, error) {
	var canvases []Canvas
	err := database.Conn().
		Where("organization_id = ?", orgID).
		Order("name ASC").
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
		Joins("JOIN organizations ON organizations.id = workflows.organization_id").
		Select(
			"workflows.id",
			"workflows.organization_id",
			"workflows.live_version_id",
			"workflows.folder_id",
			"workflows.name",
			"workflows.description",
			"workflows.created_by",
			"workflows.created_at",
			"workflows.updated_at",
			"COALESCE(workflows.deleted_at, organizations.deleted_at) AS deleted_at",
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
	err := tx.
		Unscoped().
		Where("organization_id = ?", orgID).
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
			"workflows.description",
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

// DeleteRemainingResources removes workflow execution rows still scoped to this
// canvas after all runs have been deleted via CanvasRun.DeleteChain.
//
// Run cleanup covers the normal path: events, executions, queue items, KVs, and
// requests that belong to a run. This sweep is still required because some rows
// are workflow-scoped but not run-scoped — notably trigger node requests created
// without an execution_id (CanvasNode.CreateRequest). Orphan or inconsistent rows
// (e.g. nil run_id from partial routing) are also cleared here before nodes and
// the canvas row can be removed. Deletes are capped per call via maxRecords so
// large final sweeps do not exceed statement timeouts.
func (c *Canvas) DeleteRemainingResources(db *gorm.DB, maxRecords int) (*RunDeletionSummary, bool, error) {
	summary := &RunDeletionSummary{}

	count, err := deleteRows(db, &CanvasNodeRequest{}, "workflow_id = ?", c.ID)
	if err != nil {
		return nil, false, err
	}

	summary.NodeRequests = count
	if summary.TotalRecords() >= int64(maxRecords) {
		return summary, false, nil
	}

	count, err = deleteRows(db, &CanvasNodeExecutionKV{}, "workflow_id = ?", c.ID)
	if err != nil {
		return nil, false, err
	}

	summary.NodeExecutionKVs = count
	if summary.TotalRecords() >= int64(maxRecords) {
		return summary, false, nil
	}

	count, err = deleteRows(db, &CanvasNodeQueueItem{}, "workflow_id = ?", c.ID)
	if err != nil {
		return nil, false, err
	}

	summary.NodeQueueItems = count
	if summary.TotalRecords() >= int64(maxRecords) {
		return summary, false, nil
	}

	count, err = deleteRows(db, &CanvasEvent{}, "workflow_id = ?", c.ID)
	if err != nil {
		return nil, false, err
	}

	summary.Events = count
	if summary.TotalRecords() >= int64(maxRecords) {
		return summary, false, nil
	}

	count, err = deleteRows(db, &CanvasNodeExecution{}, "workflow_id = ?", c.ID)
	if err != nil {
		return nil, false, err
	}

	summary.NodeExecutions = count
	if summary.TotalRecords() >= int64(maxRecords) {
		return summary, false, nil
	}

	count, err = deleteRows(db, &CanvasRun{}, "workflow_id = ?", c.ID)
	if err != nil {
		return nil, false, err
	}

	summary.Runs = count
	if summary.TotalRecords() >= int64(maxRecords) {
		return summary, false, nil
	}

	return summary, true, nil
}

func (c *Canvas) CountRuns(db *gorm.DB) (int64, error) {
	var count int64
	err := db.Model(&CanvasRun{}).
		Where("workflow_id = ?", c.ID).
		Count(&count).
		Error
	if err != nil {
		return 0, err
	}
	return count, nil
}
