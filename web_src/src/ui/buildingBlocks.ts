import { BuildingBlock, BuildingBlockCategory } from "./BuildingBlocksSidebar";
import {
  TriggersTrigger,
  ComponentsComponent,
  BlueprintsBlueprint,
  IntegrationsIntegrationDefinition,
} from "@/api-client";

// Flow control components that control workflow execution flow
const FLOW_COMPONENT_NAMES = new Set(["if", "filter", "approval", "wait", "timeGate"]);
const MEMORY_COMPONENT_NAMES = new Set(["addmemory", "readmemory", "updatememory", "deletememory", "upsertmemory"]);

function isMemoryBlock(block: BuildingBlock): boolean {
  return MEMORY_COMPONENT_NAMES.has((block.name || "").toLowerCase());
}

/**
 * Determines the component subtype based on the building block's type and name.
 * - Triggers are always "trigger"
 * - Flow control components (if, filter, approval, wait, timeGate) are "flow"
 * - All other components are "action"
 * - Blueprints default to "action"
 */
export function getComponentSubtype(block: BuildingBlock): "trigger" | "action" | "flow" {
  if (block.type === "trigger") {
    return "trigger";
  }

  if (block.type === "component" && block.name && FLOW_COMPONENT_NAMES.has(block.name)) {
    return "flow";
  }

  // Default to "action" for components and blueprints
  return "action";
}

// Build categories of building blocks from live data
export function buildBuildingBlockCategories(
  triggers: TriggersTrigger[],
  components: ComponentsComponent[],
  blueprints: BlueprintsBlueprint[],
  availableIntegrations: IntegrationsIntegrationDefinition[],
): BuildingBlockCategory[] {
  const deprecatedTriggerNames = new Set(["github", "semaphore"]);
  const deprecatedComponentNames = new Set(["semaphore"]);
  const filteredTriggers = triggers.filter((trigger) => !deprecatedTriggerNames.has(trigger.name ?? ""));
  const filteredComponents = components.filter((component) => !deprecatedComponentNames.has(component.name ?? ""));

  // Combine triggers and components into a single "Core" category
  const coreBlocks: BuildingBlock[] = [
    ...filteredTriggers.map((t): BuildingBlock => {
      const block: BuildingBlock = {
        name: t.name!,
        label: t.label,
        description: t.description,
        type: "trigger",
        configuration: t.configuration,
        icon: t.icon,
        color: t.color,
        exampleData: t.exampleData,
      };
      block.componentSubtype = getComponentSubtype(block);
      return block;
    }),
    ...filteredComponents.map((c): BuildingBlock => {
      const block: BuildingBlock = {
        name: c.name!,
        label: c.label,
        description: c.description,
        type: "component",
        outputChannels: c.outputChannels,
        configuration: c.configuration,
        icon: c.icon,
        color: c.color,
        exampleOutput: c.exampleOutput,
      };
      block.componentSubtype = getComponentSubtype(block);
      return block;
    }),
  ];

  const liveCategories: BuildingBlockCategory[] = [
    {
      name: "Core",
      blocks: coreBlocks,
    },
    {
      name: "Bundles",
      blocks: blueprints.map((b): BuildingBlock => {
        const block: BuildingBlock = {
          id: b.id,
          name: b.name!,
          description: b.description,
          type: "blueprint",
          outputChannels: b.outputChannels,
          configuration: b.configuration,
          icon: "component",
          color: "gray",
        };
        block.componentSubtype = getComponentSubtype(block);
        return block;
      }),
    },
  ];

  // Add a category for each available application with its components and triggers
  availableIntegrations.forEach((integration) => {
    const blocks: BuildingBlock[] = [];

    // Add triggers from this integration
    if (integration.triggers) {
      integration.triggers.forEach((t) => {
        const block: BuildingBlock = {
          name: t.name!,
          label: t.label,
          description: t.description,
          type: "trigger",
          configuration: t.configuration,
          icon: t.icon,
          color: t.color,
          integrationName: integration.name,
          exampleData: t.exampleData,
        };
        block.componentSubtype = getComponentSubtype(block);
        blocks.push(block);
      });
    }

    // Add components from this application
    if (integration.components) {
      integration.components.forEach((c) => {
        const block: BuildingBlock = {
          name: c.name!,
          label: c.label,
          description: c.description,
          type: "component",
          outputChannels: c.outputChannels,
          configuration: c.configuration,
          icon: c.icon,
          color: c.color,
          integrationName: integration.name,
          exampleOutput: c.exampleOutput,
        };
        block.componentSubtype = getComponentSubtype(block);
        blocks.push(block);
      });
    }

    // Only add the category if there are blocks (label is normalized in useAvailableIntegrations)
    if (blocks.length > 0) {
      liveCategories.push({
        name: integration.label || "Unknown Integration",
        blocks,
      });
    }
  });

  // Move all memory blocks to a dedicated "memory" category.
  const memoryBlocksByKey = new Map<string, BuildingBlock>();
  const categoriesWithoutMemory = liveCategories
    .map((category) => {
      const nonMemoryBlocks: BuildingBlock[] = [];
      category.blocks.forEach((block) => {
        if (isMemoryBlock(block)) {
          memoryBlocksByKey.set(`${block.type}:${block.name}`, block);
          return;
        }
        nonMemoryBlocks.push(block);
      });

      return {
        ...category,
        blocks: nonMemoryBlocks,
      };
    })
    .filter((category) => category.blocks.length > 0);

  const memoryCategory: BuildingBlockCategory[] =
    memoryBlocksByKey.size > 0
      ? [
          {
            name: "Memory",
            blocks: Array.from(memoryBlocksByKey.values()),
          },
        ]
      : [];

  return [...categoriesWithoutMemory, ...memoryCategory];
}

export function flattenBuildingBlocks(categories: BuildingBlockCategory[]): BuildingBlock[] {
  return categories.flatMap((c) => c.blocks);
}
