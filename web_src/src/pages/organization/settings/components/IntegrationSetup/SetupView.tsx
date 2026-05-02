import type { useIntegrationSetupController } from "./useIntegrationSetupController";
import { SetupCard } from "./SetupCard";
import { SetupHeader } from "./SetupHeader";
import { SetupLoading } from "./SetupLoading";
import { StepHistory } from "./StepHistory";

type IntegrationSetupController = ReturnType<typeof useIntegrationSetupController>;

interface SetupViewProps {
  setup: IntegrationSetupController;
}

export function SetupView({ setup }: SetupViewProps) {
  const { actions, metadata, mutations, organizationId, progress, queries, route, state } = setup;
  const activeError = getActiveError(mutations);
  const isReverting = getIsReverting(mutations);
  const discardDisabled = getDiscardDisabled(mutations);
  const canDiscard = getCanDiscard(setup);

  if (queries.isAvailableIntegrationsLoading) {
    return <SetupLoading />;
  }

  const setupHeader = (
    <SetupHeader
      integrationsHref={route.integrationsHref}
      integrationName={route.integrationName}
      iconSlug={metadata.integrationDefinition?.icon}
      setupPageTitle={progress.setupPageTitle}
      hasCreatedIntegration={Boolean(state.createdIntegration)}
    />
  );
  const setupCard = (
    <SetupCard
      activeError={activeError}
      organizationId={organizationId}
      createdIntegration={state.createdIntegration}
      currentStep={progress.currentStep}
      stepInputs={state.stepInputs}
      showSetupStepBack={progress.showSetupStepBack}
      instanceName={state.instanceName}
      integrationName={route.integrationName}
      integrationCapabilities={metadata.integrationCapabilities}
      capabilitySections={metadata.capabilitySections}
      capabilityByName={metadata.capabilityByName}
      selectedCapabilities={state.selectedCapabilities}
      isCreatePending={mutations.createMutation.isPending}
      isSubmitting={mutations.submitStepMutation.isPending}
      isReverting={isReverting}
      onInstanceNameChange={state.setInstanceName}
      onToggleCapability={actions.toggleCapabilitySelection}
      onToggleCapabilityGroup={actions.toggleCapabilityGroup}
      onCreate={actions.handleCreateIntegration}
      onStepInputChange={actions.handleStepInputChange}
      onSubmitCurrentStep={actions.handleSubmitCurrentStep}
      onSetupStepBack={() => void actions.handleSetupStepBack()}
    />
  );

  return (
    <div className="pt-6">
      {state.createdIntegration ? (
        <div className="space-y-6 px-4 sm:px-6">
          {setupHeader}
          <div className="flex flex-col gap-6 lg:flex-row lg:items-start lg:gap-8">
            <div className="min-w-0 flex-1">{setupCard}</div>
            <StepHistory
              previousSteps={state.createdIntegration.status?.setupState?.previousSteps ?? []}
              currentStep={progress.currentStep}
              onDiscard={canDiscard ? () => void actions.handleDiscardIntegration() : undefined}
              discardDisabled={discardDisabled}
            />
          </div>
        </div>
      ) : (
        <>
          {setupHeader}
          {setupCard}
        </>
      )}
    </div>
  );
}

function getActiveError(mutations: IntegrationSetupController["mutations"]) {
  return mutations.createMutation.error || mutations.submitStepMutation.error || mutations.revertStepMutation.error;
}

function getIsReverting(mutations: IntegrationSetupController["mutations"]) {
  return mutations.revertStepMutation.isPending || mutations.deleteIntegrationMutation.isPending;
}

function getDiscardDisabled(mutations: IntegrationSetupController["mutations"]) {
  return (
    mutations.deleteIntegrationMutation.isPending ||
    mutations.submitStepMutation.isPending ||
    mutations.revertStepMutation.isPending ||
    mutations.createMutation.isPending
  );
}

function getCanDiscard(setup: IntegrationSetupController) {
  return Boolean(setup.state.createdIntegration?.metadata?.id && !setup.progress.integrationReady);
}
