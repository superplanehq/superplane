import { describe, expect, it } from "bun:test";

import {
  applyColumnAutomationsOverlay,
  buildColumnAutomations,
  catalogForColumn,
  columnAutomationOpenPath,
  columnAutomationViewTabs,
} from "./columnAutomations";
import { factoryColumnAutomationViewPath } from "./factoryPagePaths";

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

it("keeps a renamed scheduled intake visible and opens its saved automation", () => {
  const rows = buildColumnAutomations("backlog", {
    columnTitle: "Backlog",
    planningEnabled: false,
    apps: [{ id: "scheduled-app", name: "Review documentation", columnKey: "backlog" }],
  });
  expect(rows).toHaveLength(1);
  expect(rows[0]).toMatchObject({ name: "Review documentation", trigger: "On schedule" });
  expect(columnAutomationOpenPath(rows[0], { organizationId: "org", factoryKey: "workspace" })).toBe(
    factoryColumnAutomationViewPath("org", "workspace", undefined, "scheduled-app"),
  );
});

it("keeps a renamed Review request visible without discovery and avoids a duplicate custom row", () => {
  const rows = buildColumnAutomations("verify", {
    columnTitle: "Verify",
    apps: [
      {
        id: "review-request-app",
        name: "Request engineering review",
        templateId: "review-request",
        columnKey: "verify",
        displayColumnKey: "done",
      },
    ],
  });
  expect(rows).toHaveLength(1);
  expect(rows[0]).toMatchObject({
    kind: "review-request",
    displayColumnKey: "done",
    executionKey: "verify",
    canvasId: "review-request-app",
    name: "Request engineering review",
  });
  expect(columnAutomationOpenPath(rows[0], { organizationId: "org", factoryKey: "workspace" })).toBe(
    factoryColumnAutomationViewPath("org", "workspace", undefined, "review-request-app"),
  );
  expect(catalogForColumn("verify").some((entry) => entry.id === "review-request")).toBe(false);
});
