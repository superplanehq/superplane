import { act, fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "bun:test";
import { AutomationRunHistory } from "./AutomationRunHistory";
import { implementationRuns } from "./fixtures";

// CopyButton owns clipboard feedback; this covers the history-to-copy wiring,
// where accidentally passing the displayed prefix would lose the real identity.
describe("AutomationRunHistory", () => {
  it("loads another page when the end is visible, without duplicate requests during loading", () => {
    let intersect: IntersectionObserverCallback = () => {};
    const original = globalThis.IntersectionObserver;
    const disconnect = vi.fn();
    globalThis.IntersectionObserver = class {
      constructor(callback: IntersectionObserverCallback) {
        intersect = callback;
      }
      observe() {}
      disconnect = disconnect;
    } as unknown as typeof IntersectionObserver;
    const onLoadMore = vi.fn();
    try {
      const { rerender, unmount } = render(
        <AutomationRunHistory runs={implementationRuns} onSelectRun={() => {}} onLoadMore={onLoadMore} />,
      );
      const entry = { isIntersecting: true } as IntersectionObserverEntry;
      act(() => intersect([entry], {} as IntersectionObserver));
      act(() => intersect([entry], {} as IntersectionObserver));
      expect(onLoadMore).toHaveBeenCalledTimes(1);
      rerender(
        <AutomationRunHistory runs={implementationRuns} onSelectRun={() => {}} onLoadMore={onLoadMore} loadingMore />,
      );
      expect(screen.getByRole("status", { name: "Loading more runs" })).toBeInTheDocument();
      unmount();
      expect(disconnect).toHaveBeenCalled();
    } finally {
      globalThis.IntersectionObserver = original;
    }
  });

  it("shows calendar days for recent runs and always keeps the absolute time", () => {
    const today = new Date();
    today.setHours(14, 6, 0, 0);
    const yesterday = new Date(today);
    yesterday.setDate(yesterday.getDate() - 1);
    const older = new Date(today);
    older.setDate(older.getDate() - 3);
    render(
      <AutomationRunHistory
        runs={[today, yesterday, older].map((date, i) => ({
          ...implementationRuns[i],
          startedAt: date.toISOString(),
        }))}
        onSelectRun={() => {}}
      />,
    );
    expect(screen.getByText("Today, 14:06")).toBeInTheDocument();
    expect(screen.getByText("Yesterday, 14:06")).toBeInTheDocument();
    expect(screen.getAllByText(/14:06/)).toHaveLength(3);
  });

  it.each(["missing-task", implementationRuns[0].id])(
    "requires explicit pagination while filtering by %s, then resumes automatic loading when cleared",
    (query) => {
      const callbacks = new Set<IntersectionObserverCallback>();
      const original = globalThis.IntersectionObserver;
      globalThis.IntersectionObserver = class {
        constructor(private callback: IntersectionObserverCallback) {}
        observe() {
          callbacks.add(this.callback);
        }
        disconnect() {
          callbacks.delete(this.callback);
        }
      } as unknown as typeof IntersectionObserver;
      const onLoadMore = vi.fn();
      const notifyBottom = () =>
        act(() => {
          for (const callback of callbacks)
            callback([{ isIntersecting: true } as IntersectionObserverEntry], {} as IntersectionObserver);
        });
      const view = (runs = implementationRuns) => (
        <AutomationRunHistory runs={runs} onSelectRun={() => {}} onLoadMore={onLoadMore} />
      );
      try {
        const { rerender } = render(view());
        fireEvent.change(screen.getByRole("textbox", { name: "Search task or run ID" }), { target: { value: query } });
        notifyBottom();
        expect(onLoadMore).not.toHaveBeenCalled();
        const loadMore = screen.getByRole("button", { name: "Load more runs" });
        expect(loadMore).toBeVisible();
        fireEvent.click(loadMore);
        expect(onLoadMore).toHaveBeenCalledTimes(1);
        rerender(view([...implementationRuns, { ...implementationRuns[1], id: "older-run" }]));
        notifyBottom();
        expect(onLoadMore).toHaveBeenCalledTimes(1);
        fireEvent.change(screen.getByRole("textbox", { name: "Search task or run ID" }), { target: { value: "" } });
        notifyBottom();
        expect(onLoadMore).toHaveBeenCalledTimes(2);
      } finally {
        globalThis.IntersectionObserver = original;
      }
    },
  );
  it("copies the complete run ID without selecting the run", async () => {
    const writeText = vi.fn(() => Promise.resolve());
    const clipboard = Object.getOwnPropertyDescriptor(navigator, "clipboard");
    Object.defineProperty(navigator, "clipboard", { configurable: true, value: { writeText } });
    const onSelectRun = vi.fn();
    const run = implementationRuns[0];
    try {
      render(<AutomationRunHistory runs={[run]} onSelectRun={onSelectRun} />);
      await act(async () => {
        fireEvent.click(screen.getByRole("button", { name: `Copy full run ID ${run.id}` }));
      });
      expect(writeText).toHaveBeenCalledWith(run.id);
      expect(onSelectRun).not.toHaveBeenCalled();
      fireEvent.click(screen.getByRole("button", { name: `Open run ${run.id}` }));
      expect(onSelectRun).toHaveBeenCalledWith(run);
    } finally {
      if (clipboard) Object.defineProperty(navigator, "clipboard", clipboard);
      else Reflect.deleteProperty(navigator, "clipboard");
    }
  });

  it("finds a run by its full ID and recovers from an empty search", () => {
    render(<AutomationRunHistory runs={implementationRuns} onSelectRun={() => {}} />);
    fireEvent.change(screen.getByRole("textbox", { name: "Search task or run ID" }), {
      target: { value: implementationRuns[1].id },
    });
    expect(screen.getAllByRole("button", { name: /^Open run/ })).toHaveLength(1);
    expect(screen.getByRole("button", { name: `Open run ${implementationRuns[1].id}` })).toBeInTheDocument();
    fireEvent.change(screen.getByRole("textbox", { name: "Search task or run ID" }), {
      target: { value: "missing-task" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Clear filters" }));
    expect(screen.getAllByRole("button", { name: /^Open run/ })).toHaveLength(implementationRuns.length);
  });
});
