import SuperplaneLogo from "@/assets/superplane.svg";

import { cn } from "@/lib/utils";

import type { OnboardingAgentCredentialChoice } from "../onboardingAgentReadiness";
import { ConnectOptionRow } from "../onboardingSteps";
import { FIRST_RUN_COPY } from "./firstRunCopy";

/** Organization BYOK: hosted models stay the choice. An own key is a text link. */
export function FirstRunModelSourceChoice({
  disabled,
  modelSource,
  onSelectModelSource,
}: {
  disabled: boolean;
  modelSource: OnboardingAgentCredentialChoice | null;
  onSelectModelSource: (source: OnboardingAgentCredentialChoice) => void;
}) {
  const copy = FIRST_RUN_COPY.agent;
  const ownKeySelected = modelSource === "own-key";
  return (
    <fieldset disabled={disabled} className="m-0 min-w-0 border-0 p-0" data-testid="first-run-model-source">
      <legend className="mb-3 text-[13px] font-medium">{copy.modelSourceHeading}</legend>
      <div className="space-y-3">
        <ConnectOptionRow
          icon={<img src={SuperplaneLogo} alt="" className="size-5 dark:brightness-0 dark:invert" />}
          title={copy.hostedModels}
          detail={copy.hostedModelsHelper}
          selected={modelSource !== "own-key"}
          disabled={disabled}
          onSelect={() => onSelectModelSource("hosted")}
        />
        <button
          type="button"
          className={cn(
            "text-left text-[13px] underline-offset-4 hover:underline disabled:pointer-events-none disabled:opacity-50",
            ownKeySelected ? "font-medium text-foreground underline" : "text-muted-foreground",
          )}
          aria-pressed={ownKeySelected}
          onClick={() => onSelectModelSource("own-key")}
        >
          {copy.ownKey}
        </button>
      </div>
    </fieldset>
  );
}
