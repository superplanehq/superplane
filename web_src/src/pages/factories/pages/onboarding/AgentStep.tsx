import superplaneLogo from "@/assets/superplane.svg";
import { useOrganizationLLMSpend } from "@/hooks/useOrganizationLLMSpend";
import { cn } from "@/lib/utils";
import { parseWorkOrderMetric } from "@/pages/factories/lib/workOrderUsage";
import { useState } from "react";

import { AGENT_PROVIDER_IDS, hostedCreditGrantCopy, type AgentProviderId } from "./onboardingAgentReadiness";
import { AGENT_OPTIONS, type IntegrationId } from "./onboardingFixtures";
import { ConnectOptionRow, IntegrationChoiceIcon } from "./onboardingSteps";
import type { OnboardingSetupApi } from "./useOnboardingSetupState";

const BYOK_OPTIONS = AGENT_OPTIONS.filter(
  (option): option is (typeof AGENT_OPTIONS)[number] & { id: AgentProviderId } =>
    AGENT_PROVIDER_IDS.includes(option.id as AgentProviderId),
);

export function AgentStep({
  organizationId,
  setup,
  onRequestConnect,
}: {
  organizationId: string;
  setup: OnboardingSetupApi;
  onRequestConnect: (id: IntegrationId) => void;
}) {
  const spend = useOrganizationLLMSpend(organizationId);
  const remainingCreditCents = parseWorkOrderMetric(spend.data?.remainingCreditCents);
  const [ownKeyOpen, setOwnKeyOpen] = useState(() => setup.keyProvider !== null);
  const superPlaneSelected = setup.keyProvider === null;

  return (
    <div className="grid gap-3">
      <ConnectOptionRow
        icon={<img src={superplaneLogo} alt="" className="size-5 dark:brightness-0 dark:invert" />}
        title="SuperPlane agent"
        detail="SuperPlane will run the agent on this workspace. Work starts only after you approve a ticket."
        selected={superPlaneSelected}
        onSelect={() => setup.setKeyProvider(null)}
      />
      <button
        type="button"
        className="justify-self-start text-[12px] text-muted-foreground underline-offset-4 hover:underline"
        onClick={() => setOwnKeyOpen((open) => !open)}
      >
        Use your own key
      </button>
      {ownKeyOpen ? (
        <div className="grid gap-2">
          {remainingCreditCents <= 0 ? (
            <p className={cn("text-[12px] text-amber-700 dark:text-amber-400")}>{hostedCreditGrantCopy(0)}</p>
          ) : null}
          {BYOK_OPTIONS.map((option) => (
            <ConnectOptionRow
              key={option.id}
              icon={<IntegrationChoiceIcon name={option.id} />}
              title={option.label}
              detail={option.detail}
              connectLabel={option.label}
              connected={setup.connected.has(option.id)}
              selected={setup.keyProvider === option.id}
              onSelect={() => setup.setKeyProvider(option.id)}
              onConnect={() => onRequestConnect(option.id)}
            />
          ))}
        </div>
      ) : null}
    </div>
  );
}
