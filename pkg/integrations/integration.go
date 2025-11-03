package integrations

import (
	"context"
	"errors"

	"github.com/superplanehq/superplane/pkg/crypto"
)

var ErrInvalidSignature = errors.New("invalid signature")

type AuthenticateFn func() (string, error)

type ResourceManager interface {
	//
	// Describe and list resources by their type and name.
	// Used when creating nodes for integration components.
	//
	Get(resourceType, id string) (Resource, error)
	List(resourceType string) ([]Resource, error)

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

// A generic interface for representing integration resources.
type Resource interface {
	Id() string
	Name() string
	Type() string
	URL() string
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
