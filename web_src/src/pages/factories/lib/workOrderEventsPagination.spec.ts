import { describe, expect, it } from "vitest";

import type { FactoriesListWorkOrderEventsResponse } from "@/api-client";

import { flattenWorkOrderEventsPages, getWorkOrderEventsNextPageParam } from "./workOrderEventsPagination";

describe("workOrderEventsPagination", () => {
  it("flattens events from all loaded pages", () => {
    const pages: FactoriesListWorkOrderEventsResponse[] = [
      { events: [{ type: "order.closed" }] },
      { events: [{ type: "order.opened" }] },
    ];

    expect(flattenWorkOrderEventsPages(pages)).toEqual([{ type: "order.closed" }, { type: "order.opened" }]);
  });

  it("returns lastTimestamp when more events remain", () => {
    const lastPage: FactoriesListWorkOrderEventsResponse = {
      events: [{ type: "step.execution.finished", timestamp: "2026-08-04T10:00:00.000Z" }],
      totalCount: 3,
      hasNextPage: true,
      lastTimestamp: "2026-08-04T10:00:00.000Z",
    };

    expect(getWorkOrderEventsNextPageParam(lastPage, [lastPage])).toBe("2026-08-04T10:00:00.000Z");
  });

  it("returns undefined when all events are loaded", () => {
    const page: FactoriesListWorkOrderEventsResponse = {
      events: [{ type: "order.opened" }],
      totalCount: 1,
      hasNextPage: false,
      lastTimestamp: "2026-08-04T10:00:00.000Z",
    };

    expect(getWorkOrderEventsNextPageParam(page, [page])).toBeUndefined();
  });
});
