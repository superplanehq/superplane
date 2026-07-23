import yaml from "js-yaml";

import type { IntegrationSelections } from "../InstallIntegrationsSection";
import type { FactoryDefinition } from "./types";

const INSTALL_PARAM_PATTERN = /\{\{\s*install_params\.(\w+)\s*\}\}/g;

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
  [key: string]: unknown;
};

type YamlCanvas = {
  metadata?: { name?: string; description?: string; [key: string]: unknown };
  spec?: { nodes?: YamlNode[]; [key: string]: unknown };
  [key: string]: unknown;
};

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
  installParams: Record<string, string>;
  integrations: IntegrationSelections;
}): string {
  const substituted = substituteInstallParams(args.definition.canvasYaml, args.installParams);
  const wired = wireFactoryIntegrations(substituted, args.definition.componentIntegrations, args.integrations);
  const doc = yaml.load(wired) as YamlCanvas;
  if (!doc.metadata) doc.metadata = {};
  doc.metadata.name = args.canvasName;
  return yaml.dump(doc, { lineWidth: -1, noRefs: true });
}

export function materializeFactoryConsole(definition: FactoryDefinition, canvasName: string): string {
  const doc = yaml.load(definition.consoleYaml) as {
    metadata?: { name?: string; [key: string]: unknown };
    [key: string]: unknown;
  };
  if (!doc.metadata) doc.metadata = {};
  doc.metadata.name = canvasName;
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
