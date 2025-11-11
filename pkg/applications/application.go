package applications

import (
	"net/http"

	"github.com/superplanehq/superplane/pkg/components"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/triggers"
)

type Application interface {
	/*
	 * The name of the application.
	 */
	Name() string

	/*
	 * The configuration fields of the application.
	 */
	Configuration() []configuration.Field

	/*
	 * The list of components exposed by the application.
	 */
	Components() []components.Component

	/*
	 * The list of triggers exposed by the application.
	 */
	Triggers() []triggers.Trigger

	/*
	 * Called when configuration changes.
	 */
	Sync(ctx SyncContext) SyncResponse

	/*
	 * HTTP request handler
	 */
	HandleRequest(ctx HttpRequestContext)
}

type SyncContext struct {
	Configuration  any
	BaseURL        string
	OrganizationID string
	InstallationID string
	AppContext     AppContext
}

type AppContext interface {
	GetMetadata() any
	SetMetadata(any)
	GetState() string
	SetState(string)
	NewBrowserAction(action BrowserAction)
	RemoveBrowserAction()
}

type BrowserAction struct {
	URL        string
	Method     string
	FormFields map[string]string
}

type SyncResponse struct {
	NewState         string
	StateDescription string
	Signals          []Signal
}

type Signal struct {
	Name      string
	Data      any
	Sensitive bool
}

type HttpRequestContext struct {
	Request        *http.Request
	Response       *http.ResponseWriter
	OrganizationID string
	InstallationID string
	BaseURL        string
	AppContext     AppContext
}
