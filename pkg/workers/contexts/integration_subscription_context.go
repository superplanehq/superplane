package contexts

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"gorm.io/gorm"
)

type IntegrationSubscriptionContext struct {
	tx             *gorm.DB
	registry       *registry.Registry
	node           *models.CanvasNode
	integration    *models.Integration
	subscription   *models.NodeSubscription
	integrationCtx *IntegrationContext
}

func NewIntegrationSubscriptionContext(
	tx *gorm.DB,
	registry *registry.Registry,
	subscription *models.NodeSubscription,
	node *models.CanvasNode,
	integration *models.Integration,
	integrationCtx *IntegrationContext,
) core.IntegrationSubscriptionContext {
	return &IntegrationSubscriptionContext{
		tx:             tx,
		registry:       registry,
		subscription:   subscription,
		node:           node,
		integration:    integration,
		integrationCtx: integrationCtx,
	}
}

func (c *IntegrationSubscriptionContext) Configuration() any {
	return c.subscription.Configuration.Data()
}

func (c *IntegrationSubscriptionContext) SendMessage(message any) error {
	switch c.subscription.NodeType {
	case models.NodeTypeComponent:
		return c.sendMessageToComponent(message)

	case models.NodeTypeTrigger:
		return c.sendMessageToTrigger(message)
	}

	return fmt.Errorf("node type %s does not support messages", c.subscription.NodeType)
}

func (c *IntegrationSubscriptionContext) sendMessageToComponent(message any) error {
	nodeRef := c.subscription.NodeRef.Data()
	if nodeRef.Component == nil {
		return fmt.Errorf("invalid component ref")
	}

	componentName := nodeRef.Component.Name
	component, err := c.registry.GetComponent(componentName)
	if err != nil {
		return fmt.Errorf("component %s not found", componentName)
	}

	integrationComponent, ok := component.(core.IntegrationComponent)
	if !ok {
		return fmt.Errorf("component %s is not an app component", componentName)
	}

	return integrationComponent.OnIntegrationMessage(core.IntegrationMessageContext{
		HTTP:          c.registry.HTTPContext(),
		Configuration: c.node.Configuration.Data(),
		NodeMetadata:  NewNodeMetadataContext(c.tx, c.node),
		Integration:   c.integrationCtx,
		Events:        NewEventContext(c.tx, c.node),
		Message:       message,
		Logger:        logging.WithIntegration(logging.ForNode(*c.node), *c.integration),
		FindExecutionByKV: func(key string, value string) (*core.ExecutionContext, error) {
			return c.findExecutionByKV(key, value)
		},
	})
}

func (c *IntegrationSubscriptionContext) sendMessageToTrigger(message any) error {
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
		HTTP:              c.registry.HTTPContext(),
		Configuration:     c.node.Configuration.Data(),
		NodeMetadata:      NewNodeMetadataContext(c.tx, c.node),
		Integration:       c.integrationCtx,
		Message:           message,
		Events:            NewEventContext(c.tx, c.node),
		Logger:            logging.WithIntegration(logging.ForNode(*c.node), *c.integration),
		FindExecutionByKV: c.findExecutionByKV,
	})
}

func (c *IntegrationSubscriptionContext) findExecutionByKV(key string, value string) (*core.ExecutionContext, error) {
	execution, err := models.FirstNodeExecutionByKVInTransaction(c.tx, c.node.WorkflowID, c.node.NodeID, key, value)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}

		return nil, err
	}

	return &core.ExecutionContext{
		ID:             execution.ID,
		WorkflowID:     execution.WorkflowID.String(),
		NodeID:         execution.NodeID,
		Configuration:  execution.Configuration.Data(),
		HTTP:           c.registry.HTTPContext(),
		Metadata:       NewExecutionMetadataContext(c.tx, execution),
		NodeMetadata:   NewNodeMetadataContext(c.tx, c.node),
		ExecutionState: NewExecutionStateContext(c.tx, execution),
		Requests:       NewExecutionRequestContext(c.tx, execution),
		Integration:    c.integrationCtx,
		Logger:         logging.WithExecution(logging.ForNode(*c.node), execution, nil),
		Notifications:  NewNotificationContext(c.tx, c.integration.OrganizationID, execution.WorkflowID),
	}, nil
}
