import { describe, expect, it } from "vitest";

import type { FactoriesFactoryIntake, FactoriesFactoryPrFeedbackHandler, FactoriesWorkOrder } from "@/api-client";

import {
  applyColumnAutomationsOverlay,
  buildColumnAutomations,
  catalogForColumn,
  columnAutomationsEmptyCopy,
  columnAutomationsNeedRepair,
  columnTitleForKey,
  isColumnKey,
  phaseIndexFromColumnKey,
  takenCatalogIds,
} from "./columnAutomations";
import type { LinePhaseColumn } from "./linePhaseRuns";

const GITHUB_INTAKE: FactoriesFactoryIntake = {
  id: "intake-github",
  canvasId: "app-github-issues-intake",
  name: "GitHub issues",
  source: "SOURCE_GITHUB_ISSUES",
  healthy: true,
};

const UNHEALTHY_SENTRY: FactoriesFactoryIntake = {
  id: "intake-sentry",
  canvasId: "app-sentry-intake",
  name: "Sentry exceptions",
  source: "SOURCE_SENTRY_EXCEPTIONS",
  healthy: false,
};

const DISCUSSION_HANDLER: FactoriesFactoryPrFeedbackHandler = {
  id: "prfb-discussion",
  canvasId: "app-pr-discussion",
  name: "Address PR feedback",
  source: "SOURCE_PULL_REQUEST_DISCUSSION",
  healthy: true,
};

const CHECKS_HANDLER: FactoriesFactoryPrFeedbackHandler = {
  id: "prfb-checks",
  canvasId: "app-pr-checks",
  name: "Fix pull request checks",
  source: "SOURCE_PULL_REQUEST_CHECKS",
  healthy: false,
};

const IMPLEMENT_COLUMN: LinePhaseColumn = {
  stepName: "Implement",
  stepIndex: 0,
  appId: "app-refund-implementer",
  maxParallelism: 10,
  runs: [],
  tick: null,
};

const EMPTY_PHASE: LinePhaseColumn = {
  stepName: "Review",
  stepIndex: 1,
  maxParallelism: 10,
  runs: [],
  tick: null,
};

const RUNNING_ORDER: FactoriesWorkOrder = {
  id: "wo-1",
  lineDispatches: [
    {
      stepExecutions: [
        {
          id: "exec-1",
          state: "STATE_STARTED",
          run: { appId: "app-refund-implementer" },
        },
      ],
    },
  ],
};

describe("isColumnKey", () => {
  it("accepts bookends and phase keys", () => {
    expect(isColumnKey("backlog")).toBe(true);
    expect(isColumnKey("verify")).toBe(true);
    expect(isColumnKey("done")).toBe(true);
    expect(isColumnKey("phase-0")).toBe(true);
    expect(isColumnKey("phase-12")).toBe(true);
    expect(isColumnKey("phase")).toBe(false);
    expect(isColumnKey("inbox")).toBe(false);
    expect(isColumnKey(null)).toBe(false);
  });
});

describe("phaseIndexFromColumnKey", () => {
  it("reads the step index from a phase key", () => {
    expect(phaseIndexFromColumnKey("phase-2")).toBe(2);
    expect(phaseIndexFromColumnKey("backlog")).toBeUndefined();
  });
});

describe("columnTitleForKey", () => {
  it("uses the phase name when the column is present", () => {
    expect(columnTitleForKey("backlog")).toBe("Backlog");
    expect(columnTitleForKey("phase-0", [IMPLEMENT_COLUMN])).toBe("Implement");
    expect(columnTitleForKey("phase-3", [IMPLEMENT_COLUMN])).toBe("Phase 4");
  });
});

describe("columnAutomationsEmptyCopy", () => {
  it("explains what is missing for each column", () => {
    expect(columnAutomationsEmptyCopy("Backlog", "backlog")).toBe(
      "No automations. Add an intake to create tasks from a source.",
    );
    expect(columnAutomationsEmptyCopy("Implement", "phase-0")).toBe(
      "No automations. Tasks pass through Implement without action.",
    );
    expect(columnAutomationsEmptyCopy("Verify", "verify")).toBe("No automations. Add a pull request listener.");
    expect(columnAutomationsEmptyCopy("Done", "done")).toBe("No automations. Tasks stay here when they finish.");
  });
});

describe("buildColumnAutomations", () => {
  it("builds backlog intakes and the analysis automation", () => {
    const automations = buildColumnAutomations("backlog", {
      columnTitle: "Backlog",
      intakes: [GITHUB_INTAKE],
      apps: [{ id: "app-refund-backlog", name: "Ingest" }],
    });

    expect(automations).toHaveLength(2);
    expect(automations[0]).toMatchObject({
      kind: "intake",
      trigger: "On GitHub issue",
      action: "Create a task in Backlog",
      catalogId: "github-issues",
    });
    expect(automations[1]).toMatchObject({
      kind: "analysis",
      name: "Task analysis",
      trigger: "On task in Backlog",
      action: "Score the task",
      catalogId: "analysis",
    });
  });

  it("marks an unhealthy intake as needs-repair", () => {
    const automations = buildColumnAutomations("backlog", {
      columnTitle: "Backlog",
      intakes: [UNHEALTHY_SENTRY],
    });

    expect(automations[0]?.health).toBe("needs-repair");
    expect(columnAutomationsNeedRepair(automations)).toBe(true);
  });

  it("builds a phase agent sentence from the column name", () => {
    const automations = buildColumnAutomations("phase-0", {
      columnTitle: "Implement",
      columns: [IMPLEMENT_COLUMN],
      workOrders: [RUNNING_ORDER],
    });

    expect(automations).toEqual([
      expect.objectContaining({
        kind: "agent-step",
        name: "Implement",
        trigger: "On task in Implement",
        action: "Run the Implement agent",
        runningCount: 1,
        canvasId: "app-refund-implementer",
      }),
    ]);
  });

  it("returns no automations for a phase without an app", () => {
    expect(
      buildColumnAutomations("phase-1", {
        columnTitle: "Review",
        columns: [IMPLEMENT_COLUMN, EMPTY_PHASE],
      }),
    ).toEqual([]);
  });

  it("builds verify listeners from PR feedback handlers", () => {
    const automations = buildColumnAutomations("verify", {
      columnTitle: "Verify",
      prFeedbackHandlers: [DISCUSSION_HANDLER, CHECKS_HANDLER],
    });

    expect(automations[0]).toMatchObject({
      kind: "pr-discussion",
      trigger: "On pull request comment",
      action: "Address the feedback",
    });
    expect(automations[1]).toMatchObject({
      kind: "pr-checks",
      trigger: "On failing pull request check",
      action: "Fix the checks",
      health: "needs-repair",
    });
  });

  it("builds the Done closure automation", () => {
    const automations = buildColumnAutomations("done", {
      columnTitle: "Done",
      apps: [{ id: "app-pr-closure", name: "PR Closure" }],
    });

    expect(automations).toEqual([
      expect.objectContaining({
        kind: "pr-closure",
        trigger: "On pull request merged or closed",
        action: "Complete the task",
      }),
    ]);
  });
});

describe("catalogForColumn", () => {
  it("marks unique catalog entries as taken when they are already present", () => {
    const catalog = catalogForColumn("backlog");
    const automations = buildColumnAutomations("backlog", {
      columnTitle: "Backlog",
      intakes: [GITHUB_INTAKE],
      apps: [{ id: "app-refund-backlog", name: "Ingest" }],
    });

    expect(takenCatalogIds(automations, catalog)).toEqual(["github-issues", "analysis"]);
    expect(catalog.map((entry) => entry.id)).toEqual([
      "github-issues",
      "sentry-exceptions",
      "pagerduty-incidents",
      "analysis",
    ]);
  });

  it("keeps phase catalog entries available after one agent exists", () => {
    const catalog = catalogForColumn("phase-0");
    const automations = buildColumnAutomations("phase-0", {
      columnTitle: "Implement",
      columns: [IMPLEMENT_COLUMN],
    });

    expect(takenCatalogIds(automations, catalog)).toEqual([]);
    expect(catalog.map((entry) => entry.id)).toEqual(["agent-step", "custom"]);
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
