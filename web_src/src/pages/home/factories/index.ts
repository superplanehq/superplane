import type { InstallParam } from "@/pages/install/types";

import type { FactoryDefinition } from "./types";
import factoryMeta from "./software-factory/factory.json";
import factoryParams from "./software-factory/params.json";
import softwareFactoryCanvasYaml from "./software-factory/canvas.yaml?raw";
import softwareFactoryConsoleYaml from "./software-factory/console.yaml?raw";

export type { FactoryDefinition, FactoryStartingTask, FactoryRunDefinition } from "./types";
export {
  buildFactoryRunParameters,
  materializeFactoryCanvas,
  materializeFactoryConsole,
  normalizeFactoryInstallParams,
  substituteInstallParams,
  wireFactoryIntegrations,
} from "./materializeFactoryTemplate";
export type { FactoryAgentRewrite } from "./materializeFactoryTemplate";

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
// entrypoint that the line calls in order, passing the task through.
const LINE_APP_COMPONENT_INTEGRATIONS: Record<string, string> = {
  "github.createIssueComment": "github",
  "github.createPullRequest": "github",
};

function buildOnboardingApp(args: {
  id: string;
  title: string;
  description: string;
  integrations: string[];
  componentIntegrations: Record<string, string>;
  entrypointNodeId: string;
}): FactoryDefinition {
  return {
    id: args.id,
    title: args.title,
    description: args.description,
    integrations: args.integrations,
    componentIntegrations: args.componentIntegrations,
    startingTasks: [],
    // Onboarding never triggers the entrypoint directly.
    run: {
      nodeId: args.entrypointNodeId,
      hookName: "run",
      template: "",
      parameters: {},
    },
    source: { type: "bundled" },
    installParams: factoryParams.install_params as InstallParam[],
  };
}

function buildLineApp(args: {
  id: string;
  title: string;
  description: string;
  entrypointNodeId: string;
}): FactoryDefinition {
  return buildOnboardingApp({
    ...args,
    integrations: ["github", "claude"],
    componentIntegrations: LINE_APP_COMPONENT_INTEGRATIONS,
  });
}

const EVENT_APP_COMPONENT_INTEGRATIONS: Record<string, string> = {
  "github.onIssue": "github",
  "github.onPullRequest": "github",
};

function buildEventApp(args: {
  id: string;
  title: string;
  description: string;
  triggerNodeId: string;
}): FactoryDefinition {
  return buildOnboardingApp({
    id: args.id,
    title: args.title,
    description: args.description,
    integrations: ["github"],
    componentIntegrations: EVENT_APP_COMPONENT_INTEGRATIONS,
    entrypointNodeId: args.triggerNodeId,
  });
}

/**
 * Ordered factory-line apps provisioned during onboarding. Each entry maps to a
 * bundled app template and the line step that calls its onRun entrypoint.
 * Implement opens the pull request and hands review to the wait note.
 */
export interface OnboardingLineApp {
  factoryId: string;
  entrypointNodeId: string;
}

export const ONBOARDING_LINE_APPS: OnboardingLineApp[] = [
  { factoryId: "line-planning", entrypointNodeId: "onrun-create-plan" },
  { factoryId: "line-implementation", entrypointNodeId: "onrun-implement" },
];

// Event-driven factory apps provisioned during onboarding. These listen for
// GitHub events; they are not factory line steps. Issue intake is not here: the
// workspace gets a first-class factory intake instead.
export const ONBOARDING_EVENT_APPS = ["pr-closure"] as const;

const FACTORY_BY_ID: Record<string, FactoryDefinition> = {
  "software-factory": buildSoftwareFactory(),
  "line-planning": buildLineApp({
    id: "line-planning",
    title: "Plan",
    description: "Read the task and write an implementation plan.",
    entrypointNodeId: "onrun-create-plan",
  }),
  "line-implementation": buildLineApp({
    id: "line-implementation",
    title: "Implement",
    description: "Create a branch, implement the plan, and open a pull request.",
    entrypointNodeId: "onrun-implement",
  }),
  "pr-closure": buildEventApp({
    id: "pr-closure",
    title: "PR Closure",
    description: "Close the task when the attached pull request merges or is closed without a merge.",
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
