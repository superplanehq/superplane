import type { IntegrationSetupStepDefinition } from "@/api-client";
import { Button } from "@/components/ui/button";
import { ArrowLeft, MoveRight } from "lucide-react";
import { Instructions } from "./Instructions";

interface RedirectPromptStepProps {
  step: IntegrationSetupStepDefinition;
  onBack?: () => void;
  onOpenRedirect: () => void;
  isReverting?: boolean;
  isSubmitting?: boolean;
}

export function RedirectPromptStep({
  step,
  onBack,
  onOpenRedirect,
  isReverting,
  isSubmitting,
}: RedirectPromptStepProps) {
  const redirectPrompt = step.redirectPrompt;

  return (
    <div className="space-y-4">
      <Instructions description={step.instructions} />

      <div className="flex w-fit max-w-full items-center gap-4 pt-2">
        {onBack ? (
          <Button
            type="button"
            variant="link"
            onClick={onBack}
            disabled={Boolean(isSubmitting || isReverting)}
            className="group h-auto shrink-0 gap-1.5 px-0 py-1 font-normal hover:!no-underline"
          >
            <ArrowLeft
              aria-hidden
              className="size-4 shrink-0 transition-transform duration-200 ease-out group-hover:-translate-x-1 motion-reduce:transition-none motion-reduce:group-hover:translate-x-0"
            />
            {isReverting ? "Going back..." : "Previous"}
          </Button>
        ) : null}
        <Button
          type="button"
          onClick={onOpenRedirect}
          disabled={Boolean(!redirectPrompt?.url || isSubmitting || isReverting)}
          className="group justify-center gap-2 text-sm !px-7 hover:!bg-primary"
        >
          Continue
          <MoveRight
            aria-hidden
            className="size-4 shrink-0 transition-transform duration-200 ease-out group-hover:translate-x-1 motion-reduce:transition-none motion-reduce:group-hover:translate-x-0"
          />
        </Button>
      </div>
    </div>
  );
}
