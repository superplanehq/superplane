import type { FactoriesWorkOrder } from "@/api-client";
import { QueryClient, type InfiniteData } from "@tanstack/react-query";
import { describe, expect, it } from "bun:test";

import { factoryWorkOrdersPageKey, type WorkOrdersPage } from "@/pages/factories/lib/workOrderListPagination";

import {
  patchCachedWorkOrderList,
  patchCachedWorkOrderPages,
  removeWorkOrderFromListCaches,
} from "./workOrderListCache";

function pages(
  orders: Array<{ id: string; title?: string; state?: FactoriesWorkOrder["state"] }>,
): InfiniteData<WorkOrdersPage> {
  return {
    pageParams: [undefined],
    pages: [{ orders, hasNextPage: true }],
  };
}

describe("patchCachedWorkOrderList", () => {
  it("replaces a matching row", () => {
    const next = patchCachedWorkOrderList(
      [
        { id: "wo-1", title: "Old" },
        { id: "wo-2", title: "Other" },
      ],
      "wo-1",
      {
        id: "wo-1",
        title: "New",
        checks: [],
      },
    );
    expect(next?.[0]).toMatchObject({ id: "wo-1", title: "New" });
    expect(next?.[1]).toEqual({ id: "wo-2", title: "Other" });
  });

  it("copies pull requests from the described task", () => {
    const next = patchCachedWorkOrderList([{ id: "wo-1", title: "Old", pullRequests: [] }], "wo-1", {
      id: "wo-1",
      title: "New",
      checks: [],
      pullRequests: [
        {
          id: "pr-1",
          workOrderId: "wo-1",
          number: "12",
          url: "https://github.com/acme/app/pull/12",
          state: "STATE_OPEN",
        },
      ],
    });
    expect(next?.[0]?.pullRequests).toEqual([
      {
        id: "pr-1",
        workOrderId: "wo-1",
        number: "12",
        url: "https://github.com/acme/app/pull/12",
        state: "STATE_OPEN",
      },
    ]);
  });

  it("prepends a missing row", () => {
    const next = patchCachedWorkOrderList([{ id: "wo-2", title: "Other" }], "wo-1", {
      id: "wo-1",
      title: "New",
      checks: [],
    });
    expect(next?.map((order) => order.id)).toEqual(["wo-1", "wo-2"]);
  });
});

describe("patchCachedWorkOrderPages", () => {
  it("updates a row that belongs on this page", () => {
    const next = patchCachedWorkOrderPages(
      pages([{ id: "wo-1", title: "Old", state: "STATE_DRAFT" }]),
      "wo-1",
      { id: "wo-1", title: "New", state: "STATE_DRAFT", checks: [] },
      ["STATE_DRAFT"],
    );
    expect(next?.pages[0]?.orders[0]).toMatchObject({ id: "wo-1", title: "New" });
  });

  it("prepends a new row onto the first page", () => {
    const next = patchCachedWorkOrderPages(
      pages([{ id: "wo-2", title: "Other", state: "STATE_DRAFT" }]),
      "wo-1",
      { id: "wo-1", title: "New", state: "STATE_DRAFT", checks: [] },
      ["STATE_DRAFT"],
    );
    expect(next?.pages[0]?.orders.map((order) => order.id)).toEqual(["wo-1", "wo-2"]);
  });

  it("does not prepend a row that is outside the user filter", () => {
    const next = patchCachedWorkOrderPages(
      pages([{ id: "wo-2", title: "Mine", state: "STATE_DRAFT" }]),
      "wo-1",
      { id: "wo-1", title: "Other", state: "STATE_DRAFT", checks: [], assignees: [{ id: "other" }] },
      ["STATE_DRAFT"],
      { userId: "me", unassigned: false },
    );
    expect(next?.pages[0]?.orders.map((order) => order.id)).toEqual(["wo-2"]);
  });

  it("removes a row that left this state", () => {
    const next = patchCachedWorkOrderPages(
      pages([{ id: "wo-1", title: "Old", state: "STATE_DRAFT" }]),
      "wo-1",
      { id: "wo-1", title: "New", state: "STATE_OPEN", checks: [] },
      ["STATE_DRAFT"],
    );
    expect(next?.pages[0]?.orders).toEqual([]);
  });
});

describe("removeWorkOrderFromListCaches", () => {
  it("removes the task from the full list and from paged lists", () => {
    const queryClient = new QueryClient();
    const listKey = ["factories", "org-1", "factory-1", "work-orders"] as const;
    const pageKey = factoryWorkOrdersPageKey("org-1", "factory-1", ["STATE_OPEN"]);
    queryClient.setQueryData(listKey, [
      { id: "wo-1", title: "Gone" },
      { id: "wo-2", title: "Other" },
    ]);
    queryClient.setQueryData(
      pageKey,
      pages([
        { id: "wo-1", title: "Gone" },
        { id: "wo-2", title: "Other" },
      ]),
    );

    removeWorkOrderFromListCaches(queryClient, "org-1", "factory-1", "wo-1");

    expect(queryClient.getQueryData(listKey)).toEqual([{ id: "wo-2", title: "Other" }]);
    expect(queryClient.getQueryData<InfiniteData<WorkOrdersPage>>(pageKey)?.pages[0]?.orders).toEqual([
      { id: "wo-2", title: "Other" },
    ]);
  });
});
