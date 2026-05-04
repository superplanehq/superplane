import type { IntegrationSetupStepDefinition } from "@/api-client";
import { Button } from "@/components/ui/button";
import { Instructions } from "./Instructions";
import { Check, CheckCheck } from "lucide-react";

interface DoneStepProps {
  step: IntegrationSetupStepDefinition;
  onFinish: () => void;
  isSubmitting?: boolean;
}

export function DoneStep({ step, onFinish, isSubmitting }: DoneStepProps) {
  return (
    <div className="space-y-4">
      <Instructions description={step.instructions} />

      <div className="flex w-fit max-w-full items-center gap-4 pt-2">
        <Button
          type="button"
          onClick={onFinish}
          disabled={Boolean(isSubmitting)}
          className="group justify-center gap-2 text-sm !px-7 hover:!bg-primary"
        >
          {isSubmitting ? "Finishing..." : "Finish"}
          <span className="relative inline-flex size-4 shrink-0 items-center justify-center" aria-hidden>
            <Check className="absolute size-4 transition-opacity duration-200 ease-out group-hover:opacity-0 motion-reduce:transition-none motion-reduce:opacity-100 motion-reduce:group-hover:opacity-100" />
            <CheckCheck className="absolute size-4 opacity-0 transition-opacity duration-200 ease-out group-hover:opacity-100 motion-reduce:transition-none motion-reduce:opacity-0 motion-reduce:group-hover:opacity-0" />
          </span>
        </Button>
      </div>
    </div>
  );
}
