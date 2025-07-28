package integrations

import (
	"context"

	"github.com/superplanehq/superplane/pkg/executors"
)

type BuildFn func(ctx context.Context, URL string, authenticate AuthenticateFn) (Integration, error)

type AuthenticateFn func() (string, error)

type Integration interface {
	Get(resourceType, id string, parentIDs ...string) (Resource, error)
	Create(resourceType string, params any) (Resource, error)
	List(resourceType string, parentIDs ...string) ([]Resource, error)

	SetupWebhook(options WebhookOptions) ([]Resource, error)

	Check(resourceType, id string) (StatefulResource, error)
	HandleWebhook([]byte) (StatefulResource, error)

	Executor(resource Resource) (Executor, error)
}

type Executor interface {
	Validate(context.Context, []byte) error
	Execute([]byte, executors.ExecutionParameters) (Resource, error)
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
