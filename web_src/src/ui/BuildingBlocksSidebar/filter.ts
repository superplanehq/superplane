import type { BuildingBlock, BuildingBlockCategory } from "./types";

export type TypeFilter = "all" | "trigger" | "component";

/**
 * Normalizes an integration name for comparison by lowercasing and stripping
 * any non-alphanumeric characters, so that values like "GitHub" and "github"
 * (or "google-cloud" and "googlecloud") are treated as the same integration.
 */
export function normalizeIntegrationName(value?: string): string {
  return (value || "").toLowerCase().replace(/[^a-z0-9]/g, "");
}

const TYPE_SORT_ORDER: Record<BuildingBlock["type"], number> = {
  trigger: 0,
  component: 1,
};

/**
 * Builds the lowercased text a block is matched against. It combines the
 * category name (the section subheader shown above the block, e.g. "GitHub")
 * with the block's own label, name and description, so a query that spans the
 * subheader and the block — like "github status" for the "On Status" trigger
 * under "GitHub" — still finds the block.
 */
function blockSearchText(category: BuildingBlockCategory, block: BuildingBlock): string {
  return [category.name, block.label, block.name, block.description].filter(Boolean).join(" ").toLowerCase();
}

/**
 * Filters a category's blocks by search term + type filter and returns them
 * in the same order the sidebar renders them. Used by both `CategorySection`
 * (to decide what to draw) and by the keyboard shortcut (to pick the first
 * visible block when the user presses Enter). Keeping the two paths on a
 * shared helper is what makes "Enter drops the top item" match the eye.
 *
 * The query is split into whitespace-separated tokens and a block matches only
 * when every token is found in its combined search text (see
 * `blockSearchText`). This lets a query mix the section subheader with block
 * text in any order, e.g. "github status" matches the "On Status" trigger in
 * the "GitHub" category.
 */
export function filterBlocksInCategory(
  category: BuildingBlockCategory,
  searchTerm: string,
  typeFilter: TypeFilter = "all",
): BuildingBlock[] {
  const tokens = searchTerm.trim().toLowerCase().split(/\s+/).filter(Boolean);

  const baseBlocks =
    tokens.length === 0
      ? category.blocks || []
      : (category.blocks || []).filter((block) => {
          const text = blockSearchText(category, block);
          return tokens.every((token) => text.includes(token));
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
