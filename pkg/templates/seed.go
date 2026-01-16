package templates

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
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
)

//go:embed templates/*.yaml
var templateAssets embed.FS

func SeedTemplates(registry *registry.Registry) error {
	entries, err := fs.ReadDir(templateAssets, "templates")
	if err != nil {
		return fmt.Errorf("read template assets: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		data, err := templateAssets.ReadFile("templates/" + entry.Name())
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

		if err := ensureTemplateWorkflow(registry, &workflow); err != nil {
			return fmt.Errorf("ensure template %s: %w", workflow.Metadata.Name, err)
		}
	}

	return nil
}

func ensureTemplateWorkflow(registry *registry.Registry, template *pb.Workflow) error {
	_, err := models.FindWorkflowTemplateByName(template.Metadata.Name)
	if err == nil {
		return nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	return createTemplateWorkflow(registry, template)
}

func createTemplateWorkflow(registry *registry.Registry, template *pb.Workflow) error {
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

	return database.Conn().Transaction(func(tx *gorm.DB) error {
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
	})
}
