import { useAccount } from "@/contexts/useAccount";
import { useAccountOrganizations } from "@/hooks/useAccountOrganizations";
import { useIntegrationResources } from "@/hooks/useIntegrations";
import { organizationMatchesRoute } from "@/lib/accountOrganizations";
import { posthog } from "@/posthog";
import { useNavigate } from "react-router";

import { useFactoriesLayout } from "../../layout/factoriesLayoutContext";
import { FirstRunAnalysisHost } from "./first-run/FirstRunAnalysisHost";
import { FirstRunBitbucketChooseScreen } from "./first-run/FirstRunBitbucketChooseScreen";
import { FirstRunBitbucketForgeScreen } from "./first-run/FirstRunBitbucketForgeScreen";
import { FirstRunHostScreen } from "./first-run/FirstRunHostScreen";
import { FirstRunAgentScreen } from "./first-run/FirstRunAgentScreen";
import { FirstRunChooseScreen } from "./first-run/FirstRunChooseScreen";
import { FirstRunConnectScreen } from "./first-run/FirstRunConnectScreen";
import { FIRST_RUN_STEP_COUNT } from "./first-run/FirstRunShell";
import { FirstRunTicketsScreen } from "./first-run/FirstRunTicketsScreen";
import type { FirstRunChrome } from "./first-run/firstRunTypes";
import { FIRST_RUN_COPY } from "./first-run/firstRunCopy";
import { FirstRunWelcomeScreen } from "./first-run/FirstRunWelcomeScreen";
import type { FirstRunSphereProps } from "./first-run/FirstRunSpherePane";
import { repositorySphereFor, sphereFor } from "./first-run/firstRunSphereFor";
import { afterOnboardingPath } from "./useFinishOnboarding";
import { useGitHubInstallationChecks } from "./useGitHubOnboarding";
import {
  useFirstRunSetupFlow,
  type FirstRunScreen,
  type FirstRunSetupFlow,
  type OnboardingPageModel,
} from "./useFirstRunSetupFlow";
const STEP_INDEX_FOR_SCREEN: Record<FirstRunScreen, number> = {
  welcome: 0,
  host: 1,
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
  host: "welcome",
  connect: "host",
  choose: "connect",
  tickets: "choose",
  agent: "tickets",
};

const BACK_SCREEN_AGENT_FIRST: Partial<Record<FirstRunScreen, FirstRunScreen>> = {
  ...BACK_SCREEN,
  agent: "choose",
  tickets: "agent",
};

function firstNameOf(name: string | undefined): string | undefined {
  const first = name?.trim().split(/\s+/)[0];
  return first || undefined;
}

function signOut() {
  posthog.reset();
  window.location.href = "/logout";
}

function backActionFor(target: FirstRunScreen, flow: FirstRunSetupFlow): (() => void) | undefined {
  const backScreen = (flow.agentBeforeTickets ? BACK_SCREEN_AGENT_FIRST : BACK_SCREEN)[target];
  // Bitbucket has no separate connect screen.
  if (backScreen === "connect" && (flow.vcsHost === "bitbucket" || !flow.accountConnectionRequired)) {
    return () => flow.goToScreen("host");
  }
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
      githubAvailable={flow.vcsHost !== "bitbucket"}
      ticketSource={flow.ticketSource}
      chrome={chrome}
      sphere={sphere}
      continueLabel={ticketsContinueLabel(flow.ticketsFinishSetup)}
      continuePending={flow.agentGatePending}
      saving={flow.blockingAction === "saving-ticket-source" || finishing}
      savingLabel={finishing ? FIRST_RUN_COPY.finish.saving : FIRST_RUN_COPY.tickets.saving}
      githubConnected={model.setup.connected.has("github")}
      onConnectGitHub={() => void flow.connectGitHubIssues()}
      jiraConnected={model.setup.connected.has("jira")}
      jiraProjects={model.jiraProjects}
      jiraProjectsLoading={model.jiraProjectsLoading}
      jiraProjectsError={model.jiraProjectsError}
      jiraProjectId={model.jiraProjectId}
      linearConnected={model.setup.connected.has("linear")}
      linearProjects={model.linearProjects}
      linearProjectsLoading={model.linearProjectsLoading}
      linearProjectsError={model.linearProjectsError}
      linearProjectIds={model.linearProjectIds}
      onSelectTicketSource={flow.selectTicketSource}
      onConnectJira={() => void flow.connectJira()}
      onSelectJiraProject={model.setJiraProjectId}
      onRetryJiraProjects={model.retryJiraProjects}
      onConnectLinear={() => void flow.connectLinear()}
      onToggleLinearProject={model.toggleLinearProject}
      onRetryLinearProjects={model.retryLinearProjects}
      onAnalyzeTickets={() => void flow.continueFromTickets()}
    />
  );
}

function bitbucketForgePhase(
  identityLinked: boolean,
  repositoryCount: number,
  installedWorkspaceCount: number,
): "connect" | "grant" | "choose" {
  if (!identityLinked) return "connect";
  if (repositoryCount === 0 && installedWorkspaceCount === 0) return "grant";
  return "choose";
}

function BitbucketChooseHost({
  organizationId,
  flow,
  model,
  chrome,
}: {
  organizationId: string;
  flow: FirstRunSetupFlow;
  model: OnboardingPageModel;
  chrome: FirstRunChrome;
}) {
  const connected = model.setup.connected.has("bitbucket");
  const resources = useIntegrationResources(organizationId, model.bitbucketIntegrationId, "repository", undefined, {
    enabled: connected && !flow.bitbucketForgeConfigured,
  });
  const repositories = (resources.data ?? []).map((resource) => resource.name ?? "").filter(Boolean);
  const sphere = connected ? repositorySphereFor(model.setup.selectedRepo) : sphereFor("connect", null);
  if (flow.bitbucketOnboardingPending || flow.bitbucketForgeConfigured || flow.bitbucketLookupFailed) {
    return (
      <FirstRunBitbucketForgeScreen
        phase={bitbucketForgePhase(
          flow.bitbucketIdentityLinked,
          flow.bitbucketRepositories.length,
          flow.bitbucketInstalledWorkspaces.length,
        )}
        connectHref={flow.bitbucketConnectHref}
        repositories={flow.bitbucketRepositories}
        installedWorkspaces={flow.bitbucketInstalledWorkspaces}
        selectedRepository={model.setup.selectedRepo}
        granting={flow.blockingAction === "opening-bitbucket"}
        saving={flow.blockingAction === "saving-repository"}
        loading={flow.bitbucketOnboardingPending}
        loadError={flow.bitbucketLoadError}
        lookupFailed={flow.bitbucketLookupFailed}
        retrying={flow.bitbucketLookupRetrying}
        attemptActive={flow.bitbucketInstallationAttemptActive}
        attemptTimedOut={flow.bitbucketInstallationAttemptTimedOut}
        login={flow.bitbucketLogin}
        providerUserId={flow.bitbucketProviderUserId}
        identities={flow.bitbucketIdentities}
        switchingAccount={flow.blockingAction === "switching-bitbucket-account"}
        switchFailed={flow.bitbucketSwitchFailed}
        chrome={chrome}
        sphere={sphere}
        onRetryLookup={flow.retryBitbucketLookup}
        onGrantAccess={() => void flow.grantBitbucketAccess()}
        installUrl={flow.bitbucketInstallUrl}
        onCopyInstallLink={() => flow.copyBitbucketInstallLink()}
        onSelectRepository={model.setup.selectRepo}
        onClearRepository={model.setup.clearRepository}
        onContinue={() => void flow.continueFromRepository()}
        onSelectIdentity={(providerUserId) => void flow.selectBitbucketIdentity(providerUserId)}
        onConnectAnotherAccount={() => window.location.assign(flow.bitbucketConnectHref)}
      />
    );
  }
  return (
    <FirstRunBitbucketChooseScreen
      connected={connected}
      repositories={repositories}
      selectedRepository={model.setup.selectedRepo}
      loading={resources.isLoading}
      loadError={resources.isError}
      saving={flow.blockingAction === "saving-repository"}
      chrome={chrome}
      sphere={sphere}
      onConnect={flow.connectBitbucket}
      onSelectRepository={model.setup.selectRepo}
      onContinue={() => void flow.continueFromRepository()}
    />
  );
}

/** The Git host choice, the connection, and the repository choice. */
function VcsScreen({
  screen,
  organizationId,
  flow,
  model,
  chrome,
}: {
  screen: "host" | "connect" | "choose";
  organizationId: string;
  flow: FirstRunSetupFlow;
  model: OnboardingPageModel;
  chrome: FirstRunChrome;
}) {
  const setup = model.setup;
  if (screen === "host") {
    return (
      <FirstRunHostScreen
        selectedHost={flow.vcsHost}
        bitbucketAvailable={flow.bitbucketAvailable}
        bitbucketFeatureLoading={flow.bitbucketFeatureLoading}
        chrome={chrome}
        sphere={sphereFor("host", setup.selectedRepo)}
        onChooseHost={flow.chooseVcsHost}
      />
    );
  }

  if (screen === "connect") {
    return (
      <FirstRunConnectScreen
        loading={flow.repositoriesLoading}
        connecting={flow.blockingAction === "opening-github"}
        createApp={!flow.appConfigured}
        connectError={flow.connectError}
        chrome={chrome}
        sphere={sphereFor("connect", setup.selectedRepo)}
        onConnectGitHub={() => void (flow.appConfigured ? flow.connectGitHub() : flow.createGitHubApp())}
      />
    );
  }

  if (flow.vcsHost === "bitbucket") {
    return <BitbucketChooseHost organizationId={organizationId} flow={flow} model={model} chrome={chrome} />;
  }

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
      chrome={chrome}
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
  useGitHubInstallationChecks(organizationId, flow.screen === "choose" && flow.vcsHost !== "bitbucket" && !destination);
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
        onGetStarted={() => flow.goToScreen("host")}
      />
    );
  }

  if (flow.screen === "host" || flow.screen === "connect" || flow.screen === "choose") {
    return (
      <VcsScreen
        screen={flow.screen}
        organizationId={organizationId}
        flow={flow}
        model={model}
        chrome={chromeFor(flow.screen)}
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
    <FirstRunAgentScreen
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
      codex={{
        offered: Boolean(model.codexSubscriptionOffered),
        connection: model.codexConnection,
        ready: Boolean(model.codexSubscriptionReady),
        starting: Boolean(model.codexStarting),
        canceling: Boolean(model.codexCanceling),
        onStart: model.startCodexConnection,
        onCancel: model.cancelCodexConnection,
      }}
      onRequestConnect={model.requestConnect}
      onContinue={() => {
        if (flow.agentBeforeTickets && !flow.credentialChoice) flow.selectCredentialChoice("hosted");
        void flow.continueFromAgent();
      }}
    />
  );
}
