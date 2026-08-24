import type { InstallParam } from "@/pages/install/types";

import type { FactoryDefinition } from "./types";
import factoryMeta from "./software-factory/factory.json";
import factoryParams from "./software-factory/params.json";
import softwareFactoryCanvasYaml from "./software-factory/canvas.yaml?raw";
import softwareFactoryConsoleYaml from "./software-factory/console.yaml?raw";
import lineAppConsoleYaml from "./line-apps/console.yaml?raw";
import eventAppConsoleYaml from "./line-apps/event-app.console.yaml?raw";
import planningCanvasYaml from "./line-apps/planning.canvas.yaml?raw";
import implementationCanvasYaml from "./line-apps/implementation.canvas.yaml?raw";
import prCanvasYaml from "./line-apps/pr.canvas.yaml?raw";
import prClosureCanvasYaml from "./line-apps/pr-closure.canvas.yaml?raw";
import issueIntakeCanvasYaml from "./line-apps/issue-intake.canvas.yaml?raw";
import issueIntakeConsoleYaml from "./line-apps/issue-intake.console.yaml?raw";
import sentryIssueIntakeCanvasYaml from "./line-apps/sentry-issue-intake.canvas.yaml?raw";
import sentryIssueIntakeConsoleYaml from "./line-apps/sentry-issue-intake.console.yaml?raw";

export type { FactoryDefinition, FactoryStartingTask, FactoryRunDefinition } from "./types";
export {
  buildFactoryRunParameters,
  materializeFactoryCanvas,
  materializeFactoryConsole,
  normalizeFactoryInstallParams,
  substituteInstallParams,
  wireFactoryIntegrations,
} from "./materializeFactoryTemplate";

function buildSoftwareFactory(): FactoryDefinition {
  return {
    id: factoryMeta.id,
    title: factoryMeta.title,
    description: factoryMeta.description,
    integrations: factoryMeta.integrations,
    componentIntegrations: factoryMeta.componentIntegrations,
    startingTasks: factoryMeta.startingTasks,
    agentSuggestions: factoryMeta.agentSuggestions,
    run: factoryMeta.run as FactoryDefinition["run"],
    source: factoryMeta.source as FactoryDefinition["source"],
    installParams: factoryParams.install_params as InstallParam[],
    canvasYaml: softwareFactoryCanvasYaml,
    consoleYaml: softwareFactoryConsoleYaml,
  };
}

// Onboarding provisions a factory line as separate, focused apps — one per
// phase — mirroring the production setup. Each app exposes a single onRun
// entrypoint that the line calls in order, passing the work order through.
const LINE_APP_COMPONENT_INTEGRATIONS: Record<string, string> = {
  "github.addIssueLabel": "github",
  "github.createIssueComment": "github",
  "github.createPullRequest": "github",
};

function buildOnboardingApp(args: {
  id: string;
  title: string;
  description: string;
  canvasYaml: string;
  consoleYaml: string;
  integrations: string[];
  componentIntegrations: Record<string, string>;
  entrypointNodeId: string;
  runTemplate?: string;
}): FactoryDefinition {
  return {
    id: args.id,
    title: args.title,
    description: args.description,
    integrations: args.integrations,
    componentIntegrations: args.componentIntegrations,
    startingTasks: [],
    // Onboarding never triggers the entrypoint. Apps that expose a manual
    // Start template name it here so the UI can run them on demand.
    run: {
      nodeId: args.entrypointNodeId,
      hookName: "run",
      template: args.runTemplate ?? "",
      parameters: {},
    },
    source: { type: "bundled" },
    installParams: factoryParams.install_params as InstallParam[],
    canvasYaml: args.canvasYaml,
    consoleYaml: args.consoleYaml,
  };
}

function buildLineApp(args: {
  id: string;
  title: string;
  description: string;
  canvasYaml: string;
  entrypointNodeId: string;
}): FactoryDefinition {
  return buildOnboardingApp({
    ...args,
    integrations: ["github", "claude"],
    componentIntegrations: LINE_APP_COMPONENT_INTEGRATIONS,
    consoleYaml: lineAppConsoleYaml,
  });
}

const EVENT_APP_COMPONENT_INTEGRATIONS: Record<string, string> = {
  "github.onIssue": "github",
  "github.onPullRequest": "github",
  "sentry.getIssue": "sentry",
  "sentry.onIssue": "sentry",
};

function buildEventApp(args: {
  id: string;
  title: string;
  description: string;
  canvasYaml: string;
  consoleYaml?: string;
  triggerNodeId: string;
  integrations?: string[];
  runTemplate?: string;
}): FactoryDefinition {
  return buildOnboardingApp({
    id: args.id,
    title: args.title,
    description: args.description,
    canvasYaml: args.canvasYaml,
    consoleYaml: args.consoleYaml ?? eventAppConsoleYaml,
    integrations: args.integrations ?? ["github"],
    componentIntegrations: EVENT_APP_COMPONENT_INTEGRATIONS,
    entrypointNodeId: args.triggerNodeId,
    runTemplate: args.runTemplate,
  });
}

/**
 * Ordered factory-line apps provisioned during onboarding. Each entry maps to a
 * bundled app template and the line step that calls its onRun entrypoint.
 */
export interface OnboardingLineApp {
  factoryId: string;
  entrypointNodeId: string;
}

export const ONBOARDING_LINE_APPS: OnboardingLineApp[] = [
  { factoryId: "line-planning", entrypointNodeId: "onrun-create-plan" },
  { factoryId: "line-implementation", entrypointNodeId: "onrun-implement" },
  { factoryId: "line-pr", entrypointNodeId: "onrun-open-pr" },
];

// PR closure is required by the delivery line. Ingestion apps are optional and
// are installed only when the user selects one on the post-setup page.
export const ONBOARDING_EVENT_APPS = ["pr-closure"] as const;

/** Bundled apps that turn external issues into draft work orders. */
export const INGESTION_FACTORY_ID = "issue-intake";
export const SENTRY_INGESTION_FACTORY_ID = "sentry-issue-intake";

const FACTORY_BY_ID: Record<string, FactoryDefinition> = {
  "software-factory": buildSoftwareFactory(),
  "line-planning": buildLineApp({
    id: "line-planning",
    title: "Planning",
    description: "Read the work order and write an implementation plan.",
    canvasYaml: planningCanvasYaml,
    entrypointNodeId: "onrun-create-plan",
  }),
  "line-implementation": buildLineApp({
    id: "line-implementation",
    title: "Implementation",
    description: "Create a branch and implement the plan with an agent.",
    canvasYaml: implementationCanvasYaml,
    entrypointNodeId: "onrun-implement",
  }),
  "line-pr": buildLineApp({
    id: "line-pr",
    title: "PR Creation",
    description: "Generate a pull request title and body, then open a draft PR.",
    canvasYaml: prCanvasYaml,
    entrypointNodeId: "onrun-open-pr",
  }),
  "issue-intake": buildEventApp({
    id: "issue-intake",
    title: "Issue Ingestion",
    description:
      "Create a draft work order from new issues and from a 10-minute backlog scan, then attach an implementation plan.",
    canvasYaml: issueIntakeCanvasYaml,
    consoleYaml: issueIntakeConsoleYaml,
    triggerNodeId: "manual-start",
    runTemplate: "Scan backlog now",
    integrations: ["github", "claude"],
  }),
  "sentry-issue-intake": buildEventApp({
    id: "sentry-issue-intake",
    title: "Sentry Issue Ingestion",
    description: "Create a draft work order with an implementation plan when Sentry reports a new issue.",
    canvasYaml: sentryIssueIntakeCanvasYaml,
    consoleYaml: sentryIssueIntakeConsoleYaml,
    triggerNodeId: "on-sentry-issue",
    integrations: ["sentry", "github", "claude"],
  }),
  "pr-closure": buildEventApp({
    id: "pr-closure",
    title: "PR Closure",
    description: "Close the work order when the attached pull request merges or is closed without a merge.",
    canvasYaml: prClosureCanvasYaml,
    triggerNodeId: "on-pr-closed",
  }),
};

export const DEFAULT_FACTORY_ID = "software-factory";

export function getFactoryDefinition(id: string = DEFAULT_FACTORY_ID): FactoryDefinition {
  const definition = FACTORY_BY_ID[id];
  if (!definition) {
    throw new Error(`Unknown factory definition: ${id}`);
  }
  return definition;
}

export function listFactoryDefinitions(): FactoryDefinition[] {
  return Object.values(FACTORY_BY_ID);
}
