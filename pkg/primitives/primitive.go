package primitives

const DefaultBranchName = "default"

type ConfigurationField struct {
	Name        string
	Type        string
	Description string
	Required    bool
}

type Primitive interface {
	Name() string
	Description() string
	Outputs(configuration any) []string
	Configuration() []ConfigurationField
	Execute(ctx ExecutionContext) (*Result, error)
}

type ExecutionContext struct {
	Data          any
	Configuration any
}

type Result struct {
	Branches map[string][]any
}
