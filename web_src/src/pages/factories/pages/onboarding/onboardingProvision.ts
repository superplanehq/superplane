import type {
  FactoriesFactory,
  FactoriesFactoryIntake,
  FactoriesFactoryIntakeSource,
  FactoriesFactoryLine,
  FactoriesFactoryPrFeedbackHandler,
  FactoriesUpdateFactoryOnboardingBody,
  FactoryLineStep,
} from "@/api-client";
import type { IntegrationSelections } from "@/pages/home/InstallIntegrationsSection";
import { ONBOARDING_EVENT_APPS, ONBOARDING_LINE_APPS, type FactoryAgentRewrite } from "@/pages/home/factories";
import type { InstallFactoryInput } from "@/pages/home/useInstallFactory";

export const DEFAULT_LINE_NAME = "plan-and-implement";

export const GITHUB_INTAKE_SOURCE: FactoriesFactoryIntakeSource = "SOURCE_GITHUB_ISSUES";

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
  agentRewrite?: FactoryAgentRewrite;
  installFactory: InstallOnboardingApp;
}): Promise<{ canvasId: string; canvasName: string }> {
  const installed = await args.installFactory({
    factoryId: args.appFactoryId,
    workspaceFactoryId: args.factoryId,
    integrations: args.selections,
    installParams: {
      appRepository: args.appRepository,
      backlogRepository: args.backlogRepository,
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
  agentRewrite?: FactoryAgentRewrite;
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
      agentRewrite: args.agentRewrite,
      installFactory: args.installFactory,
    });
    steps.push({
      type: "runApp",
      app: { app: installed.canvasId, entrypoint: app.entrypointNodeId },
    });
  }
  return steps;
}

// Event apps listen for GitHub events and are not factory line steps. Install
// them even when the line already exists, so a retry after a failed finish
// still creates PR Closure.
export async function provisionEventApps(args: {
  factoryId: string;
  selections: IntegrationSelections;
  appRepository: string;
  backlogRepository: string;
  agentRewrite?: FactoryAgentRewrite;
  installFactory: InstallOnboardingApp;
}): Promise<void> {
  for (const appFactoryId of ONBOARDING_EVENT_APPS) {
    await installOnboardingApp({
      factoryId: args.factoryId,
      appFactoryId,
      selections: args.selections,
      appRepository: args.appRepository,
      backlogRepository: args.backlogRepository,
      agentRewrite: args.agentRewrite,
      installFactory: args.installFactory,
    });
  }
}

export type ListFactoryIntakes = () => Promise<FactoriesFactoryIntake[]>;

export type CreateFactoryIntake = (input: { source: FactoriesFactoryIntakeSource }) => Promise<FactoriesFactoryIntake>;

// The GitHub intake scores new issues and opens a work order for the ones it
// trusts. The backend reads the connection and the backlog repository from the
// saved onboarding config, so this runs after the wizard choices are stored. A
// retried finish must not add a second copy.
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

export type ListFactoryPRFeedbackHandlers = () => Promise<FactoriesFactoryPrFeedbackHandler[]>;

export type CreateFactoryPRFeedbackHandler = (input: {
  repository?: string;
}) => Promise<FactoriesFactoryPrFeedbackHandler>;

// The PR feedback handler addresses review comments after a work-order PR
// opens. Match by repository so a retried finish does not add a second copy.
export async function provisionPRFeedbackHandler(args: {
  listHandlers: ListFactoryPRFeedbackHandlers;
  createHandler: CreateFactoryPRFeedbackHandler;
  repository: string;
}): Promise<FactoriesFactoryPrFeedbackHandler> {
  const handlers = await args.listHandlers();
  const existing = handlers.find((handler) => handler.settings?.subject?.repository === args.repository);
  if (existing) {
    return existing;
  }

  return args.createHandler({ repository: args.repository });
}

export async function provisionLine(args: {
  factory: FactoriesFactory | null;
  savedLineId?: string;
  savedAppId?: string;
  selections: IntegrationSelections;
  appRepository: string;
  backlogRepository: string;
  agentRewrite?: FactoryAgentRewrite;
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
    agentRewrite: args.agentRewrite,
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
