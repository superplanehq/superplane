import { act, renderHook } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import {
  canvasMemoryHiddenColumnsStorageKey,
  useCanvasMemoryColumnVisibility,
} from "@/hooks/useCanvasMemoryColumnVisibility";

const canvasId = "canvas-1";

afterEach(() => {
  window.localStorage.clear();
});

describe("useCanvasMemoryColumnVisibility", () => {
  it("starts with all columns visible when nothing is stored", () => {
    const { result } = renderHook(() => useCanvasMemoryColumnVisibility(canvasId, "ns", ["a", "b", "c"]));
    expect(result.current.visibleColumns).toEqual(["a", "b", "c"]);
    expect(result.current.hidden.size).toBe(0);
  });

  it("hydrates hidden columns from localStorage", () => {
    window.localStorage.setItem(canvasMemoryHiddenColumnsStorageKey(canvasId), JSON.stringify({ ns: ["b"] }));
    const { result } = renderHook(() => useCanvasMemoryColumnVisibility(canvasId, "ns", ["a", "b", "c"]));
    expect(result.current.visibleColumns).toEqual(["a", "c"]);
    expect(result.current.hidden.has("b")).toBe(true);
  });

  it("toggles a column and persists to localStorage", () => {
    const { result } = renderHook(() => useCanvasMemoryColumnVisibility(canvasId, "ns", ["a", "b", "c"]));

    act(() => result.current.toggle("b"));
    expect(result.current.visibleColumns).toEqual(["a", "c"]);
    expect(JSON.parse(window.localStorage.getItem(canvasMemoryHiddenColumnsStorageKey(canvasId))!)).toEqual({
      ns: ["b"],
    });

    act(() => result.current.toggle("b"));
    expect(result.current.visibleColumns).toEqual(["a", "b", "c"]);
    expect(window.localStorage.getItem(canvasMemoryHiddenColumnsStorageKey(canvasId))).toBeNull();
  });

  it("hideAll hides every column; showAll restores them", () => {
    const { result } = renderHook(() => useCanvasMemoryColumnVisibility(canvasId, "ns", ["a", "b"]));

    act(() => result.current.hideAll());
    expect(result.current.visibleColumns).toEqual([]);
    expect(result.current.hidden.size).toBe(2);

    act(() => result.current.showAll());
    expect(result.current.visibleColumns).toEqual(["a", "b"]);
    expect(result.current.hidden.size).toBe(0);
  });

  it("does not clobber other namespaces in the same canvas", () => {
    const nsA = renderHook(() => useCanvasMemoryColumnVisibility(canvasId, "ns-a", ["a", "b"]));
    const nsB = renderHook(() => useCanvasMemoryColumnVisibility(canvasId, "ns-b", ["x", "y"]));

    act(() => nsA.result.current.toggle("a"));
    act(() => nsB.result.current.toggle("y"));

    const stored = JSON.parse(window.localStorage.getItem(canvasMemoryHiddenColumnsStorageKey(canvasId))!);
    expect(stored).toEqual({ "ns-a": ["a"], "ns-b": ["y"] });
  });

  it("ignores malformed stored values", () => {
    window.localStorage.setItem(canvasMemoryHiddenColumnsStorageKey(canvasId), "not json");
    const { result } = renderHook(() => useCanvasMemoryColumnVisibility(canvasId, "ns", ["a", "b"]));
    expect(result.current.visibleColumns).toEqual(["a", "b"]);
  });
});
