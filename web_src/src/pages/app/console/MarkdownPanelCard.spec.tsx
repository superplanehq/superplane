import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, expect, it, vi } from "vitest";

import { canvasKeys, type CanvasMemoryEntry, type ConsolePanel } from "@/hooks/useCanvasData";

import { ConsoleContextProvider } from "./ConsoleContextProvider";
import { MarkdownPanelCard } from "./MarkdownPanelCard";

vi.mock("@/components/AgentSidebar/widgets/NodeChip", () => ({
  NodeChipFromLink: ({
    nodeId,
    rawLabel,
    canvasId,
    organizationId,
  }: {
    nodeId: string;
    rawLabel?: string;
    canvasId: string;
    organizationId: string;
  }) => (
    <button type="button" data-testid="node-chip">
      {rawLabel}:{nodeId}:{canvasId}:{organizationId}
    </button>
  ),
}));

function renderMarkdown(body: string) {
  // The MarkdownPanelCard always issues queries through `useMarkdownVariables`,
  // so even tests that don't touch the variable system need a QueryClient and
  // dashboard context in the tree.
  return renderWithVariables({
    panel: { id: "md-test", type: "markdown", content: { body } },
  });
}

interface RenderWithVariablesOptions {
  panel: ConsolePanel;
  memoryEntries?: CanvasMemoryEntry[];
}

/**
 * Render the markdown panel inside a QueryClient + dashboard context so the
 * `useMarkdownVariables` hook can issue (mocked) queries. Memory entries are
 * preloaded into the React Query cache so the test environment doesn't need
 * to mock the network layer.
 */
function renderWithVariables({ panel, memoryEntries = [] }: RenderWithVariablesOptions) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  queryClient.setQueryData(canvasKeys.canvasMemoryEntries("canvas-1"), memoryEntries);
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

describe("MarkdownPanelCard rendering", () => {
  it("renders a GFM pipe table from hand-written markdown", () => {
    renderMarkdown("| Service | Status |\n| --- | --- |\n| api | passed |\n| web | failed |\n");

    const view = screen.getByTestId("console-markdown");
    const table = view.querySelector("table");
    expect(table).not.toBeNull();
    const headers = table!.querySelectorAll("th");
    expect(headers).toHaveLength(2);
    expect(headers[0].textContent).toBe("Service");
    expect(headers[1].textContent).toBe("Status");
    const rows = table!.querySelectorAll("tbody tr");
    expect(rows).toHaveLength(2);
    expect(rows[0].textContent).toMatch(/api/);
    expect(rows[1].textContent).toMatch(/failed/);
  });

  it("renders node links as chips using console canvas context", () => {
    renderMarkdown("Open [Deploy](node:deploy-node) before continuing.");

    expect(screen.getByTestId("node-chip")).toHaveTextContent("Deploy:deploy-node:canvas-1:org-1");
    expect(screen.queryByRole("link", { name: "Deploy" })).not.toBeInTheDocument();
  });

  it("preserves <details>/<summary> accordions and the open attribute", () => {
    renderMarkdown("<details open>\n<summary>Troubleshooting</summary>\n\nFlush the cache.\n\n</details>");

    const view = screen.getByTestId("console-markdown");
    const details = view.querySelector("details");
    expect(details).not.toBeNull();
    expect(details!.hasAttribute("open")).toBe(true);
    const summary = details!.querySelector("summary");
    expect(summary?.textContent).toBe("Troubleshooting");
    expect(details!.textContent).toMatch(/Flush the cache/);
  });

  it("strips unsafe raw HTML like <script> tags", () => {
    renderMarkdown("Hello <script>window.__pwned = true;</script> world");

    const view = screen.getByTestId("console-markdown");
    expect(view.querySelector("script")).toBeNull();
    expect(view.textContent).toMatch(/Hello/);
    expect(view.textContent).toMatch(/world/);
  });

  it("strips inline event handlers from allowed tags", () => {
    renderMarkdown('<a href="https://example.com" onclick="alert(1)">link</a>');

    const view = screen.getByTestId("console-markdown");
    const anchor = view.querySelector("a");
    expect(anchor).not.toBeNull();
    expect(anchor!.getAttribute("onclick")).toBeNull();
  });
});

describe("MarkdownPanelCard variable interpolation", () => {
  it("interpolates a memory variable's field into the body and title", async () => {
    const panel: ConsolePanel = {
      id: "panel-1",
      type: "markdown",
      content: {
        title: "Latest deploy of {{ recipe.service }}",
        body: "Service: **{{ recipe.service }}**\n\nStatus: {{ recipe.status }}",
        variables: [{ name: "recipe", source: { kind: "memory", namespace: "deploys" } }],
      },
    };
    renderWithVariables({
      panel,
      memoryEntries: [
        {
          id: "row-old",
          namespace: "deploys",
          values: { service: "api", status: "passed" },
          source: "node",
          createdAt: "2026-06-01T00:00:00Z",
        },
        {
          id: "row-new",
          namespace: "deploys",
          values: { service: "web", status: "failed" },
          source: "node",
          createdAt: "2026-06-04T00:00:00Z",
        },
      ],
    });

    const view = await waitFor(() => screen.getByTestId("console-markdown"));
    expect(view.textContent).toMatch(/Service: web/);
    expect(view.textContent).toMatch(/Status: failed/);
    // Picks the most recent memory row by createdAt by default.
    expect(screen.getByText(/Latest deploy of web/)).toBeTruthy();
  });

  it("applies property-equality matches before sorting", async () => {
    const panel: ConsolePanel = {
      id: "panel-2",
      type: "markdown",
      content: {
        body: "Approved: {{ recipe.service }}",
        variables: [
          {
            name: "recipe",
            source: {
              kind: "memory",
              namespace: "deploys",
              matches: [{ field: "status", value: "passed" }],
            },
          },
        ],
      },
    };
    renderWithVariables({
      panel,
      memoryEntries: [
        {
          id: "row-old",
          namespace: "deploys",
          values: { service: "api", status: "passed" },
          source: "node",
          createdAt: "2026-06-01T00:00:00Z",
        },
        {
          id: "row-new",
          namespace: "deploys",
          values: { service: "web", status: "failed" },
          source: "node",
          createdAt: "2026-06-04T00:00:00Z",
        },
      ],
    });
    const view = await waitFor(() => screen.getByTestId("console-markdown"));
    expect(view.textContent).toMatch(/Approved: api/);
  });

  it("resolves a list-mode memory variable and renders it via join(map())", async () => {
    // End-to-end coverage for list mode: the variable resolves to the full
    // sorted array, `rows.map(...)` runs the CEL macro, and `join(..., sep)`
    // flattens it into a string — the documented pattern for rendering lists.
    const panel: ConsolePanel = {
      id: "panel-list",
      type: "markdown",
      content: {
        body: 'Services: {{ join(rows.map(item, item.service), ", ") }}',
        variables: [{ name: "rows", source: { kind: "memory", namespace: "deploys", mode: "list" } }],
      },
    };
    renderWithVariables({
      panel,
      memoryEntries: [
        {
          id: "row-old",
          namespace: "deploys",
          values: { service: "api" },
          source: "node",
          createdAt: "2026-06-01T00:00:00Z",
        },
        {
          id: "row-new",
          namespace: "deploys",
          values: { service: "web" },
          source: "node",
          createdAt: "2026-06-04T00:00:00Z",
        },
      ],
    });
    const view = await waitFor(() => screen.getByTestId("console-markdown"));
    // Sorted createdAt desc by default, so the newest row comes first.
    expect(view.textContent).toMatch(/Services: web, api/);
  });

  it("renders the original markdown when a referenced variable has no data", async () => {
    const panel: ConsolePanel = {
      id: "panel-3",
      type: "markdown",
      content: {
        body: "Service: {{ recipe.service }}",
        variables: [{ name: "recipe", source: { kind: "memory", namespace: "missing" } }],
      },
    };
    renderWithVariables({ panel, memoryEntries: [] });
    const view = await waitFor(() => screen.getByTestId("console-markdown"));
    // Empty rather than a thrown error or stack trace.
    expect(view.textContent?.trim()).toBe("Service:");
  });

  it("resolves the first row for duplicate variable names, matching save semantics", async () => {
    // Two variables share the name `recipe`. `normalizeDraftVariables` keeps
    // only the first on save, so the preview/render must resolve the first too
    // (and not the shadowed last one) to avoid showing a value that won't
    // survive a save.
    const panel: ConsolePanel = {
      id: "panel-dup",
      type: "markdown",
      content: {
        body: "Service: {{ recipe.service }}",
        variables: [
          { name: "recipe", source: { kind: "memory", namespace: "first" } },
          { name: "recipe", source: { kind: "memory", namespace: "second" } },
        ],
      },
    };
    renderWithVariables({
      panel,
      memoryEntries: [
        { id: "row-first", namespace: "first", values: { service: "api" }, source: "node" },
        { id: "row-second", namespace: "second", values: { service: "web" }, source: "node" },
      ],
    });
    const view = await waitFor(() => screen.getByTestId("console-markdown"));
    expect(view.textContent).toMatch(/Service: api/);
    expect(view.textContent).not.toMatch(/web/);
  });
});
