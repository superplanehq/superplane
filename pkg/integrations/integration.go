package integrations

import (
	"context"
	"errors"
	"net/http"

	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/executors"
)

var ErrInvalidSignature = errors.New("invalid signature")

type AuthenticateFn func() (string, error)

type EventHandler interface {

	//
	// Convert the webhook data into a stateful resource.
	// Used by the pending events worker to update execution resources.
	// Used in conjunction with Status() to update the status of an execution resource.
	//
	Status(string, []byte) (StatefulResource, error)

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
	// Describe a resource by its type and name.
	// Used when creating event sources or stage executors,
	// to validate that the resource reference in the event source
	// or executor really exists.
	//
	Get(resourceType, id string) (Resource, error)

	//
	// Get the status of a resource created by the executor.
	// Used by the execution resource poller. Ideally, not needed at all, since the status
	// should be received in a webhook, through WebhookStatus().
	//
	Status(resourceType, id string, parentResource Resource) (StatefulResource, error)

	//
	// Configure the webhook for a integration resource.
	// This method might be called multiple times for the same parent resource,
	// so it should also update webhook-related resources, if needed.
	//
	SetupWebhook(options WebhookOptions) ([]Resource, error)
}

type Executor interface {

	//
	// Validates the executor spec.
	// Used during stage creation to validate that the executor spec is valid.
	//
	Validate(context.Context, []byte) error

	//
	// Triggers a new execution.
	//
	Execute([]byte, executors.ExecutionParameters) (StatefulResource, error)
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
}

// Similar to Resource, but with additional state information.
type StatefulResource interface {
	Id() string
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

type WebhookOptions struct {
	Parent     Resource
	Children   []Resource
	ID         string
	URL        string
	Key        []byte
	EventTypes []string
	Internal   bool
}
