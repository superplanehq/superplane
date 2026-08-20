import { describe, expect, it } from "vitest";
import { buildBuildingBlockCategories } from "./buildingBlocks";

const triggers = [
  { name: "onBroadcast", label: "On Broadcast" },
  { name: "onRun", label: "On Run" },
];

const components = [
  { name: "deploy", label: "Deploy" },
  { name: "broadcastMessage", label: "Broadcast Message" },
  { name: "runApp", label: "Run App" },
  { name: "createWorkOrder", label: "Create Work Order" },
  { name: "updateWorkOrderArtifact", label: "Update Work Order Artifact" },
  { name: "runnerJS", label: "Run JavaScript" },
  { name: "runnerBash", label: "Run Bash" },
  { name: "runnerPython", label: "Run Python" },
  { name: "runnerClaudeCode", label: "Run Claude Code" },
  { name: "runnerCodex", label: "Run Codex" },
  { name: "runnerOpenRouter", label: "Run OpenRouter Agent" },
  { name: "runner", label: "Run Shell Commands" },
  { name: "display", label: "Display" },
  { name: "addmemory", label: "Add Memory" },
];

describe("buildBuildingBlockCategories", () => {
  it("orders categories as Core, Runners, Debugging, Memory, then SuperPlane", () => {
    const categories = buildBuildingBlockCategories(triggers, components, []);

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
    ]);
    expect(categories.find((category) => category.name === "Runners")?.blocks.map((block) => block.name)).toEqual([
      "runner",
      "runnerBash",
      "runnerJS",
      "runnerPython",
      "runnerClaudeCode",
      "runnerCodex",
      "runnerOpenRouter",
    ]);
    expect(categories.find((category) => category.name === "Core")?.blocks.map((block) => block.name)).toEqual([
      "deploy",
    ]);
  });

  it("includes factory components in SuperPlane for factory-owned apps", () => {
    const categories = buildBuildingBlockCategories(triggers, components, [], { isFactoryApp: true });

    expect(categories.find((category) => category.name === "SuperPlane")?.blocks.map((block) => block.name)).toEqual([
      "onBroadcast",
      "onRun",
      "broadcastMessage",
      "createWorkOrder",
      "runApp",
      "updateWorkOrderArtifact",
    ]);
  });
});
