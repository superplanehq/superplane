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

  it("assigns totalCount to backlog and done columns and verify when no stages exist", () => {
    const columns = buildMobileBoardColumns(PLAN_LINE, [BACKLOG_DRAFT, DONE_ORDER], REFUND_FACTORY_APPS, {
      backlog: 40,
      open: 30,
      done: 80,
    });

    expect(columns.find((c) => c.key === "backlog")?.totalCount).toBe(40);
    expect(columns.find((c) => c.key === "done")?.totalCount).toBe(80);
    expect(columns.find((c) => c.key === "verify")?.totalCount).toBeUndefined();
  });

  it("assigns open totalCount to verify when line has no steps", () => {
    const columns = buildMobileBoardColumns(
      { ...PLAN_LINE, steps: [] },
      [BACKLOG_DRAFT, DONE_ORDER],
      REFUND_FACTORY_APPS,
      { backlog: 10, open: 5, done: 20 },
    );

    expect(columns.find((c) => c.key === "verify")?.totalCount).toBe(5);
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
