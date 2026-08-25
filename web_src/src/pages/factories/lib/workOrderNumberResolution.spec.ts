import { describe, expect, it } from "vitest";
import type { FactoriesWorkOrder } from "@/api-client";
import {
  canonicalWorkOrderNumber,
  findWorkOrderByRunId,
  latestDispatchForLine,
  resolveWorkOrderByNumber,
  workOrderRouteNeedsCanonicalRedirect,
} from "./workOrderNumberResolution";

const ORDERS: FactoriesWorkOrder[] = [
  { id: "order-1", number: "42", key: "SP-42", title: "Fix things" },
  { id: "order-2", number: "7", key: "SP-7", title: "Ship things" },
];

describe("resolveWorkOrderByNumber", () => {
  it("matches an exact number", () => {
    const resolution = resolveWorkOrderByNumber(ORDERS, "42", false);
    expect(resolution).toEqual({ status: "found", order: ORDERS[0], matchedBy: "number" });
  });

  it("tolerates leading zeros", () => {
    const resolution = resolveWorkOrderByNumber(ORDERS, "007", false);
    expect(resolution.status).toBe("found");
    expect(resolution.order).toBe(ORDERS[1]);
    expect(resolution.matchedBy).toBe("number");
  });

  it("falls back to a legacy id match when the segment isn't numeric", () => {
    const resolution = resolveWorkOrderByNumber(ORDERS, "order-2", false);
    expect(resolution).toEqual({ status: "found", order: ORDERS[1], matchedBy: "id" });
  });

  it("returns not-found once loaded and nothing matches", () => {
    expect(resolveWorkOrderByNumber(ORDERS, "999", false)).toEqual({
      status: "not-found",
      order: null,
      matchedBy: null,
    });
  });

  it("returns loading instead of not-found while the list is still fetching", () => {
    expect(resolveWorkOrderByNumber([], "42", true)).toEqual({ status: "loading", order: null, matchedBy: null });
  });

  it("rejects zero and negative numbers", () => {
    expect(resolveWorkOrderByNumber(ORDERS, "0", false).status).toBe("not-found");
    expect(resolveWorkOrderByNumber(ORDERS, "-1", false).status).toBe("not-found");
  });
});

describe("workOrderRouteNeedsCanonicalRedirect", () => {
  it("is false for an exact number match", () => {
    const resolution = resolveWorkOrderByNumber(ORDERS, "42", false);
    expect(workOrderRouteNeedsCanonicalRedirect(resolution, "42")).toBe(false);
  });

  it("is true for a leading-zero number", () => {
    const resolution = resolveWorkOrderByNumber(ORDERS, "007", false);
    expect(workOrderRouteNeedsCanonicalRedirect(resolution, "007")).toBe(true);
  });

  it("is true for a legacy id match", () => {
    const resolution = resolveWorkOrderByNumber(ORDERS, "order-2", false);
    expect(workOrderRouteNeedsCanonicalRedirect(resolution, "order-2")).toBe(true);
  });
});

describe("canonicalWorkOrderNumber", () => {
  it("returns the numeric string for a resolved order", () => {
    expect(canonicalWorkOrderNumber(ORDERS[0])).toBe("42");
  });

  it("returns null when there is no order or number", () => {
    expect(canonicalWorkOrderNumber(null)).toBeNull();
    expect(canonicalWorkOrderNumber({ id: "x" })).toBeNull();
  });
});

describe("findWorkOrderByRunId", () => {
  it("returns the order whose dispatch ran this canvas run", () => {
    const withRun: FactoriesWorkOrder = {
      id: "order-run",
      number: "9",
      lineDispatches: [
        {
          id: "d-old",
          stepExecutions: [{ id: "e-old", run: { id: "run-old" } }],
        },
        {
          id: "d-new",
          stepExecutions: [{ id: "e-new", run: { id: "run-new" } }],
        },
      ],
    };

    expect(findWorkOrderByRunId([ORDERS[0], withRun], "run-new")?.id).toBe("order-run");
    expect(findWorkOrderByRunId([withRun], "missing")).toBeUndefined();
    expect(findWorkOrderByRunId([withRun], "  ")).toBeUndefined();
  });
});

describe("latestDispatchForLine", () => {
  it("picks the newest dispatch on the viewed line", () => {
    const order: FactoriesWorkOrder = {
      id: "order-1",
      lineDispatches: [
        {
          id: "d-other",
          createdAt: "2026-08-21T10:00:00.000Z",
          line: { id: "line-other" },
          stepExecutions: [{ id: "e-other", run: { id: "run-other" } }],
        },
        {
          id: "d-old",
          createdAt: "2026-08-21T11:00:00.000Z",
          line: { id: "line-1" },
          stepExecutions: [{ id: "e-old", run: { id: "run-old" } }],
        },
        {
          id: "d-new",
          createdAt: "2026-08-21T12:00:00.000Z",
          line: { id: "line-1" },
          stepExecutions: [{ id: "e-new", run: { id: "run-new" } }],
        },
      ],
    };

    expect(latestDispatchForLine(order, "line-1")?.id).toBe("d-new");
    expect(latestDispatchForLine(order)?.id).toBe("d-new");
  });
});
