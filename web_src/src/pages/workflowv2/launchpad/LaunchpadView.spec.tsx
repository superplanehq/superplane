import { fireEvent, render, screen, act } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

// Mock react-grid-layout/legacy with a passthrough component so we don't have
// to deal with width measurement, drag handlers, or CSS imports during tests.
// The shim still exposes onLayoutChange so we can simulate a drag.
vi.mock("react-grid-layout/legacy", () => ({
  ReactGridLayout: ({
    children,
    onLayoutChange,
    layout,
  }: {
    children: ReactNode;
    onLayoutChange?: (layout: Array<{ i: string; x: number; y: number; w: number; h: number }>) => void;
    layout?: Array<{ i: string; x: number; y: number; w: number; h: number }>;
  }) => (
    <div data-testid="rgl" data-layout={JSON.stringify(layout ?? [])}>
      {/* A dedicated trigger so simulating a layout change doesn't fight with
          clicks on the panel chrome (delete button, drag handle, etc.). */}
      <button
        type="button"
        data-testid="rgl-simulate-drag"
        onClick={() => {
          if (onLayoutChange && layout?.[0]) {
            onLayoutChange([{ ...layout[0], x: 1, y: 1, w: 4, h: 4 }]);
          }
        }}
      />
      {children}
    </div>
  ),
}));

// Avoid running the real markdown pipeline in tests. Surface the nodeRefs
// context as data attributes / a simulator button so tests can assert that
// LaunchpadView wires triggerTemplates and onTriggerTemplateRun through to
// the markdown layer. The button matches a hardcoded slug pair for tests
// that opt into the run-chip flow.
type MockTriggerInfo = { name: string; payload: unknown };
type MockNodeStatusInfo = { status: string; badgeColor?: string; label?: string };
type MockNodeRefs = {
  triggerTemplates?: Record<string, Record<string, MockTriggerInfo>>;
  onTriggerTemplateRun?: (input: { nodeSlug: string; templateSlug: string }) => void;
  nodeStatuses?: Record<string, MockNodeStatusInfo | undefined>;
};

// WidgetBlock pulls in useQueryClient transitively. The fill-mode tests only
// care about the wrapping markdown view's `data-fill` attribute, so we
// short-circuit to a simple stub.
vi.mock("@/ui/Markdown/WidgetBlock", () => ({
  WidgetBlock: ({ body, fill }: { body: string; fill?: boolean }) => (
    <div data-testid="widget-block" data-fill={fill ? "yes" : "no"}>
      <pre>{body}</pre>
    </div>
  ),
}));

vi.mock("@/ui/Markdown/CanvasMarkdown", () => ({
  CanvasMarkdown: ({ children, nodeRefs }: { children: string; nodeRefs?: MockNodeRefs }) => {
    const triggerKey = "my-trigger";
    const templateKey = "hello-world";
    const hasTemplate = Boolean(nodeRefs?.triggerTemplates?.[triggerKey]?.[templateKey]);
    const canRun = Boolean(nodeRefs?.onTriggerTemplateRun);
    const statusKeys = nodeRefs?.nodeStatuses ? Object.keys(nodeRefs.nodeStatuses).sort().join(",") : "";
    return (
      <div
        data-testid="markdown"
        data-has-template={hasTemplate ? "yes" : "no"}
        data-can-run={canRun ? "yes" : "no"}
        data-status-keys={statusKeys}
      >
        <pre>{children}</pre>
        {hasTemplate && canRun ? (
          <button
            type="button"
            data-testid="run-chip"
            onClick={(e) => {
              e.stopPropagation();
              nodeRefs?.onTriggerTemplateRun?.({ nodeSlug: triggerKey, templateSlug: templateKey });
            }}
          >
            run
          </button>
        ) : null}
      </div>
    );
  },
}));

// jsdom doesn't ship ResizeObserver; the component falls back gracefully but
// also uses clientWidth which is 0 by default. We stub ResizeObserver to a
// no-op and force a non-zero containerWidth via Object.defineProperty below.
class StubResizeObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
}
beforeEach(() => {
  vi.useFakeTimers();
  (globalThis as unknown as { ResizeObserver: typeof StubResizeObserver }).ResizeObserver = StubResizeObserver;
});

import { LaunchpadView } from "./LaunchpadView";
import type { LaunchpadLayoutItem, LaunchpadPanel } from "@/hooks/useCanvasData";

function setContainerWidth(width: number) {
  // The component reads clientWidth from its container ref. We stub the getter
  // on HTMLElement so any rendered div reports the desired width.
  Object.defineProperty(HTMLElement.prototype, "clientWidth", {
    configurable: true,
    get: () => width,
  });
}

describe("LaunchpadView", () => {
  beforeEach(() => {
    setContainerWidth(800);
  });

  const makePanel = (id: string, body = ""): LaunchpadPanel => ({
    id,
    type: "markdown",
    content: { body },
  });
  const makeLayout = (id: string, w = 6, h = 6): LaunchpadLayoutItem => ({ i: id, x: 0, y: 0, w, h });

  it("renders the empty state and add button when no panels exist", () => {
    const onChange = vi.fn();
    render(<LaunchpadView panels={[]} layout={[]} isLoading={false} readOnly={false} onChange={onChange} />);
    expect(screen.getByTestId("launchpad-empty-state")).toBeInTheDocument();
    expect(screen.getAllByTestId("launchpad-add-panel").length).toBeGreaterThan(0);
  });

  it("clicking 'Add panel' creates a new markdown panel and persists after debounce", () => {
    const onChange = vi.fn();
    render(<LaunchpadView panels={[]} layout={[]} isLoading={false} readOnly={false} onChange={onChange} />);
    fireEvent.click(screen.getAllByTestId("launchpad-add-panel")[0]);

    act(() => {
      vi.runAllTimers();
    });

    expect(onChange).toHaveBeenCalledTimes(1);
    const [args] = onChange.mock.calls[0];
    expect(args.panels).toHaveLength(1);
    expect(args.panels[0].type).toBe("markdown");
    expect(args.layout).toHaveLength(1);
    expect(args.layout[0].i).toBe(args.panels[0].id);
  });

  it("renders one chrome per panel with a flat icon cluster (no kebab menu)", () => {
    render(
      <LaunchpadView
        panels={[makePanel("p1"), makePanel("p2")]}
        layout={[makeLayout("p1"), makeLayout("p2")]}
        isLoading={false}
        readOnly={false}
        onChange={vi.fn()}
      />,
    );
    expect(screen.getByTestId("launchpad-panel-p1")).toBeInTheDocument();
    expect(screen.getByTestId("launchpad-panel-p2")).toBeInTheDocument();
    expect(screen.getAllByTestId("launchpad-drag-handle")).toHaveLength(2);
    expect(screen.getAllByTestId("launchpad-panel-actions")).toHaveLength(2);
    expect(screen.getAllByTestId("launchpad-edit-panel")).toHaveLength(2);
    expect(screen.queryByTestId("launchpad-toggle-auto-height")).toBeNull();
    expect(screen.getAllByTestId("launchpad-delete-panel-button")).toHaveLength(2);
    // The kebab menu has been removed in favor of direct icon buttons.
    expect(screen.queryByTestId("launchpad-panel-menu")).toBeNull();
  });

  it("hides edit affordances and the add button when readOnly is true", () => {
    render(
      <LaunchpadView
        panels={[makePanel("p1", "# hi")]}
        layout={[makeLayout("p1")]}
        isLoading={false}
        readOnly={true}
        onChange={vi.fn()}
      />,
    );
    expect(screen.queryByTestId("launchpad-add-panel")).toBeNull();
    expect(screen.queryByTestId("launchpad-edit-panel")).toBeNull();
    expect(screen.queryByTestId("launchpad-delete-panel-button")).toBeNull();
    expect(screen.queryByTestId("launchpad-drag-handle")).toBeNull();
    expect(screen.queryByTestId("launchpad-panel-actions")).toBeNull();
  });

  it("clicking the delete icon goes straight to the confirm dialog", async () => {
    vi.useRealTimers();
    const onChange = vi.fn();
    const user = userEvent.setup();
    render(
      <LaunchpadView
        panels={[makePanel("p1"), makePanel("p2")]}
        layout={[makeLayout("p1"), makeLayout("p2")]}
        isLoading={false}
        readOnly={false}
        onChange={onChange}
      />,
    );

    await user.click(screen.getAllByTestId("launchpad-delete-panel-button")[0]);

    expect(screen.getByTestId("launchpad-delete-confirm")).toBeInTheDocument();
    await user.click(screen.getByTestId("launchpad-delete-confirm-action"));

    // Wait past the save debounce (350ms) so onChange is invoked.
    await new Promise((r) => setTimeout(r, 500));

    expect(onChange).toHaveBeenCalledTimes(1);
    const [args] = onChange.mock.calls[0];
    expect(args.panels).toHaveLength(1);
    expect(args.panels[0].id).toBe("p2");
    expect(args.layout.map((l: LaunchpadLayoutItem) => l.i)).toEqual(["p2"]);
  });

  it("clicking the header Edit button puts the markdown panel in edit mode", () => {
    render(
      <LaunchpadView
        panels={[makePanel("p1", "# hi")]}
        layout={[makeLayout("p1")]}
        isLoading={false}
        readOnly={false}
        onChange={vi.fn()}
      />,
    );
    expect(screen.queryByTestId("launchpad-markdown-editor")).toBeNull();
    fireEvent.click(screen.getByTestId("launchpad-edit-panel"));
    expect(screen.getByTestId("launchpad-markdown-editor")).toBeInTheDocument();
  });

  it("forwards nodeStatuses through nodeRefs into the markdown layer", () => {
    const nodeRefs = {
      nodes: { deploy: "Deploy", verify: "Verify" },
      nodeStatuses: {
        deploy: { status: "running", badgeColor: "bg-blue-500", label: "RUNNING" },
        verify: { status: "none", badgeColor: "bg-gray-400", label: "no runs" },
      },
    };
    render(
      <LaunchpadView
        panels={[makePanel("p1", "Deploy is @deploy:status")]}
        layout={[makeLayout("p1")]}
        isLoading={false}
        readOnly={false}
        nodeRefs={nodeRefs}
        onChange={vi.fn()}
      />,
    );

    const markdown = screen.getByTestId("markdown");
    expect(markdown.getAttribute("data-status-keys")).toBe("deploy,verify");
  });

  it("forwards triggerTemplates + onTriggerTemplateRun through nodeRefs into the markdown layer", () => {
    const onTriggerTemplateRun = vi.fn();
    const nodeRefs = {
      nodes: { "my-trigger": "My trigger" },
      triggerTemplates: {
        "my-trigger": {
          "hello-world": { name: "Hello World", payload: { greet: "hi" } },
        },
      },
      onTriggerTemplateRun,
    };
    render(
      <LaunchpadView
        panels={[makePanel("p1", "[[run:my-trigger:hello-world]]")]}
        layout={[makeLayout("p1")]}
        isLoading={false}
        readOnly={false}
        nodeRefs={nodeRefs}
        onChange={vi.fn()}
      />,
    );

    const runChip = screen.getByTestId("run-chip");
    fireEvent.click(runChip);
    expect(onTriggerTemplateRun).toHaveBeenCalledWith({
      nodeSlug: "my-trigger",
      templateSlug: "hello-world",
    });
  });

  it("layout change from the grid library triggers a debounced save with the new layout", () => {
    const onChange = vi.fn();
    render(
      <LaunchpadView
        panels={[makePanel("p1")]}
        layout={[makeLayout("p1")]}
        isLoading={false}
        readOnly={false}
        onChange={onChange}
      />,
    );
    // Simulate a drag via our mocked grid.
    fireEvent.click(screen.getByTestId("rgl-simulate-drag"));
    act(() => {
      vi.runAllTimers();
    });
    expect(onChange).toHaveBeenCalledTimes(1);
    const [args] = onChange.mock.calls[0];
    expect(args.layout[0]).toMatchObject({ i: "p1", x: 1, y: 1, w: 4, h: 4 });
  });

  it("renders a panel with a heading + single widget in fill mode", () => {
    render(
      <LaunchpadView
        panels={[
          {
            id: "p1",
            type: "markdown",
            content: { body: "# Avg time\n\n```widget\nsource: x\nrender: { kind: number }\n```" },
          },
        ]}
        layout={[makeLayout("p1")]}
        isLoading={false}
        readOnly={false}
        canvasId="canvas-1"
        onChange={vi.fn()}
      />,
    );
    const view = screen.getByTestId("launchpad-markdown-view");
    expect(view.getAttribute("data-fill")).toBe("true");
  });

  it("renders a panel with multiple widget fences in regular (non-fill) mode", () => {
    render(
      <LaunchpadView
        panels={[
          {
            id: "p1",
            type: "markdown",
            content: {
              body: "```widget\nsource: a\n```\n\n```widget\nsource: b\n```",
            },
          },
        ]}
        layout={[makeLayout("p1")]}
        isLoading={false}
        readOnly={false}
        canvasId="canvas-1"
        onChange={vi.fn()}
      />,
    );
    const view = screen.getByTestId("launchpad-markdown-view");
    expect(view.getAttribute("data-fill")).toBe("false");
  });
});
