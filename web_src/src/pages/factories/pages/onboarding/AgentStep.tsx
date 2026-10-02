import { useOrganizationWorkspaceUsage } from "@/hooks/useOrganizationWorkspaceUsage";
import { cn } from "@/lib/utils";
import { parseWorkOrderMetric } from "@/pages/factories/lib/workOrderUsage";
import { KeyRound } from "lucide-react";

import { hostedCreditGrantCopy, shouldShowHostedCreditGrant } from "./onboardingAgentReadiness";
import { AGENT_OPTIONS, CUSTOM_PROVIDER_OPTION, type IntegrationId } from "./onboardingFixtures";
import { ConnectOptionRow, IntegrationChoiceIcon } from "./onboardingSteps";
import type { OnboardingSetupApi } from "./useOnboardingSetupState";

export function AgentStep({
  organizationId,
  setup,
  showHostedCredit = true,
  showCustomProvider = false,
  onRequestConnect,
}: {
  organizationId: string;
  setup: OnboardingSetupApi;
  showHostedCredit?: boolean;
  showCustomProvider?: boolean;
  onRequestConnect: (id: IntegrationId) => void;
}) {
  const spend = useOrganizationWorkspaceUsage(organizationId);
  const grantTotalCents = parseWorkOrderMetric(spend.data?.grantTotalCents);
  const remainingCreditCents = parseWorkOrderMetric(spend.data?.remainingCreditCents);
  const showGrant = showHostedCredit && shouldShowHostedCreditGrant(grantTotalCents);

  return (
    <div className="grid gap-3">
      {showGrant ? (
        <div className="rounded-lg border border-border bg-muted/30 px-4 py-3" data-testid="hosted-credit-grant">
          <p className="text-[13px] font-medium tracking-[-0.01em]">SuperPlane-hosted credit</p>
          <p
            className={cn(
              "mt-1 text-[12px]",
              remainingCreditCents > 0 ? "text-muted-foreground" : "text-amber-700 dark:text-amber-400",
            )}
          >
            {hostedCreditGrantCopy(remainingCreditCents)}
          </p>
        </div>
      ) : null}
      <div className="grid gap-2">
        {(showCustomProvider ? [...AGENT_OPTIONS, CUSTOM_PROVIDER_OPTION] : AGENT_OPTIONS).map((option) => (
          <ConnectOptionRow
            key={option.id}
            icon={
              option.id === "customLlm" ? (
                <KeyRound className="size-5" aria-hidden />
              ) : (
                <IntegrationChoiceIcon name={option.id} />
              )
            }
            title={option.label}
            detail={option.detail}
            connectLabel={option.label}
            connected={setup.connected.has(option.id)}
            soon={option.soon}
            onSelect={() => undefined}
            onConnect={() => onRequestConnect(option.id)}
          />
        ))}
      </div>
    </div>
  );
}
