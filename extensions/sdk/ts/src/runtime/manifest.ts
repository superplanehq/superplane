import { DEFAULT_OUTPUT_CHANNEL, type ComponentDefinition, type ExtensionDefinition, type IntegrationDefinition, type TriggerDefinition } from "../block-definitions.js";
import type {
  ActionDefinition,
  ComponentBlock,
  ConfigurationField,
  IntegrationBlock,
  ManifestV1,
  RuntimeDescriptor,
  TriggerBlock,
} from "../manifest-schema.js";

export function deriveManifest(definition: ExtensionDefinition): ManifestV1 {
  validateExtensionDefinition(definition);

  return {
    apiVersion: "spx/v1",
    kind: "extension",
    metadata: {
      id: definition.metadata.id,
      name: definition.metadata.name,
      version: definition.metadata.version,
      description: definition.metadata.description,
    },
    runtime: definition.runtime ?? defaultRuntime(),
    integrations: (definition.integrations ?? []).map(serializeIntegration),
    components: (definition.components ?? []).map(serializeComponent),
    triggers: (definition.triggers ?? []).map(serializeTrigger),
  };
}

export function validateExtensionDefinition(definition: ExtensionDefinition): void {
  assertNonEmpty(definition.metadata.id, "extension metadata.id");
  assertNonEmpty(definition.metadata.name, "extension metadata.name");
  assertNonEmpty(definition.metadata.version, "extension metadata.version");

  const seenNames = new Set<string>();

  for (const integration of definition.integrations ?? []) {
    assertUniqueName(seenNames, integration.name);
    assertNonEmpty(integration.label, `integration ${integration.name} label`);
    assertNonEmpty(integration.description, `integration ${integration.name} description`);
    assertArray(integration.configuration, `integration ${integration.name} configuration`);
    assertActionDefinitions(integration.actions ?? [], `integration ${integration.name} actions`);
  }

  for (const component of definition.components ?? []) {
    assertUniqueName(seenNames, component.name);
    assertNonEmpty(component.label, `component ${component.name} label`);
    assertNonEmpty(component.description, `component ${component.name} description`);
    assertArray(component.configuration, `component ${component.name} configuration`);
    assertActionDefinitions(component.actions ?? [], `component ${component.name} actions`);

    if (usesIntegrationResource(component.configuration) && !component.integration) {
      throw new Error(`component ${component.name} uses integration-resource fields and must declare integration`);
    }
  }

  for (const trigger of definition.triggers ?? []) {
    assertUniqueName(seenNames, trigger.name);
    assertNonEmpty(trigger.label, `trigger ${trigger.name} label`);
    assertNonEmpty(trigger.description, `trigger ${trigger.name} description`);
    assertArray(trigger.configuration, `trigger ${trigger.name} configuration`);
    assertActionDefinitions(trigger.actions ?? [], `trigger ${trigger.name} actions`);

    if (usesIntegrationResource(trigger.configuration) && !trigger.integration) {
      throw new Error(`trigger ${trigger.name} uses integration-resource fields and must declare integration`);
    }
  }
}

function serializeIntegration(integration: IntegrationDefinition): IntegrationBlock {
  return {
    name: integration.name,
    label: integration.label,
    icon: integration.icon,
    description: integration.description,
    instructions: integration.instructions,
    configuration: cloneFields(integration.configuration),
    actions: cloneActions(integration.actions ?? []),
    resourceTypes: [...(integration.resourceTypes ?? [])],
  };
}

function serializeComponent(component: ComponentDefinition): ComponentBlock {
  return {
    name: component.name,
    integration: component.integration,
    label: component.label,
    description: component.description,
    documentation: component.documentation,
    icon: component.icon,
    color: component.color,
    outputChannels: [...(component.outputChannels ?? [DEFAULT_OUTPUT_CHANNEL])],
    configuration: cloneFields(component.configuration),
    actions: cloneActions(component.actions ?? []),
  };
}

function serializeTrigger(trigger: TriggerDefinition): TriggerBlock {
  return {
    name: trigger.name,
    integration: trigger.integration,
    label: trigger.label,
    description: trigger.description,
    documentation: trigger.documentation,
    icon: trigger.icon,
    color: trigger.color,
    exampleData: trigger.exampleData,
    configuration: cloneFields(trigger.configuration),
    actions: cloneActions(trigger.actions ?? []),
  };
}

function cloneFields(fields: readonly ConfigurationField[]): ConfigurationField[] {
  return fields.map((field) => structuredClone(field));
}

function cloneActions(actions: readonly ActionDefinition[]): ActionDefinition[] {
  return actions.map((action) => ({
    ...structuredClone(action),
    parameters: cloneFields(action.parameters),
  }));
}

function assertUniqueName(seenNames: Set<string>, name: string): void {
  assertNonEmpty(name, "block name");
  if (seenNames.has(name)) {
    throw new Error(`duplicate block name ${name}`);
  }
  seenNames.add(name);
}

function assertNonEmpty(value: string | undefined, label: string): void {
  if (!value || !value.trim()) {
    throw new Error(`${label} is required`);
  }
}

function assertArray(value: readonly unknown[], label: string): void {
  if (!Array.isArray(value)) {
    throw new Error(`${label} must be an array`);
  }
}

function assertActionDefinitions(actions: readonly ActionDefinition[], label: string): void {
  if (!Array.isArray(actions)) {
    throw new Error(`${label} must be an array`);
  }

  for (const action of actions) {
    assertNonEmpty(action.name, `${label} action name`);
    assertNonEmpty(action.description, `${label} action description`);
    assertArray(action.parameters, `${label} action parameters`);
  }
}

function usesIntegrationResource(fields: readonly ConfigurationField[]): boolean {
  return fields.some((field) => field.type === "integration-resource");
}

function defaultRuntime(): RuntimeDescriptor {
  return {
    profile: "portable-web-v1",
  };
}
