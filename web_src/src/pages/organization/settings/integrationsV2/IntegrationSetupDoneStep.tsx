import type { IntegrationSetupStepDefinition } from "@/api-client";
import { Button } from "@/components/ui/button";
import { IntegrationInstructionsV2 } from "@/ui/IntegrationInstructionsV2";

interface IntegrationSetupDoneStepProps {
  step: IntegrationSetupStepDefinition;
  onBack?: () => void;
  onFinish: () => void;
  isReverting?: boolean;
  isSubmitting?: boolean;
}

export function IntegrationSetupDoneStep({
  step,
  onBack,
  onFinish,
  isReverting,
  isSubmitting,
}: IntegrationSetupDoneStepProps) {
  return (
    <div className="space-y-4">
      <div>
        <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
          {step.label?.trim() || "Setup complete"}
        </h2>
        <IntegrationInstructionsV2 description={step.instructions} className="mt-3" />
      </div>

      <div className="flex flex-wrap items-center gap-3 pt-2">
        <Button
          type="button"
          variant="outline"
          onClick={onBack}
          disabled={Boolean(isSubmitting || isReverting || !onBack)}
        >
          {isReverting ? "Going back..." : "Previous"}
        </Button>
        <Button type="button" onClick={onFinish} disabled={Boolean(isSubmitting || isReverting)}>
          {isSubmitting ? "Finishing..." : "Finish"}
        </Button>
      </div>
    </div>
  );
}
