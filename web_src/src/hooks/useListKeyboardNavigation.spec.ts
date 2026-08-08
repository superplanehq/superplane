import { act, renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { useListKeyboardNavigation } from "./useListKeyboardNavigation";

function keyEvent(key: string, init: Partial<KeyboardEvent> = {}) {
  return {
    key,
    metaKey: false,
    ctrlKey: false,
    altKey: false,
    shiftKey: false,
    preventDefault: vi.fn(),
    ...init,
  } as unknown as React.KeyboardEvent<HTMLElement>;
}

describe("useListKeyboardNavigation", () => {
  it("starts on the first item and moves down", () => {
    const { result } = renderHook(() => useListKeyboardNavigation({ itemCount: 3 }));

    expect(result.current.activeIndex).toBe(0);

    act(() => result.current.handleKeyDown(keyEvent("ArrowDown")));
    expect(result.current.activeIndex).toBe(1);
  });

  it("wraps from the last item to the first and back", () => {
    const { result } = renderHook(() => useListKeyboardNavigation({ itemCount: 2 }));

    act(() => result.current.handleKeyDown(keyEvent("ArrowDown")));
    act(() => result.current.handleKeyDown(keyEvent("ArrowDown")));
    expect(result.current.activeIndex).toBe(0);

    act(() => result.current.handleKeyDown(keyEvent("ArrowUp")));
    expect(result.current.activeIndex).toBe(1);
  });

  it("jumps to the first and last item with Home and End", () => {
    const { result } = renderHook(() => useListKeyboardNavigation({ itemCount: 5 }));

    act(() => result.current.handleKeyDown(keyEvent("End")));
    expect(result.current.activeIndex).toBe(4);

    act(() => result.current.handleKeyDown(keyEvent("Home")));
    expect(result.current.activeIndex).toBe(0);
  });

  it("activates the highlighted item on Enter", () => {
    const onActivate = vi.fn();
    const { result } = renderHook(() => useListKeyboardNavigation({ itemCount: 3, onActivate }));

    act(() => result.current.handleKeyDown(keyEvent("ArrowDown")));
    act(() => result.current.handleKeyDown(keyEvent("Enter")));

    expect(onActivate).toHaveBeenCalledWith(1);
  });

  it("ignores navigation keys pressed with a modifier so global shortcuts still work", () => {
    const onActivate = vi.fn();
    const { result } = renderHook(() => useListKeyboardNavigation({ itemCount: 3, onActivate }));

    const event = keyEvent("ArrowDown", { metaKey: true });
    act(() => result.current.handleKeyDown(event));

    expect(result.current.activeIndex).toBe(0);
    expect(event.preventDefault).not.toHaveBeenCalled();
  });

  it("reports an empty list as having no active item", () => {
    const onActivate = vi.fn();
    const { result } = renderHook(() => useListKeyboardNavigation({ itemCount: 0, onActivate }));

    expect(result.current.activeIndex).toBe(-1);

    act(() => result.current.handleKeyDown(keyEvent("Enter")));
    expect(onActivate).not.toHaveBeenCalled();
  });

  it("clamps the active index when the list shrinks under it", () => {
    const { rerender, result } = renderHook(({ itemCount }) => useListKeyboardNavigation({ itemCount }), {
      initialProps: { itemCount: 5 },
    });

    act(() => result.current.handleKeyDown(keyEvent("End")));
    expect(result.current.activeIndex).toBe(4);

    rerender({ itemCount: 2 });
    expect(result.current.activeIndex).toBe(1);
  });

  it("leaves unrelated keys alone", () => {
    const { result } = renderHook(() => useListKeyboardNavigation({ itemCount: 3 }));

    const event = keyEvent("a");
    act(() => result.current.handleKeyDown(event));

    expect(event.preventDefault).not.toHaveBeenCalled();
    expect(result.current.activeIndex).toBe(0);
  });
});
