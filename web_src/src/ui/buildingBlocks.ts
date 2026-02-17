import { BuildingBlock, BuildingBlockCategory } from "./BuildingBlocksSidebar";
import {
  TriggersTrigger,
  ComponentsComponent,
  BlueprintsBlueprint,
  IntegrationsIntegrationDefinition,
} from "@/api-client";
import { mockBuildingBlockCategories } from "@/ui/CanvasPage/storybooks/buildingBlocks";

// Flow control components that control workflow execution flow
const FLOW_COMPONENT_NAMES = new Set(["if", "filter", "approval", "wait", "timeGate"]);

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

// Build categories of building blocks from live data and merge with mocks (deduped)
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
        isLive: true,
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
        isLive: true,
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
          isLive: true,
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
          isLive: true,
          integrationName: integration.name,
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
          isLive: true,
          integrationName: integration.name,
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

  // Merge mock building blocks with live ones while avoiding duplicates
  // Dedupe key: `${type}:${name}`
  const byCategory = new Map<string, { blocks: Map<string, BuildingBlock>; order: string[] }>();

  const addCategoryIfMissing = (name: string) => {
    if (!byCategory.has(name)) {
      byCategory.set(name, { blocks: new Map(), order: [] });
    }
  };

  const addBlocks = (categoryName: string, blocks: BuildingBlock[]) => {
    addCategoryIfMissing(categoryName);
    const entry = byCategory.get(categoryName)!;
    blocks.forEach((blk) => {
      const key = `${blk.type}:${blk.name}`;
      if (!entry.blocks.has(key)) {
        // Ensure componentSubtype is set if not already present
        if (!blk.componentSubtype) {
          blk.componentSubtype = getComponentSubtype(blk);
        }
        entry.blocks.set(key, blk);
        entry.order.push(key);
      }
    });
  };

  // Seed with live categories first to prioritize real components
  liveCategories.forEach((cat) => addBlocks(cat.name, cat.blocks));
  // Merge in mocks
  mockBuildingBlockCategories.forEach((cat) => addBlocks(cat.name, cat.blocks));

  // Materialize back to array with stable order (live-first, then mock additions)
  const merged: BuildingBlockCategory[] = [];
  byCategory.forEach((value, key) => {
    merged.push({
      name: key,
      blocks: value.order.map((k) => value.blocks.get(k)!).filter(Boolean),
    });
  });

  return merged;
}

export function flattenBuildingBlocks(categories: BuildingBlockCategory[]): BuildingBlock[] {
  return categories.flatMap((c) => c.blocks);
}
