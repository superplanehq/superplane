package impl

import "github.com/superplanehq/superplane/pkg/core"

// DummyIntegrationSetupProviderOptions configures a DummyIntegrationSetupProvider for tests.
// Nil function fields fall back to stub behavior (empty step, nil next step, nil errors).
type DummyIntegrationSetupProviderOptions struct {
	CapabilityGroups []core.CapabilityGroup

	FirstStep func(core.SetupStepContext) core.SetupStep

	OnStepSubmit func(core.SetupStepContext) (*core.SetupStep, error)

	OnStepRevert func(core.SetupStepContext) error

	OnPropertyUpdate func(core.PropertyUpdateContext) (*core.SetupStep, error)

	OnSecretUpdate func(core.SecretUpdateContext) (*core.SetupStep, error)

	OnCapabilityUpdate func(core.CapabilityUpdateContext) (*core.SetupStep, error)
}

// DummyIntegrationSetupProvider is a configurable IntegrationSetupProvider for tests.
type DummyIntegrationSetupProvider struct {
	opts DummyIntegrationSetupProviderOptions
}

func NewDummyIntegrationSetupProvider(opts DummyIntegrationSetupProviderOptions) *DummyIntegrationSetupProvider {
	return &DummyIntegrationSetupProvider{opts: opts}
}

// NewStubIntegrationSetupProvider returns an IntegrationSetupProvider with empty defaults
// (same behavior as the historical StubIntegrationSetupProvider).
func NewStubIntegrationSetupProvider() *DummyIntegrationSetupProvider {
	return NewDummyIntegrationSetupProvider(DummyIntegrationSetupProviderOptions{})
}

func (p *DummyIntegrationSetupProvider) CapabilityGroups() []core.CapabilityGroup {
	return p.opts.CapabilityGroups
}

func (p *DummyIntegrationSetupProvider) FirstStep(ctx core.SetupStepContext) core.SetupStep {
	if p.opts.FirstStep != nil {
		return p.opts.FirstStep(ctx)
	}
	return core.SetupStep{}
}

func (p *DummyIntegrationSetupProvider) OnStepSubmit(ctx core.SetupStepContext) (*core.SetupStep, error) {
	if p.opts.OnStepSubmit != nil {
		return p.opts.OnStepSubmit(ctx)
	}
	return nil, nil
}

func (p *DummyIntegrationSetupProvider) OnStepRevert(ctx core.SetupStepContext) error {
	if p.opts.OnStepRevert != nil {
		return p.opts.OnStepRevert(ctx)
	}
	return nil
}

func (p *DummyIntegrationSetupProvider) OnPropertyUpdate(ctx core.PropertyUpdateContext) (*core.SetupStep, error) {
	if p.opts.OnPropertyUpdate != nil {
		return p.opts.OnPropertyUpdate(ctx)
	}
	return nil, nil
}

func (p *DummyIntegrationSetupProvider) OnSecretUpdate(ctx core.SecretUpdateContext) (*core.SetupStep, error) {
	if p.opts.OnSecretUpdate != nil {
		return p.opts.OnSecretUpdate(ctx)
	}
	return nil, nil
}

func (p *DummyIntegrationSetupProvider) OnCapabilityUpdate(ctx core.CapabilityUpdateContext) (*core.SetupStep, error) {
	if p.opts.OnCapabilityUpdate != nil {
		return p.opts.OnCapabilityUpdate(ctx)
	}
	return nil, nil
}
