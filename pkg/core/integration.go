package core

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/oidc"
)

type Integration interface {
	/*
	 * The name of the integration.
	 */
	Name() string

	/*
	 * Display name for the integration.
	 */
	Label() string

	/*
	 * The icon used by the integration.
	 */
	Icon() string

	/*
	 * A description of what the integration does.
	 */
	Description() string

	/*
	 * Markdown-formatted instructions shown in the connection modal.
	 */
	Instructions() string

	/*
	 * The configuration fields of the integration.
	 */
	Configuration() []configuration.Field

	/*
	 * The list of components exposed by the integration.
	 */
	Components() []Component

	/*
	 * The list of triggers exposed by the integration.
	 */
	Triggers() []Trigger

	/*
	 * Called when configuration changes.
	 */
	Sync(ctx SyncContext) error

	/*
	 * Called when the integration is deleted.
	 */
	Cleanup(ctx IntegrationCleanupContext) error

	/*
	 * The list of actions exposed by the integration.
	 */
	Actions() []Action

	/*
	 * Execute an action - defined in Actions() -
	 * on the integration.
	 */
	HandleAction(ctx IntegrationActionContext) error

	/*
	 * List resources of a given type.
	 */
	ListResources(resourceType string, ctx ListResourcesContext) ([]IntegrationResource, error)

	/*
	 * HTTP request handler
	 */
	HandleRequest(ctx HTTPRequestContext)

	/*
	 * Used to compare webhook configurations.
	 * If the configuration is the same,
	 * the system will reuse the existing webhook.
	 */
	CompareWebhookConfig(a, b any) (bool, error)

	/*
	 * Set up webhooks through the integration, in the external system.
	 * This is called by the webhook provisioner, for pending webhook records.
	 */
	SetupWebhook(ctx SetupWebhookContext) (any, error)

	/*
	 * Delete webhooks through the integration, in the external system.
	 * This is called by the webhook cleanup worker, for webhook records that were deleted.
	 */
	CleanupWebhook(ctx CleanupWebhookContext) error
}

type IntegrationComponent interface {

	/*
	 * IntegrationComponent inherits all the methods from Component interface,
	 * and adds a couple more, which are only applicable to app components.
	 */
	Component

	OnIntegrationMessage(ctx IntegrationMessageContext) error
}

type IntegrationTrigger interface {

	/*
	 * Inherits all the methods from Trigger interface,
	 * and adds a couple more, which are only applicable to integration triggers.
	 */
	Trigger

	OnIntegrationMessage(ctx IntegrationMessageContext) error
}

type IntegrationMessageContext struct {
	Message       any
	Configuration any
	Logger        *logrus.Entry
	Integration   IntegrationContext
	Events        EventContext
}

type IntegrationResource struct {
	Type string
	Name string
	ID   string
}

type ListResourcesContext struct {
	Logger      *logrus.Entry
	HTTP        HTTPContext
	Integration IntegrationContext
}

type SetupWebhookContext struct {
	HTTP        HTTPContext
	Webhook     WebhookContext
	Logger      *logrus.Entry
	Integration IntegrationContext
}

type CleanupWebhookContext struct {
	HTTP        HTTPContext
	Webhook     WebhookContext
	Integration IntegrationContext
}

type WebhookOptions struct {
	ID            string
	URL           string
	Secret        []byte
	Configuration any
	Metadata      any
}

type SyncContext struct {
	Configuration   any
	BaseURL         string
	WebhooksBaseURL string
	OrganizationID  string
	HTTP            HTTPContext
	Integration     IntegrationContext
	OIDC            oidc.Provider
}

type IntegrationCleanupContext struct {
	Configuration  any
	BaseURL        string
	OrganizationID string
	Logger         *logrus.Entry
	HTTP           HTTPContext
	Integration    IntegrationContext
}

type IntegrationActionContext struct {
	Name          string
	Parameters    any
	Configuration any
	Logger        *logrus.Entry
	Requests      RequestContext
	Integration   IntegrationContext
}

/*
 * IntegrationContext allows components to access integration information.
 */
type IntegrationContext interface {

	//
	// Control the metadata and config of the integration
	//
	ID() uuid.UUID
	GetMetadata() any
	SetMetadata(any)
	GetConfig(name string) ([]byte, error)

	//
	// Control the state of the integration
	//
	Ready()
	Error(message string)

	//
	// Control the browser action of the integration
	//
	NewBrowserAction(action BrowserAction)
	RemoveBrowserAction()

	//
	// Control the secrets of the integration
	//
	SetSecret(name string, value []byte) error
	GetSecrets() ([]IntegrationSecret, error)

	/*
	 * Request a new webhook from the integration.
	 * Called from the components/triggers Setup().
	 */
	RequestWebhook(configuration any) error

	/*
	 * Subscribe to integration events.
	 */
	Subscribe(any) (*uuid.UUID, error)

	/*
	 * Schedule actions for the integration.
	 */
	ScheduleResync(interval time.Duration) error
	ScheduleActionCall(actionName string, parameters any, interval time.Duration) error

	/*
	 * List integration subscriptions from nodes.
	 */
	ListSubscriptions() ([]IntegrationSubscriptionContext, error)
}

type IntegrationSubscriptionContext interface {
	Configuration() any
	SendMessage(any) error
}

type IntegrationSecret struct {
	Name  string
	Value []byte
}

type BrowserAction struct {
	Description string
	URL         string
	Method      string
	FormFields  map[string]string
}

type HTTPRequestContext struct {
	Logger          *logrus.Entry
	Request         *http.Request
	Response        http.ResponseWriter
	OrganizationID  string
	BaseURL         string
	WebhooksBaseURL string
	HTTP            HTTPContext
	Integration     IntegrationContext
}

/*
 * WebhookContext allows implementations to read/manage Webhook records.
 */
type WebhookContext interface {
	GetID() string
	GetURL() string
	GetSecret() ([]byte, error)
	GetMetadata() any
	GetConfiguration() any
	SetSecret([]byte) error
}
