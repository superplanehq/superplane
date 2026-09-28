import { describe, expect, it } from "bun:test";

import {
  boardDoneResultsForStatuses,
  backlogColumnPaging,
  draftsVisibleForStatusFilter,
  factoryWorkOrdersPageKey,
  flattenWorkOrdersPages,
  getWorkOrdersNextPageParam,
  shouldLoadHiddenArchivedPage,
  uniqueWorkOrdersById,
  workOrderMatchesPageQuery,
  workOrderMatchesUser,
  workOrdersPageFromResponse,
  workOrdersPageQueryFromKey,
} from "./workOrderListPagination";

describe("boardDoneResultsForStatuses", () => {
  it("keeps completed and failed when the board is not narrowed to one of them", () => {
    expect(boardDoneResultsForStatuses([])).toEqual(["RESULT_COMPLETED", "RESULT_FAILED"]);
    expect(boardDoneResultsForStatuses(["completed", "failed"])).toEqual(["RESULT_COMPLETED", "RESULT_FAILED"]);
    expect(boardDoneResultsForStatuses(["waiting"])).toEqual(["RESULT_COMPLETED", "RESULT_FAILED"]);
  });

  it("asks only for failed tasks when Failed is the status filter", () => {
    expect(boardDoneResultsForStatuses(["failed"])).toEqual(["RESULT_FAILED"]);
  });

  it("asks only for completed tasks when Completed is the status filter", () => {
    expect(boardDoneResultsForStatuses(["completed"])).toEqual(["RESULT_COMPLETED"]);
  });

  it("asks only for rejected tasks when Archived is the status filter", () => {
    expect(boardDoneResultsForStatuses(["archived"])).toEqual(["RESULT_REJECTED"]);
  });

  it("keeps the selected done result when Archived is also selected", () => {
    expect(boardDoneResultsForStatuses(["failed", "archived"])).toEqual(["RESULT_FAILED", "RESULT_REJECTED"]);
    expect(boardDoneResultsForStatuses(["completed", "failed", "archived"])).toEqual([
      "RESULT_COMPLETED",
      "RESULT_FAILED",
      "RESULT_REJECTED",
    ]);
  });
});

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

  it("keeps the first row when two pages share an id", () => {
    expect(
      flattenWorkOrdersPages([
        { orders: [{ id: "wo-1", title: "first" }], hasNextPage: true },
        { orders: [{ id: "wo-1", title: "second" }, { id: "wo-2" }], hasNextPage: false },
      ]).map((order) => order.id),
    ).toEqual(["wo-1", "wo-2"]);
    expect(
      uniqueWorkOrdersById([
        { id: "wo-1", title: "first" },
        { id: "wo-1", title: "second" },
      ]),
    ).toEqual([{ id: "wo-1", title: "first" }]);
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
      results: [],
      lineId: undefined,
    });
    expect(workOrdersPageQueryFromKey(factoryWorkOrdersPageKey("org-1", "factory-1", ["STATE_DRAFT"]))).toEqual({
      userId: undefined,
      unassigned: false,
      results: [],
      lineId: undefined,
    });
  });

  it("stores results on the page key", () => {
    const key = factoryWorkOrdersPageKey("org-1", "factory-1", ["STATE_CLOSED"], {
      results: ["RESULT_FAILED", "RESULT_COMPLETED"],
    });
    expect(workOrdersPageQueryFromKey(key)).toEqual({
      userId: undefined,
      unassigned: false,
      results: ["RESULT_COMPLETED", "RESULT_FAILED"],
      lineId: undefined,
    });
  });

  it("stores the line on the page key", () => {
    const key = factoryWorkOrdersPageKey("org-1", "factory-1", ["STATE_CLOSED"], {
      lineId: "line-1",
      results: ["RESULT_FAILED"],
    });
    expect(workOrdersPageQueryFromKey(key)).toEqual({
      userId: undefined,
      unassigned: false,
      results: ["RESULT_FAILED"],
      lineId: "line-1",
    });
  });

  it("matches a user on assignee or creator", () => {
    expect(workOrderMatchesUser({ assignees: [{ id: "me" }] }, "me")).toBe(true);
    expect(workOrderMatchesUser({ createdBy: { user: { id: "me" } }, assignees: [] }, "me")).toBe(true);
    expect(workOrderMatchesUser({ assignees: [{ id: "other" }] }, "me")).toBe(false);
  });

  it("matches unassigned or the selected user", () => {
    expect(workOrderMatchesPageQuery({ assignees: [] }, { userId: "alex", unassigned: true, results: [] })).toBe(true);
    expect(
      workOrderMatchesPageQuery({ assignees: [{ id: "alex" }] }, { userId: "alex", unassigned: true, results: [] }),
    ).toBe(true);
    expect(
      workOrderMatchesPageQuery({ assignees: [{ id: "zoe" }] }, { userId: "alex", unassigned: true, results: [] }),
    ).toBe(false);
  });
});

describe("backlogColumnPaging", () => {
  const drafts = { hasMore: true, isLoading: false, onLoadMore: () => undefined };
  const closed = { hasMore: true, isLoading: false, onLoadMore: () => undefined };

  it("pages drafts only when Archived is off", () => {
    expect(backlogColumnPaging({ includeArchived: false, showDrafts: true, drafts, closed })).toBe(drafts);
  });

  it("pages the closed query from Backlog when Archived is on", () => {
    let draftLoads = 0;
    let closedLoads = 0;
    const paging = backlogColumnPaging({
      includeArchived: true,
      showDrafts: false,
      drafts: {
        ...drafts,
        onLoadMore: () => {
          draftLoads += 1;
        },
      },
      closed: {
        ...closed,
        onLoadMore: () => {
          closedLoads += 1;
        },
      },
    });

    paging.onLoadMore();

    expect(draftLoads).toBe(0);
    expect(closedLoads).toBe(1);
    expect(paging.hasMore).toBe(true);
  });

  it("loads draft and closed pages together when both are visible", () => {
    let draftLoads = 0;
    let closedLoads = 0;
    const paging = backlogColumnPaging({
      includeArchived: true,
      showDrafts: true,
      drafts: {
        hasMore: true,
        isLoading: false,
        onLoadMore: () => {
          draftLoads += 1;
        },
      },
      closed: {
        hasMore: false,
        isLoading: true,
        onLoadMore: () => {
          closedLoads += 1;
        },
      },
    });

    paging.onLoadMore();

    expect(draftLoads).toBe(1);
    expect(closedLoads).toBe(0);
    expect(paging.hasMore).toBe(true);
    expect(paging.isLoading).toBe(true);
  });
});

describe("shouldLoadHiddenArchivedPage", () => {
  it("loads the next closed page while no archived task is visible", () => {
    expect(
      shouldLoadHiddenArchivedPage({
        includeArchived: true,
        visibleArchivedCount: 0,
        hasNextPage: true,
        isLoading: false,
        isError: false,
      }),
    ).toBe(true);
  });

  it("stops when an archived task is visible, a page is in flight, or the query failed", () => {
    expect(
      shouldLoadHiddenArchivedPage({
        includeArchived: true,
        visibleArchivedCount: 1,
        hasNextPage: true,
        isLoading: false,
        isError: false,
      }),
    ).toBe(false);
    expect(
      shouldLoadHiddenArchivedPage({
        includeArchived: true,
        visibleArchivedCount: 0,
        hasNextPage: true,
        isLoading: true,
        isError: false,
      }),
    ).toBe(false);
    expect(
      shouldLoadHiddenArchivedPage({
        includeArchived: true,
        visibleArchivedCount: 0,
        hasNextPage: true,
        isLoading: false,
        isError: true,
      }),
    ).toBe(false);
    expect(draftsVisibleForStatusFilter([])).toBe(true);
    expect(draftsVisibleForStatusFilter(["archived"])).toBe(false);
    expect(draftsVisibleForStatusFilter(["archived", "draft"])).toBe(true);
  });
});
