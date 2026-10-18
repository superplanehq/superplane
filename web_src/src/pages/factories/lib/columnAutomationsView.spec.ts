import { describe, expect, it } from "bun:test";

import { applyColumnAutomationsOverlay, buildColumnAutomations, columnAutomationViewTabs } from "./columnAutomations";

describe("columnAutomationViewTabs", () => {
  it("shows only Automation when the canvas has no form and no agent", () => {
    expect(columnAutomationViewTabs({ hasGeneral: false, hasAgent: false })).toEqual(["automation"]);
  });

  it("shows Agent then Automation when the canvas has an agent", () => {
    expect(columnAutomationViewTabs({ hasGeneral: false, hasAgent: true })).toEqual(["agent", "automation"]);
  });

  it("shows General, Agent, then Automation when a form and an agent exist", () => {
    expect(columnAutomationViewTabs({ hasGeneral: true, hasAgent: true })).toEqual(["general", "agent", "automation"]);
  });

  it("shows General then Automation when a form exists without an agent", () => {
    expect(columnAutomationViewTabs({ hasGeneral: true, hasAgent: false })).toEqual(["general", "automation"]);
  });
});

describe("applyColumnAutomationsOverlay", () => {
  it("adds extra rows and applies disable and remove", () => {
    const base = buildColumnAutomations("done", {
      columnTitle: "Done",
      apps: [{ id: "app-pr-closure", name: "PR Closure" }],
    });
    const extra = [
      {
        ...base[0]!,
        id: "extra-1",
        name: "Extra",
      },
    ];

    const next = applyColumnAutomationsOverlay(base, extra, [`closure-app-pr-closure`], ["extra-1"]);

    expect(next).toHaveLength(1);
    expect(next[0]?.health).toBe("disabled");
  });
});
