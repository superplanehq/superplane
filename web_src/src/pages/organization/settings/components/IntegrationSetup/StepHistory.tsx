import type { IntegrationSetupStepDefinition } from "@/api-client";
import { Button } from "@/components/ui/button";
import { CheckCheck, CircleDashed, Trash2 } from "lucide-react";

interface StepHistoryProps {
  previousSteps: Array<IntegrationSetupStepDefinition>;
  currentStep: IntegrationSetupStepDefinition | null;
  onDiscard?: () => void;
  discardDisabled?: boolean;
}

export function StepHistory({ previousSteps, currentStep, onDiscard, discardDisabled = false }: StepHistoryProps) {
  const hasTrail = previousSteps.length > 0 || Boolean(currentStep);

  return (
    <nav className="flex w-full shrink-0 flex-col lg:flex-none lg:w-44 lg:pt-5" aria-label="Setup steps">
      {hasTrail ? (
        <ol className="space-y-2">
          {previousSteps.map((step, index) => (
            <li
              key={step.name ? `${step.name}-${index}` : `completed-${index}`}
              className="grid grid-cols-[auto_minmax(0,1fr)] items-start gap-x-2 text-[11px] leading-snug"
            >
              <CheckCheck
                className="col-start-1 row-start-1 size-3.5 shrink-0 translate-y-px text-green-600 dark:text-green-400"
                strokeWidth={2}
                aria-hidden
              />
              <span className="col-start-2 row-start-1 min-w-0 text-gray-700 dark:text-gray-300">
                {step.label?.trim() || step.name || "Step"}
              </span>
            </li>
          ))}
          {currentStep ? (
            <li className="grid grid-cols-[auto_minmax(0,1fr)] items-start gap-x-2 text-[11px] leading-snug">
              {currentStep.type === "DONE" ? (
                <CheckCheck
                  className="col-start-1 row-start-1 size-3.5 shrink-0 translate-y-px text-green-600 dark:text-green-400"
                  strokeWidth={2}
                  aria-hidden
                />
              ) : (
                <CircleDashed
                  className="col-start-1 row-start-1 size-3.5 shrink-0 translate-y-px text-primary"
                  strokeWidth={2}
                  aria-hidden
                />
              )}
              <span
                className={
                  currentStep.type === "DONE"
                    ? "col-start-2 row-start-1 min-w-0 text-gray-700 dark:text-gray-300"
                    : "col-start-2 row-start-1 min-w-0 font-medium text-gray-900 dark:text-gray-100"
                }
              >
                {currentStep.label?.trim() || currentStep.name || "Current step"}
              </span>
            </li>
          ) : null}
        </ol>
      ) : (
        <p className="text-[10px] leading-snug text-gray-500 dark:text-gray-400">
          Complete a step to build your trail.
        </p>
      )}

      {onDiscard ? (
        <div className="mt-5 border-t border-gray-200 pt-4 dark:border-gray-700">
          <div className="flex justify-center">
            <Button
              type="button"
              variant="link"
              onClick={() => onDiscard()}
              disabled={discardDisabled}
              className="h-auto gap-1.5 px-0 py-0 text-xs font-medium leading-none text-gray-600 hover:text-red-600 dark:text-gray-300 dark:hover:text-red-400 has-[>svg]:px-0 hover:!no-underline"
            >
              <Trash2 aria-hidden className="size-[1em] shrink-0 opacity-80" />
              Discard
            </Button>
          </div>
        </div>
      ) : null}
    </nav>
  );
}
