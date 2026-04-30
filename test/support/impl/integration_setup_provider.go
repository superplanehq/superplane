package impl

import "github.com/superplanehq/superplane/pkg/core"

// StubIntegrationSetupProvider is a no-op IntegrationSetupProvider for tests that only need a
// registered provider instance (e.g. registry setup-flow coverage).
type StubIntegrationSetupProvider struct{}

func NewStubIntegrationSetupProvider() *StubIntegrationSetupProvider {
	return &StubIntegrationSetupProvider{}
}

func (StubIntegrationSetupProvider) CapabilityGroups() []core.CapabilityGroup {
	return nil
}

func (StubIntegrationSetupProvider) FirstStep(ctx core.SetupStepContext) core.SetupStep {
	_ = ctx
	return core.SetupStep{}
}

func (StubIntegrationSetupProvider) OnStepSubmit(ctx core.SetupStepContext) (*core.SetupStep, error) {
	_ = ctx
	return nil, nil
}

func (StubIntegrationSetupProvider) OnStepRevert(ctx core.SetupStepContext) error {
	_ = ctx
	return nil
}

func (StubIntegrationSetupProvider) OnPropertyUpdate(ctx core.PropertyUpdateContext) (*core.SetupStep, error) {
	_ = ctx
	return nil, nil
}

func (StubIntegrationSetupProvider) OnSecretUpdate(ctx core.SecretUpdateContext) (*core.SetupStep, error) {
	_ = ctx
	return nil, nil
}

func (StubIntegrationSetupProvider) OnCapabilityUpdate(ctx core.CapabilityUpdateContext) (*core.SetupStep, error) {
	_ = ctx
	return nil, nil
}
