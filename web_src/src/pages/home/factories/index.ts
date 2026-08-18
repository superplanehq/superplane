import type { InstallParam } from "@/pages/install/types";

import type { FactoryDefinition } from "./types";
import factoryMeta from "./software-factory/factory.json";
import factoryParams from "./software-factory/params.json";
import softwareFactoryCanvasYaml from "./software-factory/canvas.yaml?raw";
import softwareFactoryConsoleYaml from "./software-factory/console.yaml?raw";
import lineAppConsoleYaml from "./line-apps/console.yaml?raw";
import planningCanvasYaml from "./line-apps/planning.canvas.yaml?raw";
import implementationCanvasYaml from "./line-apps/implementation.canvas.yaml?raw";
import prCanvasYaml from "./line-apps/pr.canvas.yaml?raw";

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

function buildLineApp(args: {
  id: string;
  title: string;
  description: string;
  canvasYaml: string;
  entrypointNodeId: string;
}): FactoryDefinition {
  return {
    id: args.id,
    title: args.title,
    description: args.description,
    integrations: ["github", "claude"],
    componentIntegrations: LINE_APP_COMPONENT_INTEGRATIONS,
    startingTasks: [],
    // Onboarding never triggers the entrypoint directly; the factory line does.
    run: {
      nodeId: args.entrypointNodeId,
      hookName: "run",
      template: "",
      parameters: {},
    },
    source: { type: "bundled" },
    installParams: factoryParams.install_params as InstallParam[],
    canvasYaml: args.canvasYaml,
    consoleYaml: lineAppConsoleYaml,
  };
}

/**
 * Ordered factory-line apps provisioned during onboarding. Each entry maps to a
 * bundled app template and the line step that calls its onRun entrypoint.
 */
export interface OnboardingLineApp {
  factoryId: string;
  entrypointNodeId: string;
  lineStepName: string;
}

export const ONBOARDING_LINE_APPS: OnboardingLineApp[] = [
  { factoryId: "line-planning", entrypointNodeId: "onrun-create-plan", lineStepName: "Create Implementation Plan" },
  { factoryId: "line-implementation", entrypointNodeId: "onrun-implement", lineStepName: "Implement" },
  { factoryId: "line-pr", entrypointNodeId: "onrun-open-pr", lineStepName: "Open Pull Request" },
];

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
