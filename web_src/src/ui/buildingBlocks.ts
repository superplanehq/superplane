import type { BuildingBlock, BuildingBlockCategory } from "@/lib/index/types";
import type {
  ComponentsComponent,
  IntegrationsIntegrationDefinition,
  TriggersTrigger,
  WidgetsWidget,
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
  if (block.type === "TYPE_TRIGGER") {
    return "trigger";
  }

  if (block.type === "TYPE_COMPONENT" && block.name && FLOW_COMPONENT_NAMES.has(block.name)) {
    return "flow";
  }

  // Default to "action" for components and blueprints
  return "action";
}

function withSubtype<T extends BuildingBlock>(block: T): T {
  return {
    ...block,
    componentSubtype: getComponentSubtype(block),
  };
}

function createTriggerBlock(trigger: TriggersTrigger, integrationName?: string): BuildingBlock {
  return withSubtype({
    ...trigger,
    type: "TYPE_TRIGGER",
    name: trigger.name!,
    integrationName,
  });
}

function createComponentBlock(component: ComponentsComponent, integrationName?: string): BuildingBlock {
  return withSubtype({
    ...component,
    type: "TYPE_COMPONENT",
    name: component.name!,
    integrationName,
  });
}

function createWidgetBlock(widget: WidgetsWidget, integrationName?: string): BuildingBlock {
  return withSubtype({
    ...widget,
    type: "TYPE_WIDGET",
    name: widget.name!,
    integrationName,
  });
}

function createWidgetBlockFromComponent(component: ComponentsComponent, integrationName?: string): BuildingBlock {
  return createWidgetBlock(
    {
      name: component.name,
      label: component.label,
      description: component.description,
      icon: component.icon,
      color: component.color,
      configuration: component.configuration,
    },
    integrationName,
  );
}

function dedupeBlocks(blocks: BuildingBlock[]): BuildingBlock[] {
  const seen = new Set<string>();
  const deduped: BuildingBlock[] = [];

  blocks.forEach((block) => {
    const key = `${block.type}:${block.integrationName || ""}:${block.name}`;
    if (seen.has(key)) {
      return;
    }

    seen.add(key);
    deduped.push(block);
  });

  return deduped;
}

// Build categories of building blocks from live data
export function buildBuildingBlockCategories(
  triggers: TriggersTrigger[],
  components: ComponentsComponent[],
  availableIntegrations: IntegrationsIntegrationDefinition[],
  widgets: WidgetsWidget[] = [],
): BuildingBlockCategory[] {
  const deprecatedTriggerNames = new Set(["github", "semaphore"]);
  const deprecatedComponentNames = new Set(["semaphore"]);
  const filteredTriggers = triggers.filter((trigger) => trigger.name && !deprecatedTriggerNames.has(trigger.name));
  const filteredComponents = components.filter(
    (component) => component.name && !deprecatedComponentNames.has(component.name),
  );
  const filteredWidgets = widgets.filter((widget) => widget.name);

  // Combine triggers and components into a single "Core" category
  const coreBlocks: BuildingBlock[] = dedupeBlocks([
    ...filteredTriggers.map((trigger) => createTriggerBlock(trigger)),
    ...filteredComponents.map((component) =>
      component.name === "annotation" ? createWidgetBlockFromComponent(component) : createComponentBlock(component),
    ),
    ...filteredWidgets.map((widget) => createWidgetBlock(widget)),
  ]);

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
        if (!t.name) {
          return;
        }
        blocks.push(createTriggerBlock(t, integration.name));
      });
    }

    // Add components from this application
    if (integration.components) {
      integration.components.forEach((c) => {
        if (!c.name) {
          return;
        }
        blocks.push(
          c.name === "annotation"
            ? createWidgetBlockFromComponent(c, integration.name)
            : createComponentBlock(c, integration.name),
        );
      });
    }

    // Only add the category if there are blocks (label is normalized in useAvailableIntegrations)
    const dedupedBlocks = dedupeBlocks(blocks);
    if (dedupedBlocks.length > 0) {
      liveCategories.push({
        name: integration.label || "Unknown Integration",
        blocks: dedupedBlocks,
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
