package contexts

import (
	"fmt"

	"github.com/google/uuid"
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

	node := c.node
	tx := c.tx
	httpCtx := c.registry.HTTPContext()
	logger := logging.WithIntegration(logging.ForNode(*c.node), *c.integration)

	return integrationComponent.OnIntegrationMessage(core.IntegrationMessageContext{
		HTTP:          httpCtx,
		Configuration: node.Configuration.Data(),
		NodeMetadata:  NewNodeMetadataContext(tx, node),
		Integration:   c.integrationCtx,
		Events:        NewEventContext(tx, node),
		Message:       message,
		Logger:        logger,

		// FindExecutionByKV allows integration components to locate an existing
		// execution by a key-value pair. This enables components that receive
		// async completion events through the integration message path (e.g.,
		// AWS EventBridge) to resolve and finish running executions, rather than
		// only being able to create new root events via Events.Emit().
		FindExecutionByKV: func(key string, value string) (*core.ExecutionContext, error) {
			execution, err := models.FirstNodeExecutionByKVInTransaction(tx, node.WorkflowID, node.NodeID, key, value)
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
				HTTP:           httpCtx,
				Metadata:       NewExecutionMetadataContext(tx, execution),
				NodeMetadata:   NewNodeMetadataContext(tx, node),
				ExecutionState: NewExecutionStateContext(tx, execution),
				Requests:       NewExecutionRequestContext(tx, execution),
				Logger:         logging.WithExecution(logger, execution, nil),
				Notifications:  NewNotificationContext(tx, uuid.Nil, execution.WorkflowID),
			}, nil
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
		HTTP:          c.registry.HTTPContext(),
		Configuration: c.node.Configuration.Data(),
		NodeMetadata:  NewNodeMetadataContext(c.tx, c.node),
		Integration:   c.integrationCtx,
		Message:       message,
		Events:        NewEventContext(c.tx, c.node),
		Logger:        logging.WithIntegration(logging.ForNode(*c.node), *c.integration),
	})
}
