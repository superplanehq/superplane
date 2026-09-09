import { useBindGitHubInstallation } from "@/hooks/useBindGitHubInstallation";
import { useMe } from "@/hooks/useMe";
import { useRecheckGitHubInstallRequest } from "@/hooks/useRecheckGitHubInstallRequest";
import { getApiErrorMessage } from "@/lib/errors";
import { hostedGitHubInstallURL, type PendingGitHubInstallation } from "@/lib/hostedGitHubInstall";
import {
  GITHUB_SETUP_INTEGRATION_PARAM,
  GITHUB_SETUP_ORG_PARAM,
  GITHUB_SETUP_REQUEST_PARAM,
  GITHUB_SETUP_REQUEST_VALUE,
} from "@/lib/integrationSetupReturn";
import {
  githubAccountPickerFromConnection,
  pendingGitHubAccountPicker,
  pendingGitHubRequestConnection,
  type PendingGitHubAccountPicker,
} from "@/lib/startDirectGitHubConnect";
import { showErrorToast } from "@/lib/toast";
import { useCallback, useEffect, useRef, useState } from "react";
import { useSearchParams } from "react-router";

import { useFactoriesLayout } from "../../layout/factoriesLayoutContext";
import type { FirstRunTicketSource } from "./first-run/firstRunTypes";
import { type IntegrationId, type IssuesChoiceId, type WizardStepId } from "./onboardingFixtures";
import { isWizardStepId } from "./onboardingStatus";
import type { useOnboardingPageModel } from "./useOnboardingPageModel";

export type OnboardingPageModel = ReturnType<typeof useOnboardingPageModel>;
export type FirstRunScreen = "welcome" | "connect" | "choose" | "tickets" | "agent";
type FirstRunBlockingAction =
  | "opening-github"
  | "binding-github"
  | "saving-github-connection"
  | "saving-repository"
  | "saving-ticket-source"
  | "finishing-setup";

export const DEFAULT_TICKET_SOURCE: FirstRunTicketSource = "github-issues";
const DEFAULT_ISSUES_CHOICE: IssuesChoiceId = "vcs";
const SCREEN_FOR_STEP: Record<WizardStepId, FirstRunScreen> = {
  vcs: "connect",
  repo: "choose",
  issues: "tickets",
  agent: "agent",
  name: "agent",
};
const STEP_FOR_SCREEN: Partial<Record<FirstRunScreen, WizardStepId>> = {
  connect: "vcs",
  choose: "repo",
  tickets: "issues",
  agent: "agent",
};

function initialFirstRunScreen(searchParams: URLSearchParams): FirstRunScreen {
  const requestedStep = searchParams.get("step");
  if (isWizardStepId(requestedStep)) return SCREEN_FOR_STEP[requestedStep];
  return searchParams.get(GITHUB_SETUP_REQUEST_PARAM) === GITHUB_SETUP_REQUEST_VALUE ? "connect" : "welcome";
}

function startConnectOnPicker(searchParams: URLSearchParams): boolean {
  const step = searchParams.get("step");
  if (step === "vcs") return true;
  return step === null && searchParams.get(GITHUB_SETUP_REQUEST_PARAM) === GITHUB_SETUP_REQUEST_VALUE;
}

function screenWithoutAgent(screen: FirstRunScreen, skipAgentScreen: boolean): FirstRunScreen {
  return screen === "agent" && skipAgentScreen ? "tickets" : screen;
}

function useFirstRunBlockingAction() {
  const lock = useRef(false);
  const [action, setAction] = useState<FirstRunBlockingAction | null>(null);
  const begin = useCallback((next: FirstRunBlockingAction): boolean => {
    if (lock.current) return false;
    lock.current = true;
    setAction(next);
    return true;
  }, []);
  const finish = useCallback(() => {
    lock.current = false;
    setAction(null);
  }, []);
  const run = useCallback(
    async (next: FirstRunBlockingAction, operation: () => Promise<void>) => {
      if (!begin(next)) return;
      try {
        await operation();
      } finally {
        finish();
      }
    },
    [begin, finish],
  );
  return { action, busy: action !== null, begin, finish, setAction, run };
}

function useGitHubConnectionState(model: OnboardingPageModel, organizationId: string) {
  const { data: me, isPending: meLoading } = useMe(true, organizationId);
  const [searchParams, setSearchParams] = useSearchParams();
  const callbackIntegrationId = searchParams.get(GITHUB_SETUP_INTEGRATION_PARAM)?.trim() || undefined;
  const resolved = resolveGitHubConnection(model, me?.id, callbackIntegrationId);
  const requestConnection = resolved.requestConnection;
  const callbackRequestPending = model.githubConnectionsLoading && hasRequestMarker(searchParams);

  useEffect(() => {
    if (model.githubConnectionsLoading || requestConnection || !hasRequestMarker(searchParams)) return;
    const next = new URLSearchParams(searchParams);
    next.delete(GITHUB_SETUP_REQUEST_PARAM);
    next.delete(GITHUB_SETUP_ORG_PARAM);
    next.delete(GITHUB_SETUP_INTEGRATION_PARAM);
    setSearchParams(next, { replace: true });
  }, [model.githubConnectionsLoading, requestConnection, searchParams, setSearchParams]);

  return {
    ...resolved,
    callbackIntegrationId,
    installRequested: Boolean(requestConnection) || callbackRequestPending,
    githubOrganizations: requestedOrganizations(requestConnection, callbackRequestPending, searchParams),
    sourcesLoading: meLoading || model.githubConnectionsLoading,
    startOnPicker: startConnectOnPicker(searchParams),
    initialScreen: initialFirstRunScreen(searchParams),
  };
}

function resolveGitHubConnection(model: OnboardingPageModel, userId?: string, callbackIntegrationId?: string) {
  const requestConnection = pendingGitHubRequestConnection(
    model.githubConnections.allInstances,
    userId,
    callbackIntegrationId,
  );
  const preferredId = requestConnection?.id ?? callbackIntegrationId;
  const preferredConnection = model.githubConnections.allInstances.find((item) => item.metadata?.id === preferredId);
  const selectedConnection = model.githubConnections.readyInstances.find(
    (item) => item.metadata?.id === model.selectedVcsConnectionId,
  );
  const accountPicker =
    githubAccountPickerFromConnection(preferredConnection, userId) ??
    pendingGitHubAccountPicker(model.githubConnections.allInstances, userId, requestConnection?.id) ??
    githubAccountPickerFromConnection(selectedConnection, userId);
  return { requestConnection, accountPicker };
}

function hasRequestMarker(searchParams: URLSearchParams): boolean {
  return searchParams.get(GITHUB_SETUP_REQUEST_PARAM) === GITHUB_SETUP_REQUEST_VALUE;
}

function requestedOrganizations(
  request: ReturnType<typeof pendingGitHubRequestConnection>,
  callbackRequestPending: boolean,
  searchParams: URLSearchParams,
): string[] {
  if (request) return request.requests.map((item) => item.accountLogin).filter(Boolean);
  if (!callbackRequestPending) return [];
  const account = searchParams.get(GITHUB_SETUP_ORG_PARAM)?.trim();
  return account ? [account] : [];
}

function useFirstRunNavigation(
  model: OnboardingPageModel,
  skipAgentScreen: boolean,
  connection: ReturnType<typeof useGitHubConnectionState>,
) {
  const [openedScreen, setOpenedScreen] = useState<FirstRunScreen>(connection.initialScreen);
  const [pickerOpen, setPickerOpen] = useState(connection.startOnPicker);
  const openStep = useRef(model.openSection);

  useEffect(() => {
    if (model.openSection === openStep.current) return;
    openStep.current = model.openSection;
    setOpenedScreen(SCREEN_FOR_STEP[model.openSection]);
  }, [model.openSection]);

  const goToScreen = (next: FirstRunScreen, connectStage: "button" | "picker" = "button") => {
    if (next === "connect") setPickerOpen(connectStage === "picker" || Boolean(connection.requestConnection));
    const step = STEP_FOR_SCREEN[next];
    if (step) {
      openStep.current = step;
      model.setOpenSection(step);
    }
    setOpenedScreen(next);
  };

  return {
    screen: screenWithoutAgent(openedScreen, skipAgentScreen),
    pickerShowing: pickerOpen && Boolean(connection.accountPicker),
    pickerLoading: pickerOpen && !connection.accountPicker && connection.sourcesLoading,
    closePicker: () => setPickerOpen(false),
    goToScreen,
  };
}

function waitForBrowserPaint(): Promise<void> {
  return new Promise((resolve) => window.requestAnimationFrame(() => resolve()));
}

function useFirstRunCommands(
  model: OnboardingPageModel,
  skipAgentScreen: boolean,
  connection: ReturnType<typeof useGitHubConnectionState>,
  navigation: ReturnType<typeof useFirstRunNavigation>,
  blocking: ReturnType<typeof useFirstRunBlockingAction>,
) {
  const continueFromRepository = () =>
    blocking.run("saving-repository", async () => {
      const repository = model.setup.selectedRepo;
      if (!repository) return;
      model.setup.commitRepoStep();
      if (await model.saveRepository(repository)) navigation.goToScreen("tickets");
    });
  const continueFromTickets = () =>
    blocking.run("saving-ticket-source", async () => {
      model.setup.setIssuesChoice(DEFAULT_ISSUES_CHOICE);
      model.setup.commitIssuesStep();
      if (!(await model.saveIssues(DEFAULT_ISSUES_CHOICE))) return;
      if (!skipAgentScreen) return navigation.goToScreen("agent");
      blocking.setAction("finishing-setup");
      await model.finish(DEFAULT_ISSUES_CHOICE);
    });
  const connectGitHub = () =>
    blocking.run("opening-github", async () => {
      await waitForBrowserPaint();
      await model.requestConnect("github", connection.requestConnection?.id ?? connection.callbackIntegrationId);
    });
  const finishSetup = () =>
    blocking.run("finishing-setup", async () => {
      await model.finish();
    });
  const installOnAnotherAccount = () => {
    const state = connection.accountPicker?.state;
    const slug = connection.accountPicker?.appSlug;
    if (!state || !slug) return Promise.resolve();
    return blocking.run("opening-github", async () => {
      await waitForBrowserPaint();
      window.location.assign(hostedGitHubInstallURL(slug, state));
    });
  };
  const selectTicketSource = (source: FirstRunTicketSource) => {
    if (source === DEFAULT_TICKET_SOURCE) model.setup.setIssuesChoice(DEFAULT_ISSUES_CHOICE);
  };
  return {
    connectGitHub,
    continueFromRepository,
    continueFromTickets,
    finishSetup,
    installOnAnotherAccount,
    selectTicketSource,
  };
}

function useSelectBoundGithubConnection(
  boundIntegrationId: string | null,
  model: OnboardingPageModel,
  onSelected: () => void,
  onFinished: () => void,
) {
  const selecting = useRef<string | null>(null);
  useEffect(() => {
    if (!boundIntegrationId || selecting.current === boundIntegrationId) return;
    if (!model.githubConnections.readyInstances.some((item) => item.metadata?.id === boundIntegrationId)) return;
    selecting.current = boundIntegrationId;
    void model
      .selectVcsConnection(boundIntegrationId)
      .then((saved) => {
        if (saved) onSelected();
      })
      .finally(() => {
        selecting.current = null;
        onFinished();
      });
  }, [boundIntegrationId, model, onFinished, onSelected]);
}

function useGitHubInstallationBinding(
  organizationId: string,
  model: OnboardingPageModel,
  accountPicker: PendingGitHubAccountPicker | undefined,
  goToScreen: (screen: FirstRunScreen) => void,
  blocking: ReturnType<typeof useFirstRunBlockingAction>,
) {
  const bindInstallation = useBindGitHubInstallation(organizationId);
  const [boundIntegrationId, setBoundIntegrationId] = useState<string | null>(null);
  const [bindingInstallationId, setBindingInstallationId] = useState<string>();
  const finish = () => {
    setBoundIntegrationId(null);
    setBindingInstallationId(undefined);
    blocking.finish();
  };
  useSelectBoundGithubConnection(
    boundIntegrationId,
    model,
    () => {
      blocking.setAction("saving-github-connection");
      goToScreen("choose");
    },
    finish,
  );

  const useInstallation = async (installation: PendingGitHubInstallation) => {
    if (!accountPicker?.state || !blocking.begin("binding-github")) return;
    setBindingInstallationId(installation.id);
    try {
      await bindInstallation.mutateAsync({ state: accountPicker.state, installationId: installation.id });
      if (accountPicker.id) return setBoundIntegrationId(accountPicker.id);
      goToScreen("choose");
      finish();
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to connect the GitHub account"));
      finish();
    }
  };
  return { bindingInstallationId, useInstallation };
}

function useRepositoryErrorToast(error: unknown) {
  const reported = useRef<unknown>(null);
  useEffect(() => {
    if (!error || reported.current === error) return;
    reported.current = error;
    showErrorToast(getApiErrorMessage(error, "Failed to load repositories"));
  }, [error]);
}

export function useFreshConnectionsOnConnectScreen(screen: FirstRunScreen, refresh: () => Promise<unknown>) {
  useEffect(() => {
    if (screen === "connect") void refresh();
  }, [screen, refresh]);
}

export function useFirstRunSetupFlow(model: OnboardingPageModel) {
  const { organizationId } = useFactoriesLayout();
  const skipAgentScreen = model.hostedAgentReady;
  const blocking = useFirstRunBlockingAction();
  const connection = useGitHubConnectionState(model, organizationId);
  const navigation = useFirstRunNavigation(model, skipAgentScreen, connection);
  const commands = useFirstRunCommands(model, skipAgentScreen, connection, navigation, blocking);
  const binding = useGitHubInstallationBinding(
    organizationId,
    model,
    connection.accountPicker,
    navigation.goToScreen,
    blocking,
  );
  useRepositoryErrorToast(model.repositoriesError);
  // Recheck while a request waits, and also while the picker is open: an
  // install request made on GitHub without a callback (for example when the
  // callback URL was unreachable) only surfaces through this sync.
  useRecheckGitHubInstallRequest(
    organizationId,
    connection.requestConnection?.id ?? connection.accountPicker?.id,
    navigation.screen === "connect" && (connection.installRequested || navigation.pickerShowing),
  );

  return {
    ...navigation,
    ...commands,
    ...binding,
    skipAgentScreen,
    installRequested: connection.installRequested,
    githubOrganizations: connection.githubOrganizations,
    requestIntegrationId: connection.requestConnection?.id ?? connection.callbackIntegrationId,
    accountPicker: connection.accountPicker,
    blockingAction: blocking.action,
    busy: blocking.busy || model.saving,
  };
}

export type FirstRunSetupFlow = ReturnType<typeof useFirstRunSetupFlow>;
export type { IntegrationId };
