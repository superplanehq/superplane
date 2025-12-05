package integrations

import (
	"context"
	"errors"
	"net/http"

	"github.com/superplanehq/superplane/pkg/crypto"
)

var ErrInvalidSignature = errors.New("invalid signature")

type AuthenticateFn func() (string, error)

type EventHandler interface {

	//
	// Convert the webhook data into an event.
	// Used by the HTTP server when receiving events
	// for a resource from the integration.
	//
	Handle(data []byte, header http.Header) (Event, error)

	//
	// List of event types supported by the integration.
	//
	EventTypes() []string
}

type ResourceManager interface {
	//
	// Describe and list resources by their type and name.
	// Used when creating event sources or stage executors,
	// to validate that the resource reference in the event source
	// or executor really exists.
	//
	Get(resourceType, id string) (Resource, error)
	List(resourceType string) ([]Resource, error)

	//
	// Cancel a resource.
	// Used by the execution poller to cancel execution resources,
	// when an execution is cancelled.
	//
	Cancel(resourceType, id string, parentResource Resource) error

	//
	// Set up webhooks in the integration.
	// `any` type is used because the configuration and the metadata
	// are integration-specific.
	//
	SetupWebhook(options WebhookOptions) (any, error)

	//
	// Clean up webhooks in the integration.
	// `any` type is used because the configuration and the metadata
	// are integration-specific.
	//
	CleanupWebhook(options WebhookOptions) error
}

type WebhookOptions struct {
	ID            string
	Resource      Resource
	URL           string
	Secret        []byte
	Configuration any
	Metadata      any
}

type OIDCVerifier interface {
	Verify(ctx context.Context, verifier *crypto.OIDCVerifier, token string, options VerifyTokenOptions) error
}

type VerifyTokenOptions struct {
	IntegrationURL string
	ParentResource string
	ChildResource  string
}

// A generic interface for representing integration resources.
type Resource interface {
	Id() string
	Name() string
	Type() string
	URL() string
}

// Similar to Resource, but with additional state information.
type StatefulResource interface {
	Id() string
	URL() string
	Type() string
	Finished() bool
	Successful() bool
}

// Used to represent events received from the integration.
// Returned by HandleWebhook().
type Event interface {
	Type() string

	//
	// The signature for the event payload.
	// The value returned here is used by the HTTP server
	// to verify the integrity of the event payload.
	//
	// Usually, this value is what is present in the
	// X-*-Signature-256 HTTP header on the webhook request.
	//
	Signature() string
}
