import type {
  CanvasesCanvas,
  FactoriesFactory,
  FactoriesFactoryLine,
  FactoriesUpdateFactoryOnboardingBody,
  FactoryLineStep,
} from "@/api-client";
import type { IntegrationSelections } from "@/pages/home/InstallIntegrationsSection";
import { ONBOARDING_EVENT_APPS, ONBOARDING_LINE_APPS } from "@/pages/home/factories";
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
type CanvasNode = NonNullable<NonNullable<CanvasesCanvas["spec"]>["nodes"]>[number];

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

function readNodeConfiguration(node: CanvasNode): Record<string, unknown> {
  const configuration = node.configuration;
  return configuration && typeof configuration === "object" ? configuration : {};
}

function hasAction(nodes: CanvasNode[], component: string, nodeId: string): boolean {
  return nodes.some((node) => node.component === component && node.id === nodeId);
}

function hasTriggerAction(nodes: CanvasNode[], nodeId: string, expectedAction: string): boolean {
  return nodes.some((node) => {
    if (node.component !== "github.onIssue" && node.component !== "github.onPullRequest") {
      return false;
    }
    if (node.id !== nodeId) {
      return false;
    }
    const actions = readNodeConfiguration(node).actions;
    return Array.isArray(actions) && actions.includes(expectedAction);
  });
}

function isIssueIntakeCanvas(canvas: CanvasesCanvas): boolean {
  const nodes = canvas.spec?.nodes ?? [];
  return (
    hasTriggerAction(nodes, "on-issue-labeled", "labeled") &&
    hasTriggerAction(nodes, "on-issue-assigned", "assigned") &&
    hasAction(nodes, "findWorkOrder", "find-existing-work-order") &&
    hasAction(nodes, "createWorkOrder", "create-work-order")
  );
}

function isPrClosureCanvas(canvas: CanvasesCanvas): boolean {
  const nodes = canvas.spec?.nodes ?? [];
  return (
    hasTriggerAction(nodes, "on-pr-closed", "closed") &&
    hasAction(nodes, "findWorkOrder", "find-work-order") &&
    hasAction(nodes, "updateWorkOrderArtifact", "stamp-pr-merged") &&
    hasAction(nodes, "updateWorkOrderArtifact", "stamp-pr-closed") &&
    hasAction(nodes, "updateWorkOrderStatus", "complete-work-order") &&
    hasAction(nodes, "updateWorkOrderStatus", "reject-work-order")
  );
}

export function identifyProvisionedEventApp(canvas: CanvasesCanvas): OnboardingEventAppId | undefined {
  if (isIssueIntakeCanvas(canvas)) {
    return "issue-intake";
  }
  if (isPrClosureCanvas(canvas)) {
    return "pr-closure";
  }
  return undefined;
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
  existingAppFactoryIds?: Iterable<OnboardingEventAppId>;
  loadExistingAppFactoryIds?: () => Promise<Set<OnboardingEventAppId>>;
}): Promise<void> {
  const provisionedAppFactoryIds = new Set<OnboardingEventAppId>(args.existingAppFactoryIds ?? []);
  if (args.loadExistingAppFactoryIds) {
    for (const appFactoryId of await args.loadExistingAppFactoryIds()) {
      provisionedAppFactoryIds.add(appFactoryId);
    }
  }

  for (const appFactoryId of ONBOARDING_EVENT_APPS) {
    if (provisionedAppFactoryIds.has(appFactoryId)) {
      continue;
    }

    await installOnboardingApp({
      factoryId: args.factoryId,
      appFactoryId,
      selections: args.selections,
      appRepository: args.appRepository,
      backlogRepository: args.backlogRepository,
      installFactory: args.installFactory,
    });
    provisionedAppFactoryIds.add(appFactoryId);
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
