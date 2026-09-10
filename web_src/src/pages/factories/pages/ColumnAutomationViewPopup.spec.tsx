import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import type * as CanvasDataModule from "@/hooks/useCanvasData";
import { prepareData } from "@/pages/app/workflowPageHelpers";
import { TooltipProvider } from "@/ui/tooltip";

import { ColumnAutomationViewPopup } from "./ColumnAutomationViewPopup";
import { PLANNING_REVIEW_DRAFT } from "./planningReviewMockup";
import type { IntakeAutomationGraph } from "./useIntakeAutomationCanvas";

const { useInfiniteCanvasRuns } = vi.hoisted(() => ({
  useInfiniteCanvasRuns: vi.fn(),
}));

vi.mock("@monaco-editor/react", () => ({
  Editor: ({ value, onChange }: { value?: string; onChange?: (value: string | undefined) => void }) => (
    <textarea value={value ?? ""} onChange={(event) => onChange?.(event.target.value)} />
  ),
}));

vi.mock("@/hooks/useCanvasData", async (importOriginal) => {
  const actual = await importOriginal<typeof CanvasDataModule>();
  return {
    ...actual,
    useInfiniteCanvasRuns,
  };
});

useInfiniteCanvasRuns.mockReturnValue({
  data: {
    pages: [
      {
        runs: [
          {
            id: "run-implement-1",
            canvasId: "app-refund-implementer",
            state: "STATE_STARTED",
            createdAt: "2026-05-01T12:00:00Z",
            rootEvent: { nodeId: "on-run", customName: "Add refund reconciliation test" },
          },
        ],
      },
    ],
  },
  isPending: false,
  isError: false,
  hasNextPage: false,
  isFetchingNextPage: false,
  fetchNextPage: vi.fn(),
  refetch: vi.fn(),
});

const EDIT_HREF = "/org-1/workspaces/RF/apps/app-refund-implementer?configure=1&agent=1&from=lines&lineId=line-plan";

function implementGraph(): IntakeAutomationGraph {
  const { nodes, edges } = prepareData(
    {
      metadata: { id: "app-refund-implementer", name: "Implement", factoryId: "factory-1" },
      spec: {
        nodes: [
          { id: "on-run", name: "On run", type: "TYPE_TRIGGER", component: "onRun" },
          { id: "agent", name: "Implement From Task Description", type: "TYPE_ACTION", component: "runnerClaudeCode" },
        ],
        edges: [{ channel: "default", sourceId: "on-run", targetId: "agent" }],
      },
    },
    [{ name: "onRun", label: "On run" }],
    [{ name: "runnerClaudeCode", label: "Claude Code" }],
    {},
    {},
    {},
    "app-refund-implementer",
    new QueryClient(),
    null,
    "live",
  );
  return {
    nodes,
    edges,
    factoryId: "factory-1",
    specNodes: [{ id: "on-run", name: "On run", type: "TYPE_TRIGGER", component: "webhook" }],
  };
}

function renderPopup(props: Partial<Parameters<typeof ColumnAutomationViewPopup>[0]> = {}) {
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <ThemeProvider>
          <TooltipProvider>
            <ColumnAutomationViewPopup
              title="Implement"
              graph={implementGraph()}
              editHref={EDIT_HREF}
              onClose={vi.fn()}
              {...props}
            />
          </TooltipProvider>
        </ThemeProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

afterEach(() => {
  localStorage.clear();
});

describe("ColumnAutomationViewPopup", () => {
  it("shows the automation canvas in the board popup", () => {
    renderPopup();

    expect(screen.getByTestId("column-automation-view")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Implement" })).toBeInTheDocument();
    const canvas = screen.getByTestId("column-automation-view-canvas");
    expect(canvas).toHaveAccessibleName("Automation");
    expect(within(canvas).getAllByText("On run").length).toBeGreaterThan(0);
    expect(within(canvas).getAllByText("Implement From Task Description").length).toBeGreaterThan(0);
    expect(screen.queryByRole("button", { name: "Automation menu" })).not.toBeInTheDocument();
    const edit = within(canvas).getByRole("link", { name: "Edit automation" });
    expect(edit).toHaveAttribute("href", EDIT_HREF);
    expect(edit.className).toMatch(/absolute/);
    expect(edit.className).toMatch(/right-2/);
    expect(document.querySelector(".sp-canvas-editing")).toBeNull();
    expect(screen.queryByTestId("canvas-runs-sidebar")).not.toBeInTheDocument();
  });

  it("lists canvas runs beside the automation when a canvas id is given", () => {
    renderPopup({
      canvasId: "app-refund-implementer",
      runHrefFor: (runId) => `/org-1/workspaces/RF/apps/app-refund-implementer?run=${runId}`,
    });

    const sidebar = within(screen.getByTestId("column-automation-view-canvas")).getByTestId("canvas-runs-sidebar");
    expect(within(sidebar).getByText("Add refund reconciliation test")).toBeInTheDocument();
    expect(within(sidebar).getByRole("link", { name: "Add refund reconciliation test" })).toHaveAttribute(
      "href",
      "/org-1/workspaces/RF/apps/app-refund-implementer?run=run-implement-1",
    );
  });

  it("closes from the popup chrome", async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    renderPopup({ onClose });

    await user.click(screen.getByRole("button", { name: "Close" }));
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("shows loading copy when the canvas is not ready", () => {
    renderPopup({ graph: { nodes: [], edges: [] }, loading: true, editHref: undefined });

    expect(screen.getByTestId("column-automation-view-canvas")).toHaveTextContent("The automation is loading.");
  });

  it("hides tabs when the automation has no form and no agent", () => {
    renderPopup();

    expect(screen.queryByTestId("column-automation-view-tab-agent")).not.toBeInTheDocument();
    expect(screen.queryByTestId("column-automation-view-tab-automation")).not.toBeInTheDocument();
  });

  it("shows Agent and Automation tabs when the canvas has an agent", async () => {
    const user = userEvent.setup();
    renderPopup({
      agent: { draft: PLANNING_REVIEW_DRAFT, organizationId: "org-1", onSave: vi.fn() },
    });

    expect(screen.getByTestId("column-automation-view-tab-agent")).toHaveAttribute("data-state", "active");
    expect(screen.getByTestId("column-automation-view-tab-automation")).toHaveTextContent("Automation");
    expect(screen.getByTestId("planning-review-editor")).toBeInTheDocument();
    expect(screen.queryByTestId("planning-review-nav")).not.toBeInTheDocument();
    expect(screen.getByTestId("planning-review-settings")).toBeInTheDocument();
    expect(screen.getByText("Concurrency")).toBeInTheDocument();
    expect(screen.getByText("Model used")).toBeInTheDocument();
    expect(screen.getByTestId("planning-review-step-kind-0")).toHaveTextContent("Bash");
    expect(screen.getByTestId("planning-review-save")).toHaveTextContent("Save Agent");
    expect(screen.queryByTestId("planning-review-automation-note")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Cancel" })).not.toBeInTheDocument();
    expect(screen.queryByTestId("column-automation-view-edit")).not.toBeInTheDocument();

    await user.click(screen.getByTestId("column-automation-view-tab-automation"));
    expect(screen.getByTestId("column-automation-view-canvas")).toBeInTheDocument();
    expect(
      within(screen.getByTestId("column-automation-view-canvas")).getByRole("link", { name: "Edit automation" }),
    ).toHaveAttribute("href", EDIT_HREF);
  });

  it("puts General first when a form and an agent exist", () => {
    renderPopup({
      general: <p data-testid="column-automation-view-general">Name and filters</p>,
      agent: { draft: PLANNING_REVIEW_DRAFT, onSave: vi.fn() },
      initialTab: "general",
    });

    const tabs = screen.getAllByRole("tab").map((tab) => tab.textContent);
    expect(tabs).toEqual(["General", "Agent", "Automation"]);
    expect(screen.getByTestId("column-automation-view-general")).toBeInTheDocument();
  });
});
