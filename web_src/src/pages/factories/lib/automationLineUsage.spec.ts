import { describe, expect, it } from "vitest";

import type { FactoriesFactoryLine } from "@/api-client";

import { linesUsingAutomation } from "./automationLineUsage";

const planner = "app-refund-planner";
const verifier = "app-refund-verifier";

function line(id: string, name: string, appIds: string[]): FactoriesFactoryLine {
  return {
    id,
    name,
    steps: appIds.map((appId, index) => ({
      name: `step-${index}`,
      app: { app: appId, entrypoint: "on-run" },
    })),
  };
}

describe("linesUsingAutomation", () => {
  const plan = line("line-plan", "plan-and-implement", [planner, "app-refund-implementer", verifier]);
  const hotfix = line("line-hotfix", "hotfix", [verifier]);
  const empty = line("line-empty", "empty", []);

  it("returns lines that reference the automation, in factory order", () => {
    expect(linesUsingAutomation([plan, hotfix, empty], verifier)).toEqual([
      { id: "line-plan", name: "plan-and-implement" },
      { id: "line-hotfix", name: "hotfix" },
    ]);
  });

  it("returns one entry when a line uses the automation in several steps", () => {
    const twice = line("line-twice", "twice", [planner, planner]);
    expect(linesUsingAutomation([twice], planner)).toEqual([{ id: "line-twice", name: "twice" }]);
  });

  it("returns an empty list when no line uses the automation", () => {
    expect(linesUsingAutomation([plan, hotfix], "app-unused")).toEqual([]);
  });

  it("skips lines without an id", () => {
    const unnamed: FactoriesFactoryLine = {
      name: "ghost",
      steps: [{ name: "plan", app: { app: planner, entrypoint: "on-run" } }],
    };
    expect(linesUsingAutomation([unnamed, plan], planner)).toEqual([{ id: "line-plan", name: "plan-and-implement" }]);
  });

  it("ignores blank app ids", () => {
    expect(linesUsingAutomation([plan], "  ")).toEqual([]);
    expect(linesUsingAutomation([plan], undefined)).toEqual([]);
  });
});
