import { render } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect } from "vitest";

import { ConsoleContextProvider } from "../ConsoleContextProvider";
import { WidgetTable } from "./WidgetTable";
import type { WidgetTableRender } from "./types";

function renderProgress(tableRender: WidgetTableRender, rows: unknown[]) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <MemoryRouter>
      <QueryClientProvider client={queryClient}>
        <ConsoleContextProvider canvasId="canvas-1" organizationId="org-1" nodes={[]} canRunNodes={false}>
          <WidgetTable render={tableRender} rows={rows} isLoading={false} />
        </ConsoleContextProvider>
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

describe("WidgetTable progress column", () => {
  it("renders a bar sized to the current/target ratio with the default percent label", () => {
    const view = renderProgress(
      {
        kind: "table",
        columns: [{ field: "done", format: "progress", progressTarget: "total" }],
      },
      [{ id: "r-1", done: 5, total: 10 }],
    );
    const fill = view.container.querySelector('[data-testid="widget-progress-fill"]') as HTMLElement | null;
    expect(fill).not.toBeNull();
    expect(fill!.style.width).toBe("50%");
    expect(view.container.querySelector('[data-testid="widget-progress-label"]')!.textContent).toBe("50%");
    view.unmount();
  });

  it("supports the number label mode (`5/10`)", () => {
    const view = renderProgress(
      {
        kind: "table",
        columns: [{ field: "done", format: "progress", progressTarget: "total", progressLabel: "number" }],
      },
      [{ id: "r-1", done: 5, total: 10 }],
    );
    expect(view.container.querySelector('[data-testid="widget-progress-label"]')!.textContent).toBe("5/10");
    view.unmount();
  });

  it("omits the label entirely when `progressLabel: none`", () => {
    const view = renderProgress(
      {
        kind: "table",
        columns: [{ field: "done", format: "progress", progressTarget: "total", progressLabel: "none" }],
      },
      [{ id: "r-1", done: 5, total: 10 }],
    );
    expect(view.container.querySelector('[data-testid="widget-progress-label"]')).toBeNull();
    view.unmount();
  });

  it("clamps the bar at 100% while the label still reports the real overshoot value", () => {
    const view = renderProgress(
      {
        kind: "table",
        columns: [
          { field: "done", format: "progress", progressTarget: "total", label: "Percent" },
          { field: "done", format: "progress", progressTarget: "total", progressLabel: "number", label: "Number" },
        ],
      },
      [{ id: "r-1", done: 12, total: 10 }],
    );
    const fills = view.container.querySelectorAll('[data-testid="widget-progress-fill"]');
    expect(fills).toHaveLength(2);
    expect((fills[0] as HTMLElement).style.width).toBe("100%");
    expect((fills[1] as HTMLElement).style.width).toBe("100%");

    const labels = view.container.querySelectorAll('[data-testid="widget-progress-label"]');
    expect(labels[0].textContent).toBe("120%");
    expect(labels[1].textContent).toBe("12/10");

    const tracks = view.container.querySelectorAll('[role="progressbar"]');
    expect(tracks[0].getAttribute("aria-valuenow")).toBe("100");
    expect(tracks[0].getAttribute("aria-valuetext")).toBe("120%");
    view.unmount();
  });

  it("accepts a numeric literal as `progressTarget` without needing a row field", () => {
    const view = renderProgress(
      {
        kind: "table",
        columns: [{ field: "score", format: "progress", progressTarget: "100" }],
      },
      [{ id: "r-1", score: 25 }],
    );
    const fill = view.container.querySelector('[data-testid="widget-progress-fill"]') as HTMLElement | null;
    expect(fill!.style.width).toBe("25%");
    view.unmount();
  });

  it("supports {{ CEL }} expressions in `progressTarget`", () => {
    const view = renderProgress(
      {
        kind: "table",
        columns: [{ field: "done", format: "progress", progressTarget: "{{ base * 2 }}" }],
      },
      [{ id: "r-1", done: 5, base: 5 }],
    );
    const fill = view.container.querySelector('[data-testid="widget-progress-fill"]') as HTMLElement | null;
    expect(fill!.style.width).toBe("50%");
    view.unmount();
  });

  it("renders an empty track and em-dash placeholder when values are unresolvable", () => {
    const view = renderProgress(
      {
        kind: "table",
        columns: [{ field: "done", format: "progress", progressTarget: "total" }],
      },
      [{ id: "r-1", done: null, total: null }],
    );
    expect(view.container.querySelector('[data-testid="widget-progress-fill"]')).toBeNull();
    expect(view.container.querySelector('[data-testid="widget-progress-track"]')).not.toBeNull();
    expect(view.container.querySelector('[data-testid="widget-progress-label"]')!.textContent).toBe("—");
    view.unmount();
  });

  it("marks the bar with role='progressbar' and rounded aria-valuenow for a11y / tooltip", () => {
    const view = renderProgress(
      {
        kind: "table",
        columns: [{ field: "done", format: "progress", progressTarget: "total" }],
      },
      [{ id: "r-1", done: 3, total: 8 }],
    );
    const track = view.container.querySelector('[role="progressbar"]');
    expect(track).not.toBeNull();
    expect(track!.getAttribute("aria-valuenow")).toBe("38");
    expect(track!.getAttribute("aria-valuemin")).toBe("0");
    expect(track!.getAttribute("aria-valuemax")).toBe("100");
    expect(track!.getAttribute("aria-valuetext")).toBe("37.5%");
    view.unmount();
  });
});
