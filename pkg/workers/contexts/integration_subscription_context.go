package contexts

import (
	"errors"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"gorm.io/gorm"
)

const datadogIntegrationApp = "datadog"

type IntegrationSubscriptionContext struct {
	tx             *gorm.DB
	registry       *registry.Registry
	node           *models.CanvasNode
	integration    *models.Integration
	subscription   *models.NodeSubscription
	integrationCtx *IntegrationContext
	onNewEvents    func([]models.CanvasEvent)
}

func NewIntegrationSubscriptionContext(
	tx *gorm.DB,
	registry *registry.Registry,
	subscription *models.NodeSubscription,
	node *models.CanvasNode,
	integration *models.Integration,
	integrationCtx *IntegrationContext,
	onNewEvents func([]models.CanvasEvent),
) core.IntegrationSubscriptionContext {
	return &IntegrationSubscriptionContext{
		tx:             tx,
		registry:       registry,
		subscription:   subscription,
		node:           node,
		integration:    integration,
		integrationCtx: integrationCtx,
		onNewEvents:    onNewEvents,
	}
}

func (c *IntegrationSubscriptionContext) Configuration() any {
	return c.subscription.Configuration.Data()
}

func (c *IntegrationSubscriptionContext) SendMessage(message any) error {
	switch c.subscription.NodeType {
	case models.NodeTypeComponent:
		return c.sendMessageToAction(message)

	case models.NodeTypeTrigger:
		return c.sendMessageToTrigger(message)
	}

	return fmt.Errorf("node type %s does not support messages", c.subscription.NodeType)
}

func (c *IntegrationSubscriptionContext) sendMessageToAction(message any) error {
	nodeRef := c.subscription.NodeRef.Data()
	if nodeRef.Component == nil {
		return fmt.Errorf("invalid component ref")
	}

	name := nodeRef.Component.Name
	action, err := c.registry.GetAction(name)
	if err != nil {
		return fmt.Errorf("action %s not found", name)
	}

	integrationAction, ok := action.(core.IntegrationAction)
	if !ok {
		return fmt.Errorf("action %s is not an app action", name)
	}

	return integrationAction.OnIntegrationMessage(core.IntegrationMessageContext{
		HTTP:          c.registry.HTTPContextInTransaction(c.tx),
		Configuration: c.node.Configuration.Data(),
		NodeMetadata:  NewNodeMetadataContext(c.tx, c.node),
		Integration:   c.integrationCtx,
		Events:        NewEventContext(c.tx, c.node, nil, c.onNewEvents),
		Message:       message,
		Logger:        logging.WithIntegration(logging.ForNode(*c.node), *c.integration),
		FindExecutionByKV: func(key string, value string) (*core.ExecutionContext, error) {
			return c.findExecutionByKV(key, value)
		},
	})
}

func (c *IntegrationSubscriptionContext) sendMessageToTrigger(message any) error {
	skip, err := SkipPausedIntakeFeed(c.tx, c.node.WorkflowID)
	if err != nil {
		return err
	}
	if skip {
		return nil
	}

	if _, err := models.FindLiveCanvasVersionInTransaction(c.tx, c.node.WorkflowID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}

	nodeRef := c.subscription.NodeRef.Data()
	if nodeRef.Trigger == nil {
		return fmt.Errorf("invalid trigger ref")
	}

	triggerName := nodeRef.Trigger.Name
	trigger, err := c.registry.GetTrigger(triggerName)
	if err != nil {
		return fmt.Errorf("trigger %s not found", triggerName)
	}

	integrationTrigger, ok := trigger.(core.IntegrationTrigger)
	if !ok {
		return fmt.Errorf("trigger %s is not an app trigger", trigger.Name())
	}

	return integrationTrigger.OnIntegrationMessage(core.IntegrationMessageContext{
		HTTP:              c.registry.HTTPContextInTransaction(c.tx),
		Configuration:     c.node.Configuration.Data(),
		NodeMetadata:      NewNodeMetadataContext(c.tx, c.node),
		Integration:       c.integrationCtx,
		Message:           message,
		Events:            NewEventContext(c.tx, c.node, nil, c.onNewEvents),
		Logger:            c.triggerLogger(),
		FindExecutionByKV: c.findExecutionByKV,
	})
}

func (c *IntegrationSubscriptionContext) triggerLogger() *log.Entry {
	logger := logging.WithIntegration(logging.ForNode(*c.node), *c.integration)
	if c.integration.AppName != datadogIntegrationApp {
		return logger
	}
	return logger.WithFields(c.datadogDeliveryLogFields())
}

func (c *IntegrationSubscriptionContext) datadogDeliveryLogFields() log.Fields {
	fields := log.Fields{
		"organization_id":   c.integration.OrganizationID.String(),
		"organization_name": "",
		"integration_id":    c.integration.ID.String(),
		"workspace_id":      "",
		"workspace_name":    "",
		"intake_id":         "",
		"intake_name":       "",
	}

	organization, err := models.FindOrganizationByIDInTransaction(c.tx, c.integration.OrganizationID.String())
	if err == nil && organization != nil {
		fields["organization_name"] = organization.Name
	}

	canvas, canvasErr := models.FindCanvasInTransaction(c.tx, c.integration.OrganizationID, c.node.WorkflowID)
	if canvasErr == nil && canvas != nil && canvas.FactoryID != nil {
		fields["workspace_id"] = canvas.FactoryID.String()
		factory, factoryErr := models.FindFactory(c.tx, c.integration.OrganizationID, *canvas.FactoryID)
		if factoryErr == nil && factory != nil {
			fields["workspace_name"] = factory.Name
		}
	}

	intake, intakeErr := models.FindFactoryIntakeByCanvasID(c.tx, c.node.WorkflowID)
	if intakeErr != nil || intake == nil {
		return fields
	}

	fields["intake_id"] = intake.ID.String()
	fields["intake_name"] = intakeCanvasName(canvas, intake)
	return fields
}

func intakeCanvasName(canvas *models.Canvas, intake *models.FactoryIntake) string {
	if canvas != nil && canvas.Name != "" {
		return canvas.Name
	}
	if intake == nil {
		return ""
	}
	return intake.Name()
}

func (c *IntegrationSubscriptionContext) findExecutionByKV(key string, value string) (*core.ExecutionContext, error) {
	execution, err := models.FirstNodeExecutionByKVInTransaction(c.tx, c.node.WorkflowID, c.node.NodeID, key, value)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}

		return nil, err
	}

	canvasName := ""
	if workflow, err := models.FindCanvasWithoutOrgScopeInTransaction(c.tx, execution.WorkflowID); err == nil && workflow != nil {
		canvasName = workflow.Name
	}

	return &core.ExecutionContext{
		ID:             execution.ID,
		WorkflowID:     execution.WorkflowID.String(),
		OrganizationID: c.integration.OrganizationID.String(),
		CanvasName:     canvasName,
		NodeID:         execution.NodeID,
		NodeName:       c.node.Name,
		Configuration:  execution.Configuration.Data(),
		HTTP:           c.registry.HTTPContextInTransaction(c.tx),
		Metadata:       NewExecutionMetadataContext(c.tx, execution),
		NodeMetadata:   NewNodeMetadataContext(c.tx, c.node),
		ExecutionState: NewExecutionStateContext(c.tx, execution, c.onNewEvents),
		Requests:       NewExecutionRequestContext(c.tx, execution),
		Integration:    c.integrationCtx,
		Logger:         logging.WithExecution(logging.ForNode(*c.node), execution),
		CanvasMemory:   NewCanvasMemoryContext(c.tx, execution.WorkflowID),
	}, nil
}
