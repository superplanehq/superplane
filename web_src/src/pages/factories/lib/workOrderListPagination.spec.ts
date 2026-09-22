import { describe, expect, it } from "bun:test";

import {
  factoryWorkOrdersPageKey,
  flattenWorkOrdersPages,
  getWorkOrdersNextPageParam,
  workOrderMatchesPageQuery,
  workOrderMatchesUser,
  workOrdersPageFromResponse,
  workOrdersPageQueryFromKey,
} from "./workOrderListPagination";

describe("workOrderListPagination", () => {
  it("stops when the page has no further rows", () => {
    expect(
      getWorkOrdersNextPageParam({
        orders: [{ id: "wo-1" }],
        hasNextPage: false,
      }),
    ).toBeUndefined();
  });

  it("uses the last row as the next keyset cursor", () => {
    expect(
      getWorkOrdersNextPageParam({
        orders: [{ id: "wo-2" }, { id: "wo-1" }],
        hasNextPage: true,
      }),
    ).toEqual({ beforeId: "wo-1" });
  });

  it("flattens loaded pages in order", () => {
    expect(
      flattenWorkOrdersPages([
        { orders: [{ id: "wo-2" }], hasNextPage: true },
        { orders: [{ id: "wo-1" }], hasNextPage: false },
      ]).map((order) => order.id),
    ).toEqual(["wo-2", "wo-1"]);
  });

  it("maps a list response onto a page", () => {
    expect(workOrdersPageFromResponse({ orders: [{ id: "wo-1" }], hasNextPage: true })).toEqual({
      orders: [{ id: "wo-1" }],
      hasNextPage: true,
    });
  });

  it("stores user and unassigned on the page key", () => {
    const key = factoryWorkOrdersPageKey("org-1", "factory-1", ["STATE_DRAFT"], {
      userId: "user-1",
      unassigned: true,
    });
    expect(workOrdersPageQueryFromKey(key)).toEqual({
      userId: "user-1",
      unassigned: true,
    });
    expect(workOrdersPageQueryFromKey(factoryWorkOrdersPageKey("org-1", "factory-1", ["STATE_DRAFT"]))).toEqual({
      userId: undefined,
      unassigned: false,
    });
  });

  it("matches a user on assignee or creator", () => {
    expect(workOrderMatchesUser({ assignees: [{ id: "me" }] }, "me")).toBe(true);
    expect(workOrderMatchesUser({ createdBy: { user: { id: "me" } }, assignees: [] }, "me")).toBe(true);
    expect(workOrderMatchesUser({ assignees: [{ id: "other" }] }, "me")).toBe(false);
  });

  it("matches unassigned or the selected user", () => {
    expect(workOrderMatchesPageQuery({ assignees: [] }, { userId: "alex", unassigned: true })).toBe(true);
    expect(workOrderMatchesPageQuery({ assignees: [{ id: "alex" }] }, { userId: "alex", unassigned: true })).toBe(true);
    expect(workOrderMatchesPageQuery({ assignees: [{ id: "zoe" }] }, { userId: "alex", unassigned: true })).toBe(false);
  });
});
