import type { FactoriesWorkOrder, FactoriesWorkOrderCheck } from "@/api-client";

import { findWorkOrderByRunId, resolveWorkOrderByNumber } from "../../lib/workOrderNumberResolution";
import { canvasKeyForPhase, parseSplitRunCanvasKey, type SplitRunCanvasKey } from "./splitRunCanvases";
import {
  SPLIT_RUN_RUNNING,
  splitRunFixtureForWorkOrder,
  type SplitRunFixture,
  type SplitRunPhase,
} from "./splitRunMocks";

export type SplitRunQuery = {
  from: string | null;
  lineId: string | null;
  runId: string | null;
  orderNumber: string | null;
  canvasKey?: SplitRunCanvasKey;
};

export function readSplitRunQuery(searchParams: URLSearchParams): SplitRunQuery {
  return {
    from: searchParams.get("from"),
    lineId: searchParams.get("lineId"),
    runId: searchParams.get("run"),
    orderNumber: searchParams.get("orderNumber") ?? searchParams.get("orderId"),
    canvasKey: parseSplitRunCanvasKey(searchParams.get("canvas")),
  };
}

export function resolveSplitRunOrder(
  workOrders: FactoriesWorkOrder[],
  orderNumber: string | null,
  runId: string | null,
  isLoading: boolean,
): FactoriesWorkOrder | null {
  const byNumber = resolveWorkOrderByNumber(workOrders, orderNumber ?? undefined, isLoading).order;
  return byNumber ?? findWorkOrderByRunId(workOrders, runId) ?? null;
}

export function fixtureForSplitRunPage(
  order: FactoriesWorkOrder | null,
  orderChecks: FactoriesWorkOrderCheck[],
  lineId: string | null,
  prFeedbackRunHref?: string,
): SplitRunFixture | null {
  if (!order) {
    return null;
  }
  return splitRunFixtureForWorkOrder(order, { checks: orderChecks, lineId, demoArtifacts: false, prFeedbackRunHref });
}

export function phaseForSplitRunCanvas(fixture: SplitRunFixture | null, canvasKey?: SplitRunCanvasKey): SplitRunPhase {
  const fallback = implementFallbackPhase();
  if (!fixture) {
    return fallback;
  }
  if (canvasKey) {
    return fixture.phases.find((entry) => canvasKeyForPhase(entry) === canvasKey) ?? fixture.phases[0] ?? fallback;
  }
  return fixture.phases.find((entry) => entry.id === fixture.currentPhaseId) ?? fixture.phases[0] ?? fallback;
}

/** Bind the route app. Keep the mapped phase run; do not reuse a leftover URL run. */
export function splitRunPhaseOnRoute(phase: SplitRunPhase, appId: string): SplitRunPhase {
  return {
    ...phase,
    appId: phase.appId ?? appId,
    runId: phase.runId,
  };
}

export function splitRunPageTitle(waitingForOrder: boolean, isLoading: boolean, canvasTitle: string): string {
  if (!waitingForOrder) {
    return canvasTitle;
  }
  return isLoading ? "Loading run" : "Run not found";
}

export function splitRunMissingCopy(isLoading: boolean, failed = false): { title: string; body: string } {
  if (isLoading && !failed) {
    return { title: "Loading run", body: "Loading this run." };
  }
  if (failed) {
    return { title: "Run not found", body: "SuperPlane cannot load this canvas or run." };
  }
  return { title: "Run not found", body: "This run is not on the workspace." };
}

function implementFallbackPhase(): SplitRunPhase {
  const phase = SPLIT_RUN_RUNNING.phases.find((entry) => entry.id === "implement");
  if (!phase) {
    throw new Error("SPLIT_RUN_RUNNING is missing the implement phase");
  }
  return phase;
}
