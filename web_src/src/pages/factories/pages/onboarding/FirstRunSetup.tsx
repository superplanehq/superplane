import { LoadingButton } from "@/components/ui/loading-button";
import { useAccount } from "@/contexts/useAccount";
import { useAccountOrganizations } from "@/hooks/useAccountOrganizations";
import { useMe } from "@/hooks/useMe";
import { organizationMatchesRoute } from "@/lib/accountOrganizations";
import { getApiErrorMessage } from "@/lib/errors";
import {
  hostedGitHubInstallRequested,
  hostedGitHubInstallRequestedAccount,
  type PendingGitHubInstallation,
} from "@/lib/hostedGitHubInstall";
import { useBindGitHubInstallation } from "@/hooks/useBindGitHubInstallation";
import { useRecheckGitHubInstallRequest } from "@/hooks/useRecheckGitHubInstallRequest";
import {
  githubAccountPickerFromConnection,
  pendingGitHubAccountPicker,
  type PendingGitHubAccountPicker,
} from "@/lib/startDirectGitHubConnect";
import {
  GITHUB_SETUP_ORG_PARAM,
  GITHUB_SETUP_REQUEST_PARAM,
  GITHUB_SETUP_REQUEST_VALUE,
} from "@/lib/integrationSetupReturn";
import { showErrorToast } from "@/lib/toast";
import { posthog } from "@/posthog";
import { useEffect, useRef, useState } from "react";
import { useSearchParams } from "react-router";

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
import { isWizardStepId } from "./onboardingStatus";
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

function initialFirstRunScreen(searchParams: URLSearchParams): FirstRunScreen {
  const requestedStep = searchParams.get("step");
  // A leftover githubSetup=request on a later step must not reopen Connect.
  if (isWizardStepId(requestedStep)) {
    return SCREEN_FOR_STEP[requestedStep];
  }
  if (searchParams.get(GITHUB_SETUP_REQUEST_PARAM) === GITHUB_SETUP_REQUEST_VALUE) {
    return "connect";
  }
  return "welcome";
}

function startConnectOnPicker(searchParams: URLSearchParams): boolean {
  const step = searchParams.get("step");
  if (step === "vcs") return true;
  return step === null && searchParams.get(GITHUB_SETUP_REQUEST_PARAM) === GITHUB_SETUP_REQUEST_VALUE;
}

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

// The reverse path walks the exact screens in reverse order, back to the
// welcome screen. The connect screen has two pages (the Connect GitHub page
// and the account picker), so `backActionFor` in FirstRunSetup handles the
// connect and choose screens itself.
const BACK_SCREEN: Partial<Record<FirstRunScreen, FirstRunScreen>> = {
  connect: "welcome",
  tickets: "choose",
  agent: "tickets",
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
 * Saves the connection the account picker bound, once the refreshed
 * connection list reports it ready. The bind callback runs before the list
 * re-renders, so an effect makes the save see the ready connection. The
 * repository screen opens only after the save, so a fast repository pick
 * cannot store the prior connection.
 */
function useSelectBoundGithubConnection(
  boundIntegrationId: string | null,
  clearBoundIntegrationId: () => void,
  model: OnboardingPageModel,
  onSelected: () => void,
) {
  const readyInstances = model.githubConnections.readyInstances;
  const selectConnection = model.selectVcsConnection;

  useEffect(() => {
    if (!boundIntegrationId) return;
    const bound = readyInstances.some((instance) => instance.metadata?.id === boundIntegrationId);
    if (!bound) return;
    clearBoundIntegrationId();
    void selectConnection(boundIntegrationId).then((saved) => {
      if (saved) onSelected();
    });
  }, [boundIntegrationId, clearBoundIntegrationId, readyInstances, selectConnection, onSelected]);
}

/**
 * The GitHub account picker reads the connection list from the cache. An
 * install or bind finished outside this document, or a browser Back, can
 * leave that list stale, so entering the connect screen refetches it.
 */
function useFreshConnectionsOnConnectScreen(screen: FirstRunScreen, refresh: () => Promise<unknown>) {
  useEffect(() => {
    if (screen !== "connect") return;
    void refresh();
  }, [screen, refresh]);
}

/**
 * The connect step is two pages: the Connect GitHub page with the connect
 * button, and the account picker. Get started always opens the connect
 * button page; only a GitHub round trip (or Back from the repository screen)
 * opens the picker. So the user picks the GitHub account on every forward
 * pass, and Back walks repository, picker, connect, welcome in order.
 */
function useConnectStage(
  startOnPicker: boolean,
  accountPicker: PendingGitHubAccountPicker | undefined,
  sourcesLoading: boolean,
) {
  const [pickerOpen, setPickerOpen] = useState(startOnPicker);
  return {
    pickerShowing: pickerOpen && Boolean(accountPicker),
    // A GitHub round trip reloads the page, so the picker data arrives after
    // the first render. The screen shows a placeholder until the connection
    // list settles, instead of flashing the connect button first.
    pickerLoading: pickerOpen && !accountPicker && sourcesLoading,
    setPickerOpen,
  };
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
  const { organizationId } = useFactoriesLayout();
  const { data: me, isPending: meLoading } = useMe(true, organizationId);
  const [searchParams] = useSearchParams();
  const setup = model.setup;
  const setOpenSection = model.setOpenSection;

  const [openedScreen, setOpenedScreen] = useState<FirstRunScreen>(() => initialFirstRunScreen(searchParams));
  const openStep = useRef(model.openSection);
  const skipAgentScreen = model.hostedAgentReady;

  useRepositoryErrorToast(model.repositoriesError);

  // Setup selects the connection GitHub returns with, then opens the next step.
  useEffect(() => {
    if (model.openSection === openStep.current) return;
    openStep.current = model.openSection;
    setOpenedScreen(SCREEN_FOR_STEP[model.openSection]);
  }, [model.openSection]);

  // Pass the /me user id, not account.id. startedByUserID is the SuperPlane
  // user. The /account id is the account, so a match would hide the picker.
  // A bound connection keeps its picker data, so Back from the repository
  // screen offers the accounts again instead of a dead connected state.
  const selectedConnection = model.githubConnections.readyInstances.find(
    (instance) => instance.metadata?.id === model.selectedVcsConnectionId,
  );
  const accountPicker =
    pendingGitHubAccountPicker(model.githubConnections.allInstances, me?.id) ??
    githubAccountPickerFromConnection(selectedConnection, me?.id);

  // Only a GitHub round trip or a waiting install request lands on the
  // account picker page. A fresh pass opens the Connect GitHub page.
  const stage = useConnectStage(
    startConnectOnPicker(searchParams),
    accountPicker,
    meLoading || model.githubConnectionsLoading,
  );

  const goToScreen = (next: FirstRunScreen, connectStage: "button" | "picker" = "button") => {
    if (next === "connect") {
      stage.setPickerOpen(connectStage === "picker");
    }
    const step = STEP_FOR_SCREEN[next];
    if (step) {
      // Keeps the provider return URL on the step the user is answering.
      openStep.current = step;
      setOpenSection(step);
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

  // An approved install request binds outside the wizard round trip, so the
  // waiting screen rechecks GitHub through the connection until it is ready.
  // The welcome screen stays first even while a request waits; Get started
  // opens the connect screen, which shows the waiting state.
  useRecheckGitHubInstallRequest(organizationId, model.githubConnections.allInstances);

  const githubOrganization =
    searchParams.get(GITHUB_SETUP_ORG_PARAM)?.trim() ||
    model.githubConnections.allInstances
      .map((instance) => hostedGitHubInstallRequestedAccount(instance.status?.metadata))
      .find((account) => account !== "") ||
    "";
  // Binding through a page redirect reloads the whole app and walks the user
  // through the connect screen again. Binding in place opens the repository
  // screen directly once the connection is ready.
  const bindInstallation = useBindGitHubInstallation(organizationId);
  const [boundIntegrationId, setBoundIntegrationId] = useState<string | null>(null);
  useSelectBoundGithubConnection(
    boundIntegrationId,
    () => setBoundIntegrationId(null),
    model,
    () => goToScreen("choose"),
  );
  const useInstallation = (installation: PendingGitHubInstallation) => {
    const state = accountPicker?.state;
    const pendingId = accountPicker?.id;
    if (!state || bindInstallation.isPending) return;
    bindInstallation.mutate(
      { state, installationId: installation.id },
      {
        onSuccess: () => {
          if (!pendingId) {
            goToScreen("choose");
            return;
          }
          setBoundIntegrationId(pendingId);
        },
        onError: (error) => showErrorToast(getApiErrorMessage(error, "Failed to connect the GitHub account")),
      },
    );
  };

  return {
    screen: screenWithoutAgent(openedScreen, skipAgentScreen),
    skipAgentScreen,
    installRequested,
    githubOrganization,
    accountPicker,
    pickerShowing: stage.pickerShowing,
    pickerLoading: stage.pickerLoading,
    closePicker: () => stage.setPickerOpen(false),
    bindingInstallationId: bindInstallation.isPending ? bindInstallation.variables?.installationId : undefined,
    useInstallation,
    goToScreen,
    continueFromRepository,
    continueFromTickets,
    selectTicketSource,
  };
}

type FirstRunFlow = ReturnType<typeof useFirstRunSetupFlow>;

/**
 * Back walks the exact screens in reverse order: repository, account picker,
 * Connect GitHub, welcome. The picker is a page of the connect screen, so
 * Back on the picker closes it instead of changing screens.
 */
function backActionFor(target: FirstRunScreen, flow: FirstRunFlow): (() => void) | undefined {
  if (target === "connect" && flow.pickerShowing) {
    return flow.closePicker;
  }
  if (target === "choose") {
    return () => flow.goToScreen("connect", "picker");
  }
  const backScreen = BACK_SCREEN[target];
  return backScreen ? () => flow.goToScreen(backScreen) : undefined;
}

/** Picker data for the connect screen. The Connect GitHub page passes none. */
function pickerPropsFor(flow: FirstRunFlow) {
  if (!flow.pickerShowing) {
    return {};
  }
  return {
    pendingInstallations: flow.accountPicker?.installations,
    githubState: flow.accountPicker?.state,
    githubAppSlug: flow.accountPicker?.appSlug,
    githubLogin: flow.accountPicker?.githubLogin,
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
  useFreshConnectionsOnConnectScreen(flow.screen, model.refreshGithubConnections);
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
      stepIndex: STEP_INDEX_FOR_SCREEN[target],
      stepCount: flow.skipAgentScreen ? FIRST_RUN_STEP_COUNT - 1 : FIRST_RUN_STEP_COUNT,
      onBack: backActionFor(target, flow),
    };
  };

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
        loading={flow.pickerLoading}
        installRequested={flow.installRequested}
        githubOrganization={flow.githubOrganization}
        {...pickerPropsFor(flow)}
        bindingInstallationId={flow.bindingInstallationId}
        chrome={chromeFor("connect")}
        onConnectGitHub={() => model.requestConnect("github")}
        onUseInstallation={flow.useInstallation}
      />
    );
  }

  if (flow.screen === "choose") {
    return (
      <FirstRunChooseScreen
        repositories={model.repositories}
        selectedRepository={setup.selectedRepo}
        loading={model.repositoriesLoading}
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
