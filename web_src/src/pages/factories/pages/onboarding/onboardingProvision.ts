import type {
  FactoryApp,
  FactoriesFactory,
  FactoriesFactoryLine,
  FactoriesUpdateFactoryOnboardingBody,
  FactoryLineStep,
} from "@/api-client";
import type { IntegrationSelections } from "@/pages/home/InstallIntegrationsSection";
import { getFactoryDefinition, ONBOARDING_EVENT_APPS, ONBOARDING_LINE_APPS } from "@/pages/home/factories";
import type { InstallFactoryInput } from "@/pages/home/useInstallFactory";

export const DEFAULT_LINE_NAME = "Software delivery";

const PRIMARY_LINE_APP_ENTRYPOINT = ONBOARDING_LINE_APPS[0].entrypointNodeId;

export type InstallOnboardingApp = (
  input: InstallFactoryInput,
) => Promise<{ canvasId: string; canvasName: string } | undefined>;

export type UpdateOnboarding = (input: FactoriesUpdateFactoryOnboardingBody) => Promise<unknown>;

export interface ProvisionedLine {
  lineId: string;
  primaryAppId: string;
}

type OnboardingEventAppId = (typeof ONBOARDING_EVENT_APPS)[number];

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
  });
  if (!installed?.canvasId) throw new Error(`Failed to create the ${args.appFactoryId} app`);
  return installed;
}

function isMatchingEventAppName(name: string | undefined, expectedName: string): boolean {
  if (!name) return false;
  if (name === expectedName) return true;
  const escapedName = expectedName.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  return new RegExp(`^${escapedName} \\(\\d+\\)$`).test(name);
}

function findProvisionedEventApp(
  apps: FactoryApp[],
  appFactoryId: OnboardingEventAppId,
): FactoryApp | undefined {
  const definition = getFactoryDefinition(appFactoryId);
  return apps.find(
    (app) =>
      app.description === definition.description &&
      isMatchingEventAppName(app.name ?? undefined, definition.title),
  );
}

// Install each bundled app in order and return the line steps that call them.
// installFactory clears its pending-canvas ref after each success, so the
// sequential calls create distinct canvases.
async function provisionLineApps(args: {
  factoryId: string;
  selections: IntegrationSelections;
  appRepository: string;
  backlogRepository: string;
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
  installFactory: InstallOnboardingApp;
  existingApps?: FactoryApp[];
  loadExistingApps?: () => Promise<FactoryApp[]>;
}): Promise<void> {
  const provisionedApps = [...(args.existingApps ?? (args.loadExistingApps ? await args.loadExistingApps() : []))];

  for (const appFactoryId of ONBOARDING_EVENT_APPS) {
    if (findProvisionedEventApp(provisionedApps, appFactoryId)) {
      continue;
    }

    const installed = await installOnboardingApp({
      factoryId: args.factoryId,
      appFactoryId,
      selections: args.selections,
      appRepository: args.appRepository,
      backlogRepository: args.backlogRepository,
      installFactory: args.installFactory,
    });

    provisionedApps.push({
      id: installed.canvasId,
      name: installed.canvasName,
      description: getFactoryDefinition(appFactoryId).description,
    });
  }
}

export async function provisionLine(args: {
  factory: FactoriesFactory | null;
  savedLineId?: string;
  savedAppId?: string;
  selections: IntegrationSelections;
  appRepository: string;
  backlogRepository: string;
  installFactory: InstallOnboardingApp;
  createLine: (input: { name: string; steps: FactoryLineStep[] }) => Promise<FactoriesFactoryLine>;
  updateOnboarding: UpdateOnboarding;
}): Promise<ProvisionedLine> {
  const existing = findProvisionedLine(args.factory);
  const lineId = args.savedLineId ?? existing?.id;
  if (lineId) {
    const primaryAppId = args.savedAppId ?? existing?.steps?.[0]?.app?.app;
    if (primaryAppId) return { lineId, primaryAppId };
  }

  const steps = await provisionLineApps({
    factoryId: args.factory?.id ?? "",
    selections: args.selections,
    appRepository: args.appRepository,
    backlogRepository: args.backlogRepository,
    installFactory: args.installFactory,
  });
  const primaryAppId = steps[0]?.app?.app;
  if (!primaryAppId) throw new Error("Software delivery apps were not created");

  const line = await args.createLine({ name: DEFAULT_LINE_NAME, steps });
  if (!line.id) throw new Error("Software delivery line was not created");
  await args.updateOnboarding({ provisionedAppId: primaryAppId, provisionedLineId: line.id });
  return { lineId: line.id, primaryAppId };
}
