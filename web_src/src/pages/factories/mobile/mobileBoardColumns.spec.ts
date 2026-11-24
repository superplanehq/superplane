import { describe, expect, it } from "bun:test";

import type { FactoriesWorkOrder } from "@/api-client";

import { REFUND_FACTORY_APPS, REFUND_FACTORY_LINES, REFUND_LINE_PLAN_ID } from "../__fixtures__/factoryPageResponses";
import { activeColumnIndex, buildMobileBoardColumns } from "./mobileBoardColumns";

const PLAN_LINE = REFUND_FACTORY_LINES[0];

const BACKLOG_DRAFT: FactoriesWorkOrder = {
  id: "wo-draft",
  number: "11",
  title: "Draft in backlog",
  state: "STATE_DRAFT",
  lineDispatches: [],
};

const DONE_ORDER: FactoriesWorkOrder = {
  id: "wo-done",
  number: "12",
  title: "Finished task",
  state: "STATE_CLOSED",
  result: "RESULT_COMPLETED",
  lineDispatches: [{ id: "dispatch-done", line: { id: REFUND_LINE_PLAN_ID } }],
};

describe("buildMobileBoardColumns", () => {
  it("keeps detached stages and maps compacted steps to their original lanes", () => {
    const order: FactoriesWorkOrder = {
      id: "retained-task",
      state: "STATE_OPEN",
      lineDispatches: [
        {
          id: "dispatch",
          line: { id: REFUND_LINE_PLAN_ID },
          stepExecutions: [
            {
              id: "execution",
              stepIndex: 0,
              stageKey: "phase-1",
              state: "STATE_STARTED",
              run: { id: "run", appId: PLAN_LINE.steps![1].app!.app },
            },
          ],
        },
      ],
    };
    const columns = buildMobileBoardColumns(
      {
        ...PLAN_LINE,
        steps: [PLAN_LINE.steps![1]],
        stages: [
          { id: "empty", key: "phase-0", name: "Implement", kind: "STAGE_KIND_PHASE" },
          { id: "retained", key: "phase-1", name: "Release", kind: "STAGE_KIND_PHASE", stepIndex: 0 },
        ],
      },
      [order],
      REFUND_FACTORY_APPS,
    );
    expect(columns.map((column) => [column.key, column.title])).toEqual([
      ["phase-0", "Implement"],
      ["phase-1", "Release"],
    ]);
    expect(columns[0].cards).toEqual([]);
    expect(columns[0].glyph).toBeUndefined();
    expect(columns[1].cards.map((card) => card.order.id)).toEqual(["retained-task"]);
  });
  it("lays out Backlog, each line phase, Verify, and Done as one flat swipe order", () => {
    const columns = buildMobileBoardColumns(PLAN_LINE, [BACKLOG_DRAFT, DONE_ORDER], REFUND_FACTORY_APPS);

    expect(columns.map((column) => column.key)).toEqual(["backlog", "phase-0", "phase-1", "verify", "done"]);
    expect(columns.map((column) => column.paging)).toEqual(["backlog", "open", "open", "open", "done"]);
    expect(columns[0].cards.map((card) => card.order.id)).toEqual(["wo-draft"]);
    expect(columns[4].cards.map((card) => card.order.id)).toEqual(["wo-done"]);
    expect(columns[1].cards).toEqual([]);
  });

  it("gives every column an empty-state line so a blank column still reads on a phone", () => {
    const columns = buildMobileBoardColumns(PLAN_LINE, [], REFUND_FACTORY_APPS);

    for (const column of columns) {
      expect(column.emptyDescription).toContain(column.title);
    }
  });

  it("renders saved custom stages in the line stage order", () => {
    const columns = buildMobileBoardColumns(
      {
        ...PLAN_LINE,
        stages: [
          { id: "backlog", key: "backlog", name: "Incoming", kind: "STAGE_KIND_BACKLOG" },
          { id: "phase-0", key: "phase-0", name: "Build", kind: "STAGE_KIND_PHASE", stepIndex: 0 },
          { id: "custom", key: "custom-qa", name: "QA", kind: "STAGE_KIND_CUSTOM" },
          { id: "phase-1", key: "phase-1", name: "Release", kind: "STAGE_KIND_PHASE", stepIndex: 1 },
          { id: "verify", key: "verify", name: "Check", kind: "STAGE_KIND_VERIFY" },
          { id: "done", key: "done", name: "Shipped", kind: "STAGE_KIND_DONE" },
        ],
      },
      [],
      REFUND_FACTORY_APPS,
    );

    expect(columns.map((column) => [column.key, column.title])).toEqual([
      ["backlog", "Incoming"],
      ["phase-0", "Build"],
      ["custom-qa", "QA"],
      ["phase-1", "Release"],
      ["verify", "Check"],
      ["done", "Shipped"],
    ]);
    expect(columns[2].emptyDescription).toBe("No tasks in this stage.");
  });

  it("assigns the server total to Backlog and Done", () => {
    const columns = buildMobileBoardColumns(PLAN_LINE, [BACKLOG_DRAFT, DONE_ORDER], REFUND_FACTORY_APPS, {
      backlog: 40,
      done: 80,
    });

    expect(columns.find((c) => c.key === "backlog")?.totalCount).toBe(40);
    expect(columns.find((c) => c.key === "done")?.totalCount).toBe(80);
    expect(columns.find((c) => c.key === "verify")?.totalCount).toBeUndefined();
  });

  it("does not use the factory open count when Verify has no stages", () => {
    const columns = buildMobileBoardColumns(
      { ...PLAN_LINE, steps: [] },
      [BACKLOG_DRAFT, DONE_ORDER],
      REFUND_FACTORY_APPS,
      { backlog: 10, done: 20 },
    );

    expect(columns.find((c) => c.key === "verify")?.cards).toEqual([]);
    expect(columns.find((c) => c.key === "verify")?.totalCount).toBeUndefined();
  });

  it("adds an open task on a Done step to the closed total", () => {
    const line = {
      ...PLAN_LINE,
      steps: [...(PLAN_LINE.steps ?? []), { app: { app: "app-pr-closure" } }],
    };
    const openOnDone: FactoriesWorkOrder = {
      id: "wo-open-done",
      title: "Closing",
      state: "STATE_OPEN",
      lineDispatches: [
        {
          id: "dispatch-open-done",
          line: { id: REFUND_LINE_PLAN_ID },
          stepExecutions: [
            {
              id: "e-done",
              step: "Done",
              stepIndex: 2,
              state: "STATE_STARTED",
              createdAt: "2026-08-11T13:00:00.000Z",
              updatedAt: "2026-08-11T13:00:00.000Z",
              run: { id: "run-done", appId: "app-pr-closure" },
            },
          ],
        },
      ],
    };

    const columns = buildMobileBoardColumns(line, [openOnDone], REFUND_FACTORY_APPS, { done: 0 });

    expect(columns.find((c) => c.key === "done")?.cards.map((card) => card.order.id)).toEqual(["wo-open-done"]);
    expect(columns.find((c) => c.key === "done")?.totalCount).toBe(1);
  });

  it("keeps unloaded closed tasks in the Done total when an open Done-step task is visible", () => {
    const line = {
      ...PLAN_LINE,
      steps: [...(PLAN_LINE.steps ?? []), { app: { app: "app-pr-closure" } }],
    };
    const openOnDone: FactoriesWorkOrder = {
      id: "wo-open-done",
      title: "Closing",
      state: "STATE_OPEN",
      lineDispatches: [
        {
          id: "dispatch-open-done",
          line: { id: REFUND_LINE_PLAN_ID },
          stepExecutions: [
            {
              id: "e-done",
              step: "Done",
              stepIndex: 2,
              state: "STATE_STARTED",
              run: { id: "run-done", appId: "app-pr-closure" },
            },
          ],
        },
      ],
    };

    const columns = buildMobileBoardColumns(line, [DONE_ORDER, openOnDone], REFUND_FACTORY_APPS, { done: 80 });

    expect(columns.find((c) => c.key === "done")?.totalCount).toBe(81);
  });
});

describe("activeColumnIndex", () => {
  it("snaps to the nearest column and clamps to the board", () => {
    expect(activeColumnIndex(0, 390, 5)).toBe(0);
    expect(activeColumnIndex(390 * 2 + 100, 390, 5)).toBe(2);
    expect(activeColumnIndex(390 * 2 + 200, 390, 5)).toBe(3);
    expect(activeColumnIndex(390 * 9, 390, 5)).toBe(4);
    expect(activeColumnIndex(-50, 390, 5)).toBe(0);
  });

  it("falls back to the first column when there is nothing to measure", () => {
    expect(activeColumnIndex(120, 0, 5)).toBe(0);
    expect(activeColumnIndex(120, 390, 0)).toBe(0);
  });
});
