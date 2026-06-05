import { render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";

import type { DashboardPanel } from "@/hooks/useCanvasData";

import { DashboardContextProvider } from "./DashboardContextProvider";
import { MarkdownPanelCard } from "./MarkdownPanelCard";
import { useMarkdownVariables, type MarkdownVariablesResult } from "./useMarkdownVariables";

// Control the resolved/loading state directly so we can exercise the read
// path's loading gate without standing up the underlying query machinery.
vi.mock("./useMarkdownVariables", () => ({
  useMarkdownVariables: vi.fn(),
}));

function mockVariables(result: Partial<MarkdownVariablesResult>) {
  vi.mocked(useMarkdownVariables).mockReturnValue({
    vars: {},
    isLoading: false,
    errors: [],
    ...result,
  });
}

function renderCard(panel: DashboardPanel) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <MemoryRouter>
      <QueryClientProvider client={queryClient}>
        <DashboardContextProvider canvasId="canvas-1" organizationId="org-1" nodes={[]} canRunNodes={false}>
          <MarkdownPanelCard panel={panel} readOnly onDelete={() => {}} onChange={() => {}} />
        </DashboardContextProvider>
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

afterEach(() => {
  vi.clearAllMocks();
});

describe("MarkdownPanelCard variable loading gate", () => {
  it("shows a loading placeholder instead of empty node fields while run executions side-load", () => {
    // The run row already resolved (so `run` is non-null) but the per-run
    // executions backing `$["Deploy"]` are still loading.
    mockVariables({ vars: { run: { status: "passed", $: {} } }, isLoading: true });
    renderCard({
      id: "panel-1",
      type: "markdown",
      content: {
        body: 'Output: {{ run.$["Deploy"].data.url }}',
        variables: [{ name: "run", source: { kind: "run", select: "latest" } }],
      },
    });

    expect(screen.getByTestId("dashboard-markdown-loading")).toBeTruthy();
    // The interpolated (empty) body must not be rendered yet.
    expect(screen.queryByTestId("dashboard-markdown")).toBeNull();
  });

  it("renders the interpolated body once variables finish loading", () => {
    mockVariables({
      vars: { run: { $: { Deploy: { data: { url: "https://example.com/run/42" } } } } },
      isLoading: false,
    });
    renderCard({
      id: "panel-1",
      type: "markdown",
      content: {
        body: 'Output: {{ run.$["Deploy"].data.url }}',
        variables: [{ name: "run", source: { kind: "run", select: "latest" } }],
      },
    });

    expect(screen.queryByTestId("dashboard-markdown-loading")).toBeNull();
    expect(screen.getByTestId("dashboard-markdown").textContent).toMatch(/https:\/\/example\.com\/run\/42/);
  });

  it("falls back to the panel id for a templated title while it loads", () => {
    mockVariables({ vars: {}, isLoading: true });
    renderCard({
      id: "panel-title",
      type: "markdown",
      content: {
        title: "Latest deploy of {{ run.nodeName }}",
        body: "static",
        variables: [{ name: "run", source: { kind: "run", select: "latest" } }],
      },
    });

    // No half-interpolated "Latest deploy of " leaks into the header.
    expect(screen.getByText("panel-title")).toBeTruthy();
    expect(screen.queryByText(/Latest deploy of/)).toBeNull();
  });

  it("does not gate a static body when only the title references variables", () => {
    mockVariables({ vars: {}, isLoading: true });
    renderCard({
      id: "panel-static-body",
      type: "markdown",
      content: {
        title: "Run {{ run.status }}",
        body: "This body has no variables.",
        variables: [{ name: "run", source: { kind: "run", select: "latest" } }],
      },
    });

    expect(screen.queryByTestId("dashboard-markdown-loading")).toBeNull();
    expect(screen.getByTestId("dashboard-markdown").textContent).toMatch(/This body has no variables\./);
  });
});
