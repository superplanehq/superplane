import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import type * as CanvasDataModule from "@/hooks/useCanvasData";
import { prepareData } from "@/pages/app/workflowPageHelpers";
import { TooltipProvider } from "@/ui/tooltip";

import { IntakeSourceSettingsPopup } from "./IntakeSourceSettingsPopup";
import { DEFAULT_GITHUB_INTAKE_SETTINGS, type IntakeSettingsTab } from "./intakeSourceSettingsModel";
import { PLANNING_REVIEW_DRAFT } from "./planningReviewMockup";
import type { IntakeAutomationGraph } from "./useIntakeAutomationCanvas";
import type { PlanningReviewAgentSlot } from "./PlanningReviewEditor";

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
            id: "run-1",
            canvasId: "app-github-issues-intake",
            state: "STATE_FINISHED",
            result: "RESULT_PASSED",
            createdAt: "2026-05-01T12:00:00Z",
            rootEvent: { nodeId: "trigger", customName: "feat: Add console empty-state" },
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

const githubAutomationGraph = githubIntakeGraph();

/** Same pipeline the canvas editor uses, so the popup renders editor nodes. */
function githubIntakeGraph(): IntakeAutomationGraph {
  const { nodes, edges } = prepareData(
    {
      metadata: { id: "app-github-issues-intake", name: "GitHub issues", factoryId: "factory-1" },
      spec: {
        nodes: [
          { id: "trigger", name: "On Issue", type: "TYPE_TRIGGER", component: "github.onIssue" },
          {
            id: "filter",
            name: "Matches filters?",
            type: "TYPE_ACTION",
            component: "if",
            configuration: { expression: "true" },
          },
          { id: "create", name: "Create Task", type: "TYPE_ACTION", component: "createWorkOrder" },
        ],
        edges: [
          { channel: "default", sourceId: "trigger", targetId: "filter" },
          { channel: "true", sourceId: "filter", targetId: "create" },
        ],
      },
    },
    [{ name: "github.onIssue", label: "On Issue" }],
    [
      { name: "if", label: "If" },
      { name: "createWorkOrder", label: "Create Task" },
    ],
    {},
    {},
    {},
    "app-github-issues-intake",
    new QueryClient(),
    null,
    "live",
  );

  return {
    nodes,
    edges,
    factoryId: "factory-1",
    specNodes: [{ id: "trigger", name: "On Issue", type: "TYPE_TRIGGER", component: "github.onIssue" }],
  };
}

function renderPopup(
  props: {
    onSave?: (next: typeof DEFAULT_GITHUB_INTAKE_SETTINGS) => void;
    onClose?: () => void;
    editAutomationHref?: string;
    canvasId?: string;
    runHrefFor?: (runId: string) => string;
    agent?: PlanningReviewAgentSlot;
    initialTab?: IntakeSettingsTab;
  } = {},
) {
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <ThemeProvider>
          <TooltipProvider>
            <IntakeSourceSettingsPopup
              settings={DEFAULT_GITHUB_INTAKE_SETTINGS}
              automationGraph={githubAutomationGraph}
              onSave={props.onSave ?? vi.fn()}
              editAutomationHref={props.editAutomationHref}
              canvasId={props.canvasId}
              runHrefFor={props.runHrefFor}
              agent={props.agent}
              onClose={props.onClose ?? vi.fn()}
              initialTab={props.initialTab}
              fixed={false}
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

describe("IntakeSourceSettingsPopup", () => {
  it("shows the GitHub issues configuration fields", () => {
    renderPopup();

    const dialog = screen.getByTestId("intake-source-settings");
    expect(dialog).toHaveAttribute("role", "dialog");
    expect(screen.getByRole("heading", { name: "Intake GitHub issues" })).toBeInTheDocument();
    expect(screen.getByLabelText("Name")).toHaveValue("GitHub issues");
    expect(screen.getByRole("radio", { name: /Listen for new issues/ })).toBeChecked();
    expect(screen.getByRole("radio", { name: /Run on a schedule/ })).not.toBeChecked();
    expect(screen.queryByTestId("intake-confidence-value")).not.toBeInTheDocument();
    expect(screen.getByRole("radio", { name: "Include these labels" })).toBeChecked();
    expect(screen.getByRole("radio", { name: "Exclude these labels" })).not.toBeChecked();
    expect(screen.getByRole("checkbox", { name: "bug" })).not.toBeChecked();
    expect(screen.getByRole("radio", { name: "Any assignment" })).toBeChecked();
    expect(screen.getByRole("tab", { name: "General" })).toHaveAttribute("data-state", "active");
    expect(screen.getAllByRole("tab").map((tab) => tab.textContent)).toEqual(["General", "Automation"]);
    expect(screen.queryByTestId("intake-settings-tab-agent")).not.toBeInTheDocument();
    expect(screen.queryByRole("tab", { name: "Runs" })).not.toBeInTheDocument();
    expect(screen.queryByTestId("intake-source-automation")).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Edit automation" })).not.toBeInTheDocument();
  });

  it("shows the intake automation on the Automation tab", async () => {
    const user = userEvent.setup();
    renderPopup({ editAutomationHref: "/org-1/workspaces/RF/apps/app-github-issues-intake?configure=1&agent=1" });

    await user.click(screen.getByRole("tab", { name: "Automation" }));

    const automation = screen.getByTestId("intake-source-automation");
    expect(automation).toHaveAccessibleName("Automation");
    expect(within(automation).getByTestId("rf__node-trigger")).toBeInTheDocument();
    expect(within(automation).getAllByText("On Issue").length).toBeGreaterThan(0);
    expect(within(automation).getByText("Matches filters?")).toBeInTheDocument();
    expect(within(automation).getAllByText("Create Task").length).toBeGreaterThan(0);
    const headerRow = screen.getByTestId("settings-automation-header-row");
    expect(within(headerRow).getByRole("tab", { name: "General" })).toBeInTheDocument();
    expect(within(headerRow).getByRole("tab", { name: "Automation" })).toBeInTheDocument();
    expect(within(headerRow).queryByRole("link", { name: "Edit automation" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Automation menu" })).not.toBeInTheDocument();
    const edit = within(automation).getByRole("link", { name: "Edit automation" });
    expect(edit).toHaveAttribute("href", "/org-1/workspaces/RF/apps/app-github-issues-intake?configure=1&agent=1");
    expect(document.querySelector(".sp-canvas-editing")).toBeNull();
    expect(within(automation).queryByRole("button", { name: /Add next component/ })).not.toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Accepted events go to Backlog" })).not.toBeInTheDocument();
    expect(screen.queryByLabelText("Name")).not.toBeInTheDocument();
    expect(screen.queryByTestId("intake-source-settings-save")).not.toBeInTheDocument();
    expect(screen.queryByTestId("canvas-runs-sidebar")).not.toBeInTheDocument();
  });

  it("lists canvas runs beside the automation when a canvas id is given", async () => {
    const user = userEvent.setup();
    renderPopup({
      canvasId: "app-github-issues-intake",
      runHrefFor: (runId) => `/org-1/workspaces/RF/apps/app-github-issues-intake?run=${runId}`,
    });

    await user.click(screen.getByRole("tab", { name: "Automation" }));

    const sidebar = within(screen.getByTestId("intake-source-automation")).getByTestId("canvas-runs-sidebar");
    expect(within(sidebar).getByText("Runs")).toBeInTheDocument();
    expect(within(sidebar).getByText("feat: Add console empty-state")).toBeInTheDocument();
    expect(within(sidebar).getByRole("link", { name: "feat: Add console empty-state" })).toHaveAttribute(
      "href",
      "/org-1/workspaces/RF/apps/app-github-issues-intake?run=run-1",
    );
  });

  it("puts Agent after General when the intake canvas has an agent", async () => {
    const user = userEvent.setup();
    renderPopup({
      agent: { draft: PLANNING_REVIEW_DRAFT, organizationId: "org-1", onSave: vi.fn() },
    });

    expect(screen.getAllByRole("tab").map((tab) => tab.textContent)).toEqual(["General", "Agent", "Automation"]);

    await user.click(screen.getByTestId("intake-settings-tab-agent"));
    expect(screen.getByTestId("planning-review-editor")).toBeInTheDocument();
    expect(screen.getByTestId("planning-review-save")).toHaveTextContent("Save Agent");
    expect(screen.queryByTestId("planning-review-automation-note")).not.toBeInTheDocument();
  });

  it("saves the edited configuration", async () => {
    const onSave = vi.fn();
    const onClose = vi.fn();
    const user = userEvent.setup();
    renderPopup({ onSave, onClose });

    await user.clear(screen.getByLabelText("Name"));
    await user.type(screen.getByLabelText("Name"), "Acme GitHub issues");
    await user.click(screen.getByRole("radio", { name: "Exclude these labels" }));
    await user.click(screen.getByRole("checkbox", { name: "bug" }));
    await user.click(screen.getByRole("radio", { name: "Unassigned" }));
    await user.click(screen.getByRole("checkbox", { name: "Only issues from people with repository access" }));
    await user.click(screen.getByTestId("intake-source-settings-save"));

    await waitFor(() =>
      expect(onSave).toHaveBeenCalledWith({
        name: "Acme GitHub issues",
        listenMode: "listen",
        confidencePct: 65,
        labelFilterMode: "exclude",
        labels: ["bug"],
        assignment: "unassigned",
        authorsWithAccess: true,
      }),
    );
    expect(onClose).toHaveBeenCalledTimes(1);
  });
});
