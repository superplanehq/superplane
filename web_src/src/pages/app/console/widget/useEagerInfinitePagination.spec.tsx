import { renderHook } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { __resetEagerPaginationInFlight, useEagerInfinitePagination } from "./widgetPagination";

const BASE_ARGS = {
  enabled: true,
  fillTarget: 100,
  loadedRowCount: 25,
  pageCount: 1,
  hasNextPage: true as boolean | undefined,
  isFetchingNextPage: false,
  isFetching: false,
};

describe("useEagerInfinitePagination single-flight", () => {
  afterEach(() => {
    __resetEagerPaginationInFlight();
  });

  it("dispatches a single fetchNextPage across concurrent widget effects sharing a flightKey", () => {
    const fetchNextPage = vi.fn(() => new Promise<void>(() => {}));
    const flightKey = JSON.stringify(["canvases", "runs", "canvas-1", "infinite"]);

    renderHook(() => useEagerInfinitePagination({ ...BASE_ARGS, fetchNextPage, flightKey }));
    renderHook(() => useEagerInfinitePagination({ ...BASE_ARGS, fetchNextPage, flightKey }));
    renderHook(() => useEagerInfinitePagination({ ...BASE_ARGS, fetchNextPage, flightKey }));

    expect(fetchNextPage).toHaveBeenCalledTimes(1);
  });

  it("still allows concurrent fetchNextPage for different flightKeys", () => {
    const fetchA = vi.fn(() => new Promise<void>(() => {}));
    const fetchB = vi.fn(() => new Promise<void>(() => {}));

    renderHook(() => useEagerInfinitePagination({ ...BASE_ARGS, fetchNextPage: fetchA, flightKey: "canvas-1:{}" }));
    renderHook(() =>
      useEagerInfinitePagination({
        ...BASE_ARGS,
        fetchNextPage: fetchB,
        flightKey: "canvas-1:STATE_STARTED",
      }),
    );

    expect(fetchA).toHaveBeenCalledTimes(1);
    expect(fetchB).toHaveBeenCalledTimes(1);
  });

  it("releases the lock once the fetch settles so a subsequent effect can fetch again", async () => {
    let resolveFetch: () => void = () => {};
    const fetchNextPage = vi
      .fn()
      .mockImplementationOnce(
        () =>
          new Promise<void>((resolve) => {
            resolveFetch = resolve;
          }),
      )
      .mockImplementationOnce(() => new Promise<void>(() => {}));
    const flightKey = "canvas-1:{}";

    const first = renderHook(() => useEagerInfinitePagination({ ...BASE_ARGS, fetchNextPage, flightKey }));
    expect(fetchNextPage).toHaveBeenCalledTimes(1);

    // A second mount while the first is still in flight is deduped.
    renderHook(() => useEagerInfinitePagination({ ...BASE_ARGS, fetchNextPage, flightKey }));
    expect(fetchNextPage).toHaveBeenCalledTimes(1);

    resolveFetch();
    await new Promise((resolve) => setTimeout(resolve, 0));

    // After settle, a new mount can trigger a fresh fetch on the same key.
    renderHook(() => useEagerInfinitePagination({ ...BASE_ARGS, fetchNextPage, flightKey }));
    expect(fetchNextPage).toHaveBeenCalledTimes(2);

    first.unmount();
  });

  it("does not fetch or lock when disabled", () => {
    const fetchNextPage = vi.fn(() => new Promise<void>(() => {}));
    const flightKey = "canvas-1:{}";

    renderHook(() => useEagerInfinitePagination({ ...BASE_ARGS, enabled: false, fetchNextPage, flightKey }));

    expect(fetchNextPage).not.toHaveBeenCalled();
  });
});
