import { LoadingButton } from "@/components/ui/loading-button";
import { useAccount } from "@/contexts/useAccount";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";
import { posthog } from "@/posthog";
import { useEffect, useRef, useState } from "react";
import { useSearchParams } from "react-router";

import { useFactoriesLayout } from "../../layout/factoriesLayoutContext";
import { AgentStep } from "./AgentStep";
import { FirstRunChooseScreen } from "./first-run/FirstRunChooseScreen";
import { FirstRunConnectScreen } from "./first-run/FirstRunConnectScreen";
import { FirstRunHeading, FirstRunPanel, FirstRunShell } from "./first-run/FirstRunShell";
import { FirstRunTicketsScreen } from "./first-run/FirstRunTicketsScreen";
import type { FirstRunChrome, FirstRunTicketSource } from "./first-run/firstRunTypes";
import { FIRST_RUN_COPY } from "./first-run/firstRunCopy";
import { FirstRunWelcomeScreen } from "./first-run/FirstRunWelcomeScreen";
import { WIZARD_STEPS, type IntegrationId, type WizardStepId } from "./onboardingFixtures";
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
          loadingText="Finishing setup..."
          onClick={onContinue}
          data-testid="first-run-finish-setup"
        >
          Finish setup
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
  const { factory } = useFactoriesLayout();
  const [searchParams] = useSearchParams();
  const setup = model.setup;

  const [screen, setScreen] = useState<FirstRunScreen>(() => {
    const resumed = Boolean(factory?.onboarding?.vcsIntegrationId) || searchParams.get("step") !== null;
    return resumed ? SCREEN_FOR_STEP[model.openSection] : "welcome";
  });
  const openStep = useRef(model.openSection);

  useSingleGithubConnection(model);
  useRepositoryErrorToast(model.repositoriesError);

  // Setup selects the connection GitHub returns with, then opens the next step.
  useEffect(() => {
    if (model.openSection === openStep.current) return;
    openStep.current = model.openSection;
    setScreen(SCREEN_FOR_STEP[model.openSection]);
  }, [model.openSection]);

  const goToScreen = (next: FirstRunScreen) => {
    const step = STEP_FOR_SCREEN[next];
    if (step) {
      // Keeps the provider return URL on the step the user is answering.
      openStep.current = step;
      model.setOpenSection(step);
    }
    setScreen(next);
  };

  const continueFromRepository = async () => {
    const repository = setup.selectedRepo;
    if (!repository) return;
    setup.commitRepoStep();
    if (!(await model.saveRepository(repository))) return;
    goToScreen("tickets");
  };

  const continueFromTickets = async () => {
    setup.commitIssuesStep();
    if (!(await model.saveIssues())) return;
    goToScreen("agent");
  };

  const selectTicketSource = (source: FirstRunTicketSource) => {
    // Jira and Linear are not connectable yet, so the screen shows them as
    // coming soon and reports GitHub Issues only.
    if (source !== "github-issues") return;
    setup.setIssuesChoice("vcs");
  };

  return {
    screen,
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
  const { organizationId } = useFactoriesLayout();
  const flow = useFirstRunSetupFlow(model);
  const setup = model.setup;

  const chromeFor = (target: FirstRunScreen): FirstRunChrome => ({
    displayName: firstNameOf(account?.name),
    email: account?.email,
    onLogOut: signOut,
    stepIndex: STEP_INDEX_FOR_SCREEN[target],
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
        ticketSource={setup.issuesChoice === "vcs" ? "github-issues" : null}
        chrome={chromeFor("tickets")}
        continueLabel={FIRST_RUN_COPY.tickets.continue}
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
