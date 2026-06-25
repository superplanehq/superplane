package public

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/oidc"
	"gorm.io/gorm"
)

func authorizeExecutionToken(tx *gorm.DB, claims oidc.ExecutionTokenClaims) error {
	canvasID, err := uuid.Parse(claims.CanvasID)
	if err != nil {
		return fmt.Errorf("invalid canvas_id claim")
	}

	executionID, err := uuid.Parse(claims.ExecutionID)
	if err != nil {
		return fmt.Errorf("invalid execution_id claim")
	}

	canvas, err := models.FindCanvasWithoutOrgScopeInTransaction(tx, canvasID)
	if err != nil {
		return fmt.Errorf("canvas not found")
	}

	if canvas.OrganizationID.String() != claims.OrgID {
		return fmt.Errorf("canvas does not belong to organization")
	}

	execution, err := models.FindNodeExecutionWithNodeIDInTransaction(tx, canvasID, executionID, claims.NodeID)
	if err != nil {
		return fmt.Errorf("execution not found for canvas node")
	}

	if execution.NodeID != claims.NodeID {
		return fmt.Errorf("execution node mismatch")
	}

	nodes, _, err := models.FindLiveCanvasSpecInTransaction(tx, canvasID)
	if err != nil {
		return fmt.Errorf("failed to load canvas spec: %w", err)
	}

	var liveNode *models.Node
	for i := range nodes {
		if nodes[i].ID == claims.NodeID {
			liveNode = &nodes[i]
			break
		}
	}

	if liveNode == nil {
		return fmt.Errorf("node not found in live canvas")
	}

	componentName := ""
	if liveNode.Ref.Component != nil {
		componentName = liveNode.Ref.Component.Name
	}

	if claims.Component != "" && componentName != claims.Component {
		return fmt.Errorf("node component mismatch")
	}

	if claims.PipelineFile != "" {
		configPipelineFile, _ := liveNode.Configuration["pipelineFile"].(string)
		if configPipelineFile != claims.PipelineFile {
			return fmt.Errorf("node pipeline_file mismatch")
		}
	}

	if claims.ProjectID != "" {
		if err := authorizeProjectID(liveNode, claims.ProjectID); err != nil {
			return err
		}
	}

	return nil
}

func authorizeProjectID(node *models.Node, expectedProjectID string) error {
	configProject, _ := node.Configuration["project"].(string)
	if configProject == expectedProjectID {
		return nil
	}

	type projectMetadata struct {
		Project *struct {
			ID string `mapstructure:"id"`
		} `mapstructure:"project"`
	}

	metadata := projectMetadata{}
	if err := mapstructure.Decode(node.Metadata, &metadata); err != nil {
		return fmt.Errorf("failed to decode node metadata")
	}

	if metadata.Project != nil && metadata.Project.ID == expectedProjectID {
		return nil
	}

	return fmt.Errorf("node project mismatch")
}
