import SuperplaneLogo from "@/assets/superplane.svg";
import { Button } from "@/components/ui/button";

import { cn } from "@/lib/utils";
import { DARK_SURFACE_MARK_INVERT_CLASS } from "@/ui/componentSidebar/integrationIcons";

import type { OnboardingAgentCredentialChoice } from "../onboardingAgentReadiness";
import { ConnectOptionRow } from "../onboardingSteps";
import { FIRST_RUN_COPY } from "./firstRunCopy";

export function FirstRunModelSourceChoice({
  disabled,
  modelSource,
  subscriptionOffered = false,
  providerKeysOffered = true,
  onSelectModelSource,
}: {
  disabled: boolean;
  modelSource: OnboardingAgentCredentialChoice | null;
  subscriptionOffered?: boolean;
  providerKeysOffered?: boolean;
  onSelectModelSource: (source: OnboardingAgentCredentialChoice) => void;
}) {
  const copy = FIRST_RUN_COPY.agent;
  const alternativeSelected = modelSource === "own-key" || modelSource === "subscription";
  return (
    <fieldset disabled={disabled} className="m-0 min-w-0 border-0 p-0" data-testid="first-run-model-source">
      <legend className="mb-3 text-[13px] font-medium">{copy.modelSourceHeading}</legend>
      <div className="space-y-3">
        <ConnectOptionRow
          icon={<img src={SuperplaneLogo} alt="" className={cn("size-5", DARK_SURFACE_MARK_INVERT_CLASS)} />}
          title={copy.hostedModels}
          detail={copy.hostedModelsHelper}
          selected={!modelSource || modelSource === "hosted"}
          disabled={disabled}
          onSelect={() => onSelectModelSource("hosted")}
        />
        <Button
          type="button"
          variant="link"
          className={cn(
            "h-auto justify-start rounded-none p-0 text-left text-[13px] font-normal underline-offset-4 hover:underline disabled:pointer-events-none disabled:opacity-50",
            alternativeSelected ? "font-medium text-foreground underline" : "text-muted-foreground",
          )}
          aria-pressed={alternativeSelected}
          aria-expanded={alternativeSelected}
          onClick={() => onSelectModelSource(providerKeysOffered ? "own-key" : "subscription")}
        >
          {providerKeysOffered ? (subscriptionOffered ? copy.ownKeyOrSubscription : copy.ownKey) : copy.subscription}
        </Button>
      </div>
    </fieldset>
  );
}
