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
	SetupWebhook(ctx AppInstallationContext, options WebhookOptions) (any, error)

	/*
	 * Delete webhooks through the app installation, in the external system.
	 * This is called by the webhook cleanup worker, for webhook records that were deleted.
	 */
	CleanupWebhook(ctx AppInstallationContext, options WebhookOptions) error
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
	 * Schedule a sync call for the app installation.
	 */
	ScheduleResync(interval time.Duration) error
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
	AppInstallation AppInstallationContext
}
