package core

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/configuration"
)

type Application interface {
	/*
	 * The name of the application.
	 */
	Name() string

	/*
	 * Display name for the application.
	 */
	Label() string

	/*
	 * The icon used by the application.
	 */
	Icon() string

	/*
	 * A description of what the application does.
	 */
	Description() string

	/*
	 * Markdown-formatted instructions shown in the installation modal.
	 */
	InstallationInstructions() string

	/*
	 * The configuration fields of the application.
	 */
	Configuration() []configuration.Field

	/*
	 * The list of components exposed by the application.
	 */
	Components() []Component

	/*
	 * The list of triggers exposed by the application.
	 */
	Triggers() []Trigger

	/*
	 * Called when configuration changes.
	 */
	Sync(ctx SyncContext) error

	/*
	 * List resources of a given type.
	 */
	ListResources(resourceType string, ctx ListResourcesContext) ([]ApplicationResource, error)

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
	 * Set up webhooks through the app installation, in the external system.
	 * This is called by the webhook provisioner, for pending webhook records.
	 */
	SetupWebhook(ctx SetupWebhookContext) (any, error)

	/*
	 * Delete webhooks through the app installation, in the external system.
	 * This is called by the webhook cleanup worker, for webhook records that were deleted.
	 */
	CleanupWebhook(ctx CleanupWebhookContext) error
}

type AppComponent interface {

	/*
	 * AppComponent inherits all the methods from Component interface,
	 * and adds a couple more, which are only applicable to app components.
	 */
	Component

	OnAppMessage(ctx AppMessageContext) error
}

type AppTrigger interface {

	/*
	 * Inherits all the methods from Trigger interface,
	 * and adds a couple more, which are only applicable to app triggers.
	 */
	Trigger

	OnAppMessage(ctx AppMessageContext) error
}

type AppMessageContext struct {
	Message         any
	Configuration   any
	Logger          *logrus.Entry
	AppInstallation AppInstallationContext
	Events          EventContext
}

type ApplicationResource struct {
	Type string
	Name string
	ID   string
}

type ListResourcesContext struct {
	Logger          *logrus.Entry
	HTTP            HTTPContext
	AppInstallation AppInstallationContext
}

type SetupWebhookContext struct {
	HTTP            HTTPContext
	Webhook         WebhookContext
	Logger          *logrus.Entry
	AppInstallation AppInstallationContext
}

type CleanupWebhookContext struct {
	HTTP            HTTPContext
	Webhook         WebhookContext
	AppInstallation AppInstallationContext
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
	InstallationID  string
	HTTP            HTTPContext
	AppInstallation AppInstallationContext
}

/*
 * AppInstallationContext allows components to access app installation information.
 */
type AppInstallationContext interface {

	//
	// Control the metadata and config of the app installation
	//
	ID() uuid.UUID
	GetMetadata() any
	SetMetadata(any)
	GetConfig(name string) ([]byte, error)

	//
	// Control the state of the app installation
	//
	GetState() string
	SetState(state, stateDescription string)

	//
	// Control the browser action of the app installation
	//
	NewBrowserAction(action BrowserAction)
	RemoveBrowserAction()

	//
	// Control the secrets of the app installation
	//
	SetSecret(name string, value []byte) error
	GetSecrets() ([]InstallationSecret, error)

	/*
	 * Request a new webhook from the app installation.
	 * Called from the components/triggers Setup().
	 */
	RequestWebhook(configuration any) error

	/*
	 * Subscribe to app events.
	 */
	Subscribe(any) (*uuid.UUID, error)

	/*
	 * Schedule a sync call for the app installation.
	 */
	ScheduleResync(interval time.Duration) error

	/*
	 * List app installation subscriptions from nodes.
	 */
	ListSubscriptions() ([]AppSubscriptionContext, error)
}

type AppSubscriptionContext interface {
	Configuration() any
	SendMessage(any) error
}

type InstallationSecret struct {
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
	AppInstallation AppInstallationContext
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
