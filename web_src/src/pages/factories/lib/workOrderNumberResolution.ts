import type { FactoriesWorkOrder, FactoriesWorkOrderLineDispatch } from "@/api-client";
import { flattenWorkOrderExecutions } from "./workOrderExecutions";

export type WorkOrderResolutionStatus = "loading" | "found" | "not-found";

export interface WorkOrderResolution {
  status: WorkOrderResolutionStatus;
  order: FactoriesWorkOrder | null;
  /**
   * How the match was made. `"number"` is the canonical route identifier;
   * `"id"` is the legacy fallback for old links that carry the raw database
   * id. See `workOrderRouteNeedsCanonicalRedirect`.
   */
  matchedBy: "number" | "id" | null;
}

const LOADING: WorkOrderResolution = { status: "loading", order: null, matchedBy: null };
const NOT_FOUND: WorkOrderResolution = { status: "not-found", order: null, matchedBy: null };

function parsePositiveInteger(value: string): number | null {
  if (!/^\d+$/.test(value.trim())) {
    return null;
  }
  const parsed = Number(value);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : null;
}

/**
 * Resolves the `:orderNumber` route segment to a `FactoriesWorkOrder` using
 * the already-fetched work orders list (same query the sidebar's "recent
 * orders" section uses — no extra network round trip on a warm workspace).
 * Matches the factory-scoped `number` first (tolerating leading zeros, e.g.
 * `007` -> `7`), then falls back to `id` so old permalinks keep working.
 *
 * Returns `"loading"` (rather than `"not-found"`) whenever the list hasn't
 * resolved yet, so callers don't flash an error state on every deep link.
 */
export function resolveWorkOrderByNumber(
  orders: FactoriesWorkOrder[],
  routeNumber: string | undefined,
  isLoading: boolean,
): WorkOrderResolution {
  if (!routeNumber) {
    return isLoading ? LOADING : NOT_FOUND;
  }

  const parsedNumber = parsePositiveInteger(routeNumber);
  if (parsedNumber !== null) {
    const byNumber = orders.find((order) => {
      if (order.number === undefined || order.number === null || order.number === "") {
        return false;
      }
      return Number(order.number) === parsedNumber;
    });
    if (byNumber) {
      return { status: "found", order: byNumber, matchedBy: "number" };
    }
  }

  const byId = orders.find((order) => Boolean(order.id) && order.id === routeNumber);
  if (byId) {
    return { status: "found", order: byId, matchedBy: "id" };
  }

  return isLoading ? LOADING : NOT_FOUND;
}

/**
 * True when the route segment does not already match the canonical
 * `order.number` exactly — a legacy id, a leading-zero-padded number, or
 * any other non-canonical spelling. Callers `<Navigate replace>` to the
 * canonical URL in this case instead of rendering the page.
 */
export function workOrderRouteNeedsCanonicalRedirect(resolution: WorkOrderResolution, routeNumber: string): boolean {
  if (resolution.status !== "found" || !resolution.order || resolution.order.number === undefined) {
    return false;
  }
  return String(Number(resolution.order.number)) !== routeNumber;
}

/** Find the work order whose line dispatch ran this canvas run. */
export function findWorkOrderByRunId(
  orders: FactoriesWorkOrder[],
  runId: string | null | undefined,
): FactoriesWorkOrder | undefined {
  const id = runId?.trim();
  if (!id) {
    return undefined;
  }
  return orders.find((order) => flattenWorkOrderExecutions(order).some((execution) => execution.run?.id === id));
}

/** Latest dispatch on this line, or the latest dispatch on the order. */
export function latestDispatchForLine(
  order: FactoriesWorkOrder | undefined,
  lineId?: string | null,
): FactoriesWorkOrderLineDispatch | undefined {
  const dispatches = (order?.lineDispatches ?? []).filter((dispatch) => !lineId || dispatch.line?.id === lineId);
  if (dispatches.length === 0) {
    return undefined;
  }
  return dispatches.reduce((best, candidate) => {
    const bestAt = Date.parse(best.createdAt ?? "") || 0;
    const candidateAt = Date.parse(candidate.createdAt ?? "") || 0;
    return candidateAt >= bestAt ? candidate : best;
  });
}

/** Canonical route identifier for a resolved work order, or `null`. */
export function canonicalWorkOrderNumber(order: FactoriesWorkOrder | null): string | null {
  if (!order || order.number === undefined || order.number === null || order.number === "") {
    return null;
  }
  const parsed = Number(order.number);
  return Number.isFinite(parsed) ? String(parsed) : null;
}
