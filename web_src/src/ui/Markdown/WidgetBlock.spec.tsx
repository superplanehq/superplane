import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import type * as SdkGen from "@/api-client/sdk.gen";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const { canvasesListCanvasMemories, canvasesListCanvasEvents } = vi.hoisted(() => ({
  canvasesListCanvasMemories: vi.fn(),
  canvasesListCanvasEvents: vi.fn(),
}));

vi.mock("@/api-client/sdk.gen", async (importOriginal) => {
  const actual = await importOriginal<typeof SdkGen>();
  return { ...actual, canvasesListCanvasMemories, canvasesListCanvasEvents };
});

const showSuccessToast = vi.fn();
const showErrorToast = vi.fn();
vi.mock("@/lib/toast", () => ({
  showSuccessToast: (...args: unknown[]) => showSuccessToast(...args),
  showErrorToast: (...args: unknown[]) => showErrorToast(...args),
}));

// Mock recharts so we can assert on the shaped data and the props the widget
// passes to Bar / Line / Area / Pie / Cell. ResponsiveContainer is replaced with
// a fixed-size div so the chart subtree renders inside jsdom.
vi.mock("recharts", () => {
  type AnyProps = { children?: ReactNode } & Record<string, unknown>;

  const Box =
    (testid: string) =>
    ({ children, ...rest }: AnyProps) => {
      const dataAttrs: Record<string, string> = { "data-testid": testid };
      if (typeof rest.data !== "undefined") {
        dataAttrs["data-rows"] = JSON.stringify(rest.data);
      }
      return <div {...dataAttrs}>{children}</div>;
    };

  return {
    ResponsiveContainer: ({ children }: AnyProps) => (
      <div data-testid="rc-container" style={{ width: 800, height: 400 }}>
        {children}
      </div>
    ),
    BarChart: Box("rc-barchart"),
    LineChart: Box("rc-linechart"),
    AreaChart: Box("rc-areachart"),
    PieChart: Box("rc-piechart"),
    Bar: ({ dataKey, fill, stackId }: AnyProps) => (
      <div
        data-testid={`rc-bar-${String(dataKey)}`}
        data-key={String(dataKey)}
        data-fill={fill ? String(fill) : ""}
        data-stack-id={stackId ? String(stackId) : ""}
      />
    ),
    Line: ({ dataKey, stroke }: AnyProps) => (
      <div
        data-testid={`rc-line-${String(dataKey)}`}
        data-key={String(dataKey)}
        data-stroke={stroke ? String(stroke) : ""}
      />
    ),
    Area: ({ dataKey, stroke, fill }: AnyProps) => (
      <div
        data-testid={`rc-area-${String(dataKey)}`}
        data-key={String(dataKey)}
        data-stroke={stroke ? String(stroke) : ""}
        data-fill={fill ? String(fill) : ""}
      />
    ),
    Pie: ({ children, data, dataKey, nameKey }: AnyProps) => (
      <div
        data-testid="rc-pie"
        data-rows={JSON.stringify(data ?? [])}
        data-key={String(dataKey ?? "")}
        data-name-key={String(nameKey ?? "")}
      >
        {children}
      </div>
    ),
    Cell: ({ fill }: AnyProps) => <div data-testid="rc-cell" data-fill={fill ? String(fill) : ""} />,
    XAxis: () => null,
    YAxis: () => null,
    CartesianGrid: () => null,
    Tooltip: () => null,
    Legend: () => null,
  };
});

import { WidgetBlock } from "./WidgetBlock";

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
  canvasesListCanvasEvents.mockReset();
  showSuccessToast.mockReset();
  showErrorToast.mockReset();
});

describe("WidgetBlock", () => {
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

    renderWithClient(<WidgetBlock body={baseBody} canvasId="canvas-1" />);

    await waitFor(() => {
      expect(screen.getByTestId("canvas-widget-block")).toBeInTheDocument();
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

    renderWithClient(<WidgetBlock body={baseBody} canvasId="canvas-1" />);

    await waitFor(() => {
      expect(screen.getByTestId("canvas-widget-block")).toBeInTheDocument();
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

    renderWithClient(<WidgetBlock body={baseBody} canvasId="canvas-1" />);

    await waitFor(() => {
      expect(screen.getByTestId("canvas-widget-block-empty")).toBeInTheDocument();
    });
    expect(screen.getByText(/No entries in/)).toBeInTheDocument();
    expect(screen.getByText(/environments/)).toBeInTheDocument();
  });

  it("renders an inline error for invalid YAML", () => {
    renderWithClient(<WidgetBlock body={"source: memory\n  namespace: bad: indent"} canvasId="canvas-1" />);

    const error = screen.getByTestId("canvas-widget-block-error");
    expect(error).toBeInTheDocument();
    expect(error.textContent).toMatch(/Invalid widget block/);
    expect(canvasesListCanvasMemories).not.toHaveBeenCalled();
  });

  it("renders an inline error when namespace is missing", () => {
    renderWithClient(<WidgetBlock body={"source: memory"} canvasId="canvas-1" />);

    const error = screen.getByTestId("canvas-widget-block-error");
    expect(error).toBeInTheDocument();
    expect(error.textContent).toMatch(/missing `namespace`/);
    expect(canvasesListCanvasMemories).not.toHaveBeenCalled();
  });

  it("renders an inline error for unsupported source", () => {
    renderWithClient(<WidgetBlock body={"source: runs\nnamespace: x"} canvasId="canvas-1" />);

    const error = screen.getByTestId("canvas-widget-block-error");
    expect(error).toBeInTheDocument();
    expect(error.textContent).toMatch(/unsupported source/);
    expect(canvasesListCanvasMemories).not.toHaveBeenCalled();
  });

  it("renders the loading skeleton while the query is in flight", () => {
    canvasesListCanvasMemories.mockReturnValue(new Promise(() => {}));

    renderWithClient(<WidgetBlock body={baseBody} canvasId="canvas-1" />);

    expect(screen.getByTestId("canvas-widget-block-skeleton")).toBeInTheDocument();
  });

  it("renders an inline error when the API call fails", async () => {
    canvasesListCanvasMemories.mockRejectedValue(new Error("boom"));

    renderWithClient(<WidgetBlock body={baseBody} canvasId="canvas-1" />);

    const error = await screen.findByTestId("canvas-widget-block-error");
    expect(error.textContent).toMatch(/Failed to load memory: boom/);
  });
});

describe("WidgetBlock custom columns", () => {
  function bodyWithColumns(columnsYaml: string): string {
    return `source: memory\nnamespace: environments\ncolumns:\n${columnsYaml}`;
  }

  it("uses author-defined labels and field mapping (not raw keys)", async () => {
    canvasesListCanvasMemories.mockResolvedValue({
      data: {
        items: [
          {
            id: "a",
            namespace: "environments",
            values: { pr_number: "42", pr_title: "Fix bug", url: "https://example.com" },
          },
        ],
      },
    });

    renderWithClient(
      <WidgetBlock
        body={bodyWithColumns("  - label: PR\n    field: pr_number\n  - label: Title\n    field: pr_title\n")}
        canvasId="canvas-1"
      />,
    );

    await waitFor(() => {
      expect(screen.getByTestId("canvas-widget-block")).toBeInTheDocument();
    });

    const headers = screen.getAllByRole("columnheader").map((th) => th.textContent);
    expect(headers).toEqual(["PR", "Title"]);
    expect(screen.getByText("42")).toBeInTheDocument();
    expect(screen.getByText("Fix bug")).toBeInTheDocument();
    // `url` field exists in the data but is not selected by columns.
    expect(screen.queryByText("https://example.com")).toBeNull();
  });

  it("renders an empty cell when a column.field is missing on a row", async () => {
    canvasesListCanvasMemories.mockResolvedValue({
      data: {
        items: [{ id: "a", namespace: "environments", values: { name: "alpha" } }],
      },
    });

    renderWithClient(
      <WidgetBlock
        body={bodyWithColumns("  - label: Name\n    field: name\n  - label: Region\n    field: region\n")}
        canvasId="canvas-1"
      />,
    );

    await waitFor(() => {
      expect(screen.getByTestId("canvas-widget-block")).toBeInTheDocument();
    });

    const cells = screen.getAllByRole("cell");
    expect(cells).toHaveLength(2);
    expect(cells[0].textContent).toBe("alpha");
    expect(cells[1].textContent).toBe("");
  });

  it("renders format: link as an anchor with truncated display for long URLs", async () => {
    const longUrl = "https://" + "x".repeat(100);
    canvasesListCanvasMemories.mockResolvedValue({
      data: { items: [{ id: "a", namespace: "environments", values: { url: longUrl } }] },
    });

    renderWithClient(
      <WidgetBlock body={bodyWithColumns("  - label: URL\n    field: url\n    format: link\n")} canvasId="canvas-1" />,
    );

    await waitFor(() => {
      expect(screen.getByTestId("canvas-widget-block")).toBeInTheDocument();
    });

    const link = screen.getByRole("link");
    expect(link.getAttribute("href")).toBe(longUrl);
    expect(link.getAttribute("target")).toBe("_blank");
    expect(link.getAttribute("rel")).toContain("noopener");
    expect(link.textContent).toMatch(/…/);
    expect((link.textContent ?? "").length).toBeLessThanOrEqual(longUrl.length);
  });

  it("renders format: link:Open with the static label as anchor text", async () => {
    canvasesListCanvasMemories.mockResolvedValue({
      data: {
        items: [{ id: "a", namespace: "environments", values: { url: "https://example.com" } }],
      },
    });

    renderWithClient(
      <WidgetBlock
        body={bodyWithColumns("  - label: Preview\n    field: url\n    format: link:Open\n")}
        canvasId="canvas-1"
      />,
    );

    await waitFor(() => {
      expect(screen.getByTestId("canvas-widget-block")).toBeInTheDocument();
    });

    const link = screen.getByRole("link");
    expect(link.getAttribute("href")).toBe("https://example.com");
    expect(link.textContent).toContain("Open");
  });

  it("renders format: relative as 'X ago' with the absolute UTC as a title", async () => {
    const oneHourAgo = new Date(Date.now() - 1000 * 60 * 60).toISOString();
    canvasesListCanvasMemories.mockResolvedValue({
      data: { items: [{ id: "a", namespace: "environments", values: { created_at: oneHourAgo } }] },
    });

    renderWithClient(
      <WidgetBlock
        body={bodyWithColumns("  - label: Created\n    field: created_at\n    format: relative\n")}
        canvasId="canvas-1"
      />,
    );

    await waitFor(() => {
      expect(screen.getByTestId("canvas-widget-block")).toBeInTheDocument();
    });

    const cell = screen.getAllByRole("cell")[0];
    expect(cell.textContent ?? "").toMatch(/ago/);
    const inner = cell.querySelector("[title]");
    expect(inner).not.toBeNull();
    expect(inner?.getAttribute("title")).toMatch(/UTC/);
  });

  it("falls back to raw value for format: relative on unparseable input", async () => {
    canvasesListCanvasMemories.mockResolvedValue({
      data: { items: [{ id: "a", namespace: "environments", values: { created_at: "not-a-date" } }] },
    });

    renderWithClient(
      <WidgetBlock
        body={bodyWithColumns("  - label: Created\n    field: created_at\n    format: relative\n")}
        canvasId="canvas-1"
      />,
    );

    await waitFor(() => {
      expect(screen.getByTestId("canvas-widget-block")).toBeInTheDocument();
    });

    expect(screen.getByText("not-a-date")).toBeInTheDocument();
  });

  it("renders format: date as 'YYYY-MM-DD HH:mm UTC'", async () => {
    canvasesListCanvasMemories.mockResolvedValue({
      data: {
        items: [{ id: "a", namespace: "environments", values: { created_at: "2026-04-30T07:09:45Z" } }],
      },
    });

    renderWithClient(
      <WidgetBlock
        body={bodyWithColumns("  - label: Created\n    field: created_at\n    format: date\n")}
        canvasId="canvas-1"
      />,
    );

    await waitFor(() => {
      expect(screen.getByTestId("canvas-widget-block")).toBeInTheDocument();
    });

    expect(screen.getByText("2026-04-30 07:09 UTC")).toBeInTheDocument();
  });

  it("renders format: badge as a styled pill with the value", async () => {
    canvasesListCanvasMemories.mockResolvedValue({
      data: { items: [{ id: "a", namespace: "environments", values: { status: "active" } }] },
    });

    renderWithClient(
      <WidgetBlock
        body={bodyWithColumns("  - label: Status\n    field: status\n    format: badge\n")}
        canvasId="canvas-1"
      />,
    );

    await waitFor(() => {
      expect(screen.getByTestId("canvas-widget-block")).toBeInTheDocument();
    });

    const badge = screen.getByText("active");
    expect(badge.tagName).toBe("SPAN");
    expect(badge.className).toContain("bg-emerald-100");
    expect(badge.className).toContain("rounded-full");
  });

  it("renders format: code as a <code> element", async () => {
    canvasesListCanvasMemories.mockResolvedValue({
      data: { items: [{ id: "a", namespace: "environments", values: { sha: "abc123" } }] },
    });

    renderWithClient(
      <WidgetBlock body={bodyWithColumns("  - label: SHA\n    field: sha\n    format: code\n")} canvasId="canvas-1" />,
    );

    await waitFor(() => {
      expect(screen.getByTestId("canvas-widget-block")).toBeInTheDocument();
    });

    const code = screen.getByText("abc123");
    expect(code.tagName).toBe("CODE");
  });

  it("falls back to plain text and warns once for an unknown format", async () => {
    const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
    canvasesListCanvasMemories.mockResolvedValue({
      data: { items: [{ id: "a", namespace: "environments", values: { name: "alpha" } }] },
    });

    renderWithClient(
      <WidgetBlock
        body={bodyWithColumns("  - label: Name\n    field: name\n    format: bogus\n")}
        canvasId="canvas-1"
      />,
    );

    await waitFor(() => {
      expect(screen.getByTestId("canvas-widget-block")).toBeInTheDocument();
    });

    expect(screen.getByText("alpha")).toBeInTheDocument();
    expect(warn).toHaveBeenCalled();
    const message = warn.mock.calls[0]?.[0];
    expect(message).toMatch(/Unknown format "bogus"/);
    warn.mockRestore();
  });

  it("rejects malformed columns with an inline error", () => {
    renderWithClient(
      <WidgetBlock body={"source: memory\nnamespace: env\ncolumns:\n  - label: PR\n"} canvasId="canvas-1" />,
    );

    const error = screen.getByTestId("canvas-widget-block-error");
    expect(error.textContent).toMatch(/columns\[0\] missing `field`/);
    expect(canvasesListCanvasMemories).not.toHaveBeenCalled();
  });
});

describe("WidgetBlock filters", () => {
  function bodyWithWhere(whereYaml: string): string {
    return `source: memory\nnamespace: environments\nwhere:\n${whereYaml}`;
  }

  const sample = {
    data: {
      items: [
        { id: "a", namespace: "environments", values: { name: "alpha", repo: "store-js", count: "5" } },
        { id: "b", namespace: "environments", values: { name: "beta", repo: "core", count: "10" } },
        { id: "c", namespace: "environments", values: { name: "gamma", count: "" } },
      ],
    },
  };

  it("filters with op: eq", async () => {
    canvasesListCanvasMemories.mockResolvedValue(sample);
    renderWithClient(
      <WidgetBlock body={bodyWithWhere("  - field: repo\n    op: eq\n    value: core\n")} canvasId="c1" />,
    );

    await waitFor(() => {
      expect(screen.getByTestId("canvas-widget-block")).toBeInTheDocument();
    });

    expect(screen.getByText("beta")).toBeInTheDocument();
    expect(screen.queryByText("alpha")).toBeNull();
    expect(screen.queryByText("gamma")).toBeNull();
  });

  it("filters with op: neq", async () => {
    canvasesListCanvasMemories.mockResolvedValue(sample);
    renderWithClient(
      <WidgetBlock body={bodyWithWhere("  - field: repo\n    op: neq\n    value: core\n")} canvasId="c1" />,
    );

    await waitFor(() => {
      expect(screen.getByTestId("canvas-widget-block")).toBeInTheDocument();
    });

    expect(screen.getByText("alpha")).toBeInTheDocument();
    // gamma has no `repo` field -> neq excludes (non-existence ops fail closed).
    expect(screen.queryByText("gamma")).toBeNull();
    expect(screen.queryByText("beta")).toBeNull();
  });

  it("filters with op: contains and not_contains", async () => {
    canvasesListCanvasMemories.mockResolvedValue(sample);
    const { rerender } = renderWithClient(
      <WidgetBlock body={bodyWithWhere("  - field: repo\n    op: contains\n    value: store\n")} canvasId="c1" />,
    );

    await waitFor(() => {
      expect(screen.getByText("alpha")).toBeInTheDocument();
    });
    expect(screen.queryByText("beta")).toBeNull();

    rerender(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <WidgetBlock body={bodyWithWhere("  - field: repo\n    op: not_contains\n    value: store\n")} canvasId="c1" />
      </QueryClientProvider>,
    );

    await waitFor(() => {
      expect(screen.getByText("beta")).toBeInTheDocument();
    });
    expect(screen.queryByText("alpha")).toBeNull();
  });

  it("filters with op: gt and lt on numeric strings", async () => {
    canvasesListCanvasMemories.mockResolvedValue(sample);
    renderWithClient(
      <WidgetBlock body={bodyWithWhere("  - field: count\n    op: gt\n    value: 7\n")} canvasId="c1" />,
    );

    await waitFor(() => {
      expect(screen.getByText("beta")).toBeInTheDocument();
    });
    expect(screen.queryByText("alpha")).toBeNull();
  });

  it("excludes rows with non-numeric values for gt/lt", async () => {
    canvasesListCanvasMemories.mockResolvedValue({
      data: {
        items: [
          { id: "a", namespace: "environments", values: { name: "num", count: "5" } },
          { id: "b", namespace: "environments", values: { name: "junk", count: "not-a-number" } },
        ],
      },
    });
    renderWithClient(
      <WidgetBlock body={bodyWithWhere("  - field: count\n    op: lt\n    value: 100\n")} canvasId="c1" />,
    );

    await waitFor(() => {
      expect(screen.getByText("num")).toBeInTheDocument();
    });
    expect(screen.queryByText("junk")).toBeNull();
  });

  it("filters with op: exists / not_exists treating empty string as missing", async () => {
    canvasesListCanvasMemories.mockResolvedValue(sample);
    const { rerender } = renderWithClient(
      <WidgetBlock body={bodyWithWhere("  - field: repo\n    op: exists\n")} canvasId="c1" />,
    );

    await waitFor(() => {
      expect(screen.getByText("alpha")).toBeInTheDocument();
    });
    expect(screen.getByText("beta")).toBeInTheDocument();
    expect(screen.queryByText("gamma")).toBeNull();

    rerender(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <WidgetBlock body={bodyWithWhere("  - field: count\n    op: not_exists\n")} canvasId="c1" />
      </QueryClientProvider>,
    );

    await waitFor(() => {
      expect(screen.getByText("gamma")).toBeInTheDocument();
    });
    expect(screen.queryByText("alpha")).toBeNull();
    expect(screen.queryByText("beta")).toBeNull();
  });

  it("ANDs multiple where conditions together", async () => {
    canvasesListCanvasMemories.mockResolvedValue(sample);
    renderWithClient(
      <WidgetBlock
        body={bodyWithWhere(
          "  - field: repo\n    op: eq\n    value: core\n  - field: count\n    op: gt\n    value: 5\n",
        )}
        canvasId="c1"
      />,
    );

    await waitFor(() => {
      expect(screen.getByText("beta")).toBeInTheDocument();
    });
    expect(screen.queryByText("alpha")).toBeNull();
    expect(screen.queryByText("gamma")).toBeNull();
  });

  it("renders an inline error for an unknown operator", () => {
    renderWithClient(
      <WidgetBlock body={bodyWithWhere("  - field: repo\n    op: weirdop\n    value: x\n")} canvasId="c1" />,
    );

    const error = screen.getByTestId("canvas-widget-block-error");
    expect(error.textContent).toMatch(/Unknown filter operator: "weirdop"/);
    expect(canvasesListCanvasMemories).not.toHaveBeenCalled();
  });

  it("renders an inline error when value is missing for an op that requires it", () => {
    renderWithClient(<WidgetBlock body={bodyWithWhere("  - field: repo\n    op: eq\n")} canvasId="c1" />);

    const error = screen.getByTestId("canvas-widget-block-error");
    expect(error.textContent).toMatch(/where\[0\] missing `value`/);
    expect(canvasesListCanvasMemories).not.toHaveBeenCalled();
  });
});

describe("WidgetBlock backward compatibility", () => {
  it("omitting columns and where reproduces increment-1 auto-column behavior", async () => {
    canvasesListCanvasMemories.mockResolvedValue({
      data: {
        items: [
          { id: "a", namespace: "environments", values: { z: "1", a: "2" } },
          { id: "b", namespace: "environments", values: { m: "3", a: "4" } },
        ],
      },
    });

    renderWithClient(<WidgetBlock body={baseBody} canvasId="canvas-1" />);

    await waitFor(() => {
      expect(screen.getByTestId("canvas-widget-block")).toBeInTheDocument();
    });

    const headers = screen.getAllByRole("columnheader").map((th) => th.textContent);
    expect(headers).toEqual(["a", "m", "z"]);
  });
});

describe("WidgetBlock actions", () => {
  function bodyWithActions(actionsYaml: string, columnsYaml = "  - label: PR\n    field: pr_number\n"): string {
    return `source: memory\nnamespace: environments\ncolumns:\n${columnsYaml}actions:\n${actionsYaml}`;
  }

  type EmitEventFn = (input: { nodeSlug: string; channel: string; data: unknown }) => Promise<void>;

  function renderWithActions(
    body: string,
    opts?: {
      onEmitEvent?: EmitEventFn;
      nodeIds?: Record<string, string>;
      memory?: Array<{ id: string; namespace: string; values: Record<string, unknown> }>;
    },
  ) {
    canvasesListCanvasMemories.mockResolvedValue({
      data: {
        items: opts?.memory ?? [
          { id: "a", namespace: "environments", values: { pr_number: "69", pr_title: "Fix bug" } },
        ],
      },
    });
    const onEmitEvent: EmitEventFn = opts?.onEmitEvent ?? vi.fn(async () => undefined);
    const nodeIds = opts?.nodeIds ?? { destroy: "node-1" };
    const result = renderWithClient(
      <WidgetBlock body={body} canvasId="c1" nodeRefs={{ nodes: { destroy: "Destroy" }, nodeIds, onEmitEvent }} />,
    );
    return { onEmitEvent, ...result };
  }

  it("renders an Actions column when actions are present, and one button per action per row", async () => {
    const body = bodyWithActions("  - label: Destroy\n    trigger: destroy\n    variant: danger\n");
    renderWithActions(body);

    await waitFor(() => {
      expect(screen.getByTestId("canvas-widget-block")).toBeInTheDocument();
    });

    const headers = screen.getAllByRole("columnheader").map((th) => th.textContent);
    expect(headers).toEqual(["PR", "Actions"]);

    const button = screen.getByTestId("canvas-widget-block-action-destroy");
    expect(button).toBeInTheDocument();
    expect(button.textContent).toContain("Destroy");
    expect(button.getAttribute("data-variant")).toBe("danger");
    expect(button.className).toContain("bg-red-50");
  });

  it("does not render an Actions column when actions are omitted", async () => {
    canvasesListCanvasMemories.mockResolvedValue({
      data: {
        items: [{ id: "a", namespace: "environments", values: { pr_number: "1" } }],
      },
    });
    renderWithClient(
      <WidgetBlock
        body={"source: memory\nnamespace: environments\ncolumns:\n  - label: PR\n    field: pr_number\n"}
        canvasId="c1"
      />,
    );

    await waitFor(() => {
      expect(screen.getByTestId("canvas-widget-block")).toBeInTheDocument();
    });

    const headers = screen.getAllByRole("columnheader").map((th) => th.textContent);
    expect(headers).toEqual(["PR"]);
  });

  it("clicks fire onEmitEvent with the merged fill payload (no confirm)", async () => {
    const body = bodyWithActions(
      '  - label: Destroy\n    trigger: destroy\n    fill:\n      data.issue.number: "{{pr_number}}"\n',
    );
    const { onEmitEvent } = renderWithActions(body);

    await waitFor(() => {
      expect(screen.getByTestId("canvas-widget-block-action-destroy")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByTestId("canvas-widget-block-action-destroy"));

    await waitFor(() => {
      expect(onEmitEvent).toHaveBeenCalledTimes(1);
    });
    expect(onEmitEvent).toHaveBeenCalledWith({
      nodeSlug: "destroy",
      channel: "default",
      data: { data: { issue: { number: "69" } } },
    });
  });

  it("interpolates missing fields as empty string", async () => {
    const body = bodyWithActions(
      '  - label: Destroy\n    trigger: destroy\n    fill:\n      data.missing: "{{not_in_row}}"\n',
    );
    const { onEmitEvent } = renderWithActions(body);

    await waitFor(() => {
      expect(screen.getByTestId("canvas-widget-block-action-destroy")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByTestId("canvas-widget-block-action-destroy"));
    await waitFor(() => {
      expect(onEmitEvent).toHaveBeenCalled();
    });
    expect(onEmitEvent).toHaveBeenCalledWith(expect.objectContaining({ data: { data: { missing: "" } } }));
  });

  it("opens a confirm dialog when `confirm` is set; only Confirm fires the event", async () => {
    const body = bodyWithActions(
      '  - label: Destroy\n    trigger: destroy\n    confirm: "Destroy PR #{{pr_number}}?"\n',
    );
    const { onEmitEvent } = renderWithActions(body);

    await waitFor(() => {
      expect(screen.getByTestId("canvas-widget-block-action-destroy")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByTestId("canvas-widget-block-action-destroy"));

    // Dialog body has the interpolated text.
    await screen.findByText("Destroy PR #69?");
    expect(onEmitEvent).not.toHaveBeenCalled();

    // Cancel closes without firing.
    fireEvent.click(screen.getByText("Cancel"));
    await waitFor(() => {
      expect(screen.queryByText("Destroy PR #69?")).toBeNull();
    });
    expect(onEmitEvent).not.toHaveBeenCalled();

    // Reopen and confirm.
    fireEvent.click(screen.getByTestId("canvas-widget-block-action-destroy"));
    await screen.findByText("Destroy PR #69?");
    fireEvent.click(screen.getByTestId("canvas-widget-block-action-confirm-destroy"));

    await waitFor(() => {
      expect(onEmitEvent).toHaveBeenCalledTimes(1);
    });
  });

  it("shows the success toast with the action label after firing", async () => {
    const body = bodyWithActions("  - label: Destroy\n    trigger: destroy\n");
    renderWithActions(body);

    await waitFor(() => {
      expect(screen.getByTestId("canvas-widget-block-action-destroy")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByTestId("canvas-widget-block-action-destroy"));

    await waitFor(() => {
      expect(showSuccessToast).toHaveBeenCalledWith("Triggered: Destroy");
    });
  });

  it("renders an inline error near the button on API rejection (and no toast)", async () => {
    const onEmitEvent: EmitEventFn = vi.fn(async () => {
      throw new Error("nope");
    });
    const body = bodyWithActions("  - label: Destroy\n    trigger: destroy\n");
    renderWithActions(body, { onEmitEvent });

    await waitFor(() => {
      expect(screen.getByTestId("canvas-widget-block-action-destroy")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByTestId("canvas-widget-block-action-destroy"));

    const err = await screen.findByTestId("canvas-widget-block-action-error-destroy");
    expect(err.textContent).toContain("nope");
    expect(showSuccessToast).not.toHaveBeenCalled();
  });

  it("disables the button with a tooltip when the trigger slug is unknown", async () => {
    const body = bodyWithActions("  - label: Destroy\n    trigger: ghost\n");
    renderWithActions(body, { nodeIds: {} });

    await waitFor(() => {
      expect(screen.getByTestId("canvas-widget-block-action-ghost")).toBeInTheDocument();
    });
    const button = screen.getByTestId("canvas-widget-block-action-ghost") as HTMLButtonElement;
    expect(button.disabled).toBe(true);
    expect(button.getAttribute("title")).toMatch(/Trigger "ghost" not found/);
  });

  it("renders multiple buttons in a single row when multiple actions are configured", async () => {
    const body = bodyWithActions(
      "  - label: Open\n    trigger: deploy\n    variant: primary\n  - label: Destroy\n    trigger: destroy\n    variant: danger\n",
    );
    renderWithActions(body, { nodeIds: { destroy: "node-1", deploy: "node-2" } });

    await waitFor(() => {
      expect(screen.getByTestId("canvas-widget-block-action-destroy")).toBeInTheDocument();
    });

    const open = screen.getByTestId("canvas-widget-block-action-deploy");
    const destroy = screen.getByTestId("canvas-widget-block-action-destroy");
    expect(open.className).toContain("bg-blue-50");
    expect(destroy.className).toContain("bg-red-50");
  });

  it("warns and falls back when `icon` is unknown", async () => {
    const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
    const body = bodyWithActions("  - label: Destroy\n    trigger: destroy\n    icon: bogus\n");
    renderWithActions(body);

    await waitFor(() => {
      expect(screen.getByTestId("canvas-widget-block-action-destroy")).toBeInTheDocument();
    });
    expect(warn).toHaveBeenCalled();
    expect(warn.mock.calls.some((c) => String(c[0]).includes('Unknown action icon "bogus"'))).toBe(true);
    warn.mockRestore();
  });

  it("warns and falls back to default styling when `variant` is unknown", async () => {
    const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
    const body = bodyWithActions("  - label: Destroy\n    trigger: destroy\n    variant: weird\n");
    renderWithActions(body);

    await waitFor(() => {
      expect(screen.getByTestId("canvas-widget-block-action-destroy")).toBeInTheDocument();
    });
    const button = screen.getByTestId("canvas-widget-block-action-destroy");
    expect(button.getAttribute("data-variant")).toBe("default");
    expect(button.className).toContain("bg-white");
    expect(warn).toHaveBeenCalled();
    warn.mockRestore();
  });

  it("renders a parse error for a malformed action (missing trigger)", () => {
    renderWithClient(
      <WidgetBlock
        body={"source: memory\nnamespace: env\nactions:\n  - label: Destroy\n    kind: trigger\n"}
        canvasId="c1"
        nodeRefs={{ nodes: {}, nodeIds: {}, onEmitEvent: vi.fn(async () => undefined) }}
      />,
    );
    const error = screen.getByTestId("canvas-widget-block-error");
    expect(error.textContent).toMatch(/actions\[0\] missing `trigger`/);
    expect(canvasesListCanvasMemories).not.toHaveBeenCalled();
  });

  it("renders a parse error when both `kind` and `trigger` are missing", () => {
    renderWithClient(
      <WidgetBlock
        body={"source: memory\nnamespace: env\nactions:\n  - label: Destroy\n"}
        canvasId="c1"
        nodeRefs={{ nodes: {}, nodeIds: {}, onEmitEvent: vi.fn(async () => undefined) }}
      />,
    );
    const error = screen.getByTestId("canvas-widget-block-error");
    expect(error.textContent).toMatch(/actions\[0\] missing `kind`/);
    expect(canvasesListCanvasMemories).not.toHaveBeenCalled();
  });

  it("renders a parse error when fill has a non-string value", () => {
    renderWithClient(
      <WidgetBlock
        body={
          "source: memory\nnamespace: env\nactions:\n  - label: Destroy\n    trigger: destroy\n    fill:\n      data.issue.number: 42\n"
        }
        canvasId="c1"
        nodeRefs={{ nodes: {}, nodeIds: {}, onEmitEvent: vi.fn(async () => undefined) }}
      />,
    );
    const error = screen.getByTestId("canvas-widget-block-error");
    expect(error.textContent).toMatch(/actions\[0\]\.fill\.data\.issue\.number must be a string/);
  });
});

//
// v5: Executions data source.
//

type EventsPage = {
  events: Array<{
    id: string;
    nodeId: string;
    createdAt: string;
    data?: unknown;
    executions?: Array<{
      id: string;
      nodeId: string;
      state: string;
      result?: string;
      createdAt?: string;
      updatedAt?: string;
    }>;
    queueItems?: unknown[];
  }>;
  totalCount: number;
};

function mockEventsResponse(events: EventsPage["events"]) {
  canvasesListCanvasEvents.mockResolvedValue({
    data: { events, totalCount: events.length },
  });
}

const sampleRunningEvent: EventsPage["events"][number] = {
  id: "run-1",
  nodeId: "deploy-cmd",
  createdAt: "2026-01-01T10:00:00Z",
  data: { data: { issue: { number: 42 } } },
  executions: [
    {
      id: "exec-1",
      nodeId: "deploy-cmd",
      state: "STATE_STARTED",
      createdAt: "2026-01-01T10:00:00Z",
      updatedAt: "2026-01-01T10:00:30Z",
    },
  ],
  queueItems: [],
};

const samplePassedEvent: EventsPage["events"][number] = {
  id: "run-2",
  nodeId: "deploy-cmd",
  createdAt: "2026-01-01T09:00:00Z",
  data: { data: { issue: { number: 41 } } },
  executions: [
    {
      id: "exec-2",
      nodeId: "deploy-cmd",
      state: "STATE_FINISHED",
      result: "RESULT_PASSED",
      createdAt: "2026-01-01T09:00:00Z",
      updatedAt: "2026-01-01T09:00:45Z",
    },
  ],
  queueItems: [],
};

const sampleFailedEvent: EventsPage["events"][number] = {
  id: "run-3",
  nodeId: "deploy-cmd",
  createdAt: "2026-01-01T08:00:00Z",
  data: { data: { issue: { number: 40 } } },
  executions: [
    {
      id: "exec-3",
      nodeId: "deploy-cmd",
      state: "STATE_FINISHED",
      result: "RESULT_FAILED",
      createdAt: "2026-01-01T08:00:00Z",
      updatedAt: "2026-01-01T08:01:00Z",
    },
  ],
  queueItems: [],
};

const sampleCancelledEvent: EventsPage["events"][number] = {
  id: "run-4",
  nodeId: "deploy-cmd",
  createdAt: "2026-01-01T07:00:00Z",
  data: { data: { issue: { number: 39 } } },
  executions: [
    {
      id: "exec-4",
      nodeId: "deploy-cmd",
      state: "STATE_FINISHED",
      result: "RESULT_CANCELLED",
      createdAt: "2026-01-01T07:00:00Z",
      updatedAt: "2026-01-01T07:00:30Z",
    },
  ],
  queueItems: [],
};

describe("WidgetBlock executions source", () => {
  it("renders rows from useInfiniteCanvasEvents and computes status correctly", async () => {
    mockEventsResponse([sampleRunningEvent, samplePassedEvent, sampleFailedEvent, sampleCancelledEvent]);

    renderWithClient(
      <WidgetBlock
        body={
          "source: executions\ncolumns:\n  - label: PR\n    field: root.data.data.issue.number\n  - label: Status\n    field: status\n    format: badge\n"
        }
        canvasId="c1"
      />,
    );

    await waitFor(() => {
      expect(screen.getByTestId("canvas-widget-block")).toBeInTheDocument();
    });
    expect(screen.getByText("42")).toBeInTheDocument();
    expect(screen.getByText("41")).toBeInTheDocument();
    expect(screen.getByText("40")).toBeInTheDocument();
    expect(screen.getByText("39")).toBeInTheDocument();

    expect(screen.getByText("running").className).toContain("bg-blue-100");
    expect(screen.getByText("passed").className).toContain("bg-emerald-100");
    expect(screen.getByText("failed").className).toContain("bg-red-100");
    expect(screen.getByText("cancelled").className).toContain("bg-slate-100");
  });

  it("filters by trigger", async () => {
    mockEventsResponse([sampleRunningEvent, { ...samplePassedEvent, id: "run-other", nodeId: "other-trigger" }]);

    renderWithClient(
      <WidgetBlock
        body={"source: executions\ntrigger: deploy-cmd\ncolumns:\n  - label: ID\n    field: root.id\n"}
        canvasId="c1"
      />,
    );
    await waitFor(() => {
      expect(screen.getByTestId("canvas-widget-block")).toBeInTheDocument();
    });
    expect(screen.getByText("run-1")).toBeInTheDocument();
    expect(screen.queryByText("run-other")).toBeNull();
  });

  it("filters by status (after normalization)", async () => {
    mockEventsResponse([sampleRunningEvent, samplePassedEvent]);

    renderWithClient(
      <WidgetBlock
        body={"source: executions\nstatus: passed\ncolumns:\n  - label: ID\n    field: root.id\n"}
        canvasId="c1"
      />,
    );
    await waitFor(() => {
      expect(screen.getByTestId("canvas-widget-block")).toBeInTheDocument();
    });
    expect(screen.queryByText("run-1")).toBeNull();
    expect(screen.getByText("run-2")).toBeInTheDocument();
  });

  it("truncates rows with limit", async () => {
    mockEventsResponse([sampleRunningEvent, samplePassedEvent, sampleFailedEvent, sampleCancelledEvent]);

    renderWithClient(
      <WidgetBlock
        body={"source: executions\nlimit: 2\ncolumns:\n  - label: ID\n    field: root.id\n"}
        canvasId="c1"
      />,
    );
    await waitFor(() => {
      expect(screen.getByTestId("canvas-widget-block")).toBeInTheDocument();
    });
    expect(screen.getByText("run-1")).toBeInTheDocument();
    expect(screen.getByText("run-2")).toBeInTheDocument();
    expect(screen.queryByText("run-3")).toBeNull();
    expect(screen.queryByText("run-4")).toBeNull();
  });

  it("renders the empty state when no events match", async () => {
    mockEventsResponse([sampleRunningEvent]);

    renderWithClient(<WidgetBlock body={"source: executions\ntrigger: nope\n"} canvasId="c1" />);
    await waitFor(() => {
      expect(screen.getByTestId("canvas-widget-block-empty")).toBeInTheDocument();
    });
    expect(screen.getByTestId("canvas-widget-block-empty").textContent).toContain("No runs");
  });

  it("rejects an unsupported status", () => {
    renderWithClient(<WidgetBlock body={"source: executions\nstatus: weird\n"} canvasId="c1" />);
    const err = screen.getByTestId("canvas-widget-block-error");
    expect(err.textContent).toMatch(/unsupported status "weird"/);
  });

  it("auto-derives default columns when none are specified", async () => {
    mockEventsResponse([samplePassedEvent]);
    renderWithClient(<WidgetBlock body={"source: executions\n"} canvasId="c1" />);
    await waitFor(() => {
      expect(screen.getByTestId("canvas-widget-block")).toBeInTheDocument();
    });
    const headers = screen.getAllByRole("columnheader").map((th) => th.textContent);
    expect(headers).toContain("Run");
    expect(headers).toContain("Status");
    expect(headers).toContain("Started");
  });
});

//
// v5: Execution actions
//

describe("WidgetBlock execution actions", () => {
  it("kind: cancel resolves the running execution and calls onExecutionAction", async () => {
    mockEventsResponse([sampleRunningEvent]);
    const onExecutionAction = vi.fn(async () => undefined);

    renderWithClient(
      <WidgetBlock
        body={
          "source: executions\ncolumns:\n  - label: ID\n    field: root.id\nactions:\n  - label: Cancel\n    kind: cancel\n    variant: danger\n"
        }
        canvasId="c1"
        nodeRefs={{ onExecutionAction }}
      />,
    );

    await waitFor(() => {
      expect(screen.getByTestId("canvas-widget-block-action-cancel")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByTestId("canvas-widget-block-action-cancel"));

    await waitFor(() => {
      expect(onExecutionAction).toHaveBeenCalledWith({
        kind: "cancel",
        nodeId: "deploy-cmd",
        executionId: "exec-1",
      });
    });
  });

  it("kind: cancel button is hidden when no execution is running", async () => {
    mockEventsResponse([samplePassedEvent]);
    renderWithClient(
      <WidgetBlock
        body={
          "source: executions\ncolumns:\n  - label: ID\n    field: root.id\nactions:\n  - label: Cancel\n    kind: cancel\n"
        }
        canvasId="c1"
        nodeRefs={{ onExecutionAction: vi.fn(async () => undefined) }}
      />,
    );
    await waitFor(() => {
      expect(screen.getByTestId("canvas-widget-block")).toBeInTheDocument();
    });
    expect(screen.queryByTestId("canvas-widget-block-action-cancel")).toBeNull();
  });

  it("kind: approve resolves the matching node execution and dispatches", async () => {
    const evt: EventsPage["events"][number] = {
      ...sampleRunningEvent,
      executions: [
        ...(sampleRunningEvent.executions ?? []),
        {
          id: "exec-approval",
          nodeId: "approval-gate",
          state: "STATE_STARTED",
          createdAt: "2026-01-01T10:00:30Z",
          updatedAt: "2026-01-01T10:00:30Z",
        },
      ],
    };
    mockEventsResponse([evt]);
    const onExecutionAction = vi.fn(async () => undefined);

    renderWithClient(
      <WidgetBlock
        body={
          "source: executions\ncolumns:\n  - label: ID\n    field: root.id\nactions:\n  - label: Approve\n    kind: approve\n    node: approval-gate\n"
        }
        canvasId="c1"
        nodeRefs={{ onExecutionAction }}
      />,
    );

    await waitFor(() => {
      expect(screen.getByTestId("canvas-widget-block-action-approve-approval-gate")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByTestId("canvas-widget-block-action-approve-approval-gate"));

    await waitFor(() => {
      expect(onExecutionAction).toHaveBeenCalledWith({
        kind: "approve",
        nodeId: "approval-gate",
        executionId: "exec-approval",
      });
    });
  });

  it("kind: push-through dispatches with push-through kind", async () => {
    const evt: EventsPage["events"][number] = {
      ...sampleRunningEvent,
      executions: [
        ...(sampleRunningEvent.executions ?? []),
        {
          id: "exec-gate",
          nodeId: "approval-gate",
          state: "STATE_STARTED",
          createdAt: "2026-01-01T10:00:30Z",
          updatedAt: "2026-01-01T10:00:30Z",
        },
      ],
    };
    mockEventsResponse([evt]);
    const onExecutionAction = vi.fn(async () => undefined);

    renderWithClient(
      <WidgetBlock
        body={
          "source: executions\ncolumns:\n  - label: ID\n    field: root.id\nactions:\n  - label: Force\n    kind: push-through\n    node: approval-gate\n"
        }
        canvasId="c1"
        nodeRefs={{ onExecutionAction }}
      />,
    );

    await waitFor(() => {
      expect(screen.getByTestId("canvas-widget-block-action-push-through-approval-gate")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByTestId("canvas-widget-block-action-push-through-approval-gate"));

    await waitFor(() => {
      expect(onExecutionAction).toHaveBeenCalledWith({
        kind: "push-through",
        nodeId: "approval-gate",
        executionId: "exec-gate",
      });
    });
  });

  it("hides approve when no matching execution exists", async () => {
    mockEventsResponse([samplePassedEvent]);
    renderWithClient(
      <WidgetBlock
        body={
          "source: executions\ncolumns:\n  - label: ID\n    field: root.id\nactions:\n  - label: Approve\n    kind: approve\n    node: approval-gate\n"
        }
        canvasId="c1"
        nodeRefs={{ onExecutionAction: vi.fn(async () => undefined) }}
      />,
    );
    await waitFor(() => {
      expect(screen.getByTestId("canvas-widget-block")).toBeInTheDocument();
    });
    expect(screen.queryByTestId(/canvas-widget-block-action-approve/)).toBeNull();
  });

  it('uses `show: status == "running"` to gate the button per row', async () => {
    mockEventsResponse([sampleRunningEvent, samplePassedEvent]);
    renderWithClient(
      <WidgetBlock
        body={
          'source: executions\ncolumns:\n  - label: ID\n    field: root.id\nactions:\n  - label: Cancel\n    kind: cancel\n    show: status == "running"\n'
        }
        canvasId="c1"
        nodeRefs={{ onExecutionAction: vi.fn(async () => undefined) }}
      />,
    );
    await waitFor(() => {
      expect(screen.getByTestId("canvas-widget-block")).toBeInTheDocument();
    });
    // Only one button should be rendered (for the running row).
    const buttons = screen.getAllByTestId("canvas-widget-block-action-cancel");
    expect(buttons).toHaveLength(1);
  });

  it("renders trigger and approve actions side-by-side", async () => {
    const evt: EventsPage["events"][number] = {
      ...sampleRunningEvent,
      executions: [
        ...(sampleRunningEvent.executions ?? []),
        {
          id: "exec-gate",
          nodeId: "approval-gate",
          state: "STATE_STARTED",
          createdAt: "2026-01-01T10:00:30Z",
          updatedAt: "2026-01-01T10:00:30Z",
        },
      ],
    };
    mockEventsResponse([evt]);
    const onEmitEvent = vi.fn(async () => undefined);
    const onExecutionAction = vi.fn(async () => undefined);

    renderWithClient(
      <WidgetBlock
        body={
          "source: executions\ncolumns:\n  - label: ID\n    field: root.id\nactions:\n  - label: Reprovision\n    kind: trigger\n    trigger: redeploy\n  - label: Approve\n    kind: approve\n    node: approval-gate\n"
        }
        canvasId="c1"
        nodeRefs={{
          nodeIds: { redeploy: "node-redeploy" },
          onEmitEvent,
          onExecutionAction,
        }}
      />,
    );
    await waitFor(() => {
      expect(screen.getByTestId("canvas-widget-block-action-redeploy")).toBeInTheDocument();
    });
    expect(screen.getByTestId("canvas-widget-block-action-approve-approval-gate")).toBeInTheDocument();
  });
});

//
// v5: Render: chart
//

describe("WidgetBlock render: chart", () => {
  function readBarChart() {
    const chart = screen.getByTestId("rc-barchart");
    const rows = JSON.parse(chart.getAttribute("data-rows") ?? "[]") as Array<Record<string, unknown>>;
    return { chart, rows };
  }

  it("renders a BarChart with shaped rows for memory data", async () => {
    canvasesListCanvasMemories.mockResolvedValue({
      data: {
        items: [
          { id: "a", namespace: "envs", values: { name: "alpha", duration: 30 } },
          { id: "b", namespace: "envs", values: { name: "beta", duration: 60 } },
        ],
      },
    });

    renderWithClient(
      <WidgetBlock
        body={
          "source: memory\nnamespace: envs\nrender:\n  kind: chart\n  chart:\n    type: bar\n    x: name\n    y: duration\n    label: Provisioning\n"
        }
        canvasId="c1"
      />,
    );

    await waitFor(() => {
      expect(screen.getByTestId("canvas-widget-block-chart")).toBeInTheDocument();
    });
    const { rows } = readBarChart();
    expect(rows).toEqual([
      { x: "alpha", y: 30 },
      { x: "beta", y: 60 },
    ]);
    const bar = screen.getByTestId("rc-bar-y");
    expect(bar.getAttribute("data-fill")).toBe("var(--chart-1)");
  });

  it("renders a LineChart for executions rows", async () => {
    mockEventsResponse([
      { ...samplePassedEvent, id: "ev-a", data: { metric: 10 } },
      { ...samplePassedEvent, id: "ev-b", data: { metric: 20 } },
    ]);

    renderWithClient(
      <WidgetBlock
        body={
          "source: executions\nrender:\n  kind: chart\n  chart:\n    type: line\n    x: root.id\n    y: root.data.metric\n"
        }
        canvasId="c1"
      />,
    );
    await waitFor(() => {
      expect(screen.getByTestId("canvas-widget-block-chart")).toBeInTheDocument();
    });
    const chart = screen.getByTestId("rc-linechart");
    const rows = JSON.parse(chart.getAttribute("data-rows") ?? "[]") as Array<Record<string, unknown>>;
    expect(rows).toEqual([
      { x: "ev-a", y: 10 },
      { x: "ev-b", y: 20 },
    ]);
  });

  it("renders an AreaChart for type: area and respects color: blue", async () => {
    canvasesListCanvasMemories.mockResolvedValue({
      data: {
        items: [
          { id: "a", namespace: "envs", values: { name: "alpha", duration: 30 } },
          { id: "b", namespace: "envs", values: { name: "beta", duration: 60 } },
        ],
      },
    });
    renderWithClient(
      <WidgetBlock
        body={
          "source: memory\nnamespace: envs\nrender:\n  kind: chart\n  chart:\n    type: area\n    x: name\n    y: duration\n    color: blue\n"
        }
        canvasId="c1"
      />,
    );
    await waitFor(() => {
      expect(screen.getByTestId("canvas-widget-block-chart")).toBeInTheDocument();
    });
    expect(screen.getByTestId("rc-areachart")).toBeInTheDocument();
    const area = screen.getByTestId("rc-area-y");
    expect(area.getAttribute("data-fill")).toBe("var(--chart-1)");
    expect(area.getAttribute("data-stroke")).toBe("var(--chart-1)");
  });

  it("collapses duplicate x values with chart.aggregate: avg", async () => {
    canvasesListCanvasMemories.mockResolvedValue({
      data: {
        items: [
          { id: "a", namespace: "envs", values: { region: "us", duration: 10 } },
          { id: "b", namespace: "envs", values: { region: "us", duration: 30 } },
          { id: "c", namespace: "envs", values: { region: "eu", duration: 50 } },
        ],
      },
    });

    renderWithClient(
      <WidgetBlock
        body={
          "source: memory\nnamespace: envs\nrender:\n  kind: chart\n  chart:\n    type: bar\n    x: region\n    y: duration\n    aggregate: avg\n"
        }
        canvasId="c1"
      />,
    );
    await waitFor(() => {
      expect(screen.getByTestId("canvas-widget-block-chart")).toBeInTheDocument();
    });
    const { rows } = readBarChart();
    expect(rows).toEqual([
      { x: "us", y: 20 },
      { x: "eu", y: 50 },
    ]);
  });

  it("drops non-numeric Y values before grouping", async () => {
    canvasesListCanvasMemories.mockResolvedValue({
      data: {
        items: [
          { id: "a", namespace: "envs", values: { name: "alpha", duration: 30 } },
          { id: "b", namespace: "envs", values: { name: "beta", duration: "n/a" } },
          { id: "c", namespace: "envs", values: { name: "gamma", duration: 90 } },
        ],
      },
    });

    renderWithClient(
      <WidgetBlock
        body={
          "source: memory\nnamespace: envs\nrender:\n  kind: chart\n  chart:\n    type: bar\n    x: name\n    y: duration\n"
        }
        canvasId="c1"
      />,
    );
    await waitFor(() => {
      expect(screen.getByTestId("canvas-widget-block-chart")).toBeInTheDocument();
    });
    const { rows } = readBarChart();
    expect(rows).toEqual([
      { x: "alpha", y: 30 },
      { x: "gamma", y: 90 },
    ]);
  });

  it("renders an empty placeholder when no rows match", async () => {
    canvasesListCanvasMemories.mockResolvedValue({ data: { items: [] } });
    renderWithClient(
      <WidgetBlock
        body={
          "source: memory\nnamespace: envs\nrender:\n  kind: chart\n  chart:\n    type: bar\n    x: name\n    y: duration\n"
        }
        canvasId="c1"
      />,
    );
    await waitFor(() => {
      expect(screen.getByTestId("canvas-widget-block-chart-empty")).toBeInTheDocument();
    });
    expect(screen.queryByTestId("rc-barchart")).toBeNull();
  });

  it("rejects unknown chart types", () => {
    renderWithClient(
      <WidgetBlock
        body={
          "source: memory\nnamespace: envs\nrender:\n  kind: chart\n  chart:\n    type: pie\n    x: name\n    y: duration\n"
        }
        canvasId="c1"
      />,
    );
    expect(screen.getByTestId("canvas-widget-block-error").textContent).toContain('render.chart.type "pie"');
  });
});

describe("WidgetBlock render: chart, type: stacked-bar", () => {
  it("requires `group` and rejects without it", () => {
    renderWithClient(
      <WidgetBlock
        body={
          "source: memory\nnamespace: envs\nrender:\n  kind: chart\n  chart:\n    type: stacked-bar\n    x: name\n    y: count\n"
        }
        canvasId="c1"
      />,
    );
    expect(screen.getByTestId("canvas-widget-block-error").textContent).toContain(
      "render.chart.group is required for stacked-bar",
    );
  });

  it("pivots rows into wide format with one Bar per group value", async () => {
    canvasesListCanvasMemories.mockResolvedValue({
      data: {
        items: [
          { id: "a", namespace: "runs", values: { day: "Mon", status: "passed" } },
          { id: "b", namespace: "runs", values: { day: "Mon", status: "failed" } },
          { id: "c", namespace: "runs", values: { day: "Mon", status: "passed" } },
          { id: "d", namespace: "runs", values: { day: "Tue", status: "passed" } },
        ],
      },
    });

    renderWithClient(
      <WidgetBlock
        body={
          "source: memory\nnamespace: runs\nrender:\n  kind: chart\n  chart:\n    type: stacked-bar\n    x: day\n    y: status\n    group: status\n    aggregate: count\n"
        }
        canvasId="c1"
      />,
    );

    await waitFor(() => {
      expect(screen.getByTestId("rc-barchart")).toBeInTheDocument();
    });
    const chart = screen.getByTestId("rc-barchart");
    const rows = JSON.parse(chart.getAttribute("data-rows") ?? "[]") as Array<Record<string, unknown>>;
    expect(rows).toEqual([
      { x: "Mon", passed: 2, failed: 1 },
      { x: "Tue", passed: 1, failed: 0 },
    ]);
    expect(screen.getByTestId("rc-bar-passed")).toBeInTheDocument();
    expect(screen.getByTestId("rc-bar-failed")).toBeInTheDocument();
    expect(screen.getByTestId("rc-bar-passed").getAttribute("data-stack-id")).toBe("a");
    expect(screen.getByTestId("rc-bar-failed").getAttribute("data-stack-id")).toBe("a");
  });
});

describe("WidgetBlock render: chart, type: donut", () => {
  it("counts rows per `x` value when no aggregate is given", async () => {
    canvasesListCanvasMemories.mockResolvedValue({
      data: {
        items: [
          { id: "a", namespace: "runs", values: { status: "passed" } },
          { id: "b", namespace: "runs", values: { status: "passed" } },
          { id: "c", namespace: "runs", values: { status: "failed" } },
        ],
      },
    });

    renderWithClient(
      <WidgetBlock
        body={"source: memory\nnamespace: runs\nrender:\n  kind: chart\n  chart:\n    type: donut\n    x: status\n"}
        canvasId="c1"
      />,
    );

    await waitFor(() => {
      expect(screen.getByTestId("rc-piechart")).toBeInTheDocument();
    });
    const pie = screen.getByTestId("rc-pie");
    const data = JSON.parse(pie.getAttribute("data-rows") ?? "[]") as Array<Record<string, unknown>>;
    expect(data).toEqual([
      { name: "passed", value: 2 },
      { name: "failed", value: 1 },
    ]);
    const cells = screen.getAllByTestId("rc-cell");
    expect(cells).toHaveLength(2);
  });

  it("rejects donut with aggregate: sum but no `y`", () => {
    renderWithClient(
      <WidgetBlock
        body={
          "source: memory\nnamespace: runs\nrender:\n  kind: chart\n  chart:\n    type: donut\n    x: status\n    aggregate: sum\n"
        }
        canvasId="c1"
      />,
    );
    expect(screen.getByTestId("canvas-widget-block-error").textContent).toContain(
      "render.chart.y is required when donut uses aggregate other than count",
    );
  });
});

//
// v5: Render: number
//

describe("WidgetBlock render: number", () => {
  it("aggregate: avg renders the mean", async () => {
    canvasesListCanvasMemories.mockResolvedValue({
      data: {
        items: [
          { id: "a", namespace: "envs", values: { dur: 10 } },
          { id: "b", namespace: "envs", values: { dur: 30 } },
          { id: "c", namespace: "envs", values: { dur: 50 } },
        ],
      },
    });

    renderWithClient(
      <WidgetBlock
        body={
          "source: memory\nnamespace: envs\nrender:\n  kind: number\n  number:\n    field: dur\n    aggregate: avg\n    label: Avg duration\n"
        }
        canvasId="c1"
      />,
    );

    await waitFor(() => {
      expect(screen.getByTestId("canvas-widget-block-number")).toBeInTheDocument();
    });
    expect(screen.getByText("30")).toBeInTheDocument();
    expect(screen.getByText("Avg duration")).toBeInTheDocument();
  });

  it("aggregate: count ignores the field", async () => {
    canvasesListCanvasMemories.mockResolvedValue({
      data: {
        items: [
          { id: "a", namespace: "envs", values: {} },
          { id: "b", namespace: "envs", values: {} },
          { id: "c", namespace: "envs", values: {} },
        ],
      },
    });
    renderWithClient(
      <WidgetBlock
        body={
          "source: memory\nnamespace: envs\nrender:\n  kind: number\n  number:\n    field: anything\n    aggregate: count\n    label: Total\n"
        }
        canvasId="c1"
      />,
    );
    await waitFor(() => {
      expect(screen.getByTestId("canvas-widget-block-number")).toBeInTheDocument();
    });
    expect(screen.getByText("3")).toBeInTheDocument();
  });

  it("format: duration renders human-friendly seconds", async () => {
    canvasesListCanvasMemories.mockResolvedValue({
      data: {
        items: [
          { id: "a", namespace: "envs", values: { dur: 4980 } }, // 4980 / 1 = 4980 -> 1h 23m
        ],
      },
    });
    renderWithClient(
      <WidgetBlock
        body={
          "source: memory\nnamespace: envs\nrender:\n  kind: number\n  number:\n    field: dur\n    aggregate: avg\n    label: Avg\n    format: duration\n"
        }
        canvasId="c1"
      />,
    );
    await waitFor(() => {
      expect(screen.getByTestId("canvas-widget-block-number")).toBeInTheDocument();
    });
    expect(screen.getByText("1h 23m")).toBeInTheDocument();
  });

  it("format: percent renders as fraction × 100", async () => {
    canvasesListCanvasMemories.mockResolvedValue({
      data: { items: [{ id: "a", namespace: "envs", values: { ratio: 0.123 } }] },
    });
    renderWithClient(
      <WidgetBlock
        body={
          "source: memory\nnamespace: envs\nrender:\n  kind: number\n  number:\n    field: ratio\n    aggregate: avg\n    label: Pass rate\n    format: percent\n"
        }
        canvasId="c1"
      />,
    );
    await waitFor(() => {
      expect(screen.getByTestId("canvas-widget-block-number")).toBeInTheDocument();
    });
    expect(screen.getByText("12.3%")).toBeInTheDocument();
  });

  it("renders a dash when no rows are available", async () => {
    canvasesListCanvasMemories.mockResolvedValue({ data: { items: [] } });
    renderWithClient(
      <WidgetBlock
        body={
          "source: memory\nnamespace: envs\nrender:\n  kind: number\n  number:\n    field: dur\n    aggregate: avg\n    label: Avg\n"
        }
        canvasId="c1"
      />,
    );
    await waitFor(() => {
      expect(screen.getByTestId("canvas-widget-block-number")).toBeInTheDocument();
    });
    expect(screen.getByText("—")).toBeInTheDocument();
  });

  it("does not render a sparkline by default", async () => {
    canvasesListCanvasMemories.mockResolvedValue({
      data: {
        items: [
          { id: "a", namespace: "envs", values: { dur: 10 } },
          { id: "b", namespace: "envs", values: { dur: 20 } },
        ],
      },
    });
    renderWithClient(
      <WidgetBlock
        body={
          "source: memory\nnamespace: envs\nrender:\n  kind: number\n  number:\n    field: dur\n    aggregate: avg\n    label: Avg\n"
        }
        canvasId="c1"
      />,
    );
    await waitFor(() => {
      expect(screen.getByTestId("canvas-widget-block-number")).toBeInTheDocument();
    });
    expect(screen.queryByTestId("canvas-widget-block-number-sparkline")).toBeNull();
  });

  it("renders a sparkline area chart when sparkline: true", async () => {
    canvasesListCanvasMemories.mockResolvedValue({
      data: {
        items: [
          { id: "a", namespace: "envs", values: { dur: 10 } },
          { id: "b", namespace: "envs", values: { dur: 30 } },
          { id: "c", namespace: "envs", values: { dur: 50 } },
        ],
      },
    });
    renderWithClient(
      <WidgetBlock
        body={
          "source: memory\nnamespace: envs\nrender:\n  kind: number\n  number:\n    field: dur\n    aggregate: avg\n    label: Avg\n    sparkline: true\n"
        }
        canvasId="c1"
      />,
    );
    await waitFor(() => {
      expect(screen.getByTestId("canvas-widget-block-number-sparkline")).toBeInTheDocument();
    });
    const sparkline = screen.getByTestId("canvas-widget-block-number-sparkline");
    const area = sparkline.querySelector('[data-testid="rc-areachart"]');
    expect(area).not.toBeNull();
    const rows = JSON.parse(area!.getAttribute("data-rows") ?? "[]") as Array<Record<string, unknown>>;
    expect(rows).toEqual([
      { i: 0, y: 10 },
      { i: 1, y: 30 },
      { i: 2, y: 50 },
    ]);
  });

  it("rejects render.number.sparkline when not boolean", () => {
    renderWithClient(
      <WidgetBlock
        body={
          'source: memory\nnamespace: envs\nrender:\n  kind: number\n  number:\n    field: dur\n    aggregate: avg\n    label: Avg\n    sparkline: "yes"\n'
        }
        canvasId="c1"
      />,
    );
    expect(screen.getByTestId("canvas-widget-block-error").textContent).toContain(
      "render.number.sparkline must be a boolean",
    );
  });
});

//
// CEL expressions inside `{{ ... }}`. The expression engine itself has unit
// coverage in widgetExpr.spec.ts; this block focuses on the integration with
// the widget pipeline (filter, columns, chart, number, show, confirm).
//

describe("WidgetBlock CEL expressions", () => {
  // Pin "now" so any expression using `now` produces a deterministic result.
  // 2026-01-01T00:00:00Z = 1767225600 Unix seconds.
  const FIXED_NOW_SEC = 1767225600;
  let dateNowSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    dateNowSpy = vi.spyOn(Date, "now").mockReturnValue(FIXED_NOW_SEC * 1000);
  });

  afterEach(() => {
    dateNowSpy.mockRestore();
    vi.restoreAllMocks();
  });

  it("evaluates a CEL expression in a column field", async () => {
    canvasesListCanvasMemories.mockResolvedValue({
      data: {
        items: [
          {
            id: "a",
            namespace: "environments",
            // 5 minutes ago
            values: { pr_number: "42", created_at: String(FIXED_NOW_SEC - 300) },
          },
        ],
      },
    });

    renderWithClient(
      <WidgetBlock
        body={
          "source: memory\nnamespace: environments\ncolumns:\n" +
          "  - label: PR\n    field: pr_number\n" +
          '  - label: Age\n    field: "{{ duration(int(now) - int(created_at)) }}"\n'
        }
        canvasId="c1"
      />,
    );

    await waitFor(() => {
      expect(screen.getByTestId("canvas-widget-block")).toBeInTheDocument();
    });
    expect(screen.getByText("42")).toBeInTheDocument();
    expect(screen.getByText("5m")).toBeInTheDocument();
  });

  it("filters rows using CEL on both `field` and `value`", async () => {
    canvasesListCanvasMemories.mockResolvedValue({
      data: {
        items: [
          // 30 minutes old — under the 2h threshold, should be excluded.
          { id: "a", namespace: "environments", values: { name: "fresh", created_at: String(FIXED_NOW_SEC - 1800) } },
          // 3 hours old — older than 2h, kept.
          { id: "b", namespace: "environments", values: { name: "stale", created_at: String(FIXED_NOW_SEC - 10800) } },
        ],
      },
    });

    const body =
      "source: memory\nnamespace: environments\nwhere:\n" +
      '  - field: "{{ int(now) - int(created_at) }}"\n    op: gt\n    value: "{{ 7200 }}"\n' +
      "columns:\n  - label: Name\n    field: name\n";

    renderWithClient(<WidgetBlock body={body} canvasId="c1" />);

    await waitFor(() => {
      expect(screen.getByTestId("canvas-widget-block")).toBeInTheDocument();
    });
    expect(screen.getByText("stale")).toBeInTheDocument();
    expect(screen.queryByText("fresh")).toBeNull();
  });

  it("uses a CEL expression for chart Y to compute a derived series", async () => {
    canvasesListCanvasMemories.mockResolvedValue({
      data: {
        items: [
          // 1 minute old → 60s / 60 = 1
          { id: "a", namespace: "envs", values: { name: "alpha", created_at: String(FIXED_NOW_SEC - 60) } },
          // 4 minutes old → 240s / 60 = 4
          { id: "b", namespace: "envs", values: { name: "beta", created_at: String(FIXED_NOW_SEC - 240) } },
        ],
      },
    });

    renderWithClient(
      <WidgetBlock
        body={
          "source: memory\nnamespace: envs\nrender:\n" +
          "  kind: chart\n  chart:\n    type: bar\n" +
          "    x: name\n" +
          '    y: "{{ (int(now) - int(created_at)) / 60 }}"\n'
        }
        canvasId="c1"
      />,
    );

    await waitFor(() => {
      expect(screen.getByTestId("canvas-widget-block-chart")).toBeInTheDocument();
    });
    const chart = screen.getByTestId("rc-barchart");
    const rows = JSON.parse(chart.getAttribute("data-rows") ?? "[]") as Array<Record<string, unknown>>;
    expect(rows).toEqual([
      { x: "alpha", y: 1 },
      { x: "beta", y: 4 },
    ]);
  });

  it("uses a CEL expression for a number aggregate (success-rate ternary)", async () => {
    mockEventsResponse([
      { ...samplePassedEvent, id: "ev-1" },
      { ...samplePassedEvent, id: "ev-2" },
      { ...sampleFailedEvent, id: "ev-3" },
      { ...sampleFailedEvent, id: "ev-4" },
    ]);

    renderWithClient(
      <WidgetBlock
        body={
          "source: executions\nrender:\n" +
          "  kind: number\n  number:\n" +
          '    field: "{{ status == \\"passed\\" ? 1.0 : 0.0 }}"\n' +
          "    aggregate: avg\n    label: Success rate\n    format: percent\n"
        }
        canvasId="c1"
      />,
    );

    await waitFor(() => {
      expect(screen.getByTestId("canvas-widget-block-number")).toBeInTheDocument();
    });
    // 2 of 4 passed → 0.5 → 50.0%
    expect(screen.getByText("50.0%")).toBeInTheDocument();
  });

  it("evaluates a CEL action `show` (truthy expression)", async () => {
    canvasesListCanvasMemories.mockResolvedValue({
      data: {
        items: [
          { id: "a", namespace: "environments", values: { pr_number: "1", status: "running" } },
          { id: "b", namespace: "environments", values: { pr_number: "2", status: "idle" } },
        ],
      },
    });

    renderWithClient(
      <WidgetBlock
        body={
          "source: memory\nnamespace: environments\n" +
          "columns:\n  - label: PR\n    field: pr_number\n" +
          "actions:\n" +
          "  - label: Stop\n    trigger: stop\n" +
          "    show: \"{{ status == 'running' }}\"\n"
        }
        canvasId="c1"
        nodeRefs={{ nodes: { stop: "Stop" }, nodeIds: { stop: "n-1" }, onEmitEvent: vi.fn(async () => undefined) }}
      />,
    );

    await waitFor(() => {
      expect(screen.getByTestId("canvas-widget-block")).toBeInTheDocument();
    });
    // Exactly one Stop button should render — the row with status running.
    expect(screen.getAllByTestId("canvas-widget-block-action-stop")).toHaveLength(1);
  });

  it('keeps the legacy `show: field == "value"` simple comparator working', async () => {
    canvasesListCanvasMemories.mockResolvedValue({
      data: {
        items: [
          { id: "a", namespace: "environments", values: { pr_number: "1", status: "running" } },
          { id: "b", namespace: "environments", values: { pr_number: "2", status: "idle" } },
        ],
      },
    });

    renderWithClient(
      <WidgetBlock
        body={
          "source: memory\nnamespace: environments\n" +
          "columns:\n  - label: PR\n    field: pr_number\n" +
          "actions:\n" +
          "  - label: Stop\n    trigger: stop\n" +
          '    show: status == "running"\n'
        }
        canvasId="c1"
        nodeRefs={{ nodes: { stop: "Stop" }, nodeIds: { stop: "n-1" }, onEmitEvent: vi.fn(async () => undefined) }}
      />,
    );

    await waitFor(() => {
      expect(screen.getByTestId("canvas-widget-block")).toBeInTheDocument();
    });
    expect(screen.getAllByTestId("canvas-widget-block-action-stop")).toHaveLength(1);
  });

  it("renders a confirm template with multiple `{{ }}` segments", async () => {
    canvasesListCanvasMemories.mockResolvedValue({
      data: {
        items: [{ id: "a", namespace: "environments", values: { pr_number: "42", repo: "core" } }],
      },
    });

    renderWithClient(
      <WidgetBlock
        body={
          "source: memory\nnamespace: environments\n" +
          "columns:\n  - label: PR\n    field: pr_number\n" +
          "actions:\n" +
          "  - label: Destroy\n    trigger: destroy\n" +
          '    confirm: "Destroy PR #{{ pr_number }} in {{ upper(repo) }}?"\n'
        }
        canvasId="c1"
        nodeRefs={{
          nodes: { destroy: "Destroy" },
          nodeIds: { destroy: "n-1" },
          onEmitEvent: vi.fn(async () => undefined),
        }}
      />,
    );

    await waitFor(() => {
      expect(screen.getByTestId("canvas-widget-block-action-destroy")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByTestId("canvas-widget-block-action-destroy"));
    await screen.findByText("Destroy PR #42 in CORE?");
  });
});

//
// v5: ```query alias backward compatibility — the alias-routing test lives in
// CanvasMarkdown.spec.tsx; here we just verify that v4 behavior keeps working
// against the renamed component, which the rest of this file already does.
//
