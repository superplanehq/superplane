package core

import (
	"github.com/superplanehq/superplane/pkg/configuration"
)

type IntegrationV2 interface {
	Name() string
	Label() string
	Description() string
	FirstStep() SetupStep
	OnStepSubmit(stepName string, inputs any, ctx SetupStepContext) (*SetupStep, error)
}

type SetupStepType string

const (
	SetupStepTypeInputs         SetupStepType = "inputs"
	SetupStepTypeRedirectPrompt SetupStepType = "redirectPrompt"
)

type SetupStep struct {
	Type           SetupStepType
	Name           string
	Instructions   string
	Inputs         []configuration.Field
	RedirectPrompt *RedirectPrompt
}

type RedirectPrompt struct {
	URL      string
	Method   string
	FormData map[string]string
}

type SetupStepContext struct {
	HTTP         HTTPContext
	Secrets      IntegrationSecretStorage
	Parameters   IntegrationParameeterStorage
	Capabilities CapabilityRegistry
}

type IntegrationSecretStorage interface {
	Get(name string) (string, error)
	Create(name string, def IntegrationSecretDefinition) error
}

type IntegrationSecretDefinition struct {
	Value    []byte
	Editable bool
}

type IntegrationParameeterStorage interface {
	Get(name string) (any, error)
	Create(name string, def IntegrationParameterDefinition) error
}

type IntegrationParameterDefinition struct {
	Type     string
	Value    any
	Editable bool
}

type CapabilityRegistry interface {
	RegisterComponents(components []Component) error
	RegisterTriggers(triggers []Trigger) error
}
