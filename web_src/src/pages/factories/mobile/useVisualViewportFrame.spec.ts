import { act, renderHook } from "@testing-library/react";
import { afterEach, describe, expect, it } from "bun:test";

import { phoneShellFrame, readVisualViewportFrame, useVisualViewportFrame } from "./useVisualViewportFrame";

type ViewportStub = {
  height: number;
  offsetTop: number;
  addEventListener: (type: string, listener: EventListener) => void;
  removeEventListener: (type: string, listener: EventListener) => void;
};

function stubVisualViewport(value: ViewportStub | null): () => void {
  const descriptor = Object.getOwnPropertyDescriptor(window, "visualViewport");
  Object.defineProperty(window, "visualViewport", {
    configurable: true,
    enumerable: true,
    value,
  });
  return () => {
    if (descriptor) {
      Object.defineProperty(window, "visualViewport", descriptor);
    } else {
      Reflect.deleteProperty(window, "visualViewport");
    }
  };
}

function viewportStub(height: number, offsetTop: number): { viewport: ViewportStub; emit: (type: string) => void } {
  const listeners = new Map<string, Set<EventListener>>();
  const viewport: ViewportStub = {
    height,
    offsetTop,
    addEventListener(type, listener) {
      const set = listeners.get(type) ?? new Set();
      set.add(listener);
      listeners.set(type, set);
    },
    removeEventListener(type, listener) {
      listeners.get(type)?.delete(listener);
    },
  };
  return {
    viewport,
    emit(type) {
      listeners.get(type)?.forEach((listener) => listener(new Event(type)));
    },
  };
}

describe("useVisualViewportFrame", () => {
  const restores: Array<() => void> = [];

  afterEach(() => {
    while (restores.length > 0) {
      restores.pop()?.();
    }
  });

  it("tracks the visual viewport on resize and scroll", () => {
    const { viewport, emit } = viewportStub(700, 12);
    restores.push(stubVisualViewport(viewport));

    const { result } = renderHook(() => useVisualViewportFrame());

    expect(result.current).toEqual({ top: 12, height: 700 });

    viewport.height = 640;
    viewport.offsetTop = 40;
    act(() => {
      emit("resize");
    });
    expect(result.current).toEqual({ top: 40, height: 640 });

    viewport.height = 610;
    viewport.offsetTop = 8;
    act(() => {
      emit("scroll");
    });
    expect(result.current).toEqual({ top: 8, height: 610 });
  });

  it("falls back to dynamic viewport height and a top of 0 when the API is missing", () => {
    restores.push(stubVisualViewport(null));

    expect(() => readVisualViewportFrame()).not.toThrow();
    const frame = readVisualViewportFrame();

    expect(frame.top).toBe(0);
    expect(frame.height).toBeGreaterThan(0);
  });
});

describe("phoneShellFrame", () => {
  it("places the shell below a banner that is still visible", () => {
    expect(phoneShellFrame({ top: 20, height: 700 }, 36)).toEqual({ top: 56, height: 664 });
    expect(phoneShellFrame({ top: 20, height: 700 }, 0)).toEqual({ top: 20, height: 700 });
  });
});
