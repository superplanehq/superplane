import { describe, expect, it } from "bun:test";

import { buildColumnAutomations, catalogForColumn } from "./columnAutomations";
import type { LinePhaseColumn } from "./linePhaseRuns";

const IMPLEMENT_COLUMN: LinePhaseColumn = {
  stepName: "Implement",
  stepIndex: 0,
  appId: "app-refund-implementer",
  maxParallelism: 10,
  runs: [],
  tick: null,
};

describe("Implement and Done custom automations", () => {
  it("offers a custom automation on Implement and a blank canvas on later phases", () => {
    expect(catalogForColumn("phase-0", { allowCustom: true }).map((entry) => entry.id)).toEqual([
      "custom-verification",
    ]);
    expect(catalogForColumn("phase-1", { allowCustom: true }).map((entry) => entry.id)).toEqual([
      "agent-step",
      "custom",
    ]);
  });

  it("lists a custom automation on Implement and Done", () => {
    const implement = buildColumnAutomations("phase-0", {
      columnTitle: "Implement",
      columns: [IMPLEMENT_COLUMN],
      apps: [
        {
          id: "lint",
          name: "Lint",
          columnKey: "implement",
          templateId: "custom-verification",
          customAutomationMode: "automatic",
        },
      ],
    });
    expect(implement.map((automation) => automation.name)).toEqual(["Implement", "Lint"]);
    expect(implement[1]?.trigger).toBe("When a task enters Implement");

    const done = buildColumnAutomations("done", {
      columnTitle: "Done",
      apps: [
        {
          id: "notify",
          name: "Notify",
          columnKey: "done",
          templateId: "custom-verification",
          customAutomationMode: "manual",
        },
      ],
    });
    expect(done.map((automation) => automation.trigger)).toEqual(["Manual start on a Done task"]);
  });
});
