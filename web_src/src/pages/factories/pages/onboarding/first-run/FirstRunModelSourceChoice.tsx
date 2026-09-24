import SuperplaneLogo from "@/assets/superplane.svg";
import { KeyRound } from "lucide-react";

import type { OnboardingAgentCredentialChoice } from "../onboardingAgentReadiness";
import { ConnectOptionRow } from "../onboardingSteps";
import { FIRST_RUN_COPY } from "./firstRunCopy";

/** Organization BYOK: pick a provider key or SuperPlane-hosted models. */
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
  return (
    <fieldset disabled={disabled} className="m-0 min-w-0 border-0 p-0" data-testid="first-run-model-source">
      <legend className="mb-3 text-[13px] font-medium">{copy.modelSourceHeading}</legend>
      <div className="space-y-3">
        <ConnectOptionRow
          icon={<KeyRound className="size-5" aria-hidden />}
          title={copy.ownKey}
          detail={copy.ownKeyHelper}
          selected={modelSource === "own-key"}
          disabled={disabled}
          onSelect={() => onSelectModelSource("own-key")}
        />
        <ConnectOptionRow
          icon={<img src={SuperplaneLogo} alt="" className="size-5 dark:brightness-0 dark:invert" />}
          title={copy.hostedModels}
          detail={copy.hostedModelsHelper}
          selected={modelSource === "hosted"}
          disabled={disabled}
          onSelect={() => onSelectModelSource("hosted")}
        />
      </div>
    </fieldset>
  );
}
