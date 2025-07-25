package integrations

type AuthenticateFn func() (string, error)

type Integration interface {
	Get(resourceType, id string, parentIDs ...string) (Resource, error)
	Create(resourceType string, params any) (Resource, error)
	List(resourceType string, parentIDs ...string) ([]Resource, error)
	SetupEventSource(options EventSourceOptions) ([]Resource, error)
}

type Resource interface {
	Id() string
	Name() string
	Type() string
}

type EventSourceOptions struct {
	Resource Resource
	BaseURL  string
	ID       string
	Name     string
	Key      []byte
}
