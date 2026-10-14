import type { FactoriesFactoryLine, FactoriesWorkOrder } from "@/api-client";

import type { ColumnKey } from "../lib/columnAutomations";
import {
  buildLinePhaseBoard,
  collectLineBacklogOrders,
  collectLineDoneOrders,
  collectLineVerifyOrders,
  resolveColumnGlyph,
  visibleLineStageColumns,
  type PhaseGlyphKind,
} from "../lib/linePhaseRuns";

export type MobileBoardPagingKey = "backlog" | "open" | "done";

export type MobileBoardColumnTotalCounts = {
  backlog?: number;
  done?: number;
};

export type MobileBoardCard = {
  /** Stable React key: the task id, or the execution id on a phase column. */
  key: string;
  order: FactoriesWorkOrder;
};

export type MobileBoardColumn = {
  key: ColumnKey;
  title: string;
  cards: MobileBoardCard[];
  /** Which paged query feeds this column, for load-more on scroll. */
  paging: MobileBoardPagingKey;
  /** Live state glyph on a phase column. Bookend columns have none. */
  glyph?: PhaseGlyphKind;
  emptyDescription: string;
  /** Total work orders in this column from the server, when known. */
  totalCount?: number;
};

function cardsForOrders(orders: FactoriesWorkOrder[]): MobileBoardCard[] {
  return orders.flatMap((order) => (order.id ? [{ key: order.id, order }] : []));
}

function doneColumnTotalCount(serverTotal: number | undefined, cards: MobileBoardCard[]): number | undefined {
  if (serverTotal === undefined) {
    return undefined;
  }
  const openOnDoneStep = cards.filter((card) => card.order.state !== "STATE_CLOSED").length;
  return serverTotal + openOnDoneStep;
}

/**
 * One flat list of columns for the phone board, in the same order as the
 * desktop line board: Backlog, each line phase, Verify, Done. The phone shows
 * one column at a time, so every column gets a title and an empty-state line.
 */
export function buildMobileBoardColumns(
  line: FactoriesFactoryLine,
  workOrders: FactoriesWorkOrder[],
  apps: Array<{ id?: string; name?: string; columnKey?: string }>,
  totalCounts?: MobileBoardColumnTotalCounts,
): MobileBoardColumn[] {
  const fullBoard = buildLinePhaseBoard(line, workOrders, apps);
  const verifyOrders = collectLineVerifyOrders(fullBoard, workOrders);
  const stageColumns = visibleLineStageColumns(fullBoard, verifyOrders);
  const backlogOrders = collectLineBacklogOrders(workOrders);
  const doneCards = cardsForOrders(collectLineDoneOrders(workOrders, line, fullBoard));

  return [
    {
      key: "backlog",
      title: "Backlog",
      cards: cardsForOrders(backlogOrders),
      paging: "backlog",
      emptyDescription: "No tasks in Backlog.",
      totalCount: totalCounts?.backlog,
    },
    ...stageColumns.map(
      (column): MobileBoardColumn => ({
        key: `phase-${column.stepIndex}`,
        title: column.stepName,
        cards: column.runs.map((run) => ({ key: run.executionId, order: run.order })),
        paging: "open",
        glyph: resolveColumnGlyph(column),
        emptyDescription: `No tasks in ${column.stepName}.`,
      }),
    ),
    {
      key: "verify",
      title: "Verify",
      cards: cardsForOrders(verifyOrders),
      paging: "open",
      emptyDescription: "No tasks in Verify.",
    },
    {
      key: "done",
      title: "Done",
      cards: doneCards,
      paging: "done",
      emptyDescription: "No tasks in Done.",
      totalCount: doneColumnTotalCount(totalCounts?.done, doneCards),
    },
  ];
}

/** Index of the column currently snapped into view, from the carousel scroll offset. */
export function activeColumnIndex(scrollLeft: number, columnWidth: number, columnCount: number): number {
  if (columnWidth <= 0 || columnCount <= 0) {
    return 0;
  }
  const index = Math.round(scrollLeft / columnWidth);
  return Math.min(Math.max(index, 0), columnCount - 1);
}
