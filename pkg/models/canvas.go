package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Canvas struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	IsTemplate     bool
	Name           string
	Description    string
	CreatedBy      *uuid.UUID
	CreatedAt      *time.Time
	UpdatedAt      *time.Time
	DeletedAt      gorm.DeletedAt `gorm:"index"`
	Nodes          datatypes.JSONSlice[NodeDefinition]
	Edges          datatypes.JSONSlice[Edge]
}

func (c *Canvas) TableName() string {
	return "workflows"
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

func (c *Canvas) FindEdges(sourceID string, channel string) []Edge {
	edges := []Edge{}

	for _, edge := range c.Edges {
		if edge.SourceID == sourceID && edge.Channel == channel {
			edges = append(edges, edge)
		}
	}

	return edges
}

func (c *Canvas) SoftDelete() error {
	return c.SoftDeleteInTransaction(database.Conn())
}

func (c *Canvas) SoftDeleteInTransaction(tx *gorm.DB) error {
	now := time.Now()
	timestamp := now.Unix()

	newName := fmt.Sprintf("%s (deleted-%d)", c.Name, timestamp)
	return tx.Model(c).Updates(map[string]interface{}{
		"deleted_at": now,
		"name":       newName,
	}).Error
}

func FindCanvas(orgID, id uuid.UUID) (*Canvas, error) {
	return FindCanvasInTransaction(database.Conn(), orgID, id)
}

func FindCanvasByName(name string, organizationID uuid.UUID) (*Canvas, error) {
	var canvas Canvas
	err := database.Conn().
		Where("name = ? AND organization_id = ?", name, organizationID).
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
	err := tx.
		Where("organization_id = ?", TemplateOrganizationID).
		Where("is_template = ?", true).
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

func ListCanvases(orgID string, includeTemplates bool) ([]Canvas, error) {
	var canvases []Canvas
	var query *gorm.DB
	if includeTemplates {
		query = database.Conn().Where(
			"(organization_id = ?) OR (organization_id = ? AND is_template = ?)",
			orgID,
			TemplateOrganizationID,
			true,
		)
	} else {
		query = database.Conn().Where("organization_id = ?", orgID)
	}

	err := query.
		Order("name ASC").
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
	err := tx.
		Where("id = ?", id).
		Where("is_template = ?", true).
		First(&canvas).
		Error

	if err != nil {
		return nil, err
	}

	return &canvas, nil
}

func ListDeletedCanvases() ([]Canvas, error) {
	var canvases []Canvas
	err := database.Conn().
		Unscoped().
		Where("deleted_at IS NOT NULL").
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
		Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where("id = ?", id).
		Where("deleted_at IS NOT NULL").
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
