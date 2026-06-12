import { act, fireEvent, render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { ConsoleLayoutItem, ConsolePanel } from "@/hooks/useCanvasData";
import type { DraftConsoleDiffSummary } from "../draftConsoleDiff";

import { ConsoleContextProvider } from "./ConsoleContextProvider";
import { ConsoleView } from "./ConsoleView";

const PANEL: ConsolePanel = {
  id: "readme",
  type: "markdown",
  content: { title: "Readme", body: "Hello" },
};

const LAYOUT: ConsoleLayoutItem[] = [{ i: "readme", x: 0, y: 0, w: 6, h: 4 }];

const VISUAL_DIFF_SUMMARY: DraftConsoleDiffSummary = {
  addedCount: 1,
  updatedCount: 1,
  removedCount: 1,
  items: [
    {
      id: "readme",
      title: "Readme",
      changeType: "updated",
      lines: [
        { prefix: "meta", text: "diff --git a/console/panels/readme.yaml b/console/panels/readme.yaml" },
        { prefix: "meta", text: "--- a/console/panels/readme.yaml" },
        { prefix: "meta", text: "+++ b/console/panels/readme.yaml" },
        { prefix: "context", text: "@@ -1,0 +1,0 @@" },
        { prefix: "-", text: "content:" },
        { prefix: "-", text: "  body: Before" },
        { prefix: "+", text: "content:" },
        { prefix: "+", text: "  body: Hello" },
      ],
    },
    {
      id: "new-panel",
      title: "New Panel",
      changeType: "added",
      lines: [],
    },
    {
      id: "old-panel",
      title: "Old Panel",
      changeType: "removed",
      panel: { id: "old-panel", type: "markdown", content: { title: "Old Panel", body: "Removed" } },
      layout: { i: "old-panel", x: 6, y: 0, w: 6, h: 4 },
      lines: [],
    },
  ],
};

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

function TestWrapper({ children }: { children: React.ReactNode }) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return (
    <MemoryRouter>
      <QueryClientProvider client={queryClient}>
        <ConsoleContextProvider canvasId="canvas-1" organizationId="org-1" nodes={[]} canRunNodes={false}>
          {children}
        </ConsoleContextProvider>
      </QueryClientProvider>
    </MemoryRouter>
  );
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
    const { rerender, container } = render(
      <TestWrapper>
        <ConsoleView {...BASE_PROPS} isLoading errorMessage={undefined} />
      </TestWrapper>,
    );

    await flushAnimationFrame();

    rerender(
      <TestWrapper>
        <ConsoleView {...BASE_PROPS} isLoading={false} errorMessage={undefined} />
      </TestWrapper>,
    );

    const grid = container.querySelector(".console-grid");
    expect(grid).not.toBeNull();
    expect(grid?.classList.contains("console-grid--instant")).toBe(true);
  });

  it("renders visual diff badges and deleted panels by default", () => {
    const { container } = render(
      <TestWrapper>
        <ConsoleView
          {...BASE_PROPS}
          panels={[PANEL, { id: "new-panel", type: "markdown", content: { title: "New Panel", body: "Added" } }]}
          layout={[...LAYOUT, { i: "new-panel", x: 0, y: 4, w: 6, h: 4 }]}
          isLoading={false}
          errorMessage={undefined}
          visualDiff={{ enabled: true, summary: VISUAL_DIFF_SUMMARY }}
        />
      </TestWrapper>,
    );

    expect(screen.getByText("EDITED")).toBeTruthy();
    expect(screen.getByText("ADDED")).toBeTruthy();
    expect(screen.getByRole("button", { name: "See Readme diff" })).toHaveClass(
      "w-fit",
      "max-w-max",
      "whitespace-nowrap",
    );
    expect(screen.getByText("REMOVED")).toBeTruthy();
    expect(screen.getByText("Old Panel")).toBeTruthy();
    expect(container.querySelectorAll(".react-grid-item.console-grid-item")).toHaveLength(2);
    expect(container.querySelectorAll('[data-testid="console-removed-panel-ghost"]')).toHaveLength(1);
    expect(container.querySelectorAll('[data-testid="console-panel-diff-border"]')).toHaveLength(3);
    expect(container.querySelector('[data-testid="console-panel-diff-border"]')).toHaveClass("rounded-lg", "border-2");
    expect(screen.queryByText("See diff")).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "See Readme diff" }));

    expect(screen.getByRole("dialog")).toBeTruthy();
    expect(screen.getByText("Console panel diff for updated panel.")).toBeTruthy();
  });

  it.each([
    ["EDITED", "updated"],
    ["ADDED", "added"],
  ] as const)("hides the %s badge while its panel is editing", (label, changeType) => {
    render(
      <TestWrapper>
        <ConsoleView
          {...BASE_PROPS}
          readOnly={false}
          isLoading={false}
          errorMessage={undefined}
          visualDiff={{
            enabled: true,
            summary: {
              addedCount: changeType === "added" ? 1 : 0,
              updatedCount: changeType === "updated" ? 1 : 0,
              removedCount: 0,
              items: [{ id: "readme", title: "Readme", changeType, lines: [] }],
            },
          }}
        />
      </TestWrapper>,
    );

    expect(screen.getByText(label)).toBeTruthy();

    fireEvent.click(screen.getByTestId("console-edit-panel"));

    expect(screen.queryByText(label)).not.toBeInTheDocument();
  });

  it("replaces deleted ghost panels when a live panel occupies their layout", () => {
    const { container } = render(
      <TestWrapper>
        <ConsoleView
          {...BASE_PROPS}
          panels={[PANEL, { id: "new-panel", type: "markdown", content: { title: "New Panel", body: "Added" } }]}
          layout={[...LAYOUT, { i: "new-panel", x: 6, y: 0, w: 6, h: 4 }]}
          isLoading={false}
          errorMessage={undefined}
          visualDiff={{ enabled: true, summary: VISUAL_DIFF_SUMMARY }}
        />
      </TestWrapper>,
    );

    expect(screen.queryByText("REMOVED")).not.toBeInTheDocument();
    expect(screen.queryByText("Old Panel")).not.toBeInTheDocument();
    expect(container.querySelectorAll(".react-grid-item.console-grid-item")).toHaveLength(2);
    expect(container.querySelectorAll('[data-testid="console-removed-panel-ghost"]')).toHaveLength(0);
  });

  it("renders a deleted panel ghost immediately after local delete", () => {
    const { container } = render(
      <TestWrapper>
        <ConsoleView
          {...BASE_PROPS}
          readOnly={false}
          isLoading={false}
          errorMessage={undefined}
          visualDiff={{ enabled: true, summary: { items: [], addedCount: 0, updatedCount: 0, removedCount: 0 } }}
        />
      </TestWrapper>,
    );

    expect(screen.queryByText("REMOVED")).not.toBeInTheDocument();

    fireEvent.click(screen.getByTestId("console-delete-panel"));
    fireEvent.click(screen.getByTestId("console-delete-confirm"));

    expect(screen.getByText("REMOVED")).toBeTruthy();
    expect(screen.getByText("Readme")).toBeTruthy();
    expect(container.querySelectorAll(".react-grid-item.console-grid-item")).toHaveLength(0);
    expect(container.querySelectorAll('[data-testid="console-removed-panel-ghost"]')).toHaveLength(1);
  });
});
