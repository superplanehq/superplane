package integrations

import (
	"context"

	"github.com/superplanehq/superplane/pkg/models"
)

type BuildFn func(ctx context.Context, integration *models.Integration, authenticate AuthenticateFn) (Integration, error)
type AuthenticateFn func() (string, error)

type Integration interface {
	Get(resourceType, id string, parentIDs ...string) (Resource, error)
	Create(resourceType string, params any) (Resource, error)
	List(resourceType string, parentIDs ...string) ([]Resource, error)
	SetupWebhook(options WebhookOptions) ([]Resource, error)
}

type Resource interface {
	Id() string
	Name() string
	Type() string
}

type WebhookOptions struct {
	Resource Resource
	ID       string
	URL      string
	Key      []byte
}
