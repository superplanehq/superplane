import type { IntegrationSetupStepDefinition } from "@/api-client";
import { Button } from "@/components/ui/button";
import { IntegrationInstructionsV2 } from "@/ui/IntegrationInstructionsV2";

interface IntegrationSetupRedirectPromptStepProps {
  step: IntegrationSetupStepDefinition;
  onBack?: () => void;
  onOpenRedirect: () => void;
  onSubmit: () => void;
  isReverting?: boolean;
  isSubmitting?: boolean;
}

export function IntegrationSetupRedirectPromptStep({
  step,
  onBack,
  onOpenRedirect,
  onSubmit,
  isReverting,
  isSubmitting,
}: IntegrationSetupRedirectPromptStepProps) {
  const redirectPrompt = step.redirectPrompt;

  return (
    <div className="space-y-4">
      <div>
        <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">Step: {step.label}</h2>
        <IntegrationInstructionsV2 description={step.instructions} className="mt-3" />
      </div>

      <div className="rounded-md border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-900 p-4">
        <p className="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">Redirect URL</p>
        <p className="mt-1 text-sm break-all text-gray-800 dark:text-gray-200">{redirectPrompt?.url || "-"}</p>
      </div>

      <div className="flex items-center gap-3 pt-2">
        <Button
          type="button"
          variant="outline"
          onClick={onBack}
          disabled={Boolean(isSubmitting || isReverting || !onBack)}
        >
          {isReverting ? "Going back..." : "Previous"}
        </Button>
        <Button type="button" variant="outline" onClick={onOpenRedirect} disabled={!redirectPrompt?.url}>
          Open Redirect
        </Button>
        <Button type="button" onClick={onSubmit} disabled={Boolean(isSubmitting || isReverting)}>
          {isSubmitting ? "Saving..." : "Next"}
        </Button>
      </div>
    </div>
  );
}
