import { act, render } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { DashboardLayoutItem, DashboardPanel } from "@/hooks/useCanvasData";

import { DashboardView } from "./DashboardView";

const PANEL: DashboardPanel = {
  id: "readme",
  type: "markdown",
  content: { title: "Readme", body: "Hello" },
};

const LAYOUT: DashboardLayoutItem[] = [{ i: "readme", x: 0, y: 0, w: 6, h: 4 }];

const BASE_PROPS = {
  panels: [PANEL],
  layout: LAYOUT,
  readOnly: true,
  onChange: vi.fn(),
};

function flushAnimationFrame() {
  return act(async () => {
    await new Promise<void>((resolve) => {
      requestAnimationFrame(() => resolve());
    });
  });
}

describe("DashboardView grid transitions", () => {
  beforeEach(() => {
    globalThis.ResizeObserver = class {
      private callback: ResizeObserverCallback;

      constructor(callback: ResizeObserverCallback) {
        this.callback = callback;
      }

      observe() {
        this.callback(
          [{ contentRect: { width: 960, height: 0 } } as ResizeObserverEntry],
          this as unknown as ResizeObserver,
        );
      }

      unobserve() {}
      disconnect() {}
    } as unknown as typeof ResizeObserver;
  });

  it("does not arm transitions while loading before the grid mounts", async () => {
    const { rerender, container } = render(<DashboardView {...BASE_PROPS} isLoading errorMessage={undefined} />);

    await flushAnimationFrame();

    rerender(<DashboardView {...BASE_PROPS} isLoading={false} errorMessage={undefined} />);

    const grid = container.querySelector(".dashboard-grid");
    expect(grid).not.toBeNull();
    expect(grid?.classList.contains("dashboard-grid--instant")).toBe(true);
  });
});
