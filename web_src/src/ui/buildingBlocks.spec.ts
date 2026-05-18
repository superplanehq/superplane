import { describe, expect, it } from "vitest";
import { buildBuildingBlockCategories } from "./buildingBlocks";

describe("buildBuildingBlockCategories", () => {
  it("orders categories as Core, Debugging, then Memory", () => {
    const categories = buildBuildingBlockCategories(
      [],
      [
        { name: "deploy", label: "Deploy" },
        { name: "display", label: "Display" },
        { name: "addmemory", label: "Add Memory" },
      ],
      [],
    );

    expect(categories.map((category) => category.name)).toEqual(["Core", "Debugging", "Memory"]);
  });
});
