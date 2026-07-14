import type { IntegrationsIntegrationDefinition, ActionsAction, TriggersTrigger } from "@/api-client";
import { actionsFromCapabilities, triggersFromCapabilities } from "@/lib/capabilities";
import type { BuildingBlock, BuildingBlockCategory } from "./BuildingBlocksSidebar";

export function flattenBuildingBlocks(categories: BuildingBlockCategory[]): BuildingBlock[] {
  return categories.flatMap((c) => c.blocks);
}

export function buildBuildingBlockCategories(
  triggers: TriggersTrigger[],
  components: ActionsAction[],
  integrations: IntegrationsIntegrationDefinition[],
): BuildingBlockCategory[] {
  const runnerCategory = runners(triggers, components);
  const superplaneCategory = superplane(triggers, components);

  return [
    core(triggers, components),
    ...(runnerCategory ? [runnerCategory] : []),
    debugging(triggers, components),
    memory(triggers, components),
    ...(superplaneCategory ? [superplaneCategory] : []),
    ...buildIntegrationCategories(integrations),
  ];
}

function superplane(triggers: TriggersTrigger[], components: ActionsAction[]): BuildingBlockCategory | null {
  const blocks: BuildingBlock[] = [
    ...triggers.filter((t) => isSuperPlaneBlock(t)).map((t) => toTriggerBlock(t)),
    ...components.filter((c) => isSuperPlaneBlock(c)).map((c) => toComponentBlock(c)),
  ];

  if (blocks.length === 0) {
    return null;
  }

  blocks.sort((a, b) => {
    if (a.type !== b.type) {
      return a.type === "trigger" ? -1 : 1;
    }

    return (a.label || a.name || "").localeCompare(b.label || b.name || "");
  });

  return {
    name: "SuperPlane",
    blocks,
  };
}

function core(triggers: TriggersTrigger[], components: ActionsAction[]): BuildingBlockCategory {
  return {
    name: "Core",
    blocks: [
      ...triggers.filter((t) => isCoreComponent(t)).map((t) => toTriggerBlock(t)),
      ...components.filter((c) => isCoreComponent(c)).map((c) => toComponentBlock(c)),
    ],
  };
}

function runners(triggers: TriggersTrigger[], components: ActionsAction[]): BuildingBlockCategory | null {
  const blocks: BuildingBlock[] = [
    ...triggers.filter((t) => isRunnerBlock(t)).map((t) => toTriggerBlock(t)),
    ...components.filter((c) => isRunnerBlock(c)).map((c) => toComponentBlock(c)),
  ];

  if (blocks.length === 0) {
    return null;
  }

  blocks.sort(sortRunnerBlocks);

  return {
    name: "Runners",
    blocks,
  };
}

function debugging(triggers: TriggersTrigger[], components: ActionsAction[]): BuildingBlockCategory {
  return {
    name: "Debugging",
    blocks: [
      ...triggers.filter((t) => isDebuggingBlock(t)).map((t) => toTriggerBlock(t)),
      ...components.filter((c) => isDebuggingBlock(c)).map((c) => toComponentBlock(c)),
    ],
  };
}

function memory(triggers: TriggersTrigger[], components: ActionsAction[]): BuildingBlockCategory {
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

function toComponentBlock(component: ActionsAction, integrationName?: string): BuildingBlock {
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

const RUNNER_BLOCK_ORDER: Record<string, number> = {
  runner: 0,
  runnerBash: 1,
  runnerJS: 2,
  runnerPython: 3,
};

function sortRunnerBlocks(a: BuildingBlock, b: BuildingBlock): number {
  const aOrder = RUNNER_BLOCK_ORDER[a.name] ?? Number.POSITIVE_INFINITY;
  const bOrder = RUNNER_BLOCK_ORDER[b.name] ?? Number.POSITIVE_INFINITY;

  if (aOrder !== bOrder) {
    return aOrder - bOrder;
  }

  return (a.name || "").localeCompare(b.name || "");
}

function isRunnerBlock(component: { name?: string }): boolean {
  const name = component.name || "";
  return name === "runner" || name === "runnerJS" || name === "runnerBash" || name === "runnerPython";
}

const SUPERPLANE_BLOCK_NAMES = new Set(["onbroadcast", "broadcastmessage"]);

function isSuperPlaneBlock(component: { name?: string }): boolean {
  return SUPERPLANE_BLOCK_NAMES.has((component.name || "").toLowerCase());
}

function isCoreComponent(component: { name?: string }): boolean {
  return (
    !isMemoryBlock(component) &&
    !isDebuggingBlock(component) &&
    !isRunnerBlock(component) &&
    !isSuperPlaneBlock(component)
  );
}
