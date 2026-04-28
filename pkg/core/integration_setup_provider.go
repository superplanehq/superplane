package core

import (
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/configuration"
)

type IntegrationSetupProvider interface {

	//
	// The list of available capabilities the integration provides.
	//
	Capabilities() []Capability

	//
	// First step of the setup flow.
	//
	FirstStep(ctx SetupStepContext) SetupStep

	//
	// Called when the user submits the current step.
	//
	OnStepSubmit(ctx SetupStepContext) (*SetupStep, error)

	//
	// Called when the user reverts the current step.
	// It should revert the changes made by the step.
	//
	OnStepRevert(ctx SetupStepContext) error

	//
	// Called when the user updates a parameter.
	// A parameter update might trigger a new setup flow.
	//
	OnParameterUpdate(ctx ParameterUpdateContext) (*SetupStep, error)

	//
	// Called when the user updates a secret.
	// A secret update might trigger a new setup flow.
	//
	OnSecretUpdate(ctx SecretUpdateContext) (*SetupStep, error)
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

type ParameterUpdateContext struct {
	ParameterName string
	Value         string
	Logger        *log.Entry
	HTTP          HTTPContext
	Secrets       IntegrationSecretStorage
	Parameters    IntegrationParameterStorage
	Capabilities  CapabilityContext
}

type SecretUpdateContext struct {
	SecretName   string
	Value        string
	Logger       *log.Entry
	HTTP         HTTPContext
	Secrets      IntegrationSecretStorage
	Parameters   IntegrationParameterStorage
	Capabilities CapabilityContext
}

type RedirectPrompt struct {
	URL      string
	Method   string
	FormData map[string]string
}

type SetupStepContext struct {
	Step           string
	Inputs         any
	IntegrationID  uuid.UUID
	OrganizationID string
	HTTP           HTTPContext
	Secrets        IntegrationSecretStorage
	Parameters     IntegrationParameterStorage
	Capabilities   CapabilityContext
}

type IntegrationSecretStorage interface {
	Get(name string) (string, error)
	Delete(name string) error
	Create(name string, def IntegrationSecretDefinition) error
	Update(name string, value string) error
}

type IntegrationSecretDefinition struct {
	Label       string
	Description string
	Value       []byte
	Editable    bool
}

type IntegrationParameterStorage interface {
	Get(name string) (any, error)
	Delete(name string) error
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
	IntegrationCapabilityTypeAction  IntegrationCapabilityType = "action"
	IntegrationCapabilityTypeTrigger IntegrationCapabilityType = "trigger"

	IntegrationCapabilityStateRequested = "requested"
	IntegrationCapabilityStateEnabled   = "enabled"
	IntegrationCapabilityStateDisabled  = "disabled"
)

type CapabilityContext interface {
	Enable(capabilities ...string) error
	Disable(capabilities ...string) error
	IsRequested(capabilities ...string) (bool, error)
}

type Capability struct {
	Type           IntegrationCapabilityType `json:"type"`
	Name           string                    `json:"name"`
	Label          string                    `json:"label"`
	Description    string                    `json:"description"`
	Configuration  []configuration.Field     `json:"configuration"`
	OutputChannels []OutputChannel           `json:"outputChannels"`
}
