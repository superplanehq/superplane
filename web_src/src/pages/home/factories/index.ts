import type { FactoryDefinition, InstallParam } from "./types";
import factoryMeta from "./software-factory/factory.json";
import factoryParams from "./software-factory/params.json";
import softwareFactoryCanvasYaml from "./software-factory/canvas.yaml?raw";
import softwareFactoryConsoleYaml from "./software-factory/console.yaml?raw";

export type { FactoryDefinition, FactoryStartingTask, FactoryRunDefinition, InstallParam } from "./types";
export {
  buildFactoryRunParameters,
  factoryAppTemplateAgentFromRewrite,
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

// Onboarding provisions a factory line as focused apps. Each app exposes a
// single onRun entrypoint that the line calls in order, passing the task
// through.
export type FactoryVCSProvider = "github" | "bitbucket";

export function factoryVCSProvider(provider?: string): FactoryVCSProvider {
  return provider === "bitbucket" ? "bitbucket" : "github";
}

function lineAppComponentIntegrations(provider: FactoryVCSProvider): Record<string, string> {
  if (provider === "bitbucket") {
    return {
      "bitbucket.createPullRequest": "bitbucket",
      "bitbucket.createPullRequestComment": "bitbucket",
      "bitbucket.findPullRequest": "bitbucket",
      "bitbucket.updatePullRequest": "bitbucket",
    };
  }
  return {
    "github.createIssueComment": "github",
    "github.createPullRequest": "github",
  };
}

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

function buildLineApp(
  args: {
    id: string;
    title: string;
    description: string;
    entrypointNodeId: string;
  },
  provider: FactoryVCSProvider = "github",
): FactoryDefinition {
  return buildOnboardingApp({
    ...args,
    integrations: [provider, "claude"],
    componentIntegrations: lineAppComponentIntegrations(provider),
  });
}

function buildEventApp(
  args: {
    id: string;
    title: string;
    description: string;
    triggerNodeId: string;
  },
  provider: FactoryVCSProvider = "github",
): FactoryDefinition {
  return buildOnboardingApp({
    id: args.id,
    title: args.title,
    description: args.description,
    integrations: [provider],
    componentIntegrations:
      provider === "bitbucket"
        ? { "bitbucket.onPullRequest": "bitbucket" }
        : { "github.onIssue": "github", "github.onPullRequest": "github" },
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
  { factoryId: "line-implementation", entrypointNodeId: "onrun-implement" },
];

// Event-driven factory apps provisioned during onboarding. These listen for
// Git host events; they are not factory line steps. Issue intake is not here: the
// workspace gets a first-class factory intake instead.
export const ONBOARDING_EVENT_APPS = ["pr-closure", "risk-score"] as const;

// Bitbucket closure listens for merge and decline events.
export function onboardingEventAppsFor(vcsProvider?: string): readonly string[] {
  return factoryVCSProvider(vcsProvider) === "github" ? ONBOARDING_EVENT_APPS : ["pr-closure"];
}

const FACTORY_BY_ID: Record<string, FactoryDefinition> = {
  "software-factory": buildSoftwareFactory(),
  "line-implementation": buildLineApp({
    id: "line-implementation",
    title: "Implement",
    description: "Create a branch, implement the task, and open a pull request.",
    entrypointNodeId: "onrun-implement",
  }),
  "pr-closure": buildEventApp({
    id: "pr-closure",
    title: "PR Closure",
    description: "Close the task when the attached pull request merges or is closed without a merge.",
    triggerNodeId: "on-pr-closed",
  }),
  "risk-score": buildEventApp({
    id: "risk-score",
    title: "Merge confidence",
    description: "Score a pull request when it opens or updates.",
    triggerNodeId: "on-pr-risk",
  }),
};

export const DEFAULT_FACTORY_ID = "software-factory";

export function getFactoryDefinition(id: string = DEFAULT_FACTORY_ID, vcsProvider?: string): FactoryDefinition {
  const provider = factoryVCSProvider(vcsProvider);
  const definition = provider === "github" ? FACTORY_BY_ID[id] : factoryDefinitionForProvider(id, provider);
  if (!definition) {
    throw new Error(`Unknown factory definition: ${id}`);
  }
  return definition;
}

function factoryDefinitionForProvider(id: string, provider: FactoryVCSProvider): FactoryDefinition | undefined {
  if (id === "line-implementation") {
    return buildLineApp(
      {
        id: "line-implementation",
        title: "Implement",
        description: "Create a branch, implement the task, and open a pull request.",
        entrypointNodeId: "onrun-implement",
      },
      provider,
    );
  }
  if (id === "pr-closure") {
    return buildEventApp(
      {
        id,
        title: FACTORY_BY_ID[id].title,
        description: FACTORY_BY_ID[id].description,
        triggerNodeId: "on-pr-closed",
      },
      provider,
    );
  }
  return FACTORY_BY_ID[id];
}

export function listFactoryDefinitions(): FactoryDefinition[] {
  return Object.values(FACTORY_BY_ID);
}
