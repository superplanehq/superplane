import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import type * as SdkGen from "@/api-client/sdk.gen";
import { beforeEach, describe, expect, it, vi } from "vitest";

const { canvasesListCanvasMemories } = vi.hoisted(() => ({
  canvasesListCanvasMemories: vi.fn(),
}));

vi.mock("@/api-client/sdk.gen", async (importOriginal) => {
  const actual = await importOriginal<typeof SdkGen>();
  return { ...actual, canvasesListCanvasMemories };
});

import { QueryBlock } from "./QueryBlock";

function renderWithClient(ui: ReactNode) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
      mutations: { retry: false },
    },
  });
  return render(<QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>);
}

const baseBody = `source: memory
namespace: environments`;

beforeEach(() => {
  canvasesListCanvasMemories.mockReset();
});

describe("QueryBlock", () => {
  it("renders a table with sorted column union from filtered entries", async () => {
    canvasesListCanvasMemories.mockResolvedValue({
      data: {
        items: [
          { id: "a", namespace: "environments", values: { name: "alpha", status: "ready" } },
          { id: "b", namespace: "environments", values: { name: "beta", region: "us-east-1" } },
          { id: "c", namespace: "other", values: { name: "ignored" } },
        ],
      },
    });

    renderWithClient(<QueryBlock body={baseBody} canvasId="canvas-1" />);

    await waitFor(() => {
      expect(screen.getByTestId("canvas-query-block")).toBeInTheDocument();
    });

    const headers = screen.getAllByRole("columnheader").map((th) => th.textContent);
    // Columns sorted alphabetically across union of `values` keys.
    expect(headers).toEqual(["name", "region", "status"]);

    // Two rows (third entry filtered out by namespace).
    const rows = screen.getAllByRole("row");
    expect(rows).toHaveLength(3); // header + 2 data rows

    expect(screen.getByText("alpha")).toBeInTheDocument();
    expect(screen.getByText("beta")).toBeInTheDocument();
    expect(screen.getByText("ready")).toBeInTheDocument();
    expect(screen.getByText("us-east-1")).toBeInTheDocument();
  });

  it("stringifies non-string values", async () => {
    canvasesListCanvasMemories.mockResolvedValue({
      data: {
        items: [
          {
            id: "a",
            namespace: "environments",
            values: { count: 42, ready: true, missing: null },
          },
        ],
      },
    });

    renderWithClient(<QueryBlock body={baseBody} canvasId="canvas-1" />);

    await waitFor(() => {
      expect(screen.getByTestId("canvas-query-block")).toBeInTheDocument();
    });

    expect(screen.getByText("42")).toBeInTheDocument();
    expect(screen.getByText("true")).toBeInTheDocument();
  });

  it("renders the muted empty state when no entries match the namespace", async () => {
    canvasesListCanvasMemories.mockResolvedValue({
      data: {
        items: [{ id: "c", namespace: "other", values: { name: "ignored" } }],
      },
    });

    renderWithClient(<QueryBlock body={baseBody} canvasId="canvas-1" />);

    await waitFor(() => {
      expect(screen.getByTestId("canvas-query-block-empty")).toBeInTheDocument();
    });
    expect(screen.getByText(/No entries in/)).toBeInTheDocument();
    expect(screen.getByText(/environments/)).toBeInTheDocument();
  });

  it("renders an inline error for invalid YAML", () => {
    renderWithClient(<QueryBlock body={"source: memory\n  namespace: bad: indent"} canvasId="canvas-1" />);

    const error = screen.getByTestId("canvas-query-block-error");
    expect(error).toBeInTheDocument();
    expect(error.textContent).toMatch(/Invalid query block/);
    expect(canvasesListCanvasMemories).not.toHaveBeenCalled();
  });

  it("renders an inline error when namespace is missing", () => {
    renderWithClient(<QueryBlock body={"source: memory"} canvasId="canvas-1" />);

    const error = screen.getByTestId("canvas-query-block-error");
    expect(error).toBeInTheDocument();
    expect(error.textContent).toMatch(/missing `namespace`/);
    expect(canvasesListCanvasMemories).not.toHaveBeenCalled();
  });

  it("renders an inline error for unsupported source", () => {
    renderWithClient(<QueryBlock body={"source: runs\nnamespace: x"} canvasId="canvas-1" />);

    const error = screen.getByTestId("canvas-query-block-error");
    expect(error).toBeInTheDocument();
    expect(error.textContent).toMatch(/unsupported source/);
    expect(canvasesListCanvasMemories).not.toHaveBeenCalled();
  });

  it("renders the loading skeleton while the query is in flight", () => {
    canvasesListCanvasMemories.mockReturnValue(new Promise(() => {}));

    renderWithClient(<QueryBlock body={baseBody} canvasId="canvas-1" />);

    expect(screen.getByTestId("canvas-query-block-skeleton")).toBeInTheDocument();
  });

  it("renders an inline error when the API call fails", async () => {
    canvasesListCanvasMemories.mockRejectedValue(new Error("boom"));

    renderWithClient(<QueryBlock body={baseBody} canvasId="canvas-1" />);

    const error = await screen.findByTestId("canvas-query-block-error");
    expect(error.textContent).toMatch(/Failed to load memory: boom/);
  });
});
