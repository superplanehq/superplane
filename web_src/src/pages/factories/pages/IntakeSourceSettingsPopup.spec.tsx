import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "bun:test";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import type * as CanvasDataModule from "@/hooks/useCanvasData";
import { prepareData } from "@/pages/app/workflowPageHelpers";
import { unmockedSrc } from "@/test/unmockedModule";
import { TooltipProvider } from "@/ui/tooltip";

import { INTAKE_CONNECTION_COPY } from "./intakeConnectionModel";
import { IntakeSourceSettingsPopup, type IntakeSettingsConnection } from "./IntakeSourceSettingsPopup";
import {
  DEFAULT_GITHUB_INTAKE_SETTINGS,
  INTAKE_SETTINGS_COPY,
  type IntakeSettingsTab,
} from "./intakeSourceSettingsModel";
import { PLANNING_REVIEW_DRAFT } from "./planningReviewMockup";
import type { IntakeAutomationGraph } from "./useIntakeAutomationCanvas";
import type { PlanningReviewAgentSlot } from "./PlanningReviewEditor";
import type { LineIntakeSourceId } from "./lineIntakeModel";

const { useInfiniteCanvasRuns } = vi.hoisted(() => ({
  useInfiniteCanvasRuns: vi.fn(),
}));

vi.mock("@monaco-editor/react", () => ({
  Editor: ({ value, onChange }: { value?: string; onChange?: (value: string | undefined) => void }) => (
    <textarea value={value ?? ""} onChange={(event) => onChange?.(event.target.value)} />
  ),
}));

vi.mock("@/hooks/useCanvasData", () => {
  const actual = unmockedSrc<typeof CanvasDataModule>("hooks/useCanvasData");
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
  const { nodes, edges } = prepareData({
    workflow: {
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
    triggers: [{ name: "github.onIssue", label: "On Issue" }],
    components: [
      { name: "if", label: "If" },
      { name: "createWorkOrder", label: "Create Task" },
    ],
    nodeEventsMap: {},
    nodeExecutionsMap: {},
    nodeQueueItemsMap: {},
    workflowId: "app-github-issues-intake",
    queryClient: new QueryClient(),
    user: null,
    canvasMode: "live",
  });

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
    labelOptions?: string[];
    labelOptionsLoading?: boolean;
    sourceId?: LineIntakeSourceId;
    connection?: IntakeSettingsConnection;
    paused?: boolean;
    pauseError?: string;
    deleteError?: string;
    onPause?: () => void | Promise<void>;
    onResume?: () => void | Promise<void>;
    onDelete?: () => void | Promise<void>;
  } = {},
) {
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <ThemeProvider>
          <TooltipProvider>
            <IntakeSourceSettingsPopup
              settings={
                props.sourceId === "sentry-exceptions"
                  ? { ...DEFAULT_GITHUB_INTAKE_SETTINGS, name: "Sentry exceptions" }
                  : props.sourceId === "jira-issues"
                    ? { ...DEFAULT_GITHUB_INTAKE_SETTINGS, name: "Jira issues" }
                    : DEFAULT_GITHUB_INTAKE_SETTINGS
              }
              sourceId={props.sourceId}
              connection={props.connection}
              labelOptions={props.labelOptions ?? ["bug", "enhancement"]}
              labelOptionsLoading={props.labelOptionsLoading}
              automationGraph={githubAutomationGraph}
              onSave={props.onSave ?? vi.fn()}
              paused={props.paused}
              pauseError={props.pauseError}
              deleteError={props.deleteError}
              onPause={props.onPause}
              onResume={props.onResume}
              onDelete={props.onDelete}
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
    expect(screen.queryByLabelText("Name")).not.toBeInTheDocument();
    expect(screen.queryByRole("radio", { name: /Listen for new issues/ })).not.toBeInTheDocument();
    expect(screen.queryByRole("radio", { name: /Run on a schedule/ })).not.toBeInTheDocument();
    expect(screen.queryByTestId("intake-confidence-value")).not.toBeInTheDocument();
    expect(screen.getByRole("group", { name: "Create task when:" })).toBeInTheDocument();
    expect(screen.getByRole("checkbox", { name: "A new issue is opened" })).toBeChecked();
    expect(screen.getByRole("checkbox", { name: "A closed issue is re-opened" })).toBeChecked();
    expect(screen.getByRole("checkbox", { name: 'The "superplane" label is added to the issue' })).toBeChecked();
    expect(screen.getByRole("group", { name: "Filters" })).toBeInTheDocument();
    expect(screen.getByRole("checkbox", { name: "Issue has one of these labels" })).not.toBeChecked();
    expect(screen.queryByTestId("intake-label-options")).not.toBeInTheDocument();
    expect(screen.getByRole("checkbox", { name: "Author is a repository collaborator" })).not.toBeChecked();
    expect(screen.getByRole("tab", { name: "General" })).toHaveAttribute("data-state", "active");
    expect(screen.getAllByRole("tab").map((tab) => tab.textContent)).toEqual(["General", "Automation"]);
    expect(screen.queryByTestId("intake-settings-tab-agent")).not.toBeInTheDocument();
    expect(screen.queryByRole("tab", { name: "Runs" })).not.toBeInTheDocument();
    expect(screen.queryByTestId("intake-source-automation")).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Edit automation" })).not.toBeInTheDocument();
  });

  it("offers the labels that exist in the repository", async () => {
    const user = userEvent.setup();
    renderPopup({ labelOptions: ["bug", "needs design"] });

    await user.click(screen.getByRole("checkbox", { name: "Issue has one of these labels" }));

    const options = screen.getByTestId("intake-label-options");
    expect(within(options).getByRole("checkbox", { name: "bug" })).not.toBeChecked();
    expect(within(options).getByRole("checkbox", { name: "needs design" })).not.toBeChecked();

    await user.click(within(options).getByRole("checkbox", { name: "bug" }));
    expect(within(options).getByRole("checkbox", { name: "bug" })).toBeChecked();
  });

  it("reports when the repository has no labels", async () => {
    const user = userEvent.setup();
    renderPopup({ labelOptions: [] });

    await user.click(screen.getByRole("checkbox", { name: "Issue has one of these labels" }));

    expect(screen.getByText("No labels found in the repository. Add a label name.")).toBeInTheDocument();
  });

  it("keeps a label that the user types", async () => {
    const user = userEvent.setup();
    renderPopup();

    await user.click(screen.getByRole("checkbox", { name: "Issue has one of these labels" }));
    expect(screen.queryByRole("textbox", { name: "Issue label" })).not.toBeInTheDocument();

    await user.click(screen.getByTestId("intake-label-new"));
    expect(screen.getByTestId("intake-label-add")).toBeDisabled();

    await user.type(screen.getByRole("textbox", { name: "Issue label" }), "needs-triage");
    await user.click(screen.getByTestId("intake-label-add"));

    expect(screen.getByRole("checkbox", { name: "needs-triage" })).toBeChecked();
    expect(screen.queryByRole("textbox", { name: "Issue label" })).not.toBeInTheDocument();
  });

  it("adds a label when the user presses Enter", async () => {
    const user = userEvent.setup();
    renderPopup();

    await user.click(screen.getByRole("checkbox", { name: "Issue has one of these labels" }));
    await user.click(screen.getByTestId("intake-label-new"));
    await user.type(screen.getByRole("textbox", { name: "Issue label" }), "needs-triage{Enter}");

    expect(screen.getByRole("checkbox", { name: "needs-triage" })).toBeChecked();
  });

  it("hides the label input when the user cancels", async () => {
    const user = userEvent.setup();
    renderPopup();

    await user.click(screen.getByRole("checkbox", { name: "Issue has one of these labels" }));
    await user.click(screen.getByTestId("intake-label-new"));
    await user.type(screen.getByRole("textbox", { name: "Issue label" }), "needs-triage");
    await user.click(screen.getByTestId("intake-label-cancel"));

    expect(screen.queryByRole("textbox", { name: "Issue label" })).not.toBeInTheDocument();
    expect(screen.queryByRole("checkbox", { name: "needs-triage" })).not.toBeInTheDocument();
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
    expect(screen.queryByTestId("factory-automation-runs-sidebar")).not.toBeInTheDocument();
  });

  it("lists canvas runs beside the automation when a canvas id is given", async () => {
    const user = userEvent.setup();
    renderPopup({
      canvasId: "app-github-issues-intake",
      runHrefFor: (runId) => `/org-1/workspaces/RF/apps/app-github-issues-intake?run=${runId}`,
    });

    await user.click(screen.getByRole("tab", { name: "Automation" }));

    const sidebar = within(screen.getByTestId("intake-source-automation")).getByTestId(
      "factory-automation-runs-sidebar",
    );
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

    await user.click(screen.getByRole("checkbox", { name: "Issue has one of these labels" }));
    await user.click(within(screen.getByTestId("intake-label-options")).getByRole("checkbox", { name: "bug" }));
    await user.click(screen.getByRole("checkbox", { name: "A closed issue is re-opened" }));
    await user.click(screen.getByRole("checkbox", { name: "Author is a repository collaborator" }));
    await user.click(screen.getByTestId("intake-source-settings-save"));

    await waitFor(() =>
      expect(onSave).toHaveBeenCalledWith({
        name: "GitHub issues",
        confidencePct: 65,
        labelFilterMode: "include",
        labels: ["bug"],
        filterByLabel: true,
        assignment: "any",
        newIssues: true,
        reopenedIssues: false,
        superplaneLabelAdded: true,
        authorsWithAccess: true,
      }),
    );
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("hides the Connection section for a GitHub intake", () => {
    renderPopup();

    expect(screen.queryByTestId("intake-connection")).not.toBeInTheDocument();
  });

  it("shows Connection fields for a Jira intake that needs a live connection", () => {
    renderPopup({
      sourceId: "jira-issues",
      connection: {
        health: "HEALTH_MISSING_INTEGRATION",
        binding: { integrationId: "", resourceId: "" },
        integrations: [
          {
            metadata: { id: "jira-1", name: "Atlassian", integrationName: "jira" },
            status: { state: "ready" },
          },
        ],
        projects: [],
        onBindingChange: vi.fn(),
        onConnect: vi.fn(),
      },
    });

    expect(screen.getByTestId("intake-connection")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: INTAKE_CONNECTION_COPY.section })).toBeInTheDocument();
    expect(screen.getByTestId("intake-connection-banner")).toHaveTextContent(INTAKE_CONNECTION_COPY.missing);
    expect(screen.getByText("Choose the Jira site that SuperPlane will monitor.")).toBeInTheDocument();
    expect(screen.getByTestId("intake-connection-jira-1")).toHaveTextContent("Atlassian");
  });

  it("keeps Save disabled until the Jira project is chosen", () => {
    renderPopup({
      sourceId: "jira-issues",
      connection: {
        health: "HEALTH_MISSING_INTEGRATION",
        binding: { integrationId: "jira-1", resourceId: "" },
        integrations: [
          {
            metadata: { id: "jira-1", name: "Atlassian", integrationName: "jira" },
            status: { state: "ready" },
          },
        ],
        projects: [{ id: "ENG", name: "Engineering" }],
        saveDisabled: true,
        onBindingChange: vi.fn(),
        onConnect: vi.fn(),
      },
    });

    expect(screen.getByTestId("intake-source-settings-save")).toBeDisabled();
  });

  it("hides pause and delete for a GitHub intake", () => {
    renderPopup();

    expect(screen.queryByTestId("intake-source-settings-pause")).not.toBeInTheDocument();
    expect(screen.queryByTestId("intake-source-settings-resume")).not.toBeInTheDocument();
    expect(screen.queryByTestId("intake-source-settings-delete")).not.toBeInTheDocument();
  });

  it.each(["sentry-exceptions", "jira-issues"] as const)(
    "pauses, resumes, and deletes a %s intake after confirmation",
    async (sourceId) => {
      const onPause = vi.fn();
      const onResume = vi.fn();
      const onDelete = vi.fn();
      const user = userEvent.setup();
      renderPopup({ sourceId, onPause, onResume, onDelete });

      expect(screen.getByTestId("intake-source-settings-pause")).toHaveTextContent(INTAKE_SETTINGS_COPY.pause);
      expect(screen.getByTestId("intake-source-settings-delete")).toHaveTextContent(INTAKE_SETTINGS_COPY.delete);

      await user.click(screen.getByTestId("intake-source-settings-pause"));
      expect(onPause).toHaveBeenCalledTimes(1);

      await user.click(screen.getByTestId("intake-source-settings-delete"));
      expect(screen.getByTestId("intake-delete-dialog")).toBeInTheDocument();
      expect(onDelete).not.toHaveBeenCalled();
      await user.click(screen.getByTestId("intake-delete-cancel"));
      expect(screen.queryByTestId("intake-delete-dialog")).not.toBeInTheDocument();
      expect(onDelete).not.toHaveBeenCalled();

      await user.click(screen.getByTestId("intake-source-settings-delete"));
      await user.click(screen.getByTestId("intake-delete-confirm"));
      expect(onDelete).toHaveBeenCalledTimes(1);
    },
  );

  it.each(["sentry-exceptions", "jira-issues"] as const)("offers resume for a paused %s intake", async (sourceId) => {
    const onResume = vi.fn();
    const user = userEvent.setup();
    renderPopup({ sourceId, paused: true, onResume });

    expect(screen.queryByTestId("intake-source-settings-pause")).not.toBeInTheDocument();
    await user.click(screen.getByTestId("intake-source-settings-resume"));
    expect(onResume).toHaveBeenCalledTimes(1);
  });

  it("keeps a failed pause from rejecting and shows the error", async () => {
    const onPause = vi.fn().mockRejectedValue(new Error("pause failed"));
    const user = userEvent.setup();
    renderPopup({
      sourceId: "sentry-exceptions",
      onPause,
      pauseError: INTAKE_SETTINGS_COPY.pauseError,
    });

    await user.click(screen.getByTestId("intake-source-settings-pause"));

    expect(onPause).toHaveBeenCalledTimes(1);
    expect(screen.getByRole("alert")).toHaveTextContent(INTAKE_SETTINGS_COPY.pauseError);
  });

  it("shows a delete error in the confirmation dialog", async () => {
    const user = userEvent.setup();
    renderPopup({
      sourceId: "sentry-exceptions",
      onDelete: vi.fn().mockRejectedValue(new Error("delete failed")),
      deleteError: INTAKE_SETTINGS_COPY.deleteError,
    });

    await user.click(screen.getByTestId("intake-source-settings-delete"));

    const dialog = screen.getByTestId("intake-delete-dialog");
    expect(within(dialog).getByTestId("intake-delete-error")).toHaveTextContent(INTAKE_SETTINGS_COPY.deleteError);
    expect(within(screen.getByTestId("intake-source-settings")).queryByRole("alert")).not.toBeInTheDocument();
  });
});
