import type {
  IntegrationSetupStepDefinition,
  IntegrationsCapabilityDefinition,
  OrganizationsIntegration,
} from "@/api-client";
import type { CapabilityGroupSection } from "@/lib/capabilities";
import { Alert, AlertDescription, AlertTitle } from "@/ui/alert";
import { getApiErrorMessage } from "@/lib/errors";
import { CurrentStep } from "./CurrentStep";
import { PreCreateIntegrationSetup } from "./PreCreateIntegrationSetup";

interface SetupCardProps {
  activeError: unknown;
  organizationId: string;
  createdIntegration: OrganizationsIntegration | null;
  currentStep: IntegrationSetupStepDefinition | null;
  stepInputs: Record<string, unknown>;
  showSetupStepBack: boolean;
  instanceName: string;
  integrationName: string;
  integrationCapabilities: IntegrationsCapabilityDefinition[];
  capabilitySections: CapabilityGroupSection[];
  capabilityByName: Map<string, IntegrationsCapabilityDefinition>;
  selectedCapabilities: ReadonlySet<string>;
  isCreatePending: boolean;
  isSubmitting: boolean;
  isReverting: boolean;
  onInstanceNameChange: (value: string) => void;
  onToggleCapability: (capabilityName: string) => void;
  onToggleCapabilityGroup: (capabilityNames: string[]) => void;
  onCreate: () => void;
  onStepInputChange: (fieldName: string, value: unknown) => void;
  onSubmitCurrentStep: () => void;
  onSetupStepBack: () => void;
}

export function SetupCard({
  activeError,
  organizationId,
  createdIntegration,
  currentStep,
  stepInputs,
  showSetupStepBack,
  instanceName,
  integrationName,
  integrationCapabilities,
  capabilitySections,
  capabilityByName,
  selectedCapabilities,
  isCreatePending,
  isSubmitting,
  isReverting,
  onInstanceNameChange,
  onToggleCapability,
  onToggleCapabilityGroup,
  onCreate,
  onStepInputChange,
  onSubmitCurrentStep,
  onSetupStepBack,
}: SetupCardProps) {
  return (
    <div className="bg-white dark:bg-gray-900 rounded-lg border border-gray-300 dark:border-gray-800 p-6 space-y-6">
      {activeError ? (
        <Alert variant="destructive">
          <AlertTitle>Setup failed</AlertTitle>
          <AlertDescription>{getApiErrorMessage(activeError)}</AlertDescription>
        </Alert>
      ) : null}

      {!createdIntegration ? (
        <PreCreateIntegrationSetup
          instanceName={instanceName}
          onInstanceNameChange={onInstanceNameChange}
          integrationName={integrationName}
          integrationCapabilities={integrationCapabilities}
          capabilitySections={capabilitySections}
          capabilityByName={capabilityByName}
          selectedCapabilities={selectedCapabilities}
          onToggleCapability={onToggleCapability}
          onToggleCapabilityGroup={onToggleCapabilityGroup}
          isCreatePending={isCreatePending}
          onCreate={onCreate}
        />
      ) : (
        <CurrentStep
          organizationId={organizationId}
          currentStep={currentStep}
          values={stepInputs}
          onChange={onStepInputChange}
          onSubmit={onSubmitCurrentStep}
          onBack={showSetupStepBack ? onSetupStepBack : undefined}
          isSubmitting={isSubmitting}
          isReverting={isReverting}
        />
      )}
    </div>
  );
}
