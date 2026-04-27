import { describe, expect, it } from "vitest";
import { filterBlocksInCategory, findFirstVisibleBlock } from "./filter";
import type { BuildingBlock, BuildingBlockCategory } from "./types";

const block = (overrides: Partial<BuildingBlock> & Pick<BuildingBlock, "name" | "type">): BuildingBlock => ({
  label: overrides.name,
  ...overrides,
});

const coreCategory: BuildingBlockCategory = {
  name: "Core",
  blocks: [
    block({ name: "manual", label: "Manual Run", type: "trigger" }),
    block({ name: "filter", label: "Filter", type: "component" }),
    block({ name: "approval", label: "Approval", type: "component" }),
    block({ name: "schedule", label: "Schedule", type: "trigger" }),
  ],
};

describe("filterBlocksInCategory", () => {
  it("returns all blocks sorted triggers-first, then alphabetically when no filter is set", () => {
    const result = filterBlocksInCategory(coreCategory, "", "all");

    expect(result.map((b) => b.label)).toEqual(["Manual Run", "Schedule", "Approval", "Filter"]);
  });

  it("matches blocks by label case-insensitively", () => {
    const result = filterBlocksInCategory(coreCategory, "filt", "all");

    expect(result.map((b) => b.label)).toEqual(["Filter"]);
  });

  it("matches blocks by name", () => {
    const result = filterBlocksInCategory(coreCategory, "approv", "all");

    expect(result.map((b) => b.label)).toEqual(["Approval"]);
  });

  it("returns every block when the query matches the category name", () => {
    const result = filterBlocksInCategory(coreCategory, "core", "all");

    expect(result).toHaveLength(coreCategory.blocks.length);
  });

  it("narrows by type filter", () => {
    const result = filterBlocksInCategory(coreCategory, "", "trigger");

    expect(result.map((b) => b.label)).toEqual(["Manual Run", "Schedule"]);
  });

  it("returns an empty array when nothing matches", () => {
    const result = filterBlocksInCategory(coreCategory, "zzz", "all");

    expect(result).toEqual([]);
  });
});

describe("findFirstVisibleBlock", () => {
  const awsCategory: BuildingBlockCategory = {
    name: "AWS",
    blocks: [block({ name: "aws.ecs", label: "ECS • Execute Command", type: "component" })],
  };

  const daytonaCategory: BuildingBlockCategory = {
    name: "Daytona",
    blocks: [block({ name: "daytona.exec", label: "Execute Command", type: "component" })],
  };

  it("returns the first block of the first category that has matches", () => {
    const result = findFirstVisibleBlock([coreCategory, awsCategory, daytonaCategory], "man", "all");

    expect(result?.label).toBe("Manual Run");
  });

  it("skips categories with no matches and finds the first one that does", () => {
    const result = findFirstVisibleBlock([coreCategory, awsCategory], "ecs", "all");

    expect(result?.label).toBe("ECS • Execute Command");
  });

  it("returns null when no block matches", () => {
    const result = findFirstVisibleBlock([coreCategory, awsCategory], "zzz", "all");

    expect(result).toBeNull();
  });

  it("returns the top-of-list block for an empty query (callers gate empty-query themselves)", () => {
    // Empty query is not intrinsically a miss — filterBlocksInCategory returns
    // every block. Callers like the Enter handler treat empty as a no-op
    // before calling in, so this helper itself must still return the
    // sort-order first block.
    const result = findFirstVisibleBlock([coreCategory], "", "all");
    expect(result?.label).toBe("Manual Run");
  });

  it("respects type filter when picking the first visible block", () => {
    const result = findFirstVisibleBlock([coreCategory, awsCategory], "", "component");

    expect(result?.label).toBe("Approval");
  });
});
