import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";

import { AgentStep, type OnboardingAgentModelPicker } from "../AgentStep";
import { WIZARD_STEPS, type IntegrationId } from "../onboardingFixtures";
import {
  agentFinishReady,
  isAgentProviderConnected,
  type OnboardingAgentCredentialChoice,
} from "../onboardingAgentReadiness";
import type { OnboardingSetupApi } from "../useOnboardingSetupState";
import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunHeading, FirstRunPanel, FirstRunShell } from "./FirstRunShell";
import { FirstRunModelSourceChoice } from "./FirstRunModelSourceChoice";
import type { FirstRunChrome } from "./firstRunTypes";
import type { FirstRunSphereProps } from "./FirstRunSpherePane";

const AGENT_STEP: { id: "agent"; label: string; purpose: string } = WIZARD_STEPS[3];

export type OnboardingSubscriptionChoice = {
  offered: boolean;
  ready: boolean;
  connect: () => void;
};

type AgentModelSource = {
  offered: boolean;
  choice: OnboardingAgentCredentialChoice | null;
  onSelect: (choice: OnboardingAgentCredentialChoice) => void;
};

function agentScreenBody(modelSource: AgentModelSource, customProvider: boolean): string {
  if (modelSource.choice === "subscription") return "Connect your ChatGPT subscription to use Codex.";
  if (modelSource.choice === "own-key") {
    return customProvider ? FIRST_RUN_COPY.agent.ownKeyBodyWithCustom : FIRST_RUN_COPY.agent.ownKeyBody;
  }
  if (modelSource.offered) return FIRST_RUN_COPY.agent.modelSourceBody;
  return AGENT_STEP.purpose;
}

function finishCredentialChoice(modelSource: AgentModelSource): OnboardingAgentCredentialChoice | null {
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
  subscription,
  providerKeysOffered,
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
  subscription: OnboardingSubscriptionChoice;
  providerKeysOffered: boolean;
  showProviders: boolean;
  modelPicker?: OnboardingAgentModelPicker;
  onRequestConnect: (id: IntegrationId) => void;
}) {
  if (subscription.offered) {
    return (
      <fieldset disabled={saving || loading} className="mx-0 min-w-0 border-0 p-0">
        <FirstRunPanel>
          <div className="grid gap-4">
            <div className="flex flex-wrap gap-2" role="group" aria-label="Model source">
              <Button
                type="button"
                variant="outline"
                aria-pressed={modelSource.choice !== "subscription"}
                onClick={() => modelSource.onSelect("own-key")}
              >
                Use an API key
              </Button>
              <Button
                type="button"
                variant="outline"
                aria-pressed={modelSource.choice === "subscription"}
                onClick={() => modelSource.onSelect("subscription")}
              >
                Use a ChatGPT subscription
              </Button>
            </div>
            {modelSource.choice === "subscription" ? (
              subscription.ready ? (
                <p role="status" className="text-sm">
                  ChatGPT connected.
                </p>
              ) : (
                <Button type="button" variant="outline" onClick={subscription.connect}>
                  Connect ChatGPT
                </Button>
              )
            ) : (
              <AgentStep
                organizationId={organizationId}
                setup={setup}
                showHostedCredit={false}
                showCustomProvider={showCustomProvider}
                modelPicker={modelPicker}
                onRequestConnect={onRequestConnect}
              />
            )}
          </div>
        </FirstRunPanel>
      </fieldset>
    );
  }
  return (
    <>
      {modelSource.offered ? (
        <FirstRunPanel>
          <FirstRunModelSourceChoice
            disabled={saving || loading}
            modelSource={modelSource.choice}
            providerKeysOffered={providerKeysOffered}
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
  providerKeysOffered,
  showCustomProvider,
  modelSource,
  subscription,
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
  providerKeysOffered: boolean;
  showCustomProvider: boolean;
  modelSource: AgentModelSource;
  subscription: OnboardingSubscriptionChoice;
  modelPicker?: OnboardingAgentModelPicker;
  onRequestConnect: (id: IntegrationId) => void;
  onContinue: () => void;
}) {
  const canFinish = agentFinishReady({
    modelSourceChoice: modelSource.offered,
    credentialChoice: finishCredentialChoice(modelSource),
    providerConnected: isAgentProviderConnected(setup.connected),
    agentReady: setup.agentReady,
    hostedAgentReady,
    subscriptionReady: subscription.ready,
  });
  const showProviders =
    providerKeysOffered &&
    modelSource.choice !== "subscription" &&
    (!modelSource.offered || modelSource.choice === "own-key");
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
          subscription={subscription}
          providerKeysOffered={providerKeysOffered}
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
