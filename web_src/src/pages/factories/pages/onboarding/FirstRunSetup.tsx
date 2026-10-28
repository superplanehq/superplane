import { Button } from "@/components/ui/button";
import { FirstRunHeading, FirstRunShell } from "./first-run/FirstRunShell";
import { useAccount } from "@/contexts/useAccount";
import { useAccountOrganizations } from "@/hooks/useAccountOrganizations";
import { useIntegrationResources } from "@/hooks/useIntegrations";
import { organizationMatchesRoute } from "@/lib/accountOrganizations";
import { posthog } from "@/posthog";
import { useNavigate } from "react-router";

import { useFactoriesLayout } from "../../layout/factoriesLayoutContext";
import { FirstRunAgentScreen } from "./first-run/FirstRunAgentScreen";
import { FirstRunAnalysisHost } from "./first-run/FirstRunAnalysisHost";
import { FirstRunBitbucketChooseScreen } from "./first-run/FirstRunBitbucketChooseScreen";
import { FirstRunBitbucketForgeScreen } from "./first-run/FirstRunBitbucketForgeScreen";
import { FirstRunHostScreen } from "./first-run/FirstRunHostScreen";
import { FirstRunChooseScreen } from "./first-run/FirstRunChooseScreen";
import { FirstRunConnectScreen } from "./first-run/FirstRunConnectScreen";
import { FIRST_RUN_STEP_COUNT } from "./first-run/FirstRunShell";
import { FirstRunTicketsScreen } from "./first-run/FirstRunTicketsScreen";
import type { FirstRunChrome } from "./first-run/firstRunTypes";
import { FIRST_RUN_COPY } from "./first-run/firstRunCopy";
import { FirstRunWelcomeScreen } from "./first-run/FirstRunWelcomeScreen";
import type { FirstRunSphereProps } from "./first-run/FirstRunSpherePane";
import { repositorySphereFor, sphereFor } from "./first-run/firstRunSphereFor";
import { ownerOfRepository, repositoriesInOrganization } from "./first-run/githubOrganizations";
import { FirstRunCardScreenHost } from "./FirstRunCardScreenHost";
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
  card: 4,
  complete: 5,
  agent: 5,
};

/** How many existing issues the first import takes. */
const INITIAL_IMPORT_LIMIT = 3;

const BACK_SCREEN: Partial<Record<FirstRunScreen, FirstRunScreen>> = {
  host: "welcome",
  connect: "host",
  choose: "connect",
  tickets: "choose",
  card: "tickets",
  complete: "tickets",
  agent: "tickets",
};

// Steps that this flow does not show. The card step is absent when billing
// has no starter plans or the organization already has a paid plan.
function stepsSkipped(flow: FirstRunSetupFlow, target: FirstRunScreen): { before: number; total: number } {
  const agent = Number(flow.skipAgentScreen);
  const card = Number(!flow.hasCardStep);
  return {
    before:
      Number(target === "complete") * agent + Number(STEP_INDEX_FOR_SCREEN[target] > STEP_INDEX_FOR_SCREEN.card) * card,
    total: agent + card,
  };
}

function firstNameOf(name: string | undefined): string | undefined {
  const first = name?.trim().split(/\s+/)[0];
  return first || undefined;
}

function modelSourceOffered(model: OnboardingPageModel): boolean {
  return (model.bringYourOwnKey || model.codexSubscriptionOffered) && model.hostedModelsAvailable;
}

function signOut() {
  posthog.reset();
  window.location.href = "/logout";
}

// The last action on the tickets step scans the backlog. When the card step
// or the model step follows, the button only continues.
function ticketsContinueLabel(flow: FirstRunSetupFlow, skipInitialImport: boolean, initialWorkspace: boolean) {
  const cardStepNext = !flow.cardStep.ready || flow.cardStep.required || flow.cardStep.requiresAdmin;
  if (cardStepNext || !flow.skipAgentScreen) return FIRST_RUN_COPY.tickets.continue;
  if (!initialWorkspace && skipInitialImport) return FIRST_RUN_COPY.finish.action;
  return FIRST_RUN_COPY.tickets.analyze;
}

function backActionFor(target: FirstRunScreen, flow: FirstRunSetupFlow): (() => void) | undefined {
  const backScreen = BACK_SCREEN[target];
  // Bitbucket has no separate connect screen.
  if (backScreen === "connect" && (flow.vcsHost === "bitbucket" || !flow.accountConnectionRequired)) {
    return () => flow.goToScreen("host");
  }
  return backScreen ? () => flow.goToScreen(backScreen) : undefined;
}

function TicketsScreenHost({
  flow,
  model,
  chrome,
  sphere,
  initialWorkspace,
}: {
  initialWorkspace: boolean;
  flow: FirstRunSetupFlow;
  model: OnboardingPageModel;
  chrome: FirstRunChrome;
  sphere?: FirstRunSphereProps;
}) {
  const connecting =
    flow.blockingAction === "opening-github" ||
    flow.blockingAction === "connecting-jira" ||
    flow.blockingAction === "connecting-linear";
  // The backlog repository stays in the organization of the app repository,
  // the same filter the Git host step applies.
  const organization = ownerOfRepository(model.setup.selectedRepo);
  const repositories = organization ? repositoriesInOrganization(flow.repositories, organization) : flow.repositories;
  return (
    <FirstRunTicketsScreen
      initialImportLimit={INITIAL_IMPORT_LIMIT}
      skipInitialImport={model.setup.skipInitialImport}
      // The first workspace always imports: that batch feeds the analysis.
      onSkipInitialImportChange={initialWorkspace ? undefined : model.setup.setSkipInitialImport}
      automaticIntake={model.setup.automaticIntake}
      onAutomaticIntakeChange={model.setup.setAutomaticIntake}
      githubAvailable={flow.vcsHost !== "bitbucket"}
      ticketSource={flow.ticketSource}
      // GitHub is only a default until the user picks it on the list.
      startOnSourceList={model.setup.issuesChoice == null}
      repository={model.setup.issuesRepo ?? model.setup.selectedRepo ?? undefined}
      repositories={repositories}
      onSelectRepository={model.setup.selectIssuesRepo}
      connecting={connecting}
      chrome={chrome}
      sphere={sphere}
      continueLabel={ticketsContinueLabel(flow, model.setup.skipInitialImport, initialWorkspace)}
      continuePending={false}
      saving={flow.busy}
      savingLabel={FIRST_RUN_COPY.tickets.saving}
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
  const { organizationId, factoryId, factories, factory } = useFactoriesLayout();
  const navigate = useNavigate();
  const flow = useFirstRunSetupFlow(model);
  const destination = model.provisionedDestination;
  useGitHubInstallationChecks(organizationId, flow.screen === "choose" && flow.vcsHost !== "bitbucket" && !destination);
  const setup = model.setup;
  const accountOrganizations = useAccountOrganizations();
  const initialWorkspace = factory?.onboarding?.initial === true;

  // The placeholder workspace under setup is itself in `factories`, so
  // another workspace exists when any factory has a different id.
  const hasOtherWorkspace = factories.some((existing) => existing.id !== factoryId);
  // The organization switch is only for other organizations. Another
  // workspace in this organization uses the workspace switch.
  const otherOrganizations = (accountOrganizations.data ?? []).filter(
    (organization) => !organizationMatchesRoute(organization, organizationId),
  );

  const chromeFor = (target: FirstRunScreen): FirstRunChrome => {
    const skipped = stepsSkipped(flow, target);
    return {
      displayName: firstNameOf(account?.name),
      email: account?.email,
      onLogOut: signOut,
      organizationSwitch: otherOrganizations.length > 0 ? { currentOrganizationRouteId: organizationId } : undefined,
      workspaceSwitch: hasOtherWorkspace ? { organizationId, currentFactoryId: factoryId, factories } : undefined,
      stepIndex: STEP_INDEX_FOR_SCREEN[target] - skipped.before,
      stepCount: FIRST_RUN_STEP_COUNT - skipped.total,
      onBack: backActionFor(target, flow),
      busy: flow.busy,
    };
  };

  if (destination) {
    return (
      <FirstRunAnalysisHost
        organizationId={destination.organizationId}
        factoryId={factoryId}
        chrome={{
          ...chromeFor("complete"),
          onBack: undefined,
          stepIndex: FIRST_RUN_STEP_COUNT - 1 - stepsSkipped(flow, "complete").total,
        }}
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

  if (flow.screen === "card") {
    return (
      <FirstRunCardScreenHost
        organizationId={organizationId}
        flow={flow}
        chrome={chromeFor("card")}
        sphere={sphereFor("card", setup.selectedRepo, model.githubOwner)}
      />
    );
  }

  if (flow.screen === "complete") {
    return (
      <CompleteSetupScreen
        flow={flow}
        chrome={chromeFor("complete")}
        sphere={sphereFor("complete", setup.selectedRepo, model.githubOwner)}
      />
    );
  }

  if (flow.screen === "tickets") {
    return (
      <TicketsScreenHost
        initialWorkspace={initialWorkspace}
        flow={flow}
        model={model}
        chrome={chromeFor("tickets")}
        sphere={sphereFor("tickets", setup.selectedRepo, model.githubOwner)}
      />
    );
  }

  return (
    <AgentScreenHost
      organizationId={organizationId}
      flow={flow}
      model={model}
      chrome={chromeFor("agent")}
      sphere={sphereFor("agent", setup.selectedRepo, model.githubOwner)}
    />
  );
}

function AgentScreenHost({
  organizationId,
  flow,
  model,
  chrome,
  sphere,
}: {
  organizationId: string;
  flow: FirstRunSetupFlow;
  model: OnboardingPageModel;
  chrome: FirstRunChrome;
  sphere?: FirstRunSphereProps;
}) {
  return (
    <FirstRunAgentScreen
      organizationId={organizationId}
      setup={model.setup}
      chrome={chrome}
      sphere={sphere}
      saving={flow.busy}
      loading={model.agentLoading}
      hostedAgentReady={model.hostedAgentReady}
      providerKeysOffered={model.bringYourOwnKey}
      showCustomProvider={model.customProvider}
      modelSource={{
        offered: modelSourceOffered(model),
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
      modelPicker={model.agentModelPicker}
      onContinue={() => {
        if (model.bringYourOwnKey && !flow.credentialChoice) flow.selectCredentialChoice("hosted");
        void flow.continueFromAgent();
      }}
    />
  );
}

function CompleteSetupScreen({
  flow,
  chrome,
  sphere,
}: {
  flow: FirstRunSetupFlow;
  chrome: FirstRunChrome;
  sphere?: FirstRunSphereProps;
}) {
  return (
    <FirstRunShell testId="first-run-complete" chrome={chrome} busy={flow.busy} sphere={sphere} visual="preview">
      <FirstRunHeading headline="Complete workspace setup">
        <p>
          {flow.cardStep.failed
            ? "SuperPlane could not verify billing. Try again."
            : flow.accessError
              ? "SuperPlane could not verify model access. Try again."
              : flow.cardStep.requiresAdmin
                ? "Ask an organization admin to select a billing plan before you complete workspace setup."
                : "Continue with the models available to your organization."}
        </p>
      </FirstRunHeading>
      <Button
        disabled={flow.busy || (!flow.cardStep.ready && !flow.cardStep.failed) || flow.cardStep.requiresAdmin}
        onClick={() => void (flow.cardStep.failed ? flow.cardStep.refetch() : flow.finishSetup())}
      >
        {flow.cardStep.failed ? "Try again" : "Continue"}
      </Button>
    </FirstRunShell>
  );
}
