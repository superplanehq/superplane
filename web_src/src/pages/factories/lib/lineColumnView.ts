import type { FactoriesWorkOrder } from "@/api-client";

import { confidenceScoreFromChecks } from "./confidenceScore";
import type { LinePhaseRunCard } from "./linePhaseRuns";
import { workOrderListSource } from "./workOrderCardSource";

export const LINE_COLUMN_SORT_KEYS = ["updated", "confidence", "source", "age"] as const;

export type LineColumnSortKey = (typeof LINE_COLUMN_SORT_KEYS)[number];

/** First direction is newest, highest, A to Z, or oldest. Reverse flips that order. */
export type LineColumnSortDirection = "forward" | "reverse";

export const LINE_COLUMN_AGE_WINDOWS = ["any", "7", "30", "90", "older"] as const;

export type LineColumnAgeWindow = (typeof LINE_COLUMN_AGE_WINDOWS)[number];

export type LineColumnViewChoice = {
  sortKey: LineColumnSortKey;
  direction: LineColumnSortDirection;
  sourceIds: readonly string[];
  minimumConfidence?: number;
  noScore: boolean;
  ageWindow: LineColumnAgeWindow;
};

export type ColumnPageLoad = {
  hasMore: boolean;
  isLoading: boolean;
  isError: boolean;
};

export type ColumnViewReadiness = "idle" | "pending" | "ready";

const MS_PER_DAY = 24 * 60 * 60 * 1000;

export function defaultLineColumnView(): LineColumnViewChoice {
  return {
    sortKey: "updated",
    direction: "forward",
    sourceIds: [],
    noScore: false,
    ageWindow: "any",
  };
}

export function isDefaultLineColumnView(choice: LineColumnViewChoice): boolean {
  return (
    choice.sortKey === "updated" &&
    choice.direction === "forward" &&
    choice.sourceIds.length === 0 &&
    choice.minimumConfidence == null &&
    !choice.noScore &&
    choice.ageWindow === "any"
  );
}

export function withLineColumnSort(choice: LineColumnViewChoice, sortKey: LineColumnSortKey): LineColumnViewChoice {
  return { ...choice, sortKey };
}

export function withLineColumnDirection(
  choice: LineColumnViewChoice,
  direction: LineColumnSortDirection,
): LineColumnViewChoice {
  return { ...choice, direction };
}

export function toggleLineColumnSource(choice: LineColumnViewChoice, sourceId: string): LineColumnViewChoice {
  const sourceIds = choice.sourceIds.includes(sourceId)
    ? choice.sourceIds.filter((id) => id !== sourceId)
    : [...choice.sourceIds, sourceId];
  return { ...choice, sourceIds };
}

export function withLineColumnSources(
  choice: LineColumnViewChoice,
  sourceIds: readonly string[],
): LineColumnViewChoice {
  return { ...choice, sourceIds };
}

export function withLineColumnMinimum(
  choice: LineColumnViewChoice,
  minimumConfidence: number | undefined,
): LineColumnViewChoice {
  if (minimumConfidence == null) {
    return { ...choice, minimumConfidence: undefined };
  }
  return { ...choice, minimumConfidence, noScore: false };
}

export function withLineColumnNoScore(choice: LineColumnViewChoice, noScore: boolean): LineColumnViewChoice {
  if (!noScore) {
    return { ...choice, noScore: false };
  }
  return { ...choice, noScore: true, minimumConfidence: undefined };
}

export function withLineColumnAge(choice: LineColumnViewChoice, ageWindow: LineColumnAgeWindow): LineColumnViewChoice {
  return { ...choice, ageWindow };
}

export function columnViewPageLoads(
  column: "phase" | "verify" | "done",
  pages: { open: ColumnPageLoad; done: ColumnPageLoad },
): ColumnPageLoad[] {
  if (column === "done") {
    return [pages.open, pages.done];
  }
  return [pages.open];
}

export function columnViewReadiness(
  choice: LineColumnViewChoice,
  pages: readonly ColumnPageLoad[],
): ColumnViewReadiness {
  if (isDefaultLineColumnView(choice)) {
    return "idle";
  }
  if (pages.some((page) => page.isError)) {
    return "idle";
  }
  if (pages.some((page) => page.hasMore || page.isLoading)) {
    return "pending";
  }
  return "ready";
}

export function applyPhaseColumnView(
  runs: readonly LinePhaseRunCard[],
  choice: LineColumnViewChoice,
  now: Date,
): LinePhaseRunCard[] {
  return applyLineColumnView(
    runs.map((run) => ({
      ...run,
      order: run.order,
      updatedAt: run.execution.updatedAt ?? run.execution.createdAt,
      tieId: run.executionId,
    })),
    choice,
    now,
  );
}

export function applyOrderColumnView(
  orders: readonly FactoriesWorkOrder[],
  choice: LineColumnViewChoice,
  now: Date,
): FactoriesWorkOrder[] {
  return applyLineColumnView(
    orders.map((order) => ({
      order,
      updatedAt: order.updatedAt ?? order.createdAt,
      tieId: order.id ?? "",
    })),
    choice,
    now,
  ).map((card) => card.order);
}

export function applyLineColumnView<T extends { order: FactoriesWorkOrder; updatedAt?: string; tieId: string }>(
  cards: readonly T[],
  choice: LineColumnViewChoice,
  now: Date,
): T[] {
  const visible = cards.filter((card) => cardMatchesColumnView(card, choice, now.getTime()));
  if (isDefaultLineColumnView(choice)) {
    return [...visible];
  }
  return [...visible].sort((left, right) => compareColumnView(left, right, choice));
}

function cardMatchesColumnView(
  card: { order: FactoriesWorkOrder },
  choice: LineColumnViewChoice,
  nowMs: number,
): boolean {
  return (
    matchesSource(card.order, choice.sourceIds) &&
    matchesConfidence(card.order, choice) &&
    matchesAge(card.order, choice, nowMs)
  );
}

function matchesSource(order: FactoriesWorkOrder, sourceIds: readonly string[]): boolean {
  if (sourceIds.length === 0) {
    return true;
  }
  return sourceIds.includes(workOrderListSource(order).id);
}

function matchesConfidence(order: FactoriesWorkOrder, choice: LineColumnViewChoice): boolean {
  const score = listConfidenceScore(order);
  if (choice.noScore) {
    return score == null;
  }
  if (choice.minimumConfidence == null) {
    return true;
  }
  return score != null && score >= choice.minimumConfidence;
}

function matchesAge(order: FactoriesWorkOrder, choice: LineColumnViewChoice, nowMs: number): boolean {
  if (choice.ageWindow === "any") {
    return true;
  }
  const createdAt = createdAtMs(order);
  const olderThan = nowMs - 90 * MS_PER_DAY;
  if (choice.ageWindow === "older") {
    return createdAt < olderThan;
  }
  const days = Number(choice.ageWindow);
  return createdAt >= nowMs - days * MS_PER_DAY;
}

function compareColumnView(
  left: { order: FactoriesWorkOrder; updatedAt?: string; tieId: string },
  right: { order: FactoriesWorkOrder; updatedAt?: string; tieId: string },
  choice: LineColumnViewChoice,
): number {
  if (choice.sortKey === "updated") {
    const newestFirst = compareNewestUpdate(left, right);
    return choice.direction === "forward" ? newestFirst : -newestFirst;
  }
  if (choice.sortKey === "confidence") {
    return compareConfidence(left, right, choice.direction);
  }
  if (choice.sortKey === "source") {
    return compareSource(left, right, choice.direction);
  }
  return compareAge(left, right, choice.direction);
}

function compareConfidence(
  left: { order: FactoriesWorkOrder; updatedAt?: string; tieId: string },
  right: { order: FactoriesWorkOrder; updatedAt?: string; tieId: string },
  direction: LineColumnSortDirection,
): number {
  const leftScore = listConfidenceScore(left.order);
  const rightScore = listConfidenceScore(right.order);
  if ((leftScore == null) !== (rightScore == null)) {
    return leftScore == null ? 1 : -1;
  }
  if (leftScore != null && rightScore != null && leftScore !== rightScore) {
    const higherFirst = rightScore - leftScore;
    return direction === "forward" ? higherFirst : -higherFirst;
  }
  return compareNewestUpdate(left, right);
}

function compareSource(
  left: { order: FactoriesWorkOrder; updatedAt?: string; tieId: string },
  right: { order: FactoriesWorkOrder; updatedAt?: string; tieId: string },
  direction: LineColumnSortDirection,
): number {
  const delta = workOrderListSource(left.order).label.localeCompare(workOrderListSource(right.order).label);
  if (delta !== 0) {
    return direction === "forward" ? delta : -delta;
  }
  return compareNewestUpdate(left, right);
}

function compareAge(
  left: { order: FactoriesWorkOrder; updatedAt?: string; tieId: string },
  right: { order: FactoriesWorkOrder; updatedAt?: string; tieId: string },
  direction: LineColumnSortDirection,
): number {
  const delta = createdAtMs(left.order) - createdAtMs(right.order);
  if (delta !== 0) {
    return direction === "forward" ? delta : -delta;
  }
  return compareNewestUpdate(left, right);
}

function compareNewestUpdate(
  left: { updatedAt?: string; tieId: string },
  right: { updatedAt?: string; tieId: string },
): number {
  const leftTime = Date.parse(left.updatedAt ?? "") || 0;
  const rightTime = Date.parse(right.updatedAt ?? "") || 0;
  if (leftTime !== rightTime) {
    return rightTime - leftTime;
  }
  return right.tieId.localeCompare(left.tieId);
}

function listConfidenceScore(order: FactoriesWorkOrder): number | undefined {
  const listed = order as FactoriesWorkOrder & { checkScores?: Array<{ name?: string; score?: number }> };
  return confidenceScoreFromChecks(listed.checkScores);
}

function createdAtMs(order: FactoriesWorkOrder): number {
  return Date.parse(order.createdAt ?? "") || 0;
}
