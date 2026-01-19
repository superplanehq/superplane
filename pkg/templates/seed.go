package templates

import (
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/workflows"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/workflows"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/protobuf/encoding/protojson"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	log "github.com/sirupsen/logrus"
)

var seedLockID int64 = 1234567890

func Setup(registry *registry.Registry) error {
	log.Info("Setting up templates...")

	if err := SeedTemplates(registry); err != nil {
		log.Warnf("Failed to seed templates: %v", err)
	}

	return nil
}

func SeedTemplates(registry *registry.Registry) error {
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		locked, err := lockTemplateSeed(tx)
		if err != nil {
			return err
		}
		if !locked {
			return nil
		}
		defer unlockTemplateSeed(tx)

		if err := deleteAllTemplateWorkflows(tx); err != nil {
			return err
		}

		templateDir, err := templateDir()
		if err != nil {
			return err
		}

		entries, err := os.ReadDir(templateDir)
		if err != nil {
			return fmt.Errorf("read template assets: %w", err)
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
				continue
			}

			data, err := os.ReadFile(path.Join(templateDir, entry.Name()))
			if err != nil {
				return fmt.Errorf("read template %s: %w", entry.Name(), err)
			}

			jsonData, err := yaml.YAMLToJSON(data)
			if err != nil {
				return fmt.Errorf("parse template %s: %w", entry.Name(), err)
			}

			var workflow pb.Workflow
			if err := protojson.Unmarshal(jsonData, &workflow); err != nil {
				return fmt.Errorf("parse template %s: %w", entry.Name(), err)
			}

			if workflow.Metadata == nil {
				return fmt.Errorf("template %s missing metadata", entry.Name())
			}

			if workflow.Metadata.Name == "" {
				return fmt.Errorf("template %s missing name", entry.Name())
			}

			workflow.Metadata.IsTemplate = true

			if err := createTemplateWorkflow(tx, registry, &workflow); err != nil {
				return fmt.Errorf("create template %s: %w", workflow.Metadata.Name, err)
			}
		}

		return nil
	})
}

func deleteAllTemplateWorkflows(tx *gorm.DB) error {
	var err error

	var workflowIDs []uuid.UUID

	err = tx.
		Unscoped().
		Model(&models.Workflow{}).
		Where("organization_id = ?", models.TemplateOrganizationID).
		Where("is_template = ?", true).
		Pluck("id", &workflowIDs).Error

	if err != nil {
		return err
	}

	err = tx.
		Unscoped().
		Model(&models.WorkflowNode{}).
		Where("workflow_id IN (?)", workflowIDs).
		Delete(&models.WorkflowNode{}).Error

	if err != nil {
		return err
	}

	err = tx.
		Unscoped().
		Model(&models.Workflow{}).
		Where("organization_id = ?", models.TemplateOrganizationID).
		Where("is_template = ?", true).
		Delete(&models.Workflow{}).Error

	if err != nil {
		return err
	}

	return nil
}

func createTemplateWorkflow(tx *gorm.DB, registry *registry.Registry, template *pb.Workflow) error {
	organizationID := models.TemplateOrganizationID.String()
	nodes, edges, err := workflows.ParseWorkflow(registry, organizationID, template)
	if err != nil {
		return err
	}

	expandedNodes, err := workflows.ExpandNodes(organizationID, nodes)
	if err != nil {
		return err
	}

	now := time.Now()
	workflow := models.Workflow{
		ID:             uuid.New(),
		OrganizationID: models.TemplateOrganizationID,
		IsTemplate:     true,
		Name:           template.Metadata.Name,
		Description:    template.Metadata.Description,
		CreatedAt:      &now,
		UpdatedAt:      &now,
		Edges:          datatypes.NewJSONSlice(edges),
		Nodes:          datatypes.NewJSONSlice(expandedNodes),
	}

	if err := tx.Create(&workflow).Error; err != nil {
		return err
	}

	for _, node := range expandedNodes {
		var parentNodeID *string
		if idx := strings.Index(node.ID, ":"); idx != -1 {
			parent := node.ID[:idx]
			parentNodeID = &parent
		}

		workflowNode := models.WorkflowNode{
			WorkflowID:    workflow.ID,
			NodeID:        node.ID,
			ParentNodeID:  parentNodeID,
			Name:          node.Name,
			State:         models.WorkflowNodeStateReady,
			Type:          node.Type,
			Ref:           datatypes.NewJSONType(node.Ref),
			Configuration: datatypes.NewJSONType(node.Configuration),
			Metadata:      datatypes.NewJSONType(node.Metadata),
			CreatedAt:     &now,
			UpdatedAt:     &now,
		}

		if err := tx.Create(&workflowNode).Error; err != nil {
			return err
		}
	}

	return nil
}

func lockTemplateSeed(tx *gorm.DB) (bool, error) {
	var locked bool
	if err := tx.Raw("SELECT pg_try_advisory_lock(?)", seedLockID).Scan(&locked).Error; err != nil {
		return false, fmt.Errorf("lock template seed: %w", err)
	}
	return locked, nil
}

func unlockTemplateSeed(tx *gorm.DB) {
	_ = tx.Exec("SELECT pg_advisory_unlock(?)", seedLockID).Error
}

func templateDir() (string, error) {
	dir := os.Getenv("TEMPLATE_DIR")
	if dir == "" {
		return "", fmt.Errorf("TEMPLATE_DIR is not set")
	}

	return path.Join(dir, "canvases"), nil
}
