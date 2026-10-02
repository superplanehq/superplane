import { LoadingButton } from "@/components/ui/loading-button";
import { useAccount } from "@/contexts/useAccount";
import { useAccountOrganizations } from "@/hooks/useAccountOrganizations";
import { organizationMatchesRoute } from "@/lib/accountOrganizations";
import { posthog } from "@/posthog";
import { useNavigate } from "react-router";

import { useFactoriesLayout } from "../../layout/factoriesLayoutContext";
import { AgentStep } from "./AgentStep";
import { FirstRunAnalysisHost } from "./first-run/FirstRunAnalysisHost";
import { FirstRunModelSourceChoice } from "./first-run/FirstRunModelSourceChoice";
import { FirstRunChooseScreen } from "./first-run/FirstRunChooseScreen";
import { FirstRunConnectScreen } from "./first-run/FirstRunConnectScreen";
import { FIRST_RUN_STEP_COUNT, FirstRunHeading, FirstRunPanel, FirstRunShell } from "./first-run/FirstRunShell";
import { FirstRunTicketsScreen } from "./first-run/FirstRunTicketsScreen";
import type { FirstRunChrome } from "./first-run/firstRunTypes";
import { FIRST_RUN_COPY } from "./first-run/firstRunCopy";
import { FirstRunWelcomeScreen } from "./first-run/FirstRunWelcomeScreen";
import type { FirstRunSphereProps } from "./first-run/FirstRunSpherePane";
import { sphereFor } from "./first-run/firstRunSphereFor";
import {
  agentFinishReady,
  isAgentProviderConnected,
  type OnboardingAgentCredentialChoice,
} from "./onboardingAgentReadiness";
import { WIZARD_STEPS } from "./onboardingFixtures";
import { afterOnboardingPath } from "./useFinishOnboarding";
import {
  useFirstRunSetupFlow,
  type FirstRunScreen,
  type FirstRunSetupFlow,
  type IntegrationId,
  type OnboardingPageModel,
} from "./useFirstRunSetupFlow";
import type { OnboardingSetupApi } from "./useOnboardingSetupState";

const STEP_INDEX_FOR_SCREEN: Record<FirstRunScreen, number> = {
  welcome: 0,
  connect: 1,
  choose: 2,
  tickets: 3,
  agent: 4,
};

// A bring-your-own-key organization chooses the model source before the backlog.
const STEP_INDEX_FOR_SCREEN_AGENT_FIRST: Record<FirstRunScreen, number> = {
  ...STEP_INDEX_FOR_SCREEN,
  agent: 3,
  tickets: 4,
};

// The reverse path walks the exact screens in reverse order.
const BACK_SCREEN: Partial<Record<FirstRunScreen, FirstRunScreen>> = {
  connect: "welcome",
  choose: "connect",
  tickets: "choose",
  agent: "tickets",
};

const BACK_SCREEN_AGENT_FIRST: Partial<Record<FirstRunScreen, FirstRunScreen>> = {
  ...BACK_SCREEN,
  agent: "choose",
  tickets: "agent",
};

/**
 * The first-run screens have no coding agent screen. This screen keeps the
 * wizard step copy, because provisioning needs a connected agent.
 */
const AGENT_STEP: { id: "agent"; label: string; purpose: string } = WIZARD_STEPS[3];

function firstNameOf(name: string | undefined): string | undefined {
  const first = name?.trim().split(/\s+/)[0];
  return first || undefined;
}

function signOut() {
  posthog.reset();
  window.location.href = "/logout";
}

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

function AgentScreen({
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
    <FirstRunShell testId="first-run-agent" chrome={chrome} busy={saving || loading} width="wide" sphere={sphere}>
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
              />
            </FirstRunPanel>
          </fieldset>
        ) : null}
        <div className="space-y-3">
          <LoadingButton
            type="button"
            className="w-full"
            disabled={!canFinish || loading}
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

function backActionFor(target: FirstRunScreen, flow: FirstRunSetupFlow): (() => void) | undefined {
  const backScreen = (flow.agentBeforeTickets ? BACK_SCREEN_AGENT_FIRST : BACK_SCREEN)[target];
  return backScreen ? () => flow.goToScreen(backScreen) : undefined;
}

/** Hosted credentials provision from this screen, so it shows finish progress. */
function ticketsContinueLabel(ticketsFinishSetup: boolean): string {
  return ticketsFinishSetup ? FIRST_RUN_COPY.tickets.analyze : FIRST_RUN_COPY.tickets.continue;
}

function TicketsScreenHost({
  flow,
  model,
  saving,
  chrome,
  sphere,
}: {
  flow: FirstRunSetupFlow;
  model: OnboardingPageModel;
  saving: boolean;
  chrome: FirstRunChrome;
  sphere?: FirstRunSphereProps;
}) {
  const finishing = flow.blockingAction === "finishing-setup" || (flow.ticketsFinishSetup && saving);
  return (
    <FirstRunTicketsScreen
      ticketSource={flow.ticketSource}
      chrome={chrome}
      sphere={sphere}
      jiraAvailable={flow.jiraAvailable}
      jiraFeatureLoading={flow.jiraFeatureLoading}
      jiraChoiceBlock={flow.jiraChoiceBlock}
      continueLabel={ticketsContinueLabel(flow.ticketsFinishSetup)}
      continuePending={flow.agentGatePending}
      saving={flow.blockingAction === "saving-ticket-source" || finishing}
      savingLabel={finishing ? FIRST_RUN_COPY.finish.saving : FIRST_RUN_COPY.tickets.saving}
      jiraConnected={model.setup.connected.has("jira")}
      jiraProjects={model.jiraProjects}
      jiraProjectsLoading={model.jiraProjectsLoading}
      jiraProjectsError={model.jiraProjectsError}
      jiraProjectId={model.jiraProjectId}
      onSelectTicketSource={flow.selectTicketSource}
      onConnectJira={() => void flow.connectJira()}
      onSelectJiraProject={model.setJiraProjectId}
      onRetryJiraProjects={model.retryJiraProjects}
      onAnalyzeTickets={() => void flow.continueFromTickets()}
    />
  );
}

/**
 * Workspace setup, on the first-run screens. Each answer is saved through the
 * setup model. The last screen provisions the workspace and opens it.
 */
export function FirstRunSetup({ model }: { model: OnboardingPageModel }) {
  const { account } = useAccount();
  const { organizationId, factoryId, factories } = useFactoriesLayout();
  const navigate = useNavigate();
  const flow = useFirstRunSetupFlow(model);
  const destination = model.provisionedDestination;
  const setup = model.setup;
  const accountOrganizations = useAccountOrganizations();

  // The placeholder workspace under setup is itself in `factories`, so
  // another workspace exists when any factory has a different id.
  const hasOtherWorkspace = factories.some((existing) => existing.id !== factoryId);
  // The organization switch is only for other organizations. Another
  // workspace in this organization uses the workspace switch.
  const otherOrganizations = (accountOrganizations.data ?? []).filter(
    (organization) => !organizationMatchesRoute(organization, organizationId),
  );

  const chromeFor = (target: FirstRunScreen): FirstRunChrome => {
    return {
      displayName: firstNameOf(account?.name),
      email: account?.email,
      onLogOut: signOut,
      organizationSwitch: otherOrganizations.length > 0 ? { currentOrganizationRouteId: organizationId } : undefined,
      workspaceSwitch: hasOtherWorkspace ? { organizationId, currentFactoryId: factoryId, factories } : undefined,
      stepIndex: (flow.agentBeforeTickets ? STEP_INDEX_FOR_SCREEN_AGENT_FIRST : STEP_INDEX_FOR_SCREEN)[target],
      stepCount: flow.skipAgentScreen ? FIRST_RUN_STEP_COUNT - 1 : FIRST_RUN_STEP_COUNT,
      onBack: backActionFor(target, flow),
      busy: flow.busy,
    };
  };

  if (destination) {
    return (
      <FirstRunAnalysisHost
        organizationId={destination.organizationId}
        factoryId={factoryId}
        chrome={{ displayName: firstNameOf(account?.name), email: account?.email, onLogOut: signOut, stepIndex: 4 }}
        selectedRepo={setup.selectedRepo}
        onGoToBoard={() => navigate(afterOnboardingPath(destination), { replace: true })}
      />
    );
  }

  if (flow.screen === "welcome") {
    return (
      <FirstRunWelcomeScreen
        firstName={firstNameOf(account?.name)}
        chrome={chromeFor("welcome")}
        sphere={sphereFor("welcome", setup.selectedRepo)}
        onGetStarted={() => flow.goToScreen("connect")}
      />
    );
  }

  if (flow.screen === "connect") {
    return (
      <FirstRunConnectScreen
        loading={flow.repositoriesLoading}
        connecting={flow.blockingAction === "opening-github"}
        connectError={flow.connectError}
        chrome={chromeFor("connect")}
        sphere={sphereFor("connect", setup.selectedRepo)}
        onConnectGitHub={() => void flow.connectGitHub()}
      />
    );
  }

  if (flow.screen === "choose") {
    return (
      <FirstRunChooseScreen
        repositories={flow.repositories}
        selectedRepository={setup.selectedRepo}
        githubLogin={flow.githubLogin}
        githubUserId={flow.githubUserId}
        githubIdentities={flow.githubIdentities}
        loading={flow.repositoriesLoading}
        saving={flow.blockingAction === "saving-repository"}
        grantingAccess={flow.blockingAction === "opening-github"}
        switchingGitHubAccount={flow.blockingAction === "switching-github-account"}
        synchronizing={flow.synchronizing}
        pendingOrganizations={flow.pendingOrganizations}
        appConfigured={flow.appConfigured}
        chrome={chromeFor("choose")}
        sphere={sphereFor("choose", setup.selectedRepo, model.githubOwner)}
        onSelectRepository={setup.selectRepo}
        onClearRepository={setup.clearRepository}
        onSelectGitHubIdentity={(userId) => void flow.selectGitHubIdentity(userId)}
        onConnectAnotherGitHubAccount={() => void flow.connectGitHub()}
        onGrantAccess={() => void flow.grantGitHubAccess()}
        onContinue={() => void flow.continueFromRepository()}
      />
    );
  }

  if (flow.screen === "tickets") {
    return (
      <TicketsScreenHost
        flow={flow}
        model={model}
        saving={model.saving}
        chrome={chromeFor("tickets")}
        sphere={sphereFor("tickets", setup.selectedRepo, model.githubOwner)}
      />
    );
  }

  return (
    <AgentScreen
      organizationId={organizationId}
      setup={setup}
      chrome={chromeFor("agent")}
      sphere={sphereFor("agent", setup.selectedRepo, model.githubOwner)}
      saving={!flow.agentBeforeTickets && (flow.blockingAction === "finishing-setup" || model.saving)}
      loading={model.agentLoading}
      hostedAgentReady={model.hostedAgentReady}
      showCustomProvider={model.customProvider}
      modelSource={{
        offered: flow.agentBeforeTickets,
        choice: flow.credentialChoice,
        onSelect: flow.selectCredentialChoice,
      }}
      onRequestConnect={model.requestConnect}
      onContinue={() => {
        if (flow.agentBeforeTickets && !flow.credentialChoice) flow.selectCredentialChoice("hosted");
        void flow.continueFromAgent();
      }}
    />
  );
}
