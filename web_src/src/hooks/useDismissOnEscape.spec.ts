import { renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { useDismissOnEscape } from "@/hooks/useDismissOnEscape";

function fireKeyDown(key: string, options: Partial<KeyboardEventInit> = {}) {
  const event = new KeyboardEvent("keydown", { key, cancelable: true, bubbles: true, ...options });
  document.dispatchEvent(event);
  return event;
}

describe("useDismissOnEscape", () => {
  it("calls onDismiss once when Escape is pressed", () => {
    const onDismiss = vi.fn();
    renderHook(() => useDismissOnEscape(onDismiss));

    fireKeyDown("Escape");

    expect(onDismiss).toHaveBeenCalledTimes(1);
  });

  it("does nothing for other keys", () => {
    const onDismiss = vi.fn();
    renderHook(() => useDismissOnEscape(onDismiss));

    fireKeyDown("Enter");

    expect(onDismiss).not.toHaveBeenCalled();
  });

  it("does nothing when onDismiss is undefined", () => {
    expect(() => {
      renderHook(() => useDismissOnEscape(undefined));
      fireKeyDown("Escape");
    }).not.toThrow();
  });

  it("ignores Escape when a nested handler already called preventDefault", () => {
    const onDismiss = vi.fn();
    // Nested controls (e.g. an inline rename field) register their own
    // keydown listener before the modal frame mounts its dismiss listener,
    // matching how React's root listener runs ahead of this hook's plain
    // document listener in the real app.
    document.addEventListener(
      "keydown",
      (event) => {
        event.preventDefault();
      },
      { once: true },
    );
    renderHook(() => useDismissOnEscape(onDismiss));

    fireKeyDown("Escape");

    expect(onDismiss).not.toHaveBeenCalled();
  });

  it("removes the listener on unmount", () => {
    const onDismiss = vi.fn();
    const { unmount } = renderHook(() => useDismissOnEscape(onDismiss));

    unmount();
    fireKeyDown("Escape");

    expect(onDismiss).not.toHaveBeenCalled();
  });

  it("re-subscribes when onDismiss changes", () => {
    const first = vi.fn();
    const second = vi.fn();
    const { rerender } = renderHook(({ onDismiss }) => useDismissOnEscape(onDismiss), {
      initialProps: { onDismiss: first },
    });

    rerender({ onDismiss: second });
    fireKeyDown("Escape");

    expect(first).not.toHaveBeenCalled();
    expect(second).toHaveBeenCalledTimes(1);
  });
});
