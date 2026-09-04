import { LoadingButton } from "@/components/ui/loading-button";
import { useAccount } from "@/contexts/useAccount";
import { useAccountOrganizations } from "@/hooks/useAccountOrganizations";
import { useDeleteFactory } from "@/hooks/useFactoryData";
import { useMe } from "@/hooks/useMe";
import { organizationMatchesRoute, organizationRouteId } from "@/lib/accountOrganizations";
import { getApiErrorMessage } from "@/lib/errors";
import {
  hostedGitHubAppSlug,
  hostedGitHubInstallRequested,
  hostedGitHubInstallRequestedAccount,
} from "@/lib/hostedGitHubInstall";
import { pendingGitHubAccountPicker } from "@/lib/startDirectGitHubConnect";
import {
  GITHUB_SETUP_ORG_PARAM,
  GITHUB_SETUP_REQUEST_PARAM,
  GITHUB_SETUP_REQUEST_VALUE,
} from "@/lib/integrationSetupReturn";
import { showErrorToast } from "@/lib/toast";
import { posthog } from "@/posthog";
import { useEffect, useRef, useState } from "react";
import { useNavigate, useSearchParams } from "react-router";

import { factoryListPath } from "../../lib/factoryPagePaths";
import { useFactoriesLayout } from "../../layout/factoriesLayoutContext";
import { AgentStep } from "./AgentStep";
import { FirstRunChooseScreen } from "./first-run/FirstRunChooseScreen";
import { FirstRunConnectScreen } from "./first-run/FirstRunConnectScreen";
import { FIRST_RUN_STEP_COUNT, FirstRunHeading, FirstRunPanel, FirstRunShell } from "./first-run/FirstRunShell";
import { FirstRunTicketsScreen } from "./first-run/FirstRunTicketsScreen";
import type { FirstRunChrome, FirstRunTicketSource } from "./first-run/firstRunTypes";
import { FIRST_RUN_COPY } from "./first-run/firstRunCopy";
import { FirstRunWelcomeScreen } from "./first-run/FirstRunWelcomeScreen";
import { WIZARD_STEPS, type IntegrationId, type IssuesChoiceId, type WizardStepId } from "./onboardingFixtures";
import type { OnboardingSetupApi } from "./useOnboardingSetupState";
import type { useOnboardingPageModel } from "./useOnboardingPageModel";

type OnboardingPageModel = ReturnType<typeof useOnboardingPageModel>;

type FirstRunScreen = "welcome" | "connect" | "choose" | "tickets" | "agent";

const SCREEN_FOR_STEP: Record<WizardStepId, FirstRunScreen> = {
  vcs: "connect",
  repo: "choose",
  issues: "tickets",
  agent: "agent",
  // The first-run screens derive the workspace name from the repository, so the
  // last saved answer opens the coding agent screen.
  name: "agent",
};

const STEP_FOR_SCREEN: Partial<Record<FirstRunScreen, WizardStepId>> = {
  connect: "vcs",
  choose: "repo",
  tickets: "issues",
  agent: "agent",
};

const STEP_INDEX_FOR_SCREEN: Record<FirstRunScreen, number> = {
  welcome: 0,
  connect: 1,
  choose: 2,
  tickets: 3,
  agent: 4,
};

/**
 * Hosted credentials leave the agent screen with no question to ask, so the
 * ticket screen becomes the last screen and provisions the workspace.
 */
function screenWithoutAgent(screen: FirstRunScreen, skipAgentScreen: boolean): FirstRunScreen {
  if (screen === "agent" && skipAgentScreen) return "tickets";
  return screen;
}

/**
 * The first-run screens have no coding agent screen. This screen keeps the
 * wizard step copy, because provisioning needs a connected agent.
 */
const AGENT_STEP: { id: "agent"; label: string; purpose: string } = WIZARD_STEPS[3];

/**
 * GitHub Issues is the only source setup can connect, so the tickets screen
 * opens with it selected. Jira and Linear stay marked as coming soon.
 */
const DEFAULT_TICKET_SOURCE: FirstRunTicketSource = "github-issues";
const DEFAULT_ISSUES_CHOICE: IssuesChoiceId = "vcs";

function firstNameOf(name: string | undefined): string | undefined {
  const first = name?.trim().split(/\s+/)[0];
  return first || undefined;
}

function signOut() {
  posthog.reset();
  window.location.href = "/logout";
}

/**
 * Selects the connection when the organization has exactly one ready GitHub
 * connection. The first-run connect screen shows connection state only, so
 * setup must not wait for a choice the screen cannot offer.
 */
function useSingleGithubConnection(model: OnboardingPageModel) {
  const selected = model.selectedVcsConnectionId;
  const readyInstances = model.githubConnections.readyInstances;
  const selectConnection = model.selectVcsConnection;
  const attempted = useRef(false);

  useEffect(() => {
    if (attempted.current || selected || readyInstances.length !== 1) return;
    const integrationId = readyInstances[0]?.metadata?.id;
    if (!integrationId) return;
    attempted.current = true;
    selectConnection(integrationId);
  }, [readyInstances, selectConnection, selected]);
}

/** Reports a failed repository list, which the choose screen shows as empty. */
function useRepositoryErrorToast(error: unknown) {
  const reported = useRef<unknown>(null);

  useEffect(() => {
    if (!error || reported.current === error) return;
    reported.current = error;
    showErrorToast(getApiErrorMessage(error, "Failed to load repositories"));
  }, [error]);
}

function AgentScreen({
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

/**
 * Screen order and answer saving for workspace setup. The first-run screens
 * stay presentational, so this hook holds every step that talks to the API.
 */
function useFirstRunSetupFlow(model: OnboardingPageModel) {
  const { factory, organizationId } = useFactoriesLayout();
  const { data: me } = useMe(true, organizationId);
  const [searchParams] = useSearchParams();
  const setup = model.setup;

  const [openedScreen, setOpenedScreen] = useState<FirstRunScreen>(() => {
    const resumed = Boolean(factory?.onboarding?.vcsIntegrationId) || searchParams.get("step") !== null;
    return resumed ? SCREEN_FOR_STEP[model.openSection] : "welcome";
  });
  const openStep = useRef(model.openSection);
  const skipAgentScreen = model.hostedAgentReady;

  useSingleGithubConnection(model);
  useRepositoryErrorToast(model.repositoriesError);

  // Setup selects the connection GitHub returns with, then opens the next step.
  useEffect(() => {
    if (model.openSection === openStep.current) return;
    openStep.current = model.openSection;
    setOpenedScreen(SCREEN_FOR_STEP[model.openSection]);
  }, [model.openSection]);

  const goToScreen = (next: FirstRunScreen) => {
    const step = STEP_FOR_SCREEN[next];
    if (step) {
      // Keeps the provider return URL on the step the user is answering.
      openStep.current = step;
      model.setOpenSection(step);
    }
    setOpenedScreen(next);
  };

  const continueFromRepository = async () => {
    const repository = setup.selectedRepo;
    if (!repository) return;
    setup.commitRepoStep();
    if (!(await model.saveRepository(repository))) return;
    goToScreen("tickets");
  };

  const continueFromTickets = async () => {
    setup.setIssuesChoice(DEFAULT_ISSUES_CHOICE);
    setup.commitIssuesStep();
    if (!(await model.saveIssues(DEFAULT_ISSUES_CHOICE))) return;
    if (skipAgentScreen) {
      // The issues choice was just set above, in this same click; the setup
      // state captured when this render closed over `model.finish` still
      // holds the answer from before the click. Passing the answer here
      // keeps a single click from saving a stale, empty issues source over
      // the one `saveIssues` already stored.
      await model.finish(DEFAULT_ISSUES_CHOICE);
      return;
    }
    goToScreen("agent");
  };

  const selectTicketSource = (source: FirstRunTicketSource) => {
    // Jira and Linear are not connectable yet, so the screen shows them as
    // coming soon and reports GitHub Issues only.
    if (source !== DEFAULT_TICKET_SOURCE) return;
    setup.setIssuesChoice(DEFAULT_ISSUES_CHOICE);
  };

  const installRequested =
    searchParams.get(GITHUB_SETUP_REQUEST_PARAM) === GITHUB_SETUP_REQUEST_VALUE ||
    model.githubConnections.allInstances.some((instance) => hostedGitHubInstallRequested(instance.status?.metadata));
  const githubOrganization =
    searchParams.get(GITHUB_SETUP_ORG_PARAM)?.trim() ||
    model.githubConnections.allInstances
      .map((instance) => hostedGitHubInstallRequestedAccount(instance.status?.metadata))
      .find((account) => account !== "") ||
    "";
  // Pass the /me user id, not account.id. startedByUserID is the SuperPlane
  // user. The /account id is the account, so a match would hide the picker.
  const accountPicker = pendingGitHubAccountPicker(model.githubConnections.allInstances, me?.id);
  const githubAppSlug =
    accountPicker?.appSlug ||
    model.githubConnections.allInstances
      .map((instance) => hostedGitHubAppSlug(instance.status?.metadata))
      .find((slug) => slug !== "") ||
    "";

  return {
    screen: screenWithoutAgent(openedScreen, skipAgentScreen),
    skipAgentScreen,
    installRequested,
    githubOrganization,
    accountPicker,
    githubAppSlug,
    goToScreen,
    continueFromRepository,
    continueFromTickets,
    selectTicketSource,
  };
}

/**
 * Workspace setup, on the first-run screens. Each answer is saved through the
 * setup model. The last screen provisions the workspace and opens it.
 */
export function FirstRunSetup({ model }: { model: OnboardingPageModel }) {
  const { account } = useAccount();
  const { organizationId, factoryId, factories } = useFactoriesLayout();
  const flow = useFirstRunSetupFlow(model);
  const setup = model.setup;
  const navigate = useNavigate();
  const deleteFactory = useDeleteFactory(organizationId);
  const accountOrganizations = useAccountOrganizations();

  // The placeholder workspace under setup is itself in `factories`, so
  // another workspace exists when any factory has a different id.
  const hasOtherWorkspace = factories.some((existing) => existing.id !== factoryId);
  // The user can also belong to organizations outside the current one. Those
  // give the user somewhere to go even when this organization has no other
  // workspace yet.
  const otherOrganizations = (accountOrganizations.data ?? []).filter(
    (organization) => !organizationMatchesRoute(organization, organizationId),
  );
  const canExitSetup = hasOtherWorkspace || otherOrganizations.length > 0;

  const cancelSetup = async () => {
    // Guards against a double delete from a second click while the mutation
    // is already in flight.
    if (deleteFactory.isPending) return;
    await deleteFactory.mutateAsync(factoryId);
    // Cancelling out of the last workspace in this organization must not
    // bounce the user back into onboarding for it, so it sends them to
    // another organization instead of this one's (now onboarding) workspace list.
    if (hasOtherWorkspace) {
      navigate(factoryListPath(organizationId));
    } else {
      navigate(`/${organizationRouteId(otherOrganizations[0])}`);
    }
  };

  const chromeFor = (target: FirstRunScreen): FirstRunChrome => ({
    displayName: firstNameOf(account?.name),
    email: account?.email,
    onLogOut: canExitSetup ? undefined : signOut,
    onCancel: canExitSetup ? () => void cancelSetup() : undefined,
    stepIndex: STEP_INDEX_FOR_SCREEN[target],
    stepCount: flow.skipAgentScreen ? FIRST_RUN_STEP_COUNT - 1 : FIRST_RUN_STEP_COUNT,
  });

  if (flow.screen === "welcome") {
    return (
      <FirstRunWelcomeScreen
        firstName={firstNameOf(account?.name)}
        chrome={chromeFor("welcome")}
        onGetStarted={() => flow.goToScreen("connect")}
      />
    );
  }

  if (flow.screen === "connect") {
    return (
      <FirstRunConnectScreen
        githubConnected={setup.vcsReady}
        installRequested={flow.installRequested}
        githubOrganization={flow.githubOrganization}
        pendingInstallations={flow.accountPicker?.installations}
        githubState={flow.accountPicker?.state}
        githubAppSlug={flow.githubAppSlug}
        chrome={chromeFor("connect")}
        onConnectGitHub={() => model.requestConnect("github")}
        onContinue={() => flow.goToScreen("choose")}
      />
    );
  }

  if (flow.screen === "choose") {
    return (
      <FirstRunChooseScreen
        repositories={model.repositories}
        selectedRepository={setup.selectedRepo}
        chrome={chromeFor("choose")}
        onSelectRepository={setup.selectRepo}
        onEditConnection={() => model.requestConfigure()}
        onContinue={() => void flow.continueFromRepository()}
      />
    );
  }

  if (flow.screen === "tickets") {
    return (
      <FirstRunTicketsScreen
        ticketSource={DEFAULT_TICKET_SOURCE}
        chrome={chromeFor("tickets")}
        continueLabel={flow.skipAgentScreen ? FIRST_RUN_COPY.tickets.analyze : FIRST_RUN_COPY.tickets.continue}
        saving={flow.skipAgentScreen && model.saving}
        onSelectTicketSource={flow.selectTicketSource}
        onAnalyzeTickets={() => void flow.continueFromTickets()}
      />
    );
  }

  return (
    <AgentScreen
      organizationId={organizationId}
      setup={setup}
      chrome={chromeFor("agent")}
      saving={model.saving}
      onRequestConnect={model.requestConnect}
      onContinue={() => void model.finish()}
    />
  );
}
