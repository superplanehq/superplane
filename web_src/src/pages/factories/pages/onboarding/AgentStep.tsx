import { useOrganizationWorkspaceUsage } from "@/hooks/useOrganizationWorkspaceUsage";
import { cn } from "@/lib/utils";
import { parseWorkOrderMetric } from "@/pages/factories/lib/workOrderUsage";
import { KeyRound } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

import { hostedCreditGrantCopy, shouldShowHostedCreditGrant } from "./onboardingAgentReadiness";
import { AGENT_OPTIONS, CUSTOM_PROVIDER_OPTION, type IntegrationId } from "./onboardingFixtures";
import { ConnectOptionRow, IntegrationChoiceIcon } from "./onboardingSteps";
import type { OnboardingSetupApi } from "./useOnboardingSetupState";

export type OnboardingAgentModelPicker = {
  models: string[];
  value: string;
  loading: boolean;
  error: boolean;
  retry: () => void;
  onChange: (model: string) => void;
};

export function AgentStep({
  organizationId,
  setup,
  showHostedCredit = true,
  showCustomProvider = false,
  onRequestConnect,
  modelPicker,
}: {
  organizationId: string;
  setup: OnboardingSetupApi;
  showHostedCredit?: boolean;
  showCustomProvider?: boolean;
  onRequestConnect: (id: IntegrationId) => void;
  modelPicker?: OnboardingAgentModelPicker;
}) {
  const spend = useOrganizationWorkspaceUsage(organizationId);
  const grantTotalCents = parseWorkOrderMetric(spend.data?.grantTotalCents);
  const remainingCreditCents = parseWorkOrderMetric(spend.data?.remainingCreditCents);
  const billingEnabled = spend.data?.billingEnabled === true;
  const showGrant = showHostedCredit && shouldShowHostedCreditGrant(grantTotalCents, billingEnabled);

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
      {modelPicker ? <AgentModelSelect picker={modelPicker} /> : null}
    </div>
  );
}

function AgentModelSelect({ picker }: { picker: OnboardingAgentModelPicker }) {
  return (
    <div className="grid gap-2">
      <Label htmlFor="onboarding-agent-model">Model</Label>
      <Select
        value={picker.value}
        onValueChange={picker.onChange}
        disabled={picker.loading || picker.error || picker.models.length === 0}
      >
        <SelectTrigger id="onboarding-agent-model" aria-describedby="onboarding-agent-model-help">
          <SelectValue placeholder={picker.loading ? "Loading models..." : "Select a model"} />
        </SelectTrigger>
        <SelectContent>
          {picker.models.map((model) => (
            <SelectItem key={model} value={model}>
              {model}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <p id="onboarding-agent-model-help" className="text-xs text-muted-foreground">
        {picker.error
          ? "Unable to load models. Try again."
          : !picker.loading && picker.models.length === 0
            ? "No models are available for this key."
            : "Use this model for Planning, Implementation, and other factory agents."}
      </p>
      {picker.error ? (
        <Button type="button" variant="outline" onClick={picker.retry}>
          Retry
        </Button>
      ) : null}
    </div>
  );
}
