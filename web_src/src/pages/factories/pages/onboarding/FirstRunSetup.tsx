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
import { FirstRunBillingScreen } from "./first-run/FirstRunBillingScreen";
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
  billing: 4,
  agent: 5,
};

const BACK_SCREEN: Partial<Record<FirstRunScreen, FirstRunScreen>> = {
  host: "welcome",
  connect: "host",
  choose: "connect",
  tickets: "choose",
  billing: "tickets",
  agent: "billing",
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
  const backScreen = target === "agent" && !flow.hasBillingStep ? "tickets" : BACK_SCREEN[target];
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
  return (
    <FirstRunTicketsScreen
      initialImportLimit={initialWorkspace ? 3 : undefined}
      skipInitialImport={model.setup.skipInitialImport}
      onSkipInitialImportChange={initialWorkspace ? undefined : model.setup.setSkipInitialImport}
      githubAvailable={flow.vcsHost !== "bitbucket"}
      ticketSource={flow.ticketSource}
      chrome={chrome}
      sphere={sphere}
      continueLabel={
        flow.hasBillingStep || !flow.skipAgentScreen
          ? FIRST_RUN_COPY.tickets.continue
          : !initialWorkspace && model.setup.skipInitialImport
            ? FIRST_RUN_COPY.finish.action
            : FIRST_RUN_COPY.tickets.analyze
      }
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
      stepIndex: STEP_INDEX_FOR_SCREEN[target] - Number(target === "agent" && !flow.hasBillingStep),
      stepCount: FIRST_RUN_STEP_COUNT - Number(flow.skipAgentScreen) + Number(flow.hasBillingStep),
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
          ...chromeFor("billing"),
          onBack: undefined,
          stepIndex: FIRST_RUN_STEP_COUNT - 1 - Number(flow.skipAgentScreen) + Number(flow.hasBillingStep),
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

  if (flow.screen === "billing") {
    return (
      <OnboardingPlanScreen
        flow={flow}
        organizationId={organizationId}
        chrome={chromeFor("billing")}
        sphere={sphereFor("billing", setup.selectedRepo, model.githubOwner)}
      />
    );
  }

  if (flow.screen === "tickets") {
    return (
      <TicketsScreenHost
        initialWorkspace={factory?.onboarding?.initial === true}
        flow={flow}
        model={model}
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
      saving={flow.busy}
      loading={model.agentLoading}
      hasBillingStep={false}
      hostedAgentReady={model.hostedAgentReady}
      showCustomProvider={model.customProvider}
      modelSource={{
        offered: model.bringYourOwnKey && model.hostedModelsAvailable,
        choice: flow.credentialChoice,
        onSelect: flow.selectCredentialChoice,
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

function OnboardingPlanScreen({
  flow,
  organizationId,
  chrome,
  sphere,
}: {
  flow: FirstRunSetupFlow;
  organizationId: string;
  chrome: FirstRunChrome;
  sphere?: FirstRunSphereProps;
}) {
  if (!flow.hasBillingStep)
    return (
      <FirstRunShell testId="first-run-complete" chrome={chrome} busy={flow.busy} visual="preview">
        <FirstRunHeading headline="Complete workspace setup">
          <p>
            {flow.accessError
              ? "SuperPlane could not verify model access. Try again."
              : "Continue with the models available to your organization."}
          </p>
        </FirstRunHeading>
        <Button disabled={flow.busy} onClick={() => void flow.finishSetup()}>
          Continue
        </Button>
      </FirstRunShell>
    );
  return (
    <FirstRunBillingScreen
      organizationId={organizationId}
      chrome={chrome}
      saving={flow.busy}
      sphere={sphere}
      onContinue={flow.finishSetup}
    />
  );
}
