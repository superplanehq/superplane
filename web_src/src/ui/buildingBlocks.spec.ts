import { describe, expect, it } from "vitest";
import { buildBuildingBlockCategories } from "./buildingBlocks";

describe("buildBuildingBlockCategories", () => {
  it("orders categories as Core, Runners, Debugging, Memory, then SuperPlane", () => {
    const categories = buildBuildingBlockCategories(
      [{ name: "onBroadcast", label: "On Broadcast" }],
      [
        { name: "deploy", label: "Deploy" },
        { name: "broadcastMessage", label: "Broadcast Message" },
        { name: "runnerJS", label: "Run JavaScript" },
        { name: "runnerBash", label: "Run Bash" },
        { name: "runnerPython", label: "Run Python" },
        { name: "runnerClaudeCode", label: "Run Claude Code" },
        { name: "runner", label: "Run Shell Commands" },
        { name: "display", label: "Display" },
        { name: "addmemory", label: "Add Memory" },
      ],
      [],
    );

    expect(categories.map((category) => category.name)).toEqual([
      "Core",
      "Runners",
      "Debugging",
      "Memory",
      "SuperPlane",
    ]);
    expect(categories.find((category) => category.name === "SuperPlane")?.blocks.map((block) => block.name)).toEqual([
      "onBroadcast",
      "broadcastMessage",
    ]);
    expect(categories.find((category) => category.name === "Runners")?.blocks.map((block) => block.name)).toEqual([
      "runner",
      "runnerBash",
      "runnerJS",
      "runnerPython",
      "runnerClaudeCode",
    ]);
    expect(categories.find((category) => category.name === "Core")?.blocks.map((block) => block.name)).toEqual([
      "deploy",
    ]);
  });
});
