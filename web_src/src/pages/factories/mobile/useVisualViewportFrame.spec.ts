import { act, render, renderHook, screen } from "@testing-library/react";
import { createElement } from "react";
import { afterEach, describe, expect, it } from "bun:test";

import { PinnedPhoneShell } from "./PinnedPhoneShell";
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
    expect(phoneShellFrame({ top: 0, height: 700 }, 36)).toEqual({ top: 36, height: 664 });
    expect(phoneShellFrame({ top: 20, height: 700 }, 0)).toEqual({ top: 20, height: 700 });
  });

  it("does not reserve banner space that is already above the visible screen", () => {
    expect(phoneShellFrame({ top: 20, height: 700 }, 36)).toEqual({ top: 36, height: 684 });
    expect(phoneShellFrame({ top: 40, height: 700 }, 36)).toEqual({ top: 40, height: 700 });
  });
});

describe("PinnedPhoneShell", () => {
  const restores: Array<() => void> = [];

  afterEach(() => {
    while (restores.length > 0) {
      restores.pop()?.();
    }
  });

  it("moves the shell when a banner appears or disappears after mount", () => {
    const { viewport } = viewportStub(700, 0);
    restores.push(stubVisualViewport(viewport));

    let bannerBottom = 0;
    const region = document.createElement("div");
    region.setAttribute("data-app-content-region", "");
    region.getBoundingClientRect = () =>
      ({
        top: bannerBottom,
        left: 0,
        right: 0,
        bottom: bannerBottom,
        width: 0,
        height: 0,
        x: 0,
        y: bannerBottom,
        toJSON: () => ({}),
      }) as DOMRect;
    document.body.appendChild(region);
    restores.push(() => {
      region.remove();
    });

    const previousObserver = globalThis.ResizeObserver;
    let notifyResize = () => {};
    globalThis.ResizeObserver = class {
      constructor(callback: ResizeObserverCallback) {
        notifyResize = () => {
          callback([], this as unknown as ResizeObserver);
        };
      }

      observe() {}
      unobserve() {}
      disconnect() {}
    } as unknown as typeof ResizeObserver;
    restores.push(() => {
      globalThis.ResizeObserver = previousObserver;
    });

    render(createElement(PinnedPhoneShell, { children: "Board", testId: "phone-shell" }));
    const shell = screen.getByTestId("phone-shell");
    expect(shell).toHaveStyle({ top: "0px", height: "700px" });

    bannerBottom = 48;
    act(() => {
      notifyResize();
    });
    expect(shell).toHaveStyle({ top: "48px", height: "652px" });

    bannerBottom = 0;
    act(() => {
      notifyResize();
    });
    expect(shell).toHaveStyle({ top: "0px", height: "700px" });
  });
});
