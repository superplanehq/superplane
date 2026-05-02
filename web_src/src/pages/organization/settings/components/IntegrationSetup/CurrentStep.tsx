import type { IntegrationSetupStepDefinition } from "@/api-client";
import { openRedirectPrompt } from "@/lib/integrations";
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

  if (currentStep.type === "REDIRECT_PROMPT") {
    return (
      <RedirectPromptStep
        step={currentStep}
        onBack={onBack}
        onOpenRedirect={() => openRedirectPrompt(currentStep)}
        onSubmit={onSubmit}
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
