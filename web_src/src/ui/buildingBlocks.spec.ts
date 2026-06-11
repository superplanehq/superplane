import { describe, expect, it } from "vitest";
import { buildBuildingBlockCategories } from "./buildingBlocks";

describe("buildBuildingBlockCategories", () => {
  it("orders categories as Core, Runners, Debugging, then Memory", () => {
    const categories = buildBuildingBlockCategories(
      [],
      [
        { name: "deploy", label: "Deploy" },
        { name: "runnerJS", label: "Run JavaScript" },
        { name: "runnerBash", label: "Run Bash" },
        { name: "runnerPython", label: "Run Python" },
        { name: "runner", label: "Run Shell Commands" },
        { name: "display", label: "Display" },
        { name: "addmemory", label: "Add Memory" },
      ],
      [],
    );

    expect(categories.map((category) => category.name)).toEqual(["Core", "Runners", "Debugging", "Memory"]);
    expect(categories.find((category) => category.name === "Runners")?.blocks.map((block) => block.name)).toEqual([
      "runner",
      "runnerBash",
      "runnerJS",
      "runnerPython",
    ]);
    expect(categories.find((category) => category.name === "Core")?.blocks.map((block) => block.name)).toEqual([
      "deploy",
    ]);
  });
});
