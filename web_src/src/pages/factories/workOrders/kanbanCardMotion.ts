import type { FactoriesFactoryLine, FactoriesWorkOrder } from "@/api-client";
import { useLayoutEffect, useRef, useState } from "react";
import { flushSync } from "react-dom";

import {
  buildLinePhaseBoard,
  collectLineBacklogOrders,
  collectLineDoneOrders,
  collectLineVerifyOrders,
  visibleLineStageColumns,
} from "../lib/linePhaseRuns";
import { groupWorkOrderEntriesByLane, type WorkOrderListEntry } from "../lib/workOrderListModel";
import { WORK_ORDER_BOARD_LANES } from "../lib/workOrderProgress";

/** CSS class on `html` while a board view transition runs. */
export const KANBAN_BOARD_MOTION_ROOT_CLASS = "kanban-board-motion";

type KanbanMotionRoot = {
  classList: {
    add: (token: string) => void;
    remove: (token: string) => void;
  };
};

let kanbanMotionRootGeneration = 0;

function defaultKanbanMotionRoot(): KanbanMotionRoot | undefined {
  return typeof document === "undefined" ? undefined : document.documentElement;
}

/**
 * Marks the document root for board motion CSS. Returns a generation so an
 * older transition cannot remove the class while a newer one is still running.
 */
export function beginKanbanMotionRoot(root: KanbanMotionRoot | undefined = defaultKanbanMotionRoot()): number {
  const generation = ++kanbanMotionRootGeneration;
  root?.classList.add(KANBAN_BOARD_MOTION_ROOT_CLASS);
  return generation;
}

/** Removes the motion class only when this generation is still the latest. */
export function endKanbanMotionRoot(
  generation: number,
  root: KanbanMotionRoot | undefined = defaultKanbanMotionRoot(),
): void {
  if (generation !== kanbanMotionRootGeneration) {
    return;
  }
  root?.classList.remove(KANBAN_BOARD_MOTION_ROOT_CLASS);
}

/** View-transition type, for browsers that support typed transitions. */
export const KANBAN_BOARD_TRANSITION_TYPE = "kanban-board";

/** `view-transition-class` on each animated card. */
export const KANBAN_CARD_TRANSITION_CLASS = "kanban-card";

/** Skip motion when more cards than this enter, leave, or change column. */
export const MAX_ANIMATED_KANBAN_CHANGES = 8;

export type KanbanCardPlacement = {
  id: string;
  column: string;
};

/**
 * CSS custom-idents cannot start with a digit. Work-order ids are often
 * UUIDs, so every name gets a stable prefix.
 */
export function kanbanViewTransitionName(id: string): string {
  const safe = id.replace(/[^a-zA-Z0-9_-]/g, "_");
  return `kanban-wo-${safe}`;
}

export function kanbanBoardSignature(placements: KanbanCardPlacement[]): string {
  return placements.map((placement) => `${placement.id}:${placement.column}`).join("|");
}

export function countKanbanMembershipChanges(previous: KanbanCardPlacement[], next: KanbanCardPlacement[]): number {
  const previousById = new Map(previous.map((placement) => [placement.id, placement.column]));
  const nextById = new Map(next.map((placement) => [placement.id, placement.column]));
  let changes = 0;
  for (const [id, column] of nextById) {
    const previousColumn = previousById.get(id);
    if (previousColumn === undefined || previousColumn !== column) {
      changes += 1;
    }
  }
  for (const id of previousById.keys()) {
    if (!nextById.has(id)) {
      changes += 1;
    }
  }
  return changes;
}

export function prefersKanbanReducedMotion(
  matchMedia: ((query: string) => MediaQueryList) | undefined = typeof window !== "undefined" &&
  typeof window.matchMedia === "function"
    ? window.matchMedia.bind(window)
    : undefined,
): boolean {
  return Boolean(matchMedia?.("(prefers-reduced-motion: reduce)").matches);
}

export function canStartKanbanViewTransition(
  doc: Pick<Document, "startViewTransition"> | undefined = typeof document === "undefined" ? undefined : document,
): boolean {
  return typeof doc?.startViewTransition === "function";
}

export function shouldAnimateKanbanBoard(
  previous: KanbanCardPlacement[],
  next: KanbanCardPlacement[],
  options?: {
    reducedMotion?: boolean;
    canStartViewTransition?: boolean;
    maxChanges?: number;
  },
): boolean {
  const canStart = options?.canStartViewTransition ?? canStartKanbanViewTransition();
  if (!canStart) {
    return false;
  }
  if (options?.reducedMotion ?? prefersKanbanReducedMotion()) {
    return false;
  }
  return countKanbanMembershipChanges(previous, next) <= (options?.maxChanges ?? MAX_ANIMATED_KANBAN_CHANGES);
}

function pushOrderPlacements(placements: KanbanCardPlacement[], column: string, orders: Array<{ id?: string }>): void {
  for (const order of orders) {
    if (order.id) {
      placements.push({ id: order.id, column });
    }
  }
}

/** Left-to-right, top-to-bottom card membership for the Lines phase board. */
export function lineBoardCardPlacements(
  line: FactoriesFactoryLine,
  workOrders: FactoriesWorkOrder[],
  apps: Array<{ id?: string; name?: string }> = [],
): KanbanCardPlacement[] {
  const fullBoard = buildLinePhaseBoard(line, workOrders, apps);
  const verifyOrders = collectLineVerifyOrders(fullBoard);
  const stageColumns = visibleLineStageColumns(fullBoard, verifyOrders);
  const placements: KanbanCardPlacement[] = [];
  pushOrderPlacements(placements, "backlog", collectLineBacklogOrders(workOrders));
  for (const column of stageColumns) {
    for (const run of column.runs) {
      if (run.workOrderId) {
        placements.push({ id: run.workOrderId, column: `phase-${column.stepIndex}` });
      }
    }
  }
  pushOrderPlacements(placements, "verify", verifyOrders);
  pushOrderPlacements(placements, "done", collectLineDoneOrders(workOrders, line, fullBoard));
  return placements;
}

/** Left-to-right, top-to-bottom card membership for the Tasks status board. */
export function tasksBoardCardPlacements(entries: WorkOrderListEntry[]): KanbanCardPlacement[] {
  const grouped = groupWorkOrderEntriesByLane(entries);
  const placements: KanbanCardPlacement[] = [];
  for (const lane of WORK_ORDER_BOARD_LANES) {
    for (const entry of grouped.get(lane.id) ?? []) {
      if (entry.id) {
        placements.push({ id: entry.id, column: lane.id });
      }
    }
  }
  return placements;
}

type DisplayedBoard<T> = {
  value: T;
  placements: KanbanCardPlacement[];
  signature: string;
};

/**
 * Holds the previous board snapshot until a view transition can capture it,
 * then commits the incoming snapshot. Content-only updates skip motion.
 */
export function useKanbanDisplayedBoard<T>(incoming: T, placements: KanbanCardPlacement[]): T {
  const signature = kanbanBoardSignature(placements);
  const displayedRef = useRef<DisplayedBoard<T>>({ value: incoming, placements, signature });
  const [, setTick] = useState(0);

  if (signature === displayedRef.current.signature) {
    displayedRef.current = { value: incoming, placements, signature };
  }

  useLayoutEffect(() => {
    if (signature === displayedRef.current.signature) {
      return;
    }

    const previousPlacements = displayedRef.current.placements;
    const next: DisplayedBoard<T> = { value: incoming, placements, signature };
    let cancelled = false;
    const commit = () => {
      if (cancelled) {
        return;
      }
      displayedRef.current = next;
      setTick((tick) => tick + 1);
    };

    const animate = shouldAnimateKanbanBoard(previousPlacements, placements);
    if (!animate) {
      commit();
      return;
    }

    const motionGeneration = beginKanbanMotionRoot();
    const clearRoot = () => {
      endKanbanMotionRoot(motionGeneration);
    };

    const update = () => {
      flushSync(commit);
    };

    try {
      document.activeViewTransition?.skipTransition();
      let transition: ViewTransition;
      try {
        transition = document.startViewTransition({
          types: [KANBAN_BOARD_TRANSITION_TYPE],
          update,
        });
      } catch {
        transition = document.startViewTransition(update);
      }
      void transition.finished.finally(clearRoot);
      return () => {
        cancelled = true;
        clearRoot();
      };
    } catch {
      clearRoot();
      commit();
    }
  }, [incoming, placements, signature]);

  return displayedRef.current.value;
}
