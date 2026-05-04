import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
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

const showSuccessToast = vi.fn();
const showErrorToast = vi.fn();
vi.mock("@/lib/toast", () => ({
  showSuccessToast: (...args: unknown[]) => showSuccessToast(...args),
  showErrorToast: (...args: unknown[]) => showErrorToast(...args),
}));

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
  showSuccessToast.mockReset();
  showErrorToast.mockReset();
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

describe("QueryBlock custom columns", () => {
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
      <QueryBlock
        body={bodyWithColumns("  - label: PR\n    field: pr_number\n  - label: Title\n    field: pr_title\n")}
        canvasId="canvas-1"
      />,
    );

    await waitFor(() => {
      expect(screen.getByTestId("canvas-query-block")).toBeInTheDocument();
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
      <QueryBlock
        body={bodyWithColumns("  - label: Name\n    field: name\n  - label: Region\n    field: region\n")}
        canvasId="canvas-1"
      />,
    );

    await waitFor(() => {
      expect(screen.getByTestId("canvas-query-block")).toBeInTheDocument();
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
      <QueryBlock body={bodyWithColumns("  - label: URL\n    field: url\n    format: link\n")} canvasId="canvas-1" />,
    );

    await waitFor(() => {
      expect(screen.getByTestId("canvas-query-block")).toBeInTheDocument();
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
      <QueryBlock
        body={bodyWithColumns("  - label: Preview\n    field: url\n    format: link:Open\n")}
        canvasId="canvas-1"
      />,
    );

    await waitFor(() => {
      expect(screen.getByTestId("canvas-query-block")).toBeInTheDocument();
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
      <QueryBlock
        body={bodyWithColumns("  - label: Created\n    field: created_at\n    format: relative\n")}
        canvasId="canvas-1"
      />,
    );

    await waitFor(() => {
      expect(screen.getByTestId("canvas-query-block")).toBeInTheDocument();
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
      <QueryBlock
        body={bodyWithColumns("  - label: Created\n    field: created_at\n    format: relative\n")}
        canvasId="canvas-1"
      />,
    );

    await waitFor(() => {
      expect(screen.getByTestId("canvas-query-block")).toBeInTheDocument();
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
      <QueryBlock
        body={bodyWithColumns("  - label: Created\n    field: created_at\n    format: date\n")}
        canvasId="canvas-1"
      />,
    );

    await waitFor(() => {
      expect(screen.getByTestId("canvas-query-block")).toBeInTheDocument();
    });

    expect(screen.getByText("2026-04-30 07:09 UTC")).toBeInTheDocument();
  });

  it("renders format: badge as a styled pill with the value", async () => {
    canvasesListCanvasMemories.mockResolvedValue({
      data: { items: [{ id: "a", namespace: "environments", values: { status: "active" } }] },
    });

    renderWithClient(
      <QueryBlock
        body={bodyWithColumns("  - label: Status\n    field: status\n    format: badge\n")}
        canvasId="canvas-1"
      />,
    );

    await waitFor(() => {
      expect(screen.getByTestId("canvas-query-block")).toBeInTheDocument();
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
      <QueryBlock body={bodyWithColumns("  - label: SHA\n    field: sha\n    format: code\n")} canvasId="canvas-1" />,
    );

    await waitFor(() => {
      expect(screen.getByTestId("canvas-query-block")).toBeInTheDocument();
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
      <QueryBlock
        body={bodyWithColumns("  - label: Name\n    field: name\n    format: bogus\n")}
        canvasId="canvas-1"
      />,
    );

    await waitFor(() => {
      expect(screen.getByTestId("canvas-query-block")).toBeInTheDocument();
    });

    expect(screen.getByText("alpha")).toBeInTheDocument();
    expect(warn).toHaveBeenCalled();
    const message = warn.mock.calls[0]?.[0];
    expect(message).toMatch(/Unknown format "bogus"/);
    warn.mockRestore();
  });

  it("rejects malformed columns with an inline error", () => {
    renderWithClient(
      <QueryBlock body={"source: memory\nnamespace: env\ncolumns:\n  - label: PR\n"} canvasId="canvas-1" />,
    );

    const error = screen.getByTestId("canvas-query-block-error");
    expect(error.textContent).toMatch(/columns\[0\] missing `field`/);
    expect(canvasesListCanvasMemories).not.toHaveBeenCalled();
  });
});

describe("QueryBlock filters", () => {
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
      <QueryBlock body={bodyWithWhere("  - field: repo\n    op: eq\n    value: core\n")} canvasId="c1" />,
    );

    await waitFor(() => {
      expect(screen.getByTestId("canvas-query-block")).toBeInTheDocument();
    });

    expect(screen.getByText("beta")).toBeInTheDocument();
    expect(screen.queryByText("alpha")).toBeNull();
    expect(screen.queryByText("gamma")).toBeNull();
  });

  it("filters with op: neq", async () => {
    canvasesListCanvasMemories.mockResolvedValue(sample);
    renderWithClient(
      <QueryBlock body={bodyWithWhere("  - field: repo\n    op: neq\n    value: core\n")} canvasId="c1" />,
    );

    await waitFor(() => {
      expect(screen.getByTestId("canvas-query-block")).toBeInTheDocument();
    });

    expect(screen.getByText("alpha")).toBeInTheDocument();
    // gamma has no `repo` field -> neq excludes (non-existence ops fail closed).
    expect(screen.queryByText("gamma")).toBeNull();
    expect(screen.queryByText("beta")).toBeNull();
  });

  it("filters with op: contains and not_contains", async () => {
    canvasesListCanvasMemories.mockResolvedValue(sample);
    const { rerender } = renderWithClient(
      <QueryBlock body={bodyWithWhere("  - field: repo\n    op: contains\n    value: store\n")} canvasId="c1" />,
    );

    await waitFor(() => {
      expect(screen.getByText("alpha")).toBeInTheDocument();
    });
    expect(screen.queryByText("beta")).toBeNull();

    rerender(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <QueryBlock body={bodyWithWhere("  - field: repo\n    op: not_contains\n    value: store\n")} canvasId="c1" />
      </QueryClientProvider>,
    );

    await waitFor(() => {
      expect(screen.getByText("beta")).toBeInTheDocument();
    });
    expect(screen.queryByText("alpha")).toBeNull();
  });

  it("filters with op: gt and lt on numeric strings", async () => {
    canvasesListCanvasMemories.mockResolvedValue(sample);
    renderWithClient(<QueryBlock body={bodyWithWhere("  - field: count\n    op: gt\n    value: 7\n")} canvasId="c1" />);

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
      <QueryBlock body={bodyWithWhere("  - field: count\n    op: lt\n    value: 100\n")} canvasId="c1" />,
    );

    await waitFor(() => {
      expect(screen.getByText("num")).toBeInTheDocument();
    });
    expect(screen.queryByText("junk")).toBeNull();
  });

  it("filters with op: exists / not_exists treating empty string as missing", async () => {
    canvasesListCanvasMemories.mockResolvedValue(sample);
    const { rerender } = renderWithClient(
      <QueryBlock body={bodyWithWhere("  - field: repo\n    op: exists\n")} canvasId="c1" />,
    );

    await waitFor(() => {
      expect(screen.getByText("alpha")).toBeInTheDocument();
    });
    expect(screen.getByText("beta")).toBeInTheDocument();
    expect(screen.queryByText("gamma")).toBeNull();

    rerender(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <QueryBlock body={bodyWithWhere("  - field: count\n    op: not_exists\n")} canvasId="c1" />
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
      <QueryBlock
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
      <QueryBlock body={bodyWithWhere("  - field: repo\n    op: weirdop\n    value: x\n")} canvasId="c1" />,
    );

    const error = screen.getByTestId("canvas-query-block-error");
    expect(error.textContent).toMatch(/Unknown filter operator: "weirdop"/);
    expect(canvasesListCanvasMemories).not.toHaveBeenCalled();
  });

  it("renders an inline error when value is missing for an op that requires it", () => {
    renderWithClient(<QueryBlock body={bodyWithWhere("  - field: repo\n    op: eq\n")} canvasId="c1" />);

    const error = screen.getByTestId("canvas-query-block-error");
    expect(error.textContent).toMatch(/where\[0\] missing `value`/);
    expect(canvasesListCanvasMemories).not.toHaveBeenCalled();
  });
});

describe("QueryBlock backward compatibility", () => {
  it("omitting columns and where reproduces increment-1 auto-column behavior", async () => {
    canvasesListCanvasMemories.mockResolvedValue({
      data: {
        items: [
          { id: "a", namespace: "environments", values: { z: "1", a: "2" } },
          { id: "b", namespace: "environments", values: { m: "3", a: "4" } },
        ],
      },
    });

    renderWithClient(<QueryBlock body={baseBody} canvasId="canvas-1" />);

    await waitFor(() => {
      expect(screen.getByTestId("canvas-query-block")).toBeInTheDocument();
    });

    const headers = screen.getAllByRole("columnheader").map((th) => th.textContent);
    expect(headers).toEqual(["a", "m", "z"]);
  });
});

describe("QueryBlock actions", () => {
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
      <QueryBlock body={body} canvasId="c1" nodeRefs={{ nodes: { destroy: "Destroy" }, nodeIds, onEmitEvent }} />,
    );
    return { onEmitEvent, ...result };
  }

  it("renders an Actions column when actions are present, and one button per action per row", async () => {
    const body = bodyWithActions("  - label: Destroy\n    trigger: destroy\n    variant: danger\n");
    renderWithActions(body);

    await waitFor(() => {
      expect(screen.getByTestId("canvas-query-block")).toBeInTheDocument();
    });

    const headers = screen.getAllByRole("columnheader").map((th) => th.textContent);
    expect(headers).toEqual(["PR", "Actions"]);

    const button = screen.getByTestId("canvas-query-block-action-destroy");
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
      <QueryBlock
        body={"source: memory\nnamespace: environments\ncolumns:\n  - label: PR\n    field: pr_number\n"}
        canvasId="c1"
      />,
    );

    await waitFor(() => {
      expect(screen.getByTestId("canvas-query-block")).toBeInTheDocument();
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
      expect(screen.getByTestId("canvas-query-block-action-destroy")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByTestId("canvas-query-block-action-destroy"));

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
      expect(screen.getByTestId("canvas-query-block-action-destroy")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByTestId("canvas-query-block-action-destroy"));
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
      expect(screen.getByTestId("canvas-query-block-action-destroy")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByTestId("canvas-query-block-action-destroy"));

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
    fireEvent.click(screen.getByTestId("canvas-query-block-action-destroy"));
    await screen.findByText("Destroy PR #69?");
    fireEvent.click(screen.getByTestId("canvas-query-block-action-confirm-destroy"));

    await waitFor(() => {
      expect(onEmitEvent).toHaveBeenCalledTimes(1);
    });
  });

  it("shows the success toast with the action label after firing", async () => {
    const body = bodyWithActions("  - label: Destroy\n    trigger: destroy\n");
    renderWithActions(body);

    await waitFor(() => {
      expect(screen.getByTestId("canvas-query-block-action-destroy")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByTestId("canvas-query-block-action-destroy"));

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
      expect(screen.getByTestId("canvas-query-block-action-destroy")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByTestId("canvas-query-block-action-destroy"));

    const err = await screen.findByTestId("canvas-query-block-action-error-destroy");
    expect(err.textContent).toContain("nope");
    expect(showSuccessToast).not.toHaveBeenCalled();
  });

  it("disables the button with a tooltip when the trigger slug is unknown", async () => {
    const body = bodyWithActions("  - label: Destroy\n    trigger: ghost\n");
    renderWithActions(body, { nodeIds: {} });

    await waitFor(() => {
      expect(screen.getByTestId("canvas-query-block-action-ghost")).toBeInTheDocument();
    });
    const button = screen.getByTestId("canvas-query-block-action-ghost") as HTMLButtonElement;
    expect(button.disabled).toBe(true);
    expect(button.getAttribute("title")).toMatch(/Trigger "ghost" not found/);
  });

  it("renders multiple buttons in a single row when multiple actions are configured", async () => {
    const body = bodyWithActions(
      "  - label: Open\n    trigger: deploy\n    variant: primary\n  - label: Destroy\n    trigger: destroy\n    variant: danger\n",
    );
    renderWithActions(body, { nodeIds: { destroy: "node-1", deploy: "node-2" } });

    await waitFor(() => {
      expect(screen.getByTestId("canvas-query-block-action-destroy")).toBeInTheDocument();
    });

    const open = screen.getByTestId("canvas-query-block-action-deploy");
    const destroy = screen.getByTestId("canvas-query-block-action-destroy");
    expect(open.className).toContain("bg-blue-50");
    expect(destroy.className).toContain("bg-red-50");
  });

  it("warns and falls back when `icon` is unknown", async () => {
    const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
    const body = bodyWithActions("  - label: Destroy\n    trigger: destroy\n    icon: bogus\n");
    renderWithActions(body);

    await waitFor(() => {
      expect(screen.getByTestId("canvas-query-block-action-destroy")).toBeInTheDocument();
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
      expect(screen.getByTestId("canvas-query-block-action-destroy")).toBeInTheDocument();
    });
    const button = screen.getByTestId("canvas-query-block-action-destroy");
    expect(button.getAttribute("data-variant")).toBe("default");
    expect(button.className).toContain("bg-white");
    expect(warn).toHaveBeenCalled();
    warn.mockRestore();
  });

  it("renders a parse error for a malformed action (missing trigger)", () => {
    renderWithClient(
      <QueryBlock
        body={"source: memory\nnamespace: env\nactions:\n  - label: Destroy\n"}
        canvasId="c1"
        nodeRefs={{ nodes: {}, nodeIds: {}, onEmitEvent: vi.fn(async () => undefined) }}
      />,
    );
    const error = screen.getByTestId("canvas-query-block-error");
    expect(error.textContent).toMatch(/actions\[0\] missing `trigger`/);
    expect(canvasesListCanvasMemories).not.toHaveBeenCalled();
  });

  it("renders a parse error when fill has a non-string value", () => {
    renderWithClient(
      <QueryBlock
        body={
          "source: memory\nnamespace: env\nactions:\n  - label: Destroy\n    trigger: destroy\n    fill:\n      data.issue.number: 42\n"
        }
        canvasId="c1"
        nodeRefs={{ nodes: {}, nodeIds: {}, onEmitEvent: vi.fn(async () => undefined) }}
      />,
    );
    const error = screen.getByTestId("canvas-query-block-error");
    expect(error.textContent).toMatch(/actions\[0\]\.fill\.data\.issue\.number must be a string/);
  });
});
