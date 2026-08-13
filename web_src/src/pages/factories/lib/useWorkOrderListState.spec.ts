import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it } from "vitest";

import { useWorkOrderListState } from "./useWorkOrderListState";

describe("useWorkOrderListState", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  afterEach(() => {
    window.localStorage.clear();
  });

  it("defaults to board layout, updated ordering, and all scope", () => {
    const { result } = renderHook(() => useWorkOrderListState());
    expect(result.current.layout).toBe("board");
    expect(result.current.ordering).toBe("updated");
    expect(result.current.scope).toBe("all");
    expect(result.current.filterCount).toBe(0);
    expect(result.current.search).toBe("");
    expect(result.current.searchOpen).toBe(false);
    expect(result.current.hasActiveFilters).toBe(false);
  });

  it("persists layout and ordering across mounts", () => {
    const first = renderHook(() => useWorkOrderListState());
    act(() => {
      first.result.current.setLayout("table");
      first.result.current.setOrdering("spend");
    });
    first.unmount();

    const second = renderHook(() => useWorkOrderListState());
    expect(second.result.current.layout).toBe("table");
    expect(second.result.current.ordering).toBe("spend");
  });

  it("scope and search stay session-local (not persisted)", () => {
    const first = renderHook(() => useWorkOrderListState());
    act(() => {
      first.result.current.setScope("my");
      first.result.current.setSearch("refund");
    });
    first.unmount();

    const second = renderHook(() => useWorkOrderListState());
    expect(second.result.current.scope).toBe("all");
    expect(second.result.current.search).toBe("");
  });

  it("toggleFilter adds and removes values within one dimension", () => {
    const { result } = renderHook(() => useWorkOrderListState());
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
    const { result } = renderHook(() => useWorkOrderListState());
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
    const { result } = renderHook(() => useWorkOrderListState());
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

  it("resetView clears scope, filters, and search but keeps display preferences", () => {
    const { result } = renderHook(() => useWorkOrderListState());
    act(() => {
      result.current.setLayout("list");
      result.current.setScope("my");
      result.current.toggleFilter("statuses", "running");
      result.current.setSearch("foo");
    });
    expect(result.current.hasActiveFilters).toBe(true);
    act(() => {
      result.current.resetView();
    });
    expect(result.current.hasActiveFilters).toBe(false);
    expect(result.current.layout).toBe("list");
  });
});
