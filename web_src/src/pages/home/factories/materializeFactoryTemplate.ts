import yaml from "js-yaml";

import type { IntegrationSelections } from "../InstallIntegrationsSection";
import type { FactoryDefinition } from "./types";

const INSTALL_PARAM_PATTERN = /\{\{\s*install_params\.(\w+)\s*\}\}/g;

export const FACTORY_CANVAS_ID_PLACEHOLDER = "__FACTORY_CANVAS_ID__";

/**
 * Legacy home install passed a single `repository`. Map it onto the split
 * app/backlog params so older callers and tests keep working.
 */
export function normalizeFactoryInstallParams(params: Record<string, string>): Record<string, string> {
  const next = { ...params };
  const legacyRepository = next.repository?.trim();
  if (!legacyRepository) return next;

  if (!next.appRepository?.trim()) {
    next.appRepository = legacyRepository;
  }
  if (!next.backlogRepository?.trim()) {
    next.backlogRepository = legacyRepository;
  }
  return next;
}

export function substituteInstallParams(content: string, params: Record<string, string>): string {
  return content.replace(INSTALL_PARAM_PATTERN, (match, name: string) => {
    if (Object.prototype.hasOwnProperty.call(params, name)) {
      return params[name] ?? match;
    }
    return match;
  });
}

type YamlNode = {
  id?: string;
  component?: string;
  integration?: { id?: string; name?: string };
  configuration?: Record<string, unknown>;
  metadata?: Record<string, unknown>;
  [key: string]: unknown;
};

type YamlCanvas = {
  metadata?: { name?: string; description?: string; [key: string]: unknown };
  spec?: { nodes?: YamlNode[]; [key: string]: unknown };
  [key: string]: unknown;
};

function replacePlaceholders(content: string, replacements: Record<string, string | undefined>): string {
  let next = content;
  for (const [placeholder, value] of Object.entries(replacements)) {
    if (!value) continue;
    next = next.split(placeholder).join(value);
  }
  return next;
}

function rewriteIntegrationRefNames(value: unknown, selections: IntegrationSelections): void {
  if (Array.isArray(value)) {
    for (const item of value) rewriteIntegrationRefNames(item, selections);
    return;
  }
  if (!value || typeof value !== "object") return;

  const obj = value as Record<string, unknown>;
  const integration = obj.integration;
  if (obj.source === "integration" && integration && typeof integration === "object" && !Array.isArray(integration)) {
    const ref = integration as { name?: string };
    const typeName = typeof ref.name === "string" ? ref.name.trim() : "";
    const selection = typeName ? selections[typeName] : undefined;
    if (selection?.name) {
      ref.name = selection.name;
    }
  }

  for (const child of Object.values(obj)) {
    rewriteIntegrationRefNames(child, selections);
  }
}

export function wireFactoryIntegrations(
  canvasYaml: string,
  componentIntegrations: Record<string, string>,
  selections: IntegrationSelections,
): string {
  const doc = yaml.load(canvasYaml) as YamlCanvas;
  const nodes = doc?.spec?.nodes;
  if (!Array.isArray(nodes)) return canvasYaml;

  for (const node of nodes) {
    const component = typeof node.component === "string" ? node.component : "";
    const integrationName = componentIntegrations[component];
    if (integrationName) {
      const selection = selections[integrationName];
      if (selection) {
        node.integration = { id: selection.id, name: selection.name };
      }
    }
    // credentials / environmentFrom use integration type names in the template
    // (e.g. name: claude); rewrite them to the selected installation name.
    rewriteIntegrationRefNames(node.configuration, selections);
  }

  return yaml.dump(doc, { lineWidth: -1, noRefs: true });
}

const CLAUDE_CODE_COMPONENT = "runnerClaudeCode";
const PLANNING_AGENT_NODE_ID = "planner-agent-no-issue";
const PLANNING_AGENT_MODEL = "opus";

export type FactoryAgentRewrite = {
  component: string;
  model: string;
  credentials: { source: "hosted" } | { source: "integration"; name: string };
};

function rewriteOnboardingAgentNodes(doc: YamlCanvas, rewrite: FactoryAgentRewrite): void {
  for (const node of doc.spec?.nodes ?? []) {
    if (node.component !== CLAUDE_CODE_COMPONENT) continue;
    node.component = rewrite.component;
    const configuration = node.configuration;
    if (!configuration || typeof configuration !== "object") continue;
    if (rewrite.credentials.source === "hosted") {
      configuration.credentials = { source: "hosted" };
    } else {
      configuration.credentials = {
        source: "integration",
        integration: { name: rewrite.credentials.name },
      };
    }
    configuration.model = planningAgentModel(node.id, rewrite);
  }
}

function planningAgentModel(nodeId: string | undefined, rewrite: FactoryAgentRewrite): string {
  if (nodeId === PLANNING_AGENT_NODE_ID && rewrite.component === CLAUDE_CODE_COMPONENT) {
    return PLANNING_AGENT_MODEL;
  }
  return rewrite.model;
}

export function materializeFactoryCanvas(args: {
  definition: FactoryDefinition;
  canvasName: string;
  canvasId: string;
  installParams: Record<string, string>;
  integrations: IntegrationSelections;
  agentRewrite?: FactoryAgentRewrite;
}): string {
  const withPlaceholders = replacePlaceholders(args.definition.canvasYaml, {
    [FACTORY_CANVAS_ID_PLACEHOLDER]: args.canvasId,
  });

  const installParams = normalizeFactoryInstallParams(args.installParams);
  const substituted = substituteInstallParams(withPlaceholders, installParams);
  const wired = wireFactoryIntegrations(substituted, args.definition.componentIntegrations, args.integrations);
  const doc = yaml.load(wired) as YamlCanvas;
  if (!doc.metadata) doc.metadata = {};
  doc.metadata.name = args.canvasName;

  if (args.agentRewrite) {
    rewriteOnboardingAgentNodes(doc, args.agentRewrite);
  }

  for (const node of doc.spec?.nodes ?? []) {
    if (node.component !== "runApp") continue;
    const configuration = node.configuration;
    if (!configuration || typeof configuration !== "object") continue;
    if (configuration.app !== args.canvasId) continue;
    node.metadata = {
      ...(node.metadata ?? {}),
      app: { id: args.canvasId, name: args.canvasName },
    };
  }

  return yaml.dump(doc, { lineWidth: -1, noRefs: true });
}

export function materializeFactoryConsole(definition: FactoryDefinition, canvasName: string, canvasId: string): string {
  const withPlaceholders = replacePlaceholders(definition.consoleYaml, {
    [FACTORY_CANVAS_ID_PLACEHOLDER]: canvasId,
  });
  const doc = yaml.load(withPlaceholders) as {
    metadata?: { name?: string; canvasId?: string; [key: string]: unknown };
    [key: string]: unknown;
  };
  if (!doc.metadata) doc.metadata = {};
  doc.metadata.name = canvasName;
  doc.metadata.canvasId = canvasId;
  return yaml.dump(doc, { lineWidth: -1, noRefs: true });
}

export function buildFactoryRunParameters(
  definition: FactoryDefinition,
  startingTaskPrompt: string,
): Record<string, string> {
  const parameters: Record<string, string> = {
    template: definition.run.template,
  };
  for (const [name, source] of Object.entries(definition.run.parameters)) {
    if (source.from === "startingTaskPrompt") {
      parameters[name] = startingTaskPrompt;
    }
  }
  return parameters;
}
