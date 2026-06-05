import { act, render } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { ConsoleLayoutItem, ConsolePanel } from "@/hooks/useCanvasData";

import { ConsoleContextProvider } from "./ConsoleContextProvider";
import { ConsoleView } from "./ConsoleView";

const PANEL: ConsolePanel = {
  id: "readme",
  type: "markdown",
  content: { title: "Readme", body: "Hello" },
};

const LAYOUT: ConsoleLayoutItem[] = [{ i: "readme", x: 0, y: 0, w: 6, h: 4 }];

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

describe("ConsoleView grid transitions", () => {
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
    // The markdown panel that backs this grid issues queries through
    // `useMarkdownVariables`, so the test tree needs the standard providers
    // even for purely layout-focused assertions.
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const Wrapper = ({ children }: { children: React.ReactNode }) => (
      <MemoryRouter>
        <QueryClientProvider client={queryClient}>
          <ConsoleContextProvider canvasId="canvas-1" organizationId="org-1" nodes={[]} canRunNodes={false}>
            {children}
          </ConsoleContextProvider>
        </QueryClientProvider>
      </MemoryRouter>
    );
    const { rerender, container } = render(
      <Wrapper>
        <ConsoleView {...BASE_PROPS} isLoading errorMessage={undefined} />
      </Wrapper>,
    );

    await flushAnimationFrame();

    rerender(
      <Wrapper>
        <ConsoleView {...BASE_PROPS} isLoading={false} errorMessage={undefined} />
      </Wrapper>,
    );

    const grid = container.querySelector(".console-grid");
    expect(grid).not.toBeNull();
    expect(grid?.classList.contains("console-grid--instant")).toBe(true);
  });
});
