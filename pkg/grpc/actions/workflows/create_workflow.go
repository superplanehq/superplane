package workflows

import (
    "context"
    "time"

    log "github.com/sirupsen/logrus"
    "github.com/google/uuid"
    "github.com/superplanehq/superplane/pkg/authentication"
    "github.com/superplanehq/superplane/pkg/database"
    "github.com/superplanehq/superplane/pkg/models"
    pb "github.com/superplanehq/superplane/pkg/protos/workflows"
    "github.com/superplanehq/superplane/pkg/registry"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
    "gorm.io/datatypes"
    "gorm.io/gorm"
    "gorm.io/gorm/clause"
)

func CreateWorkflow(ctx context.Context, registry *registry.Registry, organizationID string, pbWorkflow *pb.Workflow) (*pb.CreateWorkflowResponse, error) {
    // Intentionally loud diagnostics
    log.WithFields(log.Fields{
        "org_id": organizationID,
        "name":   pbWorkflow.GetName(),
        "nodes":  len(pbWorkflow.GetNodes()),
        "edges":  len(pbWorkflow.GetEdges()),
    }).Info("[LOUD] Creating workflow (begin)")
    log.WithFields(log.Fields{
        "org_id": organizationID,
        "name":   pbWorkflow.GetName(),
    }).Warn("[LOUD] CreateWorkflow ACTION ENTER")

    userID, ok := authentication.GetUserIdFromMetadata(ctx)
    if !ok {
        return nil, status.Error(codes.Unauthenticated, "user not authenticated")
    }

	nodes, edges, err := ParseWorkflow(registry, organizationID, pbWorkflow)
	if err != nil {
		return nil, err
	}

	createdBy := uuid.MustParse(userID)

	now := time.Now()
	workflow := models.Workflow{
		ID:             uuid.New(),
		OrganizationID: uuid.MustParse(organizationID),
		Name:           pbWorkflow.Name,
		Description:    pbWorkflow.Description,
		CreatedBy:      &createdBy,
		CreatedAt:      &now,
		UpdatedAt:      &now,
		Edges:          datatypes.NewJSONSlice(edges),
	}

    err = database.Conn().Transaction(func(tx *gorm.DB) error {
        log.WithFields(log.Fields{"org_id": organizationID}).Info("[LOUD] TX: begin create workflow")
        //
        // Create the workflow record
        //
        err := tx.Clauses(clause.Returning{}).Create(&workflow).Error
        if err != nil {
            return err
        }
        log.WithFields(log.Fields{
            "org_id": organizationID,
            "id":     workflow.ID.String(),
            "name":   workflow.Name,
        }).Info("[LOUD] TX: workflow row created")

		//
		// Create the workflow node records
		//
        for _, node := range nodes {
            workflowNode := models.WorkflowNode{
                WorkflowID:    workflow.ID,
                NodeID:        node.ID,
                Name:          node.Name,
                State:         models.WorkflowNodeStateReady,
                Type:          node.Type,
                Ref:           datatypes.NewJSONType(node.Ref),
                Configuration: datatypes.NewJSONType(node.Configuration),
                CreatedAt:     &now,
                UpdatedAt:     &now,
            }

            if err := tx.Create(&workflowNode).Error; err != nil {
                return err
            }
            log.WithFields(log.Fields{
                "workflow_id": workflow.ID.String(),
                "node_id":     workflowNode.NodeID,
                "node_name":   workflowNode.Name,
                "node_type":   workflowNode.Type,
            }).Info("[LOUD] TX: workflow node row created")
        }

        return nil
    })

    if err != nil {
        return nil, err
    }

    res := &pb.CreateWorkflowResponse{
        Workflow: SerializeWorkflow(&workflow),
    }

    log.WithFields(log.Fields{
        "org_id": organizationID,
        "id":     workflow.ID.String(),
        "name":   workflow.Name,
    }).Info("[LOUD] Workflow created (done)")

    return res, nil
}
