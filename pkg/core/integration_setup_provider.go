package core

import (
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
)

type IntegrationSetupProvider interface {
	FirstStep(ctx SetupStepContext) SetupStep
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
	Label          string
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
	IntegrationID  uuid.UUID
	OrganizationID string
	HTTP           HTTPContext
	Secrets        IntegrationSecretStorage
	Parameters     IntegrationParameterStorage
	Capabilities   CapabilityRegistry
}

type IntegrationSecretStorage interface {
	Get(name string) (string, error)
	Create(name string, def IntegrationSecretDefinition) error
}

type IntegrationSecretDefinition struct {
	Value    []byte
	Editable bool
}

type IntegrationParameterStorage interface {
	Get(name string) (any, error)
	Create(def IntegrationParameterDefinition) error
}

type IntegrationParameterDefinition struct {
	Name        string
	Label       string
	Description string
	Type        string
	Value       any
	Editable    bool
}

type IntegrationCapabilityType string

const (
	IntegrationCapabilityTypeComponent IntegrationCapabilityType = "component"
	IntegrationCapabilityTypeTrigger   IntegrationCapabilityType = "trigger"
)

type CapabilityRegistry interface {
	RegisterComponents(components []Component) error
	RegisterTriggers(triggers []Trigger) error
}

type CapabilityDefinition struct {
	Type      IntegrationCapabilityType
	Component *ComponentDefinition
	Trigger   *TriggerDefinition
}

type ComponentDefinition struct {
	Name        string
	Label       string
	Description string
}

type TriggerDefinition struct {
	Name        string
	Label       string
	Description string
}
