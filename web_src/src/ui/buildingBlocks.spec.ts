import { describe, expect, it } from "vitest";
import { buildBuildingBlockCategories } from "./buildingBlocks";

describe("buildBuildingBlockCategories", () => {
  it("orders categories as Core, Runners, Debugging, Memory, then SuperPlane", () => {
    const categories = buildBuildingBlockCategories(
      [
        { name: "onBroadcast", label: "On Broadcast" },
        { name: "onRun", label: "On Run" },
      ],
      [
        { name: "deploy", label: "Deploy" },
        { name: "broadcastMessage", label: "Broadcast Message" },
        { name: "runApp", label: "Run App" },
        { name: "assignRunOutput", label: "Assign Run Output" },
        { name: "addRunError", label: "Add Run Error" },
        { name: "runnerJS", label: "Run JavaScript" },
        { name: "runnerBash", label: "Run Bash" },
        { name: "runnerPython", label: "Run Python" },
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
      "onRun",
      "broadcastMessage",
      "runApp",
      "assignRunOutput",
      "addRunError",
    ]);
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
