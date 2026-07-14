import { render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";

import type { ConsolePanel } from "@/hooks/useCanvasData";

import { ConsoleContextProvider } from "./ConsoleContextProvider";
import { MarkdownPanelCard } from "./MarkdownPanelCard";
import { useMarkdownVariables, type MarkdownVariablesResult } from "./useMarkdownVariables";
import { DOLLAR_REWRITE_IDENTIFIER } from "./widget/celExpr";

// Control the resolved/loading state directly so we can exercise the read
// path's loading gate without standing up the underlying query machinery.
vi.mock("./useMarkdownVariables", () => ({
  useMarkdownVariables: vi.fn(),
}));

function mockVariables(result: Partial<MarkdownVariablesResult>) {
  vi.mocked(useMarkdownVariables).mockReturnValue({
    vars: {},
    isLoading: false,
    baseLoading: false,
    sideloadLoading: false,
    searchingNames: [],
    errors: [],
    ...result,
  });
}

function renderCard(panel: ConsolePanel) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <MemoryRouter>
      <QueryClientProvider client={queryClient}>
        <ConsoleContextProvider canvasId="canvas-1" organizationId="org-1" nodes={[]} canRunNodes={false}>
          <MarkdownPanelCard panel={panel} readOnly onDelete={() => {}} onChange={() => {}} />
        </ConsoleContextProvider>
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
    mockVariables({ vars: { run: { status: "passed", $: {} } }, isLoading: true, sideloadLoading: true });
    renderCard({
      id: "panel-1",
      type: "markdown",
      content: {
        body: 'Output: {{ run.$["Deploy"].data.url }}',
        variables: [{ name: "run", source: { kind: "run", select: "latest" } }],
      },
    });

    expect(screen.getByTestId("console-markdown-loading")).toBeTruthy();
    // The interpolated (empty) body must not be rendered yet.
    expect(screen.queryByTestId("console-markdown")).toBeNull();
  });

  it("renders the interpolated body once variables finish loading", () => {
    // The real hook exposes run-node outputs under both `$` and the rewritten
    // `__runNodes__` identifier that `$["Deploy"]` compiles to, so the mock
    // mirrors both.
    const deployNodes = { Deploy: { data: { url: "https://example.com/run/42" } } };
    mockVariables({
      vars: { run: { $: deployNodes, [DOLLAR_REWRITE_IDENTIFIER]: deployNodes } },
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

    expect(screen.queryByTestId("console-markdown-loading")).toBeNull();
    expect(screen.getByTestId("console-markdown").textContent).toMatch(/https:\/\/example\.com\/run\/42/);
  });

  it("falls back to the panel id for a templated title while it loads", () => {
    mockVariables({ vars: {}, isLoading: true, baseLoading: true });
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
    mockVariables({ vars: {}, isLoading: true, baseLoading: true });
    renderCard({
      id: "panel-static-body",
      type: "markdown",
      content: {
        title: "Run {{ run.status }}",
        body: "This body has no variables.",
        variables: [{ name: "run", source: { kind: "run", select: "latest" } }],
      },
    });

    expect(screen.queryByTestId("console-markdown-loading")).toBeNull();
    expect(screen.getByTestId("console-markdown").textContent).toMatch(/This body has no variables\./);
  });

  it("interpolates a title using resolved fields while only the body's run-node side-load is pending", () => {
    // The base run query resolved `run.status`, but the per-run executions
    // backing the body's `$["Deploy"]` reference are still loading. The title
    // only needs the already-available `status`, so it must interpolate now
    // instead of flashing the panel id.
    mockVariables({
      vars: { run: { status: "passed", $: {} } },
      isLoading: true,
      baseLoading: false,
      sideloadLoading: true,
    });
    renderCard({
      id: "panel-mixed",
      type: "markdown",
      content: {
        title: "Run {{ run.status }}",
        body: 'Output: {{ run.$["Deploy"].data.url }}',
        variables: [{ name: "run", source: { kind: "run", select: "latest" } }],
      },
    });

    // Title resolves immediately; the body waits on the execution side-load.
    expect(screen.getByText("Run passed")).toBeTruthy();
    expect(screen.queryByText("panel-mixed")).toBeNull();
    expect(screen.getByTestId("console-markdown-loading")).toBeTruthy();
  });

  it("falls back to the panel id when a fully-loaded templated title interpolates to empty", () => {
    // Loading is finished, but the variable the title depends on resolved to
    // null, so interpolation yields an empty string. The raw `{{ }}` template
    // must not leak into the header — fall back to the stable panel id.
    mockVariables({ vars: { run: null }, isLoading: false });
    renderCard({
      id: "panel-empty-title",
      type: "markdown",
      content: {
        title: "{{ run.nodeName }}",
        body: "static body",
        variables: [{ name: "run", source: { kind: "run", select: "latest" } }],
      },
    });

    expect(screen.getByText("panel-empty-title")).toBeTruthy();
    expect(screen.queryByText(/\{\{/)).toBeNull();
  });
});
