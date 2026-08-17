import type { ReactNode } from "react";

import { AGENT_OPTIONS, integrationLabel, type AgentHarnessId, type IntegrationId } from "./onboardingFixtures";
import { ConnectOptionRow, IntegrationChoiceIcon } from "./onboardingSteps";
import type { OnboardingSetupApi } from "./useOnboardingSetupState";

export function AgentStep({
  setup,
  onRequestConnect,
  integrationControls,
}: {
  setup: OnboardingSetupApi;
  onRequestConnect: (id: IntegrationId) => void;
  integrationControls?: ReactNode;
}) {
  if (integrationControls) return integrationControls;

  return (
    <div className="grid gap-2">
      {AGENT_OPTIONS.map((option) => (
        <ConnectOptionRow
          key={option.id}
          icon={<IntegrationChoiceIcon name={option.integrationId} />}
          title={option.label}
          detail={option.detail}
          selected={setup.agent === option.id}
          connectLabel={integrationLabel(option.integrationId)}
          connected={setup.connected.has(option.integrationId)}
          soon={option.soon}
          onSelect={() => setup.setAgent(option.id as AgentHarnessId)}
          onConnect={() => onRequestConnect(option.integrationId)}
        />
      ))}
    </div>
  );
}
