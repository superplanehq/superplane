package contexts

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"gorm.io/gorm"
)

type AppSubscriptionContext struct {
	tx           *gorm.DB
	registry     *registry.Registry
	node         *models.WorkflowNode
	installation *models.AppInstallation
	subscription *models.NodeSubscription
	appCtx       *AppInstallationContext
}

func NewAppSubscriptionContext(
	tx *gorm.DB,
	registry *registry.Registry,
	subscription *models.NodeSubscription,
	node *models.WorkflowNode,
	installation *models.AppInstallation,
	appCtx *AppInstallationContext,
) core.AppSubscriptionContext {
	return &AppSubscriptionContext{
		tx:           tx,
		registry:     registry,
		subscription: subscription,
		node:         node,
		installation: installation,
		appCtx:       appCtx,
	}
}

func (c *AppSubscriptionContext) Configuration() any {
	return c.subscription.Configuration.Data()
}

func (c *AppSubscriptionContext) SendMessage(message any) error {
	switch c.subscription.NodeType {
	case models.NodeTypeComponent:
		return c.sendMessageToComponent(message)

	case models.NodeTypeTrigger:
		return c.sendMessageToTrigger(message)
	}

	return fmt.Errorf("node type %s does not support messages", c.subscription.NodeType)
}

func (c *AppSubscriptionContext) sendMessageToComponent(message any) error {
	nodeRef := c.subscription.NodeRef.Data()
	if nodeRef.Component == nil {
		return fmt.Errorf("invalid component ref")
	}

	componentName := nodeRef.Component.Name
	component, err := c.registry.GetComponent(componentName)
	if err != nil {
		return fmt.Errorf("component %s not found", componentName)
	}

	appComponent, ok := component.(core.AppComponent)
	if !ok {
		return fmt.Errorf("component %s is not an app component", componentName)
	}

	return appComponent.OnAppMessage(core.AppMessageContext{
		AppInstallation: c.appCtx,
		Events:          NewEventContext(c.tx, c.node),
		Message:         message,
		Logger:          logging.WithAppInstallation(logging.ForNode(*c.node), *c.installation),
	})
}

func (c *AppSubscriptionContext) sendMessageToTrigger(message any) error {
	nodeRef := c.subscription.NodeRef.Data()
	if nodeRef.Component == nil {
		return fmt.Errorf("invalid trigger ref")
	}

	triggerName := nodeRef.Trigger.Name
	trigger, err := c.registry.GetComponent(triggerName)
	if err != nil {
		return fmt.Errorf("trigger %s not found", triggerName)
	}

	appTrigger, ok := trigger.(core.AppTrigger)
	if !ok {
		return fmt.Errorf("trigger %s is not an app trigger", appTrigger)
	}

	return appTrigger.OnAppMessage(core.AppMessageContext{
		AppInstallation: c.appCtx,
		Message:         message,
		Events:          NewEventContext(c.tx, c.node),
		Logger:          logging.WithAppInstallation(logging.ForNode(*c.node), *c.installation),
	})
}
