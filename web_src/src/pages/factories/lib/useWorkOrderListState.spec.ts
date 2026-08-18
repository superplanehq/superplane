import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it } from "vitest";

import { EMPTY_WORK_ORDER_FILTERS } from "./workOrderListModel";
import { useWorkOrderListState } from "./useWorkOrderListState";

describe("useWorkOrderListState", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  afterEach(() => {
    window.localStorage.clear();
  });

  it("defaults to board layout, updated ordering, and all scope", () => {
    const { result } = renderHook(() => useWorkOrderListState("factory-1"));
    expect(result.current.layout).toBe("board");
    expect(result.current.ordering).toBe("updated");
    expect(result.current.scope).toBe("all");
    expect(result.current.filterCount).toBe(0);
    expect(result.current.search).toBe("");
    expect(result.current.searchOpen).toBe(false);
    expect(result.current.hasActiveFilters).toBe(false);
  });

  it("persists layout and ordering across mounts", () => {
    const first = renderHook(() => useWorkOrderListState("factory-1"));
    act(() => {
      first.result.current.setLayout("table");
      first.result.current.setOrdering("spend");
    });
    first.unmount();

    const second = renderHook(() => useWorkOrderListState("factory-1"));
    expect(second.result.current.layout).toBe("table");
    expect(second.result.current.ordering).toBe("spend");
  });

  it("persists scope across mounts for the same factory", () => {
    const first = renderHook(() => useWorkOrderListState("factory-1"));
    act(() => {
      first.result.current.setScope("my");
    });
    first.unmount();

    const second = renderHook(() => useWorkOrderListState("factory-1"));
    expect(second.result.current.scope).toBe("my");
  });

  it("persists filters across mounts for the same factory", () => {
    const first = renderHook(() => useWorkOrderListState("factory-1"));
    act(() => {
      first.result.current.toggleFilter("statuses", "running");
      first.result.current.toggleFilter("lineIds", "line-a");
    });
    first.unmount();

    const second = renderHook(() => useWorkOrderListState("factory-1"));
    expect(second.result.current.filters.statuses).toEqual(["running"]);
    expect(second.result.current.filters.lineIds).toEqual(["line-a"]);
  });

  it("search stays session-local (not persisted)", () => {
    const first = renderHook(() => useWorkOrderListState("factory-1"));
    act(() => {
      first.result.current.setSearch("refund");
    });
    first.unmount();

    const second = renderHook(() => useWorkOrderListState("factory-1"));
    expect(second.result.current.search).toBe("");
  });

  it("does not leak scope/filters between different factories", () => {
    const factoryOne = renderHook(() => useWorkOrderListState("factory-1"));
    act(() => {
      factoryOne.result.current.setScope("my");
      factoryOne.result.current.toggleFilter("statuses", "running");
    });
    factoryOne.unmount();

    const factoryTwo = renderHook(() => useWorkOrderListState("factory-2"));
    expect(factoryTwo.result.current.scope).toBe("all");
    expect(factoryTwo.result.current.filters).toEqual(EMPTY_WORK_ORDER_FILTERS);
    factoryTwo.unmount();

    const factoryOneAgain = renderHook(() => useWorkOrderListState("factory-1"));
    expect(factoryOneAgain.result.current.scope).toBe("my");
    expect(factoryOneAgain.result.current.filters.statuses).toEqual(["running"]);
  });

  it("re-reads scope/filters when factoryId changes without an unmount", () => {
    window.localStorage.setItem("sp:work-orders:scope:factory-2", "my");
    window.localStorage.setItem(
      "sp:work-orders:filters:factory-2",
      JSON.stringify({ statuses: ["failed"], lineIds: [], assigneeIds: [] }),
    );

    const { result, rerender } = renderHook(({ factoryId }) => useWorkOrderListState(factoryId), {
      initialProps: { factoryId: "factory-1" },
    });
    act(() => {
      result.current.setScope("active");
    });
    expect(result.current.scope).toBe("active");

    rerender({ factoryId: "factory-2" });

    expect(result.current.scope).toBe("my");
    expect(result.current.filters.statuses).toEqual(["failed"]);
    expect(result.current.search).toBe("");
  });

  it("falls back to empty filters when storage holds malformed data", () => {
    window.localStorage.setItem("sp:work-orders:filters:factory-1", "not json");
    const malformed = renderHook(() => useWorkOrderListState("factory-1"));
    expect(malformed.result.current.filters).toEqual(EMPTY_WORK_ORDER_FILTERS);
    malformed.unmount();

    window.localStorage.setItem("sp:work-orders:filters:factory-1", JSON.stringify({ statuses: ["not-a-status"] }));
    const wrongShape = renderHook(() => useWorkOrderListState("factory-1"));
    expect(wrongShape.result.current.filters).toEqual(EMPTY_WORK_ORDER_FILTERS);
  });

  it("toggleFilter adds and removes values within one dimension", () => {
    const { result } = renderHook(() => useWorkOrderListState("factory-1"));
    act(() => {
      result.current.toggleFilter("statuses", "running");
      result.current.toggleFilter("statuses", "failed");
    });
    expect(result.current.filters.statuses).toEqual(["running", "failed"]);
    act(() => {
      result.current.toggleFilter("statuses", "running");
    });
    expect(result.current.filters.statuses).toEqual(["failed"]);
  });

  it("clearFilterDimension only clears the dimension it targets", () => {
    const { result } = renderHook(() => useWorkOrderListState("factory-1"));
    act(() => {
      result.current.toggleFilter("statuses", "running");
      result.current.toggleFilter("lineIds", "line-a");
    });
    expect(result.current.filterCount).toBe(2);
    act(() => {
      result.current.clearFilterDimension("statuses");
    });
    expect(result.current.filters.statuses).toEqual([]);
    expect(result.current.filters.lineIds).toEqual(["line-a"]);
  });

  it("closing search also clears the query", () => {
    const { result } = renderHook(() => useWorkOrderListState("factory-1"));
    act(() => {
      result.current.openSearch();
      result.current.setSearch("refund");
    });
    expect(result.current.searchOpen).toBe(true);
    act(() => {
      result.current.closeSearch();
    });
    expect(result.current.searchOpen).toBe(false);
    expect(result.current.search).toBe("");
  });

  it("resetView clears scope, filters, and search but keeps display preferences, and the reset sticks on reload", () => {
    const first = renderHook(() => useWorkOrderListState("factory-1"));
    act(() => {
      first.result.current.setLayout("list");
      first.result.current.setScope("my");
      first.result.current.toggleFilter("statuses", "running");
      first.result.current.setSearch("foo");
    });
    expect(first.result.current.hasActiveFilters).toBe(true);
    act(() => {
      first.result.current.resetView();
    });
    expect(first.result.current.hasActiveFilters).toBe(false);
    expect(first.result.current.layout).toBe("list");
    first.unmount();

    const second = renderHook(() => useWorkOrderListState("factory-1"));
    expect(second.result.current.scope).toBe("all");
    expect(second.result.current.filters).toEqual(EMPTY_WORK_ORDER_FILTERS);
    expect(second.result.current.layout).toBe("list");
  });
});
