import { Button } from "@/components/ui/button";
import { FIRST_RUN_STEP_COUNT, FirstRunHeading, FirstRunShell } from "./first-run/FirstRunShell";
import { useAccount } from "@/contexts/useAccount";
import { useAccountOrganizations } from "@/hooks/useAccountOrganizations";
import { organizationMatchesRoute } from "@/lib/accountOrganizations";
import { posthog } from "@/posthog";
import { useNavigate } from "react-router";

import { useFactoriesLayout } from "../../layout/factoriesLayoutContext";
import { FirstRunAgentScreen } from "./first-run/FirstRunAgentScreen";
import { FirstRunAnalysisHost } from "./first-run/FirstRunAnalysisHost";
import { FirstRunTicketsScreen } from "./first-run/FirstRunTicketsScreen";
import type { FirstRunChrome } from "./first-run/firstRunTypes";
import { FIRST_RUN_COPY } from "./first-run/firstRunCopy";
import { FirstRunVcsScreen } from "./first-run/FirstRunVcsScreen";
import { FirstRunWelcomeScreen } from "./first-run/FirstRunWelcomeScreen";
import type { FirstRunSphereProps } from "./first-run/FirstRunSpherePane";
import { sphereFor } from "./first-run/firstRunSphereFor";
import { ownerOfRepository, repositoriesInOrganization } from "./first-run/githubOrganizations";
import { FirstRunCardScreenHost } from "./FirstRunCardScreenHost";
import { afterOnboardingPath } from "./useFinishOnboarding";
import { useGitHubInstallationChecks } from "./useGitHubOnboarding";
import { isFirstRunVcsScreen } from "./firstRunScreens";
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
  "ghe-host": 1,
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
  "ghe-host": "host",
  choose: "connect",
  tickets: "choose",
  card: "tickets",
  complete: "tickets",
  agent: "tickets",
};

// Steps that this flow does not show. The card step is absent when billing
// is disabled or the organization already has a paid plan.
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
  // Bitbucket and GitHub Enterprise have no github.com connect screen.
  if (
    backScreen === "connect" &&
    (flow.vcsHost === "bitbucket" || flow.githubEnterprise || !flow.accountConnectionRequired)
  ) {
    return () => flow.goToScreen(flow.githubEnterprise ? "ghe-host" : "host");
  }
  return backScreen ? () => flow.goToScreen(backScreen) : undefined;
}

function watchesGitHubInstallation(flow: FirstRunSetupFlow, hasDestination: boolean): boolean {
  return flow.screen === "choose" && flow.vcsHost !== "bitbucket" && !flow.githubEnterprise && !hasDestination;
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
  useGitHubInstallationChecks(organizationId, watchesGitHubInstallation(flow, Boolean(destination)));
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

  if (isFirstRunVcsScreen(flow.screen)) {
    return (
      <FirstRunVcsScreen
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
