import { cn } from "@/lib/utils";
import type { IntegrationInstanceSummary } from "@/pages/home/homeIntegrationStatus";
import { Check } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router";

import { useFactoriesLayout } from "../../layout/factoriesLayoutContext";
import { factoryOverviewPath } from "../../lib/factoryPagePaths";
import { AgentStep } from "./AgentStep";
import type { OnboardingRepo } from "./onboardingMocks";
import { AnalysisSidePanel } from "./AnalysisSidePanel";
import { DonePanel } from "./DonePanel";
import { Shell } from "./OnboardingShell";
import { WIZARD_STEPS, type IntegrationId, type VcsHostId, type WizardStepId } from "./onboardingFixtures";
import { IssuesStep, NameStep, RepositoryStep, StartStep, VcsStep } from "./onboardingSteps";
import { WizardStepFooter } from "./WizardStepFooter";
import { useConnectDialog } from "./useConnectDialog";
import { useOnboardingSetupState, type OnboardingSetupApi } from "./useOnboardingSetupState";
import { useOnboardingStorybook } from "./useOnboardingStorybook";

export type { WizardStepId };

const WIZARD_STEP_INDEX: Record<WizardStepId, number> = {
  vcs: 0,
  repo: 1,
  issues: 2,
  agent: 3,
  name: 4,
  start: 5,
};

function stepComplete(setup: OnboardingSetupApi, step: WizardStepId): boolean {
  switch (step) {
    case "vcs":
      return setup.vcsReady;
    case "repo":
      return setup.repoReady;
    case "issues":
      return setup.issuesReady;
    case "agent":
      return setup.agentReady;
    case "name":
      return setup.nameReady;
    case "start":
      return setup.startReady;
  }
}

function canAdvance(setup: OnboardingSetupApi, step: WizardStepId): boolean {
  switch (step) {
    case "vcs":
      return setup.vcsReady;
    case "repo":
      return setup.repoReady;
    case "issues":
      return setup.issuesReady;
    case "agent":
      return setup.agentReady;
    case "name":
      return setup.nameReady;
    case "start":
      return setup.canFinish;
  }
}

function onboardingReposFromSetup(selectedRepo: string | null, vcsHost: VcsHostId | null): OnboardingRepo[] {
  if (!selectedRepo || !vcsHost) {
    return [];
  }
  const [org, name] = selectedRepo.split("/");
  if (!org || !name) {
    return [];
  }
  return [{ id: `${vcsHost}-${org}-${name}`, name, org, provider: vcsHost }];
}

function WizardProgress({
  currentStep,
  furthestIndex,
  setup,
  onSelectStep,
}: {
  currentStep: WizardStepId;
  /** Highest step the user has opened. Later steps are not selectable yet. */
  furthestIndex: number;
  setup: OnboardingSetupApi;
  onSelectStep: (step: WizardStepId) => void;
}) {
  const currentIndex = WIZARD_STEP_INDEX[currentStep];

  return (
    <nav aria-label="Setup steps" className="overflow-x-auto">
      <ol className="flex items-center">
        {WIZARD_STEPS.map((step, index) => {
          const current = step.id === currentStep;
          // Only the steps behind the current one are done. An answer that is
          // already known (a connected agent, the placeholder name) must not
          // mark the step the user is on, or a step still ahead of it.
          const complete = index < currentIndex && stepComplete(setup, step.id);
          const reachable = index <= furthestIndex;

          return (
            <li key={step.id} className={cn("flex items-center", index > 0 && "min-w-4 flex-1")}>
              {index > 0 ? <span className="mx-2 h-px flex-1 bg-border" aria-hidden /> : null}
              <button
                type="button"
                disabled={!reachable}
                onClick={() => onSelectStep(step.id)}
                className={cn("flex shrink-0 items-center gap-2", !reachable && "cursor-not-allowed opacity-50")}
              >
                <span
                  className={cn(
                    "flex size-5 shrink-0 items-center justify-center rounded-full border text-[11px]",
                    complete
                      ? "border-emerald-600 bg-emerald-600 text-white"
                      : current
                        ? "border-foreground text-foreground"
                        : "border-border bg-background text-muted-foreground",
                  )}
                >
                  {complete ? <Check className="size-3" strokeWidth={2.5} aria-hidden /> : index + 1}
                </span>
                <span
                  className={cn(
                    "whitespace-nowrap text-[12px]",
                    current ? "font-medium text-foreground" : "text-muted-foreground",
                  )}
                >
                  {step.label}
                </span>
              </button>
            </li>
          );
        })}
      </ol>
    </nav>
  );
}

function WizardStepBody({
  step,
  setup,
  requestConnect,
  repos,
  onSelectRepository,
  githubConnections,
  selectedVcsConnectionId,
  onSelectVcsConnection,
  onCreateVcsConnection,
  onEditVcsConnection,
}: {
  step: WizardStepId;
  setup: OnboardingSetupApi;
  requestConnect: (id: IntegrationId) => void;
  repos?: string[];
  onSelectRepository: (repo: string) => void;
  githubConnections: IntegrationInstanceSummary;
  selectedVcsConnectionId?: string;
  onSelectVcsConnection: (id: string, name: string) => void;
  onCreateVcsConnection: () => void;
  onEditVcsConnection: () => void;
}) {
  switch (step) {
    case "vcs":
      return (
        <VcsStep
          github={githubConnections}
          selectedConnectionId={selectedVcsConnectionId}
          onSelectConnection={onSelectVcsConnection}
          onCreateConnection={onCreateVcsConnection}
        />
      );
    case "repo":
      return (
        <RepositoryStep
          setup={setup}
          repos={repos}
          onSelect={onSelectRepository}
          onEditConnection={onEditVcsConnection}
        />
      );
    case "issues":
      return <IssuesStep setup={setup} onRequestConnect={requestConnect} autoDiscover repos={repos} />;
    case "agent":
      return <AgentStep setup={setup} onRequestConnect={requestConnect} />;
    case "name":
      return <NameStep setup={setup} />;
    case "start":
      return <StartStep setup={setup} />;
  }
}

async function continueToStep(
  save: (() => Promise<boolean>) | undefined,
  step: WizardStepId,
  setStep: (step: WizardStepId) => void,
) {
  if (save && !(await save())) return;
  setStep(step);
}

// Going back must keep the later steps reachable from the rail.
function useFurthestStepIndex(currentIndex: number): number {
  const [furthestIndex, setFurthestIndex] = useState(currentIndex);
  useEffect(() => {
    setFurthestIndex((reached) => Math.max(reached, currentIndex));
  }, [currentIndex]);
  return Math.max(furthestIndex, currentIndex);
}

function useVcsStepNavigation(args: {
  setup: OnboardingSetupApi;
  selectedConnectionId?: string;
  selectConnection: (integrationId: string) => void;
  createConnection: () => void;
  setOpenSection: (id: WizardStepId) => void;
}) {
  const { setup, selectedConnectionId, selectConnection, createConnection: startCreate, setOpenSection } = args;
  const connectionBeforeCreate = useRef<string | null | undefined>(undefined);

  useEffect(() => {
    if (connectionBeforeCreate.current === undefined || !setup.vcsReady || !selectedConnectionId) return;
    if (selectedConnectionId === connectionBeforeCreate.current) return;
    connectionBeforeCreate.current = undefined;
    setOpenSection("repo");
  }, [selectedConnectionId, setOpenSection, setup.vcsReady]);

  const chooseConnection = (integrationId: string) => {
    setup.selectVcsHost("github");
    selectConnection(integrationId);
    setOpenSection("repo");
  };

  const createConnection = () => {
    setup.selectVcsHost("github");
    connectionBeforeCreate.current = selectedConnectionId ?? null;
    startCreate();
  };

  return { chooseConnection, createConnection };
}

function repositoryStepNavigation(args: {
  setup: OnboardingSetupApi;
  save?: (repository: string) => Promise<boolean>;
  setOpenSection: (id: WizardStepId) => void;
}) {
  const continueFromRepository = (repository: string) => {
    args.setup.commitRepoStep();
    const save = args.save;
    void continueToStep(save ? () => save(repository) : undefined, "issues", args.setOpenSection);
  };
  const selectRepository = (repository: string) => {
    args.setup.selectRepo(repository);
    continueFromRepository(repository);
  };
  return { continueFromRepository, selectRepository };
}

export function SetupSections({
  setup,
  openSection,
  setOpenSection,
  requestConnect,
  createVcsConnection,
  selectVcsConnection,
  githubConnections,
  selectedVcsConnectionId,
  requestConfigure,
  onFinish,
  onContinueName,
  onContinueRepo,
  onContinueIssues,
  repos,
  saving = false,
}: {
  setup: OnboardingSetupApi;
  openSection: WizardStepId;
  setOpenSection: (id: WizardStepId) => void;
  requestConnect: (id: IntegrationId) => void;
  createVcsConnection: () => void;
  selectVcsConnection: (integrationId: string) => void;
  githubConnections: IntegrationInstanceSummary;
  selectedVcsConnectionId?: string;
  /** Opens the connected VCS integration so the user can grant missing repositories. */
  requestConfigure?: () => void;
  onFinish: () => void | Promise<void>;
  onContinueName?: () => Promise<boolean>;
  onContinueRepo?: (repository: string) => Promise<boolean>;
  onContinueIssues?: () => Promise<boolean>;
  repos?: string[];
  saving?: boolean;
}) {
  const current = WIZARD_STEPS[WIZARD_STEP_INDEX[openSection]];
  const currentIndex = WIZARD_STEP_INDEX[openSection];
  const furthestIndex = useFurthestStepIndex(currentIndex);
  const previousStep = currentIndex > 0 ? WIZARD_STEPS[currentIndex - 1] : null;
  const nextStep = currentIndex < WIZARD_STEPS.length - 1 ? WIZARD_STEPS[currentIndex + 1] : null;
  const advanceEnabled = canAdvance(setup, openSection) && !saving;
  const vcsNavigation = useVcsStepNavigation({
    setup,
    selectedConnectionId: selectedVcsConnectionId,
    selectConnection: selectVcsConnection,
    createConnection: createVcsConnection,
    setOpenSection,
  });
  const repositoryNavigation = repositoryStepNavigation({ setup, save: onContinueRepo, setOpenSection });

  const editVcsConnection = () => {
    requestConfigure?.();
  };

  const goNext = () => {
    if (!nextStep) {
      void onFinish();
      return;
    }

    if (openSection === "name") {
      void continueToStep(onContinueName, nextStep.id, setOpenSection);
      return;
    }
    if (openSection === "repo") {
      if (setup.selectedRepo) repositoryNavigation.continueFromRepository(setup.selectedRepo);
      return;
    }
    if (openSection === "issues") {
      setup.commitIssuesStep();
      void continueToStep(onContinueIssues, nextStep.id, setOpenSection);
      return;
    }
    setOpenSection(nextStep.id);
  };

  const stepBody = (
    <WizardStepBody
      step={openSection}
      setup={setup}
      requestConnect={requestConnect}
      repos={repos}
      onSelectRepository={repositoryNavigation.selectRepository}
      githubConnections={githubConnections}
      selectedVcsConnectionId={selectedVcsConnectionId}
      onSelectVcsConnection={(integrationId) => vcsNavigation.chooseConnection(integrationId)}
      onCreateVcsConnection={vcsNavigation.createConnection}
      onEditVcsConnection={editVcsConnection}
    />
  );

  return (
    <div className="space-y-6">
      <WizardProgress
        currentStep={openSection}
        furthestIndex={furthestIndex}
        setup={setup}
        onSelectStep={setOpenSection}
      />

      {openSection === "vcs" ? (
        stepBody
      ) : (
        <section className="rounded-lg border border-border">
          {openSection !== "name" && (
            <div className="border-b border-border px-4 py-3">
              <h2 className="text-[15px] font-medium tracking-[-0.01em]">{current.label}</h2>
              <p className="mt-1 text-[13px] text-muted-foreground">{current.purpose}</p>
            </div>
          )}
          <div className="px-4 py-4">{stepBody}</div>
          <WizardStepFooter
            step={openSection}
            advanceEnabled={advanceEnabled}
            saving={saving}
            showBack={previousStep !== null}
            onBack={() => previousStep && setOpenSection(previousStep.id)}
            onNext={goNext}
          />
        </section>
      )}
    </div>
  );
}

/**
 * Storybook-only workspace setup: progressive wizard + setup.log side panel.
 * Connect uses the real IntegrationCreateDialog. Not mounted on production routes.
 */
export function OnboardingWireframe() {
  const navigate = useNavigate();
  const { organizationId, factoryId, factoryKey } = useFactoriesLayout();
  const onboarding = useOnboardingStorybook();
  const setup = useOnboardingSetupState(onboarding?.pending?.workspaceName ?? "");
  const { requestConnect, requestConfigure, dialog } = useConnectDialog(setup);
  const [openSection, setOpenSection] = useState<WizardStepId>("vcs");

  const finishSetup = () => {
    if (onboarding && factoryId) {
      onboarding.completeOnboarding(factoryId, onboardingReposFromSetup(setup.selectedRepo, setup.vcsHost));
    }
    if (organizationId && factoryKey) {
      navigate(factoryOverviewPath(organizationId, factoryKey), { replace: true });
      return;
    }
    setup.setFinished(true);
  };

  if (setup.finished) {
    return (
      <Shell className="w-full">
        <div className="mx-auto max-w-lg px-8 py-14">
          <DonePanel setup={setup} />
        </div>
        {dialog}
      </Shell>
    );
  }

  return (
    <Shell className="w-full">
      <div className="mx-auto w-full max-w-6xl px-6 py-8 lg:px-8">
        <div className="grid items-start gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(320px,400px)]">
          <SetupSections
            setup={setup}
            openSection={openSection}
            setOpenSection={setOpenSection}
            requestConnect={requestConnect}
            createVcsConnection={() => requestConnect("github")}
            selectVcsConnection={() => setup.selectVcsHost("github")}
            githubConnections={{ name: "github", allInstances: [], readyInstances: [] }}
            requestConfigure={() => {
              if (setup.vcsHost) requestConfigure(setup.vcsHost);
            }}
            onFinish={finishSetup}
          />
          <div className="lg:sticky lg:top-6 lg:self-start">
            <AnalysisSidePanel
              progress={{
                workspaceName: setup.workspaceName,
                nameReady: setup.nameReady,
                selectedRepo: setup.selectedRepo,
                vcsHost: setup.vcsHost,
                repoCommitted: setup.repoCommitted,
                issuesChoice: setup.issuesChoice,
                issuesCommitted: setup.issuesCommitted,
                agent: setup.agent,
                agentReady: setup.agentReady,
              }}
            />
          </div>
        </div>
      </div>
      {dialog}
    </Shell>
  );
}
