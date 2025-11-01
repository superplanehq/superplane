import { BuildingBlock, BuildingBlockCategory } from "./BuildingBlocksSidebar";
import { TriggersTrigger, ComponentsComponent, BlueprintsBlueprint } from "@/api-client";
import { mockBuildingBlockCategories } from "@/ui/CanvasPage/storybooks/buildingBlocks";

// Build categories of building blocks from live data and merge with mocks (deduped)
export function buildBuildingBlockCategories(
  triggers: TriggersTrigger[],
  components: ComponentsComponent[],
  blueprints: BlueprintsBlueprint[],
): BuildingBlockCategory[] {
  const liveCategories: BuildingBlockCategory[] = [
    {
      name: "Triggers",
      blocks: triggers.map(
        (t): BuildingBlock => ({
          name: t.name!,
          label: t.label,
          description: t.description,
          type: "trigger",
          configuration: t.configuration,
          icon: t.icon,
          color: t.color,
          isLive: true,
        }),
      ),
    },
    {
      name: "Primitives",
      blocks: components.map(
        (c): BuildingBlock => ({
          name: c.name!,
          label: c.label,
          description: c.description,
          type: "component",
          outputChannels: c.outputChannels,
          configuration: c.configuration,
          icon: c.icon,
          color: c.color,
          isLive: true,
        }),
      ),
    },
    {
      name: "Components",
      blocks: blueprints.map(
        (b): BuildingBlock => ({
          id: b.id,
          name: b.name!,
          description: b.description,
          type: "blueprint",
          outputChannels: b.outputChannels,
          configuration: b.configuration,
          icon: b.icon,
          color: b.color,
          isLive: true,
        }),
      ),
    },
  ];

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

