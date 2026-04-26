import type { BuildingBlock, BuildingBlockCategory } from "./types";

export type TypeFilter = "all" | "trigger" | "component";

const TYPE_SORT_ORDER: Record<BuildingBlock["type"], number> = {
  trigger: 0,
  component: 1,
};

/**
 * Filters a category's blocks by search term + type filter and returns them
 * in the same order the sidebar renders them. Used by both `CategorySection`
 * (to decide what to draw) and by the keyboard shortcut (to pick the first
 * visible block when the user presses Enter). Keeping the two paths on a
 * shared helper is what makes "Enter drops the top item" match the eye.
 */
export function filterBlocksInCategory(
  category: BuildingBlockCategory,
  searchTerm: string,
  typeFilter: TypeFilter = "all",
): BuildingBlock[] {
  const query = searchTerm.trim().toLowerCase();
  const categoryMatches = query ? (category.name || "").toLowerCase().includes(query) : true;

  const baseBlocks = categoryMatches
    ? category.blocks || []
    : (category.blocks || []).filter((block) => {
        const name = (block.name || "").toLowerCase();
        const label = (block.label || "").toLowerCase();
        return name.includes(query) || label.includes(query);
      });

  const byType = typeFilter === "all" ? baseBlocks : baseBlocks.filter((block) => block.type === typeFilter);

  return [...byType].sort((a, b) => {
    const typeComparison = TYPE_SORT_ORDER[a.type] - TYPE_SORT_ORDER[b.type];
    if (typeComparison !== 0) {
      return typeComparison;
    }
    const aName = (a.label || a.name || "").toLowerCase();
    const bName = (b.label || b.name || "").toLowerCase();
    return aName.localeCompare(bName);
  });
}

/**
 * Returns the first block the user sees, given the already-sorted category
 * list and the current filter state. Returns null when the filter produces
 * zero matches — callers should treat that as a no-op.
 */
export function findFirstVisibleBlock(
  sortedCategories: BuildingBlockCategory[],
  searchTerm: string,
  typeFilter: TypeFilter = "all",
): BuildingBlock | null {
  for (const category of sortedCategories) {
    const blocks = filterBlocksInCategory(category, searchTerm, typeFilter);
    if (blocks.length > 0) {
      return blocks[0];
    }
  }
  return null;
}
