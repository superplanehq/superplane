import { LoadingButton } from "@/components/ui/loading-button";

import { AgentStep, type OnboardingAgentModelPicker } from "../AgentStep";
import {
  agentFinishReady,
  isAgentProviderConnected,
  type OnboardingAgentCredentialChoice,
} from "../onboardingAgentReadiness";
import { WIZARD_STEPS, type IntegrationId } from "../onboardingFixtures";
import type { OnboardingSetupApi } from "../useOnboardingSetupState";
import { FirstRunModelSourceChoice } from "./FirstRunModelSourceChoice";
import { FirstRunHeading, FirstRunPanel, FirstRunShell } from "./FirstRunShell";
import type { FirstRunSphereProps } from "./FirstRunSpherePane";
import { FIRST_RUN_COPY } from "./firstRunCopy";
import type { FirstRunChrome } from "./firstRunTypes";

const AGENT_STEP: { id: "agent"; label: string; purpose: string } = WIZARD_STEPS[3];

type AgentModelSource = {
  offered: boolean;
  choice: OnboardingAgentCredentialChoice | null;
  onSelect: (choice: OnboardingAgentCredentialChoice) => void;
};

function agentScreenBody(modelSource: AgentModelSource, customProvider: boolean): string {
  if (modelSource.choice === "own-key") {
    return customProvider ? FIRST_RUN_COPY.agent.ownKeyBodyWithCustom : FIRST_RUN_COPY.agent.ownKeyBody;
  }
  if (modelSource.offered) return FIRST_RUN_COPY.agent.modelSourceBody;
  return AGENT_STEP.purpose;
}

export function FirstRunAgentScreen({
  organizationId,
  setup,
  chrome,
  sphere,
  saving,
  loading,
  hostedAgentReady,
  showCustomProvider,
  modelSource,
  onRequestConnect,
  modelPicker,
  onContinue,
}: {
  organizationId: string;
  setup: OnboardingSetupApi;
  chrome: FirstRunChrome;
  sphere?: FirstRunSphereProps;
  saving: boolean;
  loading: boolean;
  hostedAgentReady: boolean;
  showCustomProvider: boolean;
  modelSource: AgentModelSource;
  onRequestConnect: (id: IntegrationId) => void;
  modelPicker?: OnboardingAgentModelPicker;
  onContinue: () => void;
}) {
  const credentialChoice = modelSource.offered && !modelSource.choice ? "hosted" : modelSource.choice;
  const canFinish = agentFinishReady({
    modelSourceChoice: modelSource.offered,
    credentialChoice,
    providerConnected: isAgentProviderConnected(setup.connected),
    agentReady: setup.agentReady,
    hostedAgentReady,
  });
  const showProviders = !modelSource.offered || modelSource.choice === "own-key";
  return (
    <FirstRunShell
      testId="first-run-agent"
      chrome={chrome}
      busy={saving || loading}
      width="wide"
      sphere={sphere}
      visual="preview"
    >
      <FirstRunHeading headline={FIRST_RUN_COPY.agent.headline}>
        <p className="text-[13px] text-muted-foreground">{agentScreenBody(modelSource, showCustomProvider)}</p>
      </FirstRunHeading>

      <div className="mt-8 space-y-4">
        {loading ? (
          <p className="text-[13px] text-muted-foreground" role="status">
            {FIRST_RUN_COPY.agent.loading}
          </p>
        ) : null}
        {modelSource.offered ? (
          <FirstRunPanel>
            <FirstRunModelSourceChoice
              disabled={saving || loading}
              modelSource={modelSource.choice}
              onSelectModelSource={modelSource.onSelect}
            />
          </FirstRunPanel>
        ) : null}
        {showProviders ? (
          <fieldset disabled={saving || loading} className="mx-0 min-w-0 border-0 p-0">
            <FirstRunPanel>
              <AgentStep
                organizationId={organizationId}
                setup={setup}
                showHostedCredit={modelSource.choice !== "own-key"}
                showCustomProvider={showCustomProvider}
                onRequestConnect={onRequestConnect}
                modelPicker={modelPicker}
              />
            </FirstRunPanel>
          </fieldset>
        ) : null}
        <div className="space-y-3">
          <LoadingButton
            type="button"
            className="w-full"
            disabled={!canFinish || loading || Boolean(modelPicker && !modelPicker.value)}
            loading={saving}
            loadingText={FIRST_RUN_COPY.finish.saving}
            onClick={onContinue}
            data-testid="first-run-finish-setup"
          >
            {FIRST_RUN_COPY.finish.action}
          </LoadingButton>
        </div>
      </div>
    </FirstRunShell>
  );
}
