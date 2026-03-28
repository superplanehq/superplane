import { describe, expect, it } from "vitest";
import { buildBuildingBlockCategories } from "./buildingBlocks";

describe("buildBuildingBlockCategories", () => {
  it("does not merge storybook mock blocks into live categories", () => {
    const categories = buildBuildingBlockCategories([], [], [], []);

    expect(categories).toEqual([]);
  });

  it("only returns blocks provided by live inputs", () => {
    const categories = buildBuildingBlockCategories(
      [],
      [
        {
          name: "deploy",
          label: "Deploy",
          description: "Deploy the current release",
        } as any,
      ],
      [],
      [],
    );

    expect(categories).toHaveLength(1);
    expect(categories[0]?.name).toBe("Core");
    expect(categories[0]?.blocks.map((block) => block.name)).toEqual(["deploy"]);
  });
});
