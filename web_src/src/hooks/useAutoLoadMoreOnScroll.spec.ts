import { act, renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "bun:test";

import { useAutoLoadMoreOnScroll } from "./useAutoLoadMoreOnScroll";

function scroller(remainingScroll: number): HTMLElement {
  const element = document.createElement("div");
  Object.defineProperties(element, {
    scrollHeight: { configurable: true, value: 500 },
    clientHeight: { configurable: true, value: 500 - remainingScroll },
    scrollTop: { configurable: true, value: 0 },
  });
  return element;
}

describe("useAutoLoadMoreOnScroll", () => {
  it("requests a second page after the first auto-load when isLoading is omitted", async () => {
    const onLoadMore = vi.fn();
    const { result } = renderHook(() => useAutoLoadMoreOnScroll({ hasMore: true, onLoadMore }));
    const element = scroller(0);

    result.current(element);
    expect(onLoadMore).toHaveBeenCalledTimes(1);
    result.current(element);
    expect(onLoadMore).toHaveBeenCalledTimes(1);

    await act(async () => {
      await new Promise((resolve) => setTimeout(resolve, 0));
    });

    result.current(element);
    expect(onLoadMore).toHaveBeenCalledTimes(2);
  });

  it("requests another page after hasMore becomes true again when isLoading is omitted", () => {
    const onLoadMore = vi.fn();
    const { result, rerender } = renderHook(({ hasMore }) => useAutoLoadMoreOnScroll({ hasMore, onLoadMore }), {
      initialProps: { hasMore: true },
    });
    const element = scroller(0);

    result.current(element);
    expect(onLoadMore).toHaveBeenCalledTimes(1);

    rerender({ hasMore: false });
    rerender({ hasMore: true });
    result.current(element);

    expect(onLoadMore).toHaveBeenCalledTimes(2);
  });

  it("does not request another page while isLoading is true", () => {
    const onLoadMore = vi.fn();
    const { result, rerender } = renderHook(
      ({ isLoading }) => useAutoLoadMoreOnScroll({ hasMore: true, isLoading, onLoadMore }),
      { initialProps: { isLoading: false } },
    );
    const element = scroller(0);

    result.current(element);
    expect(onLoadMore).toHaveBeenCalledTimes(1);

    rerender({ isLoading: true });
    result.current(element);
    expect(onLoadMore).toHaveBeenCalledTimes(1);

    rerender({ isLoading: false });
    result.current(element);
    expect(onLoadMore).toHaveBeenCalledTimes(2);
  });

  it("does not load when the list still has scroll remaining", () => {
    const onLoadMore = vi.fn();
    const { result } = renderHook(() => useAutoLoadMoreOnScroll({ hasMore: true, onLoadMore }));

    result.current(scroller(200));

    expect(onLoadMore).not.toHaveBeenCalled();
  });
});
