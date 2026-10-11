import type {
  FactoriesFactory,
  FactoriesFactoryIntake,
  FactoriesFactoryIntakeSettings,
  FactoriesFactoryIntakeSource,
  FactoriesFactoryLine,
  FactoriesUpdateFactoryOnboardingBody,
  FactoryAutomation,
  FactoryLineStep,
} from "@/api-client";
import type { IntegrationSelections } from "@/pages/home/InstallIntegrationsSection";
import {
  factoryVCSProvider,
  getFactoryDefinition,
  onboardingEventAppsFor,
  ONBOARDING_LINE_APPS,
  type FactoryAgentRewrite,
} from "@/pages/home/factories";
import type { InstallFactoryInput } from "@/pages/home/useInstallFactory";
import type { IssuesChoiceId } from "./onboardingFixtures";
import {
  forgetOnboardingIntakeBinding,
  onboardingIntakeBinding,
  rememberOnboardingIntakeBinding,
} from "./onboardingIntakeBinding";

export const DEFAULT_LINE_NAME = "implement";

export const GITHUB_INTAKE_SOURCE: FactoriesFactoryIntakeSource = "SOURCE_GITHUB_ISSUES";
export const JIRA_INTAKE_SOURCE: FactoriesFactoryIntakeSource = "SOURCE_JIRA_ISSUES";
export const LINEAR_INTAKE_SOURCE: FactoriesFactoryIntakeSource = "SOURCE_LINEAR_ISSUES";

const PRIMARY_LINE_APP_ENTRYPOINT = ONBOARDING_LINE_APPS[0].entrypointNodeId;

export type InstallOnboardingApp = (
  input: InstallFactoryInput,
) => Promise<{ canvasId: string; canvasName: string } | undefined>;

export type UpdateOnboarding = (input: FactoriesUpdateFactoryOnboardingBody) => Promise<unknown>;

export interface ProvisionedLine {
  lineId: string;
  primaryAppId: string;
}

// A finished line has one step per bundled app, each calling the app onRun
// entrypoint. Match on the first entrypoint to recover a line provisioned by an
// earlier, interrupted attempt.
function findProvisionedLine(factory: FactoriesFactory | null): FactoriesFactoryLine | undefined {
  return factory?.lines?.find((line) =>
    line.steps?.some((step) => step.app?.entrypoint === PRIMARY_LINE_APP_ENTRYPOINT),
  );
}

async function installOnboardingApp(args: {
  factoryId: string;
  appFactoryId: string;
  selections: IntegrationSelections;
  appRepository: string;
  backlogRepository: string;
  defaultBranch: string;
  agentRewrite?: FactoryAgentRewrite;
  vcsProvider?: string;
  installFactory: InstallOnboardingApp;
}): Promise<{ canvasId: string; canvasName: string }> {
  const installed = await args.installFactory({
    factoryId: args.appFactoryId,
    vcsProvider: args.vcsProvider,
    workspaceFactoryId: args.factoryId,
    integrations: args.selections,
    installParams: {
      appRepository: args.appRepository,
      backlogRepository: args.backlogRepository,
      defaultBranch: args.defaultBranch,
    },
    startingTaskPrompt: "",
    navigateOnComplete: false,
    startInitialRun: false,
    agentRewrite: args.agentRewrite,
  });
  if (!installed?.canvasId) throw new Error(`Failed to create the ${args.appFactoryId} app`);
  return installed;
}

// Install each bundled app in order and return the line steps that call them.
// installFactory clears its pending-canvas ref after each success, so the
// sequential calls create distinct canvases.
async function provisionLineApps(args: {
  factoryId: string;
  selections: IntegrationSelections;
  appRepository: string;
  backlogRepository: string;
  defaultBranch: string;
  agentRewrite?: FactoryAgentRewrite;
  vcsProvider?: string;
  installFactory: InstallOnboardingApp;
}): Promise<FactoryLineStep[]> {
  const steps: FactoryLineStep[] = [];
  for (const app of ONBOARDING_LINE_APPS) {
    const installed = await installOnboardingApp({
      factoryId: args.factoryId,
      appFactoryId: app.factoryId,
      selections: args.selections,
      appRepository: args.appRepository,
      backlogRepository: args.backlogRepository,
      defaultBranch: args.defaultBranch,
      agentRewrite: args.agentRewrite,
      vcsProvider: args.vcsProvider,
      installFactory: args.installFactory,
    });
    steps.push({
      type: "runApp",
      app: { app: installed.canvasId, entrypoint: app.entrypointNodeId },
    });
  }
  return steps;
}

export type ListFactoryApps = () => Promise<FactoryAutomation[]>;

function escapeRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

// installFactoryCanvas names a new canvas after the factory definition title,
// then appends " (N)" (uniqueCanvasName) to dodge a name collision. Match both
// forms so a retry recognizes a copy it created under either name.
function matchesEventAppTitle(appName: string | undefined, title: string): boolean {
  if (!appName) return false;
  if (appName === title) return true;
  return new RegExp(`^${escapeRegExp(title)} \\(\\d+\\)$`).test(appName);
}

// Event apps listen for GitHub events and are not factory line steps. Skip an
// app whose title already matches an app the workspace has, so a retry after
// a failed finish does not create a second copy. Known limitation: a
// user-renamed app (to something other than "<title>" or "<title> (N)") is
// not matched, so onboarding installs another copy — see ListFactoryApps,
// which does not return the source factory id needed for an exact match.
export async function provisionEventApps(args: {
  factoryId: string;
  selections: IntegrationSelections;
  appRepository: string;
  backlogRepository: string;
  defaultBranch: string;
  agentRewrite?: FactoryAgentRewrite;
  vcsProvider?: string;
  installFactory: InstallOnboardingApp;
  listApps: ListFactoryApps;
}): Promise<void> {
  const eventApps = onboardingEventAppsFor(args.vcsProvider);
  if (eventApps.length === 0) return;

  const apps = await args.listApps();
  for (const appFactoryId of eventApps) {
    const title = getFactoryDefinition(appFactoryId).title;
    if (apps.some((app) => matchesEventAppTitle(app.name, title))) {
      continue;
    }

    await installOnboardingApp({
      factoryId: args.factoryId,
      appFactoryId,
      selections: args.selections,
      appRepository: args.appRepository,
      backlogRepository: args.backlogRepository,
      defaultBranch: args.defaultBranch,
      agentRewrite: args.agentRewrite,
      vcsProvider: args.vcsProvider,
      installFactory: args.installFactory,
    });
  }
}

export type ListFactoryIntakes = () => Promise<FactoriesFactoryIntake[]>;

export type CreateFactoryIntake = (input: {
  source: FactoriesFactoryIntakeSource;
  integrationId?: string;
  resourceId?: string;
  settings?: FactoriesFactoryIntakeSettings;
}) => Promise<FactoriesFactoryIntake>;

export type DeleteFactoryIntake = (intakeId: string) => Promise<unknown>;

function isBacklogIntake(intake: FactoriesFactoryIntake): boolean {
  return (
    intake.source === GITHUB_INTAKE_SOURCE ||
    intake.source === JIRA_INTAKE_SOURCE ||
    intake.source === LINEAR_INTAKE_SOURCE
  );
}

function backlogSourceForChoice(choice: IssuesChoiceId | null): FactoriesFactoryIntakeSource {
  if (choice === "jira") return JIRA_INTAKE_SOURCE;
  if (choice === "linear") return LINEAR_INTAKE_SOURCE;
  return GITHUB_INTAKE_SOURCE;
}

function intakeBindingMatches(intake: FactoriesFactoryIntake, integrationId: string, resourceId: string): boolean {
  const binding = onboardingIntakeBinding(intake);
  return binding.integrationId === integrationId && binding.resourceId === resourceId;
}

async function removeProvisionedIntake(
  deleteIntake: DeleteFactoryIntake,
  intake: FactoriesFactoryIntake,
): Promise<void> {
  if (!intake.id) return;
  await deleteIntake(intake.id);
  forgetOnboardingIntakeBinding(intake.id);
}

// The GitHub intake opens a task for each matching issue. The Backlog
// canvas scores those tasks. The backend reads the connection and the
// backlog repository from the saved onboarding config, so this runs after the
// wizard choices are stored. A retried finish must not add a second copy.
export async function provisionGithubIntake(args: {
  listIntakes: ListFactoryIntakes;
  createIntake: CreateFactoryIntake;
}): Promise<FactoriesFactoryIntake> {
  const intakes = await args.listIntakes();
  const existing = intakes.find((intake) => intake.source === GITHUB_INTAKE_SOURCE);
  if (existing) {
    return existing;
  }

  return args.createIntake({ source: GITHUB_INTAKE_SOURCE });
}

export async function provisionJiraIntake(args: {
  listIntakes: ListFactoryIntakes;
  createIntake: CreateFactoryIntake;
  deleteIntake: DeleteFactoryIntake;
  integrationId: string;
  resourceId: string;
  settings?: FactoriesFactoryIntakeSettings;
}): Promise<FactoriesFactoryIntake> {
  const intakes = await args.listIntakes();
  const existing = intakes.find((intake) => intake.source === JIRA_INTAKE_SOURCE);
  if (existing && intakeBindingMatches(existing, args.integrationId, args.resourceId)) {
    return existing;
  }
  if (existing) {
    await removeProvisionedIntake(args.deleteIntake, existing);
  }

  const created = await args.createIntake({
    source: JIRA_INTAKE_SOURCE,
    integrationId: args.integrationId,
    resourceId: args.resourceId,
    ...(args.settings ? { settings: args.settings } : {}),
  });
  rememberOnboardingIntakeBinding(created.id, {
    integrationId: args.integrationId,
    resourceId: args.resourceId,
  });
  return created;
}

function linearIntakeResourceId(projectIds: string[]): string {
  return projectIds.join(",");
}

export async function provisionLinearIntake(args: {
  listIntakes: ListFactoryIntakes;
  createIntake: CreateFactoryIntake;
  deleteIntake: DeleteFactoryIntake;
  integrationId: string;
  projectIds: string[];
}): Promise<FactoriesFactoryIntake> {
  const resourceId = linearIntakeResourceId(args.projectIds);
  const intakes = await args.listIntakes();
  const existing = intakes.find((intake) => intake.source === LINEAR_INTAKE_SOURCE);
  if (existing && intakeBindingMatches(existing, args.integrationId, resourceId)) {
    return existing;
  }
  if (existing) {
    await removeProvisionedIntake(args.deleteIntake, existing);
  }

  const created = await args.createIntake({
    source: LINEAR_INTAKE_SOURCE,
    integrationId: args.integrationId,
    resourceId,
    settings: { linearProjectIds: args.projectIds },
  });
  rememberOnboardingIntakeBinding(created.id, {
    integrationId: args.integrationId,
    resourceId,
  });
  return created;
}

// Create the selected backlog intake and remove a leftover intake from an
// earlier failed finish, so analysis follows the source the user chose.
export async function provisionOnboardingIntake(args: {
  /** Workspace Git host. Empty means GitHub. */
  vcsProvider?: string;
  listIntakes: ListFactoryIntakes;
  createIntake: CreateFactoryIntake;
  deleteIntake: DeleteFactoryIntake;
  issuesChoice: IssuesChoiceId | null;
  jira?: {
    integrationId: string;
    projectId: string;
    settings?: FactoriesFactoryIntakeSettings;
  };
  linear?: {
    integrationId: string;
    projectIds: string[];
  };
}): Promise<FactoriesFactoryIntake | undefined> {
  const intakes = await args.listIntakes();
  const desiredSource = backlogSourceForChoice(args.issuesChoice);
  for (const intake of intakes) {
    if (!isBacklogIntake(intake) || intake.source === desiredSource) continue;
    await removeProvisionedIntake(args.deleteIntake, intake);
  }

  if (args.issuesChoice === "jira") {
    if (!args.jira?.integrationId || !args.jira.projectId) {
      throw new Error("Connect Jira, then choose a project.");
    }
    return provisionJiraIntake({
      listIntakes: args.listIntakes,
      createIntake: args.createIntake,
      deleteIntake: args.deleteIntake,
      integrationId: args.jira.integrationId,
      resourceId: args.jira.projectId,
      settings: args.jira.settings,
    });
  }
  if (args.issuesChoice === "linear") {
    if (!args.linear?.integrationId || args.linear.projectIds.length === 0) {
      throw new Error("Connect Linear, then choose a project.");
    }
    return provisionLinearIntake({
      listIntakes: args.listIntakes,
      createIntake: args.createIntake,
      deleteIntake: args.deleteIntake,
      integrationId: args.linear.integrationId,
      projectIds: args.linear.projectIds,
    });
  }
  // GitHub Issues needs a GitHub repository, so other hosts get no intake.
  if (factoryVCSProvider(args.vcsProvider) !== "github") return undefined;
  return provisionGithubIntake({
    listIntakes: args.listIntakes,
    createIntake: args.createIntake,
  });
}

export async function provisionLine(args: {
  factory: FactoriesFactory | null;
  savedLineId?: string;
  savedAppId?: string;
  selections: IntegrationSelections;
  appRepository: string;
  backlogRepository: string;
  defaultBranch: string;
  agentRewrite?: FactoryAgentRewrite;
  vcsProvider?: string;
  installFactory: InstallOnboardingApp;
  createLine: (input: { name: string; steps: FactoryLineStep[] }) => Promise<FactoriesFactoryLine>;
  updateOnboarding: UpdateOnboarding;
}): Promise<ProvisionedLine> {
  const existing = existingProvisionedLine(args.factory, args.savedLineId, args.savedAppId);
  if (existing) {
    return existing;
  }

  const steps = await provisionLineApps({
    factoryId: args.factory?.id ?? "",
    selections: args.selections,
    appRepository: args.appRepository,
    backlogRepository: args.backlogRepository,
    defaultBranch: args.defaultBranch,
    agentRewrite: args.agentRewrite,
    vcsProvider: args.vcsProvider,
    installFactory: args.installFactory,
  });
  const primaryAppId = steps[0]?.app?.app;
  if (!primaryAppId) throw new Error("Line apps were not created");

  const line = await args.createLine({ name: DEFAULT_LINE_NAME, steps });
  if (!line.id) throw new Error("Line was not created");
  await args.updateOnboarding({ provisionedAppId: primaryAppId, provisionedLineId: line.id });
  return { lineId: line.id, primaryAppId };
}

function existingProvisionedLine(
  factory: FactoriesFactory | null,
  savedLineId?: string,
  savedAppId?: string,
): ProvisionedLine | undefined {
  const existing = findProvisionedLine(factory);
  const lineId = savedLineId ?? existing?.id;
  const primaryAppId = savedAppId ?? existing?.steps?.[0]?.app?.app;
  if (lineId && primaryAppId) {
    return { lineId, primaryAppId };
  }
  return undefined;
}
