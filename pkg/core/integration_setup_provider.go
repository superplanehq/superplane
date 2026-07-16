package core

import (
	"errors"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/configuration"
)

//
// ErrSecretNotFound is returned by secret lookups when the requested secret
// does not exist. Callers can use errors.Is to distinguish a missing secret
// (e.g. an incomplete integration setup) from an unexpected store failure.
//
var ErrSecretNotFound = errors.New("not found")

/*
 * IntegrationSetupProvider is the contract for an integration to provide its setup flow.
 * Any changes to this interface should be documented in docs/design/integration-setup-flow.md.
 */
type IntegrationSetupProvider interface {

	//
	// All the capabilities supported by the integration.
	// The grouping is a presentation matter, and per-integration states
	// are still tracked per capability.
	//
	CapabilityGroups() []CapabilityGroup

	//
	// Generate the first step of the setup flow.
	//
	FirstStep(ctx SetupStepContext) SetupStep

	//
	// Called when the user submits the current pending step.
	//
	OnStepSubmit(ctx SetupStepContext) (*SetupStep, error)

	//
	// Called when the user reverts the last successfully submitted step.
	//
	OnStepRevert(ctx SetupStepContext) error

	//
	// Called when the user updates a property.
	// A property update might trigger a new setup flow.
	//
	OnPropertyUpdate(ctx PropertyUpdateContext) (*SetupStep, error)

	//
	// Called when the user updates a secret.
	// A secret update might trigger a new setup flow.
	//
	OnSecretUpdate(ctx SecretUpdateContext) (*SetupStep, error)

	//
	// Called when the user requests new capabilities
	// from an already setup integration.
	//
	OnCapabilityUpdate(ctx CapabilityUpdateContext) (*SetupStep, error)
}

type SetupStepType string

const (
	SetupStepTypeInputs              SetupStepType = "inputs"
	SetupStepTypeCapabilitySelection SetupStepType = "capabilitySelection"
	SetupStepTypeRedirectPrompt      SetupStepType = "redirectPrompt"
	SetupStepTypeDone                SetupStepType = "done"
)

type SetupStep struct {
	Type           SetupStepType
	Name           string
	Label          string
	Instructions   string
	Inputs         []configuration.Field
	Capabilities   []string
	RedirectPrompt *RedirectPrompt
}

type PropertyUpdateContext struct {
	PropertyName string
	Value        string
	Logger       *log.Entry
	HTTP         HTTPContext
	Secrets      IntegrationSecretStorage
	Properties   IntegrationPropertyStorage
	Capabilities CapabilityContext
}

type SecretUpdateContext struct {
	SecretName   string
	Value        string
	Logger       *log.Entry
	HTTP         HTTPContext
	Secrets      IntegrationSecretStorage
	Properties   IntegrationPropertyStorage
	Capabilities CapabilityContext
}

type CapabilityUpdateContext struct {
	Changes      map[IntegrationCapabilityState][]string
	Logger       *log.Entry
	HTTP         HTTPContext
	Secrets      IntegrationSecretStorage
	Properties   IntegrationPropertyStorage
	Capabilities CapabilityContext
}

type RedirectPrompt struct {
	URL      string
	Method   string
	FormData map[string]string
}

type SetupStepContext struct {
	Step            StepInfo
	IntegrationID   uuid.UUID
	OrganizationID  string
	BaseURL         string
	WebhooksBaseURL string
	Logger          *log.Entry
	HTTP            HTTPContext
	Secrets         IntegrationSecretStorage
	Properties      IntegrationPropertyStorage
	Capabilities    CapabilityContext
}

type StepInfo struct {
	Name         string
	Inputs       any
	Capabilities []string
}

/*
 * Properties is non-sensitive information exposed by the setup flow to the user.
 * They can be editable or not. If they are editable, OnPropertyUpdate() is called when the user updates it.
 * They are also typed, so the display layers (UI, CLI) can render them accordingly.
 */
type IntegrationPropertyType string

const (
	IntegrationPropertyTypeString IntegrationPropertyType = "string"
)

type IntegrationPropertyDefinition struct {
	Type        IntegrationPropertyType `json:"type"`
	Name        string                  `json:"name"`
	Label       string                  `json:"label"`
	Description string                  `json:"description"`
	Value       any                     `json:"value"`
	Editable    bool                    `json:"editable"`
}

type IntegrationPropertyStorageReader interface {
	Get(name string) (any, error)
	GetString(name string) (string, error)
}

type IntegrationPropertyStorage interface {
	IntegrationPropertyStorageReader

	Delete(names ...string) error
	Create(def IntegrationPropertyDefinition) error
	CreateMany(defs []IntegrationPropertyDefinition) error
}

type IntegrationSecretStorageReader interface {
	Get(name string) (string, error)
}

/*
 * Secrets are sensitive information managed by the integration.
 * In some cases, this comes from the user, as a step input.
 * In other cases, this comes from the setup flow itself.
 * Secrets can be editable or not. If they are editable, OnSecretUpdate() is called when the user updates it.
 *
 * This is context that the integration receives as part of the setup flow for managing them.
 */
type IntegrationSecretStorage interface {
	IntegrationSecretStorageReader

	Delete(name string) error
	Create(def IntegrationSecretDefinition) error
	CreateMany(defs []IntegrationSecretDefinition) error
	Update(name string, value string) error
}

type IntegrationSecretDefinition struct {
	Name        string
	Label       string
	Description string
	Value       string
	Editable    bool
}

/*
 * Capabilities are the "features" that the integration provides.
 * They are typed so the different parts of the system can take only the ones they need.
 *
 * They can be in 4 states:
 * - Unavailable: capability exists, but was not requested as part of the setup flow, and the user cannot request it anymore.
 * - Available: capability exists, setup did not expose it yet, but the user can still request it.
 * - Requested: capability was requested, but setup did not enable it yet.
 * - Enabled: capability is fully enabled and ready for use.
 * - Disabled: capability was enabled during the setup flow, but has been manually disabled by the user.
 */
type IntegrationCapabilityType string
type IntegrationCapabilityState string

const (
	IntegrationCapabilityTypeAction  IntegrationCapabilityType = "action"
	IntegrationCapabilityTypeTrigger IntegrationCapabilityType = "trigger"

	IntegrationCapabilityStateRequested   IntegrationCapabilityState = "requested"
	IntegrationCapabilityStateEnabled     IntegrationCapabilityState = "enabled"
	IntegrationCapabilityStateDisabled    IntegrationCapabilityState = "disabled"
	IntegrationCapabilityStateAvailable   IntegrationCapabilityState = "available"
	IntegrationCapabilityStateUnavailable IntegrationCapabilityState = "unavailable"
)

type CapabilityContext interface {
	Request(capabilities ...string)
	Available(capabilities ...string)
	Unavailable(capabilities ...string)
	Enable(capabilities ...string)
	Disable(capabilities ...string)
	Clear()
	IsRequested(capabilities ...string) bool
	Requested() []string
	Enabled() []string
}

type CapabilityGroup struct {
	Label        string
	Capabilities []Capability
}

type Capability struct {
	Type           IntegrationCapabilityType `json:"type"`
	Name           string                    `json:"name"`
	Label          string                    `json:"label"`
	Description    string                    `json:"description"`
	Configuration  []configuration.Field     `json:"configuration"`
	OutputChannels []OutputChannel           `json:"outputChannels"`
	ExampleOutput  map[string]any            `json:"exampleOutput,omitempty"`
	ExampleData    map[string]any            `json:"exampleData,omitempty"`
}

/*
 * IntegrationSetupContext allows integrations to manage setup state
 * from outside the setup flow (e.g. from an HTTP request handler).
 */
type IntegrationSetupContext interface {
	SetStep(step SetupStep) error
}
