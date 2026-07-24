import yaml from "js-yaml";

import type { IntegrationSelections } from "../InstallIntegrationsSection";
import type { FactoryDefinition } from "./types";

const INSTALL_PARAM_PATTERN = /\{\{\s*install_params\.(\w+)\s*\}\}/g;

export const FACTORY_CANVAS_ID_PLACEHOLDER = "__FACTORY_CANVAS_ID__";

export function substituteInstallParams(content: string, params: Record<string, string>): string {
  return content.replace(INSTALL_PARAM_PATTERN, (match, name: string) => {
    if (Object.prototype.hasOwnProperty.call(params, name)) {
      return params[name] ?? match;
    }
    return match;
  });
}

type YamlNode = {
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
    if (!integrationName) continue;
    const selection = selections[integrationName];
    if (!selection) continue;
    node.integration = { id: selection.id, name: selection.name };
  }

  return yaml.dump(doc, { lineWidth: -1, noRefs: true });
}

export function materializeFactoryCanvas(args: {
  definition: FactoryDefinition;
  canvasName: string;
  canvasId: string;
  installParams: Record<string, string>;
  integrations: IntegrationSelections;
}): string {
  const withPlaceholders = replacePlaceholders(args.definition.canvasYaml, {
    [FACTORY_CANVAS_ID_PLACEHOLDER]: args.canvasId,
  });

  const substituted = substituteInstallParams(withPlaceholders, args.installParams);
  const wired = wireFactoryIntegrations(substituted, args.definition.componentIntegrations, args.integrations);
  const doc = yaml.load(wired) as YamlCanvas;
  if (!doc.metadata) doc.metadata = {};
  doc.metadata.name = args.canvasName;

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
