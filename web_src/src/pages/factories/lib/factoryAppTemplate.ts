import type { CanvasesCanvas } from "@/api-client";
import type { FactoryAgentRewrite } from "@/pages/home/factories";
import { DEFAULT_FACTORY_ID, listFactoryDefinitions, type FactoryDefinition } from "@/pages/home/factories";
import type { IntegrationSelections } from "@/pages/home/InstallIntegrationsSection";

import { primaryAgentNode, type CanvasSpecNode } from "./columnCanvasAgent";

/**
 * Matches an installed factory app's live canvas to the bundled line/event
 * app definition it was created from. Canvas records do not persist a
 * template id, so the match is structural: every bundled app keeps a stable
 * entrypoint node id, and `materializeFactoryCanvas` never renames node ids.
 * Returns null when nothing matches, including the software-factory canvas
 * itself, which reset does not support.
 */
export function resolveFactoryAppTemplate(canvas: CanvasesCanvas | null | undefined): FactoryDefinition | null {
  const nodeIds = new Set((canvas?.spec?.nodes ?? []).map((node) => node.id).filter((id): id is string => Boolean(id)));
  if (nodeIds.size === 0) return null;

  return (
    listFactoryDefinitions().find(
      (definition) => definition.id !== DEFAULT_FACTORY_ID && nodeIds.has(definition.run.nodeId),
    ) ?? null
  );
}

export type FactoryAppResetWiring = {
  installParams: Record<string, string>;
  integrations: IntegrationSelections;
  agentRewrite?: FactoryAgentRewrite;
};

const REPO_ENV_VAR = "REPO";
const DEFAULT_BRANCH_ENV_VAR = "BASE";

function findEnvValue(nodes: CanvasSpecNode[], name: string): string | undefined {
  for (const node of nodes) {
    const environment = node.configuration?.environment;
    if (!Array.isArray(environment)) continue;
    for (const entry of environment) {
      if (!entry || typeof entry !== "object") continue;
      const { name: entryName, value } = entry as { name?: unknown; value?: unknown };
      if (entryName === name && typeof value === "string" && value) {
        return value;
      }
    }
  }
  return undefined;
}

function findConfigStringField(nodes: CanvasSpecNode[], field: string): string | undefined {
  for (const node of nodes) {
    const value = node.configuration?.[field];
    if (typeof value === "string" && value) return value;
  }
  return undefined;
}

/**
 * Bundled templates only ever substitute `appRepository` and `defaultBranch`
 * (never `backlogRepository`), reading either a literal `repository`/`base`
 * field already set by a previous materialize, or the agent runner's
 * `REPO`/`BASE` environment values. Leaving a param out lets
 * `materializeFactoryCanvas` apply its own default (e.g. `defaultBranch`
 * falls back to "main").
 */
function deriveInstallParams(nodes: CanvasSpecNode[]): Record<string, string> {
  const appRepository = findConfigStringField(nodes, "repository") ?? findEnvValue(nodes, REPO_ENV_VAR);
  const defaultBranch = findConfigStringField(nodes, "base") ?? findEnvValue(nodes, DEFAULT_BRANCH_ENV_VAR);

  const params: Record<string, string> = {};
  if (appRepository) params.appRepository = appRepository;
  if (defaultBranch) params.defaultBranch = defaultBranch;
  return params;
}

/**
 * Github-wired nodes (the only integration `materializeFactoryCanvas` sets on
 * the node itself, via `componentIntegrations`) already carry the real
 * installation id/name. Reuse whichever one is already live.
 */
function deriveGithubSelection(nodes: CanvasSpecNode[]): IntegrationSelections {
  for (const node of nodes) {
    const { id, name } = node.integration ?? {};
    if (id && name) {
      return { github: { id, name, ready: true } };
    }
  }
  return {};
}

function credentialsRewrite(configuration: Record<string, unknown> | undefined): FactoryAgentRewrite["credentials"] {
  const credentials = configuration?.credentials;
  if (credentials && typeof credentials === "object") {
    const { source, integration } = credentials as { source?: unknown; integration?: { name?: unknown } };
    if (source === "hosted") {
      return { source: "hosted" };
    }
    if (source === "integration" && typeof integration?.name === "string" && integration.name) {
      return { source: "integration", name: integration.name };
    }
  }
  // The live app must already be running with valid credentials; fall back to
  // hosted only for the unexpected case where the shape does not match.
  return { source: "hosted" };
}

/**
 * The app's own agent node already reflects its real harness, model, and
 * credentials — reuse them as-is instead of re-deriving them from onboarding
 * (which would need a fresh hosted-model/credit check unrelated to reset).
 */
function deriveAgentRewrite(nodes: CanvasSpecNode[]): FactoryAgentRewrite | undefined {
  const agentNode = primaryAgentNode({ nodes });
  const model = agentNode?.configuration?.model;
  if (!agentNode?.component || typeof model !== "string" || !model) {
    return undefined;
  }

  return {
    component: agentNode.component,
    model,
    credentials: credentialsRewrite(agentNode.configuration),
  };
}

/**
 * Derives the params/integrations/agent wiring reset should use so the
 * rematerialized bundled template keeps this app's real connections instead
 * of raw `{{ install_params.* }}` placeholders.
 */
export function deriveFactoryAppResetWiring(canvas: CanvasesCanvas | null | undefined): FactoryAppResetWiring {
  const nodes = canvas?.spec?.nodes ?? [];
  return {
    installParams: deriveInstallParams(nodes),
    integrations: deriveGithubSelection(nodes),
    agentRewrite: deriveAgentRewrite(nodes),
  };
}
