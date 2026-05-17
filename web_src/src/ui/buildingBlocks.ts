import type { IntegrationsIntegrationDefinition, SuperplaneActionsAction, TriggersTrigger } from "@/api-client";
import { actionsFromCapabilities, triggersFromCapabilities } from "@/lib/capabilities";
import type { BuildingBlock, BuildingBlockCategory } from "./BuildingBlocksSidebar";

export function flattenBuildingBlocks(categories: BuildingBlockCategory[]): BuildingBlock[] {
  return categories.flatMap((c) => c.blocks);
}

export function buildBuildingBlockCategories(
  triggers: TriggersTrigger[],
  components: SuperplaneActionsAction[],
  integrations: IntegrationsIntegrationDefinition[],
): BuildingBlockCategory[] {
  return [
    core(triggers, components),
    debugging(triggers, components),
    memory(triggers, components),
    ...buildIntegrationCategories(integrations),
  ];
}

function core(triggers: TriggersTrigger[], components: SuperplaneActionsAction[]): BuildingBlockCategory {
  return {
    name: "Core",
    blocks: [
      ...triggers.filter((t) => isCoreComponent(t)).map((t) => toTriggerBlock(t)),
      ...components.filter((c) => isCoreComponent(c)).map((c) => toComponentBlock(c)),
    ],
  };
}

function debugging(triggers: TriggersTrigger[], components: SuperplaneActionsAction[]): BuildingBlockCategory {
  return {
    name: "Debugging",
    blocks: [
      ...triggers.filter((t) => isDebuggingBlock(t)).map((t) => toTriggerBlock(t)),
      ...components.filter((c) => isDebuggingBlock(c)).map((c) => toComponentBlock(c)),
    ],
  };
}

function memory(triggers: TriggersTrigger[], components: SuperplaneActionsAction[]): BuildingBlockCategory {
  return {
    name: "Memory",
    blocks: [
      ...triggers.filter((t) => isMemoryBlock(t)).map((t) => toTriggerBlock(t)),
      ...components.filter((c) => isMemoryBlock(c)).map((c) => toComponentBlock(c)),
    ],
  };
}

function buildIntegrationCategories(integrations: IntegrationsIntegrationDefinition[]): BuildingBlockCategory[] {
  return integrations.map((i) => buildIntegrationCategory(i)).filter((c) => !!c);
}

function buildIntegrationCategory(integration: IntegrationsIntegrationDefinition): BuildingBlockCategory | null {
  const blocks: BuildingBlock[] = [];
  if (!integration.capabilities) {
    return null;
  }

  const triggers = triggersFromCapabilities(integration.capabilities);
  if (triggers) {
    triggers.forEach((t) => {
      blocks.push(toTriggerBlock(t, integration.name));
    });
  }

  const actions = actionsFromCapabilities(integration.capabilities);
  if (actions) {
    actions.forEach((c) => {
      blocks.push(toComponentBlock(c, integration.name));
    });
  }

  if (blocks.length === 0) {
    return null;
  }

  return {
    name: integration.label || "Unknown Integration",
    blocks,
  };
}

function toTriggerBlock(trigger: TriggersTrigger, integrationName?: string): BuildingBlock {
  return {
    name: trigger.name!,
    label: trigger.label,
    description: trigger.description,
    type: "trigger",
    configuration: trigger.configuration,
    icon: trigger.icon,
    color: trigger.color,
    integrationName: integrationName,
  };
}

function toComponentBlock(component: SuperplaneActionsAction, integrationName?: string): BuildingBlock {
  return {
    name: component.name!,
    label: component.label,
    description: component.description,
    type: "component",
    outputChannels: component.outputChannels,
    configuration: component.configuration,
    icon: component.icon,
    color: component.color,
    integrationName: integrationName,
  };
}

const MEMORY_COMPONENT_NAMES = new Set(["addmemory", "readmemory", "updatememory", "deletememory", "upsertmemory"]);
const DEBUGGING_COMPONENT_NAMES = new Set(["noop", "display"]);

function isMemoryBlock(component: { name?: string }): boolean {
  return MEMORY_COMPONENT_NAMES.has((component.name || "").toLowerCase());
}

function isDebuggingBlock(component: { name?: string }): boolean {
  return DEBUGGING_COMPONENT_NAMES.has((component.name || "").toLowerCase());
}

function isCoreComponent(component: { name?: string }): boolean {
  return !isMemoryBlock(component) && !isDebuggingBlock(component);
}
