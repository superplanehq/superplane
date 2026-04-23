package models

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrCanvasNameAlreadyExists = errors.New("canvas name already exists")

type Canvas struct {
	ID                      uuid.UUID
	OrganizationID          uuid.UUID
	LiveVersionID           *uuid.UUID
	IsTemplate              bool
	Name                    string                                           `gorm:"column:name;->"`
	Description             string                                           `gorm:"column:description;->"`
	ChangeManagementEnabled bool                                             `gorm:"column:change_management_enabled;->"`
	ChangeRequestApprovers  datatypes.JSONSlice[CanvasChangeRequestApprover] `gorm:"column:change_request_approvers;->"`
	CreatedBy               *uuid.UUID
	CreatedAt               *time.Time
	UpdatedAt               *time.Time
	DeletedAt               gorm.DeletedAt `gorm:"index"`
}

func (c *Canvas) EffectiveChangeRequestApprovers() []CanvasChangeRequestApprover {
	if c == nil || len(c.ChangeRequestApprovers) == 0 {
		return DefaultCanvasChangeRequestApprovers()
	}

	approvers := make([]CanvasChangeRequestApprover, len(c.ChangeRequestApprovers))
	copy(approvers, c.ChangeRequestApprovers)
	return approvers
}

func (c *Canvas) TableName() string {
	return "workflows"
}

func queryCanvasWithLiveVersion(tx *gorm.DB) *gorm.DB {
	return tx.
		Model(&Canvas{}).
		Joins("JOIN workflow_versions live_version ON live_version.id = workflows.live_version_id").
		Select(
			"workflows.*",
			"live_version.name AS name",
			"live_version.description AS description",
			"live_version.change_management_enabled AS change_management_enabled",
			"live_version.change_request_approvers AS change_request_approvers",
		)
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
		if err := innerTx.Model(c).Update("deleted_at", now).Error; err != nil {
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
		Where("live_version.name = ? AND workflows.organization_id = ?", name, organizationID).
		First(&canvas).
		Error

	if err != nil {
		return nil, err
	}

	return &canvas, nil
}

func FindCanvasTemplateByName(name string) (*Canvas, error) {
	return FindCanvasTemplateByNameInTransaction(database.Conn(), name)
}

func FindCanvasTemplateByNameInTransaction(tx *gorm.DB, name string) (*Canvas, error) {
	var canvas Canvas
	err := queryCanvasWithLiveVersion(tx).
		Where("workflows.organization_id = ?", TemplateOrganizationID).
		Where("workflows.is_template = ?", true).
		Where("live_version.name = ?", name).
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

func ListCanvases(orgID string, includeTemplates bool) ([]Canvas, error) {
	var canvases []Canvas
	var query *gorm.DB
	if includeTemplates {
		query = queryCanvasWithLiveVersion(database.Conn()).Where(
			"(workflows.organization_id = ?) OR (workflows.organization_id = ? AND workflows.is_template = ?)",
			orgID,
			TemplateOrganizationID,
			true,
		)
	} else {
		query = queryCanvasWithLiveVersion(database.Conn()).Where("workflows.organization_id = ?", orgID)
	}

	err := query.
		Order("live_version.name ASC").
		Find(&canvases).
		Error

	if err != nil {
		return nil, err
	}

	return canvases, nil
}

func FindCanvasTemplate(id uuid.UUID) (*Canvas, error) {
	return FindCanvasTemplateInTransaction(database.Conn(), id)
}

func FindCanvasTemplateInTransaction(tx *gorm.DB, id uuid.UUID) (*Canvas, error) {
	var canvas Canvas
	err := queryCanvasWithLiveVersion(tx).
		Where("workflows.id = ?", id).
		Where("workflows.is_template = ?", true).
		First(&canvas).
		Error

	if err != nil {
		return nil, err
	}

	return &canvas, nil
}

func ListDeletedCanvases() ([]Canvas, error) {
	var canvases []Canvas
	err := queryCanvasWithLiveVersion(database.Conn()).
		Unscoped().
		Where("workflows.deleted_at IS NOT NULL").
		Find(&canvases).
		Error

	if err != nil {
		return nil, err
	}

	return canvases, nil
}

func ListMaybeDeletedCanvasesByOrganizationInTransaction(tx *gorm.DB, orgID uuid.UUID) ([]Canvas, error) {
	var canvases []Canvas

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
		Select(
			"workflows.id",
			"workflows.organization_id",
			"workflows.live_version_id",
			"workflows.is_template",
			"workflows.created_by",
			"workflows.created_at",
			"workflows.updated_at",
			"workflows.deleted_at",
		).
		Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where("workflows.id = ?", id).
		Where("workflows.deleted_at IS NOT NULL").
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
		canvas.ChangeManagementEnabled = liveVersion.ChangeManagementEnabled
		canvas.ChangeRequestApprovers = datatypes.NewJSONSlice(liveVersion.EffectiveChangeRequestApprovers())
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
	err := tx.
		Model(&Canvas{}).
		Where("organization_id = ?", orgID).
		Count(&count).
		Error
	if err != nil {
		return 0, err
	}

	return count, nil
}
