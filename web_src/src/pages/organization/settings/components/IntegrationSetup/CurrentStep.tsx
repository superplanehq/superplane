import type {
  IntegrationSetupStepDefinition,
  IntegrationsCapabilityDefinition,
  IntegrationsIntegrationDefinition,
} from "@/api-client";
import { openRedirectPrompt } from "@/lib/integrations";
import { CapabilitySelectionStep } from "./CapabilitySelectionStep";
import { DoneStep } from "./DoneStep";
import { InputsStep } from "./InputsStep";
import { RedirectPromptStep } from "./RedirectPromptStep";

interface CurrentStepProps {
  organizationId: string;
  currentStep: IntegrationSetupStepDefinition | null;
  values: Record<string, unknown>;
  onChange: (fieldName: string, value: unknown) => void;
  onSubmit: () => void;
  onBack?: () => void;
  isSubmitting: boolean;
  isReverting: boolean;
  integrationDefinition: IntegrationsIntegrationDefinition | undefined;
  integrationCapabilities: IntegrationsCapabilityDefinition[];
  selectedCapabilities: ReadonlySet<string>;
  onToggleCapability: (capabilityName: string) => void;
  onToggleCapabilityGroup: (capabilityNames: string[]) => void;
}

export function CurrentStep({
  organizationId,
  currentStep,
  values,
  onChange,
  onSubmit,
  onBack,
  isSubmitting,
  isReverting,
  integrationDefinition,
  integrationCapabilities,
  selectedCapabilities,
  onToggleCapability,
  onToggleCapabilityGroup,
}: CurrentStepProps) {
  if (!currentStep) {
    return null;
  }

  if (currentStep.type === "INPUTS") {
    return (
      <InputsStep
        organizationId={organizationId}
        step={currentStep}
        values={values}
        onChange={onChange}
        onSubmit={onSubmit}
        onBack={onBack}
        isSubmitting={isSubmitting}
        isReverting={isReverting}
      />
    );
  }

  if (currentStep.type === "CAPABILITY_SELECTION") {
    return (
      <CapabilitySelectionStep
        step={currentStep}
        integrationDefinition={integrationDefinition}
        integrationCapabilities={integrationCapabilities}
        selectedCapabilities={selectedCapabilities}
        onToggleCapability={onToggleCapability}
        onToggleCapabilityGroup={onToggleCapabilityGroup}
        onSubmit={onSubmit}
        onBack={onBack}
        isSubmitting={isSubmitting}
        isReverting={isReverting}
      />
    );
  }

  if (currentStep.type === "REDIRECT_PROMPT") {
    return (
      <RedirectPromptStep
        step={currentStep}
        onBack={onBack}
        onOpenRedirect={() => openRedirectPrompt(currentStep)}
        isSubmitting={isSubmitting}
        isReverting={isReverting}
      />
    );
  }

  if (currentStep.type === "DONE") {
    return <DoneStep step={currentStep} onFinish={onSubmit} isSubmitting={isSubmitting} />;
  }

  return null;
}
