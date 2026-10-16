import { LoadingButton } from "@/components/ui/loading-button";

import { AgentStep, type OnboardingAgentModelPicker } from "../AgentStep";
import { WIZARD_STEPS } from "../onboardingFixtures";
import {
  agentFinishReady,
  isAgentProviderConnected,
  type OnboardingAgentCredentialChoice,
} from "../onboardingAgentReadiness";
import type { OnboardingSetupApi } from "../useOnboardingSetupState";
import { FirstRunCodexSubscription } from "./FirstRunCodexSubscription";
import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunHeading, FirstRunPanel, FirstRunShell } from "./FirstRunShell";
import { FirstRunModelSourceChoice } from "./FirstRunModelSourceChoice";
import type { FirstRunChrome } from "./firstRunTypes";
import type { FirstRunSphereProps } from "./FirstRunSpherePane";
import type { IntegrationId } from "../useFirstRunSetupFlow";
import type { OnboardingPageModel } from "../useFirstRunSetupFlow";

const AGENT_STEP: { id: "agent"; label: string; purpose: string } = WIZARD_STEPS[3];

type AgentModelSource = {
  offered: boolean;
  choice: OnboardingAgentCredentialChoice | null;
  onSelect: (choice: OnboardingAgentCredentialChoice) => void;
};

type CodexPanel = {
  offered: boolean;
  connection: OnboardingPageModel["codexConnection"];
  ready: boolean;
  starting: boolean;
  canceling: boolean;
  onStart: () => void;
  onCancel: () => void;
};

function agentScreenBody(modelSource: AgentModelSource, customProvider: boolean): string {
  if (modelSource.choice === "own-key") {
    return customProvider ? FIRST_RUN_COPY.agent.ownKeyBodyWithCustom : FIRST_RUN_COPY.agent.ownKeyBody;
  }
  if (modelSource.offered) return FIRST_RUN_COPY.agent.modelSourceBody;
  return AGENT_STEP.purpose;
}

function finishCredentialChoice(modelSource: AgentModelSource): OnboardingAgentCredentialChoice | null {
  if (modelSource.choice === "subscription") return "subscription";
  if (modelSource.offered && !modelSource.choice) return "hosted";
  return modelSource.choice;
}

function AgentConnectionChoices({
  organizationId,
  setup,
  saving,
  loading,
  showCustomProvider,
  modelSource,
  codex,
  subscriptionSelected,
  showProviders,
  modelPicker,
  onRequestConnect,
}: {
  organizationId: string;
  setup: OnboardingSetupApi;
  saving: boolean;
  loading: boolean;
  showCustomProvider: boolean;
  modelSource: AgentModelSource;
  codex?: CodexPanel;
  subscriptionSelected: boolean;
  showProviders: boolean;
  modelPicker?: OnboardingAgentModelPicker;
  onRequestConnect: (id: IntegrationId) => void;
}) {
  return (
    <>
      {codex?.offered ? (
        <FirstRunPanel>
          <FirstRunCodexSubscription
            selected={subscriptionSelected}
            disabled={saving || loading}
            connection={codex.connection}
            starting={codex.starting}
            canceling={codex.canceling}
            onSelect={() => modelSource.onSelect("subscription")}
            onStart={codex.onStart}
            onCancel={codex.onCancel}
          />
        </FirstRunPanel>
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
              modelPicker={modelPicker}
              onRequestConnect={onRequestConnect}
            />
          </FirstRunPanel>
        </fieldset>
      ) : null}
    </>
  );
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
  codex,
  modelPicker,
  onRequestConnect,
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
  codex?: CodexPanel;
  modelPicker?: OnboardingAgentModelPicker;
  onRequestConnect: (id: IntegrationId) => void;
  onContinue: () => void;
}) {
  const subscriptionSelected = modelSource.choice === "subscription";
  const canFinish = agentFinishReady({
    modelSourceChoice: modelSource.offered,
    credentialChoice: finishCredentialChoice(modelSource),
    providerConnected: isAgentProviderConnected(setup.connected),
    agentReady: setup.agentReady,
    hostedAgentReady,
    subscriptionReady: codex?.ready,
  });
  const showProviders = !subscriptionSelected && (!modelSource.offered || modelSource.choice === "own-key");
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
        <AgentConnectionChoices
          organizationId={organizationId}
          setup={setup}
          saving={saving}
          loading={loading}
          showCustomProvider={showCustomProvider}
          modelSource={modelSource}
          codex={codex}
          subscriptionSelected={subscriptionSelected}
          showProviders={showProviders}
          modelPicker={modelPicker}
          onRequestConnect={onRequestConnect}
        />
        <div className="space-y-3">
          <LoadingButton
            type="button"
            className="w-full"
            disabled={!canFinish || loading || Boolean(showProviders && modelPicker && !modelPicker.value)}
            loading={saving}
            loadingText={FIRST_RUN_COPY.finish.saving}
            onClick={onContinue}
            data-testid="first-run-finish-setup"
          >
            {modelSource.offered ? FIRST_RUN_COPY.tickets.continue : FIRST_RUN_COPY.finish.action}
          </LoadingButton>
        </div>
      </div>
    </FirstRunShell>
  );
}
