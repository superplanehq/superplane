import type { SuperplaneActionsAction } from "@/api-client";
import { describe, expect, it } from "vitest";
import { buildBuildingBlockCategories } from "./buildingBlocks";

describe("buildBuildingBlockCategories", () => {
  it("does not merge storybook mock blocks into live categories", () => {
    const categories = buildBuildingBlockCategories([], [], []);

    expect(categories).toEqual([]);
  });

  it("only returns blocks provided by live inputs", () => {
    const component: SuperplaneActionsAction = {
      name: "deploy",
      label: "Deploy",
      description: "Deploy the current release",
    };

    const categories = buildBuildingBlockCategories([], [component], []);

    expect(categories).toHaveLength(1);
    expect(categories[0]?.name).toBe("Core");
    expect(categories[0]?.blocks.map((block) => block.name)).toEqual(["deploy"]);
  });

  it("moves display and noop into a Debugging category", () => {
    const categories = buildBuildingBlockCategories(
      [],
      [
        { name: "deploy", label: "Deploy" },
        { name: "display", label: "Display" },
        { name: "noop", label: "No Operation" },
      ],
      [],
    );

    expect(categories.map((category) => category.name)).toEqual(["Core", "Debugging"]);
    expect(categories[0]?.blocks.map((block) => block.name)).toEqual(["deploy"]);
    expect(categories[1]?.blocks.map((block) => block.name)).toEqual(["display", "noop"]);
  });

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
