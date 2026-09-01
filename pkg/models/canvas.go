package models

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrCanvasNameAlreadyExists = errors.New("canvas name already exists")

// Canvas names are unique inside the factory that owns the canvas, and unique
// per organization for canvases that no factory owns.
var canvasNameUniqueConstraints = []string{
	"workflows_organization_id_name_active_key",
	"workflows_factory_id_name_active_key",
}

type Canvas struct {
	ID                          uuid.UUID
	OrganizationID              uuid.UUID
	FactoryID                   *uuid.UUID
	LiveVersionID               *uuid.UUID
	CanvasFolderID              *uuid.UUID `gorm:"column:folder_id"`
	Name                        string
	Description                 string
	CreatedBy                   *uuid.UUID
	DismissedAgentSuggestionIDs datatypes.JSONSlice[string]
	CreatedAt                   *time.Time
	UpdatedAt                   *time.Time
	DeletedAt                   gorm.DeletedAt `gorm:"index"`
}

func (c *Canvas) TableName() string {
	return "workflows"
}

// DismissAgentSuggestion appends suggestionID to the canvas-scoped dismissal list.
func (c *Canvas) DismissAgentSuggestion(tx *gorm.DB, suggestionID string) error {
	if suggestionID == "" {
		return nil
	}
	if slices.Contains(c.DismissedAgentSuggestionIDs, suggestionID) {
		return nil
	}

	updated := append(append(datatypes.JSONSlice[string]{}, c.DismissedAgentSuggestionIDs...), suggestionID)
	now := time.Now()
	result := tx.Model(&Canvas{}).
		Where("organization_id = ? AND id = ?", c.OrganizationID, c.ID).
		Updates(map[string]any{
			"dismissed_agent_suggestion_ids": updated,
			"updated_at":                     now,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	c.DismissedAgentSuggestionIDs = updated
	c.UpdatedAt = &now
	return nil
}

func MapCanvasNameUniqueConstraintError(err error) error {
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && slices.Contains(canvasNameUniqueConstraints, pgErr.ConstraintName) {
		return ErrCanvasNameAlreadyExists
	}

	return err
}

// scopeToCanvasNameOwner narrows a query to the scope a canvas name is unique
// in: the factory when one owns the canvas, the organization otherwise.
func scopeToCanvasNameOwner(tx *gorm.DB, organizationID uuid.UUID, factoryID *uuid.UUID) *gorm.DB {
	query := tx.Where("organization_id = ?", organizationID)
	if factoryID == nil {
		return query.Where("factory_id IS NULL")
	}

	return query.Where("factory_id = ?", *factoryID)
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

// FindCanvasByName looks a canvas up in the scope its name is unique in. Pass
// the factory ID to search inside a factory, or nil to search the canvases that
// no factory owns.
func FindCanvasByName(tx *gorm.DB, organizationID uuid.UUID, factoryID *uuid.UUID, name string) (*Canvas, error) {
	var canvas Canvas
	err := scopeToCanvasNameOwner(tx, organizationID, factoryID).
		Where("name = ?", name).
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
	query := database.Conn().Model(&Canvas{}).Where("organization_id = ?", orgID).Where("name NOT IN ?", []string{PlanningCanvasName, PlanningCanvasLegacyName})

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
		Where("organization_id = ? AND name NOT IN ?", orgID, []string{PlanningCanvasName, PlanningCanvasLegacyName}).
		Order("name ASC").
		Find(&canvases).
		Error

	if err != nil {
		return nil, err
	}

	return canvases, nil
}

// AvailableCanvasName returns preferred, or preferred with the lowest " (n)"
// suffix that no other canvas in the same scope holds. A generated canvas has to
// pick a free name before insert. Pass the factory ID for a canvas the factory
// owns, or nil for an organization-level canvas.
func AvailableCanvasName(tx *gorm.DB, organizationID uuid.UUID, factoryID *uuid.UUID, preferred string) (string, error) {
	var names []string
	err := scopeToCanvasNameOwner(tx.Model(&Canvas{}), organizationID, factoryID).
		Where("name = ? OR name LIKE ?", preferred, preferred+" (%)").
		Pluck("name", &names).
		Error
	if err != nil {
		return "", err
	}

	taken := make(map[string]bool, len(names))
	for _, name := range names {
		taken[name] = true
	}

	if !taken[preferred] {
		return preferred, nil
	}

	for suffix := 2; ; suffix++ {
		candidate := fmt.Sprintf("%s (%d)", preferred, suffix)
		if !taken[candidate] {
			return candidate, nil
		}
	}
}

func ListOrganizationCanvases(tx *gorm.DB, organizationID uuid.UUID) ([]Canvas, error) {
	var canvases []Canvas
	err := tx.
		Where("organization_id = ? AND factory_id IS NULL", organizationID).
		Order("name ASC").
		Find(&canvases).
		Error
	if err != nil {
		return nil, err
	}

	return canvases, nil
}

func ListDeletedCanvases(db *gorm.DB) ([]Canvas, error) {
	var canvases []Canvas
	err := db.
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
// the canvas row can be removed. Each call deletes at most maxRecords rows total,
// using SQL LIMIT per resource type so large orphan sets cannot time out.
func (c *Canvas) DeleteRemainingResources(db *gorm.DB, maxRecords int) (*RunDeletionSummary, bool, error) {
	summary := &RunDeletionSummary{}

	type remainingResource struct {
		model any
		apply func(int64)
	}

	resources := []remainingResource{
		{model: &CanvasNodeRequest{}, apply: func(count int64) { summary.NodeRequests = count }},
		{model: &CanvasNodeExecutionKV{}, apply: func(count int64) { summary.NodeExecutionKVs = count }},
		{model: &CanvasNodeQueueItem{}, apply: func(count int64) { summary.NodeQueueItems = count }},
		{model: &CanvasEvent{}, apply: func(count int64) { summary.Events = count }},
		{model: &CanvasNodeExecution{}, apply: func(count int64) { summary.NodeExecutions = count }},
		{model: &CanvasRun{}, apply: func(count int64) { summary.Runs = count }},
	}

	for _, resource := range resources {
		if summary.TotalRecords() >= int64(maxRecords) {
			return summary, false, nil
		}

		budget := maxRecords - int(summary.TotalRecords())
		count, err := deleteRowsLimited(db, resource.model, budget, "workflow_id = ?", c.ID)
		if err != nil {
			return nil, false, err
		}

		resource.apply(count)
	}

	return summary, summary.TotalRecords() < int64(maxRecords), nil
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
