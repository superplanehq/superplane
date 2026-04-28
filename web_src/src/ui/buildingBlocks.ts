import type { BuildingBlock, BuildingBlockCategory } from "./BuildingBlocksSidebar";
import type { TriggersTrigger, SuperplaneActionsAction, IntegrationsIntegrationDefinition } from "@/api-client";

const MEMORY_COMPONENT_NAMES = new Set(["addmemory", "readmemory", "updatememory", "deletememory", "upsertmemory"]);

function isMemoryBlock(block: BuildingBlock): boolean {
  return MEMORY_COMPONENT_NAMES.has((block.name || "").toLowerCase());
}

// Build categories of building blocks from live data
export function buildBuildingBlockCategories(
  triggers: TriggersTrigger[],
  components: SuperplaneActionsAction[],
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
      };

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
      };

      return block;
    }),
  ];

  const liveCategories: BuildingBlockCategory[] = [
    {
      name: "Core",
      blocks: coreBlocks,
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
        };

        blocks.push(block);
      });
    }

    // Add components from this application
    if (integration.actions) {
      integration.actions.forEach((c) => {
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
        };

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
