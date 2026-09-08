import { LoadingButton } from "@/components/ui/loading-button";

import { AgentStep } from "../AgentStep";
import { WIZARD_STEPS, type IntegrationId } from "../onboardingFixtures";
import type { OnboardingSetupApi } from "../useOnboardingSetupState";
import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunHeading, FirstRunPanel, FirstRunShell } from "./FirstRunShell";
import type { FirstRunChrome } from "./firstRunTypes";

/**
 * Hosted credentials leave the agent screen with no question to ask, so the
 * ticket screen becomes the last screen and provisions the workspace.
 */
const AGENT_STEP: { id: "agent"; label: string; purpose: string } = WIZARD_STEPS[3];

export function FirstRunAgentScreen({
  organizationId,
  setup,
  chrome,
  saving,
  onRequestConnect,
  onContinue,
}: {
  organizationId: string;
  setup: OnboardingSetupApi;
  chrome: FirstRunChrome;
  saving: boolean;
  onRequestConnect: (id: IntegrationId) => void;
  onContinue: () => void;
}) {
  return (
    <FirstRunShell testId="first-run-agent" chrome={chrome} width="wide">
      <FirstRunHeading headline={FIRST_RUN_COPY.agent.headline}>
        <p className="text-[13px] text-muted-foreground">{AGENT_STEP.purpose}</p>
      </FirstRunHeading>

      <div className="mt-8 space-y-4">
        <FirstRunPanel>
          <AgentStep organizationId={organizationId} setup={setup} onRequestConnect={onRequestConnect} />
        </FirstRunPanel>
        <LoadingButton
          type="button"
          className="w-full"
          disabled={!setup.agentReady}
          loading={saving}
          loadingText={FIRST_RUN_COPY.finish.saving}
          onClick={onContinue}
          data-testid="first-run-finish-setup"
        >
          {FIRST_RUN_COPY.finish.action}
        </LoadingButton>
      </div>
    </FirstRunShell>
  );
}
