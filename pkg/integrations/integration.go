package integrations

import (
	"context"

	"github.com/superplanehq/superplane/pkg/executors"
)

type AuthenticateFn func() (string, error)

type Integration interface {
	Get(resourceType, id string) (Resource, error)
	Check(resourceType, id string) (StatefulResource, error)
	SetupWebhook(options WebhookOptions) ([]Resource, error)
	HandleWebhook([]byte) (StatefulResource, error)
}

type Executor interface {
	Validate(context.Context, []byte) error
	Execute([]byte, executors.ExecutionParameters) (StatefulResource, error)
}

type Resource interface {
	Id() string
	Name() string
	Type() string
}

type StatefulResource interface {
	Finished() bool
	Successful() bool
	Id() string
	Type() string
}

type WebhookOptions struct {
	Resource Resource
	ID       string
	URL      string
	Key      []byte
}
