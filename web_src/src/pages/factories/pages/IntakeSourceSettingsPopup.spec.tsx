import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "bun:test";

import githubIcon from "@/assets/icons/integrations/github.svg";
import jiraIcon from "@/assets/icons/integrations/jira.svg";
import pagerdutyIcon from "@/assets/icons/integrations/pagerduty.svg";
import productiveIcon from "@/assets/icons/integrations/productive.svg";
import sentryIcon from "@/assets/icons/integrations/sentry.svg";
import { ThemeProvider } from "@/contexts/ThemeProvider";
import type * as CanvasDataModule from "@/hooks/useCanvasData";
import { prepareData } from "@/pages/app/workflowPageHelpers";
import { unmockedSrc } from "@/test/unmockedModule";
import { TooltipProvider } from "@/ui/tooltip";

import { IntakeSourceSettingsPopup } from "./IntakeSourceSettingsPopup";
import {
  DEFAULT_GITHUB_INTAKE_SETTINGS,
  DEFAULT_PRODUCTIVE_INTAKE_SETTINGS,
  DEFAULT_SENTRY_INTAKE_SETTINGS,
  INTAKE_SETTINGS_COPY,
  type IntakeSettingsTab,
} from "./intakeSourceSettingsModel";
import { JIRA_COMPLETION_COLUMN_COPY } from "./jiraCompletionColumnCopy";
import { LINEAR_INTAKE_SETUP_COPY } from "./linearIntakeSetupCopy";
import { SENTRY_INTAKE_SETUP_COPY } from "./sentryIntakeSetupCopy";
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

vi.mock("@/hooks/useIntegrations", () => ({
  useIntegrationResources: () => ({
    data: [
      { id: "todo", name: "To Do" },
      { id: "qa", name: "QA" },
      { id: "done", name: "Done" },
    ],
    isLoading: false,
    isError: false,
  }),
}));

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
    canvasId?: string;
    agent?: PlanningReviewAgentSlot;
    canEdit?: boolean;
    initialTab?: IntakeSettingsTab;
    page?: boolean;
    labelOptions?: string[];
    labelOptionsLoading?: boolean;
    sourceId?: LineIntakeSourceId;
    settings?: typeof DEFAULT_GITHUB_INTAKE_SETTINGS;
    organizationId?: string;
    integrationId?: string;
    resourceId?: string;
    deleteError?: string;
    onDelete?: () => void | Promise<void>;
  } = {},
) {
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <ThemeProvider>
          <TooltipProvider>
            <IntakeSourceSettingsPopup
              canEdit={props.canEdit ?? true}
              accessLoading={false}
              settings={
                props.settings ??
                (props.sourceId === "sentry-exceptions"
                  ? DEFAULT_SENTRY_INTAKE_SETTINGS
                  : props.sourceId === "jira-issues"
                    ? { ...DEFAULT_GITHUB_INTAKE_SETTINGS, name: "Jira issues" }
                    : props.sourceId === "pagerduty-incidents"
                      ? { ...DEFAULT_GITHUB_INTAKE_SETTINGS, name: "PagerDuty incidents" }
                      : props.sourceId === "productive-tasks"
                        ? DEFAULT_PRODUCTIVE_INTAKE_SETTINGS
                        : DEFAULT_GITHUB_INTAKE_SETTINGS)
              }
              sourceId={props.sourceId}
              organizationId={props.organizationId}
              integrationId={props.integrationId}
              resourceId={props.resourceId}
              labelOptions={props.labelOptions ?? ["bug", "enhancement"]}
              labelOptionsLoading={props.labelOptionsLoading}
              automationGraph={githubAutomationGraph}
              onSave={props.onSave ?? vi.fn()}
              deleteError={props.deleteError}
              onDelete={props.onDelete}
              canvasId={props.canvasId}
              agent={props.agent}
              onClose={props.onClose ?? vi.fn()}
              initialTab={props.initialTab}
              page={props.page}
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
    expect(screen.getByRole("heading", { name: "Create tasks from GitHub issues" })).toBeInTheDocument();
    expect(screen.queryByLabelText("Name")).not.toBeInTheDocument();
    expect(screen.queryByRole("radio", { name: /Listen for new issues/ })).not.toBeInTheDocument();
    expect(screen.queryByRole("radio", { name: /Run on a schedule/ })).not.toBeInTheDocument();
    expect(screen.queryByTestId("intake-confidence-value")).not.toBeInTheDocument();
    expect(screen.queryByRole("checkbox", { name: "A new issue is opened" })).not.toBeInTheDocument();
    expect(screen.getByRole("radio", { name: "Only issues with selected labels" })).not.toBeChecked();
    const filters = screen.getByRole("group", { name: "Which issues to include" });
    expect(within(filters).getByRole("checkbox", { name: "Only authors with repository access" })).toBeChecked();
    expect(screen.queryByRole("combobox", { name: "Search or add a label" })).not.toBeInTheDocument();
    expect(screen.getByRole("checkbox", { name: "Include reopened issues" })).toBeChecked();
    expect(
      screen.queryByRole("checkbox", { name: 'The "superplane" label is added to the issue' }),
    ).not.toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "General" })).toHaveAttribute("data-state", "active");
    expect(screen.getAllByRole("tab").map((tab) => tab.textContent)).toEqual(["General", "Runs"]);
    expect(screen.queryByTestId("intake-settings-tab-agent")).not.toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Runs" })).toBeInTheDocument();
    expect(screen.queryByTestId("intake-source-automation")).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Edit automation" })).not.toBeInTheDocument();
  });

  it("requires a label for selected-label intake and allows switching back to all issues", async () => {
    const user = userEvent.setup();
    const onSave = vi.fn();
    renderPopup({ onSave });

    await user.click(screen.getByRole("radio", { name: "Only issues with selected labels" }));
    expect(screen.getByRole("button", { name: "Save changes" })).toBeDisabled();
    expect(screen.getByText("Select at least one label, or choose All issues.")).toBeInTheDocument();

    await user.click(screen.getByRole("combobox", { name: "Search or add a label" }));
    await user.click(screen.getByRole("option", { name: "bug" }));
    expect(screen.getByRole("button", { name: "Save changes" })).toBeEnabled();
    await user.click(screen.getByRole("button", { name: "Remove bug" }));
    expect(screen.getByRole("button", { name: "Save changes" })).toBeDisabled();

    await user.click(screen.getByRole("radio", { name: "All issues" }));
    await user.click(screen.getByRole("button", { name: "Save changes" }));
    expect(onSave).toHaveBeenCalledWith(expect.objectContaining({ filterByLabel: false, labels: [] }));
  });

  it("keeps the intake form and save action on a page without a dialog", async () => {
    const user = userEvent.setup();
    const onSave = vi.fn();
    const onClose = vi.fn();
    renderPopup({ page: true, onSave, onClose });

    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    expect(screen.getByRole("region", { name: "Create tasks from GitHub issues" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "General" })).toHaveAttribute("data-state", "active");
    await user.click(screen.getByRole("checkbox", { name: "Include reopened issues" }));
    await user.click(screen.getByTestId("intake-source-settings-save"));

    await waitFor(() => expect(onSave).toHaveBeenCalledWith(expect.objectContaining({ reopenedIssues: false })));
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it.each([
    ["github-issues", "GitHub issues", githubIcon],
    ["jira-issues", "Jira issues", jiraIcon],
    ["sentry-exceptions", "Sentry exceptions", sentryIcon],
    ["pagerduty-incidents", "PagerDuty incidents", pagerdutyIcon],
    ["productive-tasks", "Productive tasks", productiveIcon],
  ] as const)("shows the %s picture left of the title", (sourceId, name, iconSrc) => {
    renderPopup({ sourceId });

    const heading = screen.getByRole("heading", {
      name: sourceId === "github-issues" ? "Create tasks from GitHub issues" : `Intake ${name}`,
    });
    const icon = screen.getByTestId("intake-source-settings-title-icon");
    expect(icon).toHaveAttribute("alt", "");
    expect(icon).toHaveAttribute("src", iconSrc);
    expect(icon.compareDocumentPosition(heading) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
  });

  it("offers the labels that exist in the repository", async () => {
    const user = userEvent.setup();
    renderPopup({ labelOptions: ["bug", "superplane", "needs design"] });

    await user.click(screen.getByRole("radio", { name: "Only issues with selected labels" }));

    await user.click(screen.getByRole("combobox", { name: "Search or add a label" }));
    expect(screen.getAllByRole("option").map((option) => option.textContent)).toEqual([
      "superplane",
      "bug",
      "needs design",
    ]);
    await user.click(screen.getByRole("combobox", { name: "Search or add a label" }));
    await user.click(screen.getByRole("option", { name: "bug" }));
    expect(screen.queryByRole("option", { name: "bug" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Remove bug" })).toBeInTheDocument();
  });

  it("offers only the superplane label when the repository has no labels", async () => {
    const user = userEvent.setup();
    renderPopup({ labelOptions: [] });

    await user.click(screen.getByRole("radio", { name: "Only issues with selected labels" }));

    await user.click(screen.getByRole("combobox", { name: "Search or add a label" }));
    expect(screen.getAllByRole("option").map((option) => option.textContent)).toEqual(["superplane"]);
  });

  it("keeps a label that the user types", async () => {
    const user = userEvent.setup();
    renderPopup();

    expect(screen.queryByRole("combobox", { name: "Search or add a label" })).not.toBeInTheDocument();

    await user.click(screen.getByRole("radio", { name: "Only issues with selected labels" }));
    await user.type(screen.getByRole("combobox", { name: "Search or add a label" }), "needs-triage");
    await user.click(screen.getByRole("option", { name: 'Add label "needs-triage"' }));

    expect(screen.getByRole("button", { name: "Remove needs-triage" })).toBeInTheDocument();
    expect(screen.getByRole("combobox", { name: "Search or add a label" })).toHaveValue("");
  });

  it("adds a label when the user presses Enter", async () => {
    const user = userEvent.setup();
    renderPopup();

    await user.click(screen.getByRole("radio", { name: "Only issues with selected labels" }));
    await user.type(screen.getByRole("combobox", { name: "Search or add a label" }), "needs-triage{ArrowDown}{Enter}");

    expect(screen.getByRole("button", { name: "Remove needs-triage" })).toBeInTheDocument();
  });

  it("puts Agent after General when the intake canvas has an agent", async () => {
    const user = userEvent.setup();
    renderPopup({
      agent: { draft: PLANNING_REVIEW_DRAFT, organizationId: "org-1", onSave: vi.fn() },
    });

    expect(screen.getAllByRole("tab").map((tab) => tab.textContent)).toEqual(["General", "Agent", "Runs"]);

    await user.click(screen.getByTestId("intake-settings-tab-agent"));
    expect(screen.getByTestId("planning-review-editor")).toBeInTheDocument();
    expect(screen.getByTestId("planning-review-save")).toHaveTextContent("Save Agent");
    expect(screen.queryByTestId("planning-review-automation-note")).not.toBeInTheDocument();
  });

  it("does not expose the agent editor in read-only settings", () => {
    renderPopup({
      canEdit: false,
      initialTab: "agent",
      agent: { draft: PLANNING_REVIEW_DRAFT, organizationId: "org-1", onSave: vi.fn() },
    });

    expect(screen.getAllByRole("tab").map((tab) => tab.textContent)).toEqual(["General", "Runs"]);
    expect(screen.queryByTestId("planning-review-editor")).not.toBeInTheDocument();
    expect(screen.getByRole("checkbox", { name: "Include reopened issues" })).toBeDisabled();
  });

  it("saves the edited configuration", async () => {
    const onSave = vi.fn();
    const onClose = vi.fn();
    const user = userEvent.setup();
    renderPopup({ onSave, onClose });

    await user.click(screen.getByRole("radio", { name: "Only issues with selected labels" }));
    await user.click(screen.getByRole("combobox", { name: "Search or add a label" }));
    await user.click(screen.getByRole("option", { name: "superplane" }));
    await user.click(screen.getByRole("combobox", { name: "Search or add a label" }));
    await user.click(screen.getByRole("option", { name: "bug" }));
    await user.click(screen.getByRole("checkbox", { name: "Include reopened issues" }));
    await user.click(screen.getByRole("checkbox", { name: "Only authors with repository access" }));
    await user.click(screen.getByTestId("intake-source-settings-save"));

    await waitFor(() =>
      expect(onSave).toHaveBeenCalledWith({
        ...DEFAULT_GITHUB_INTAKE_SETTINGS,
        labels: ["superplane", "bug"],
        filterByLabel: true,
        reopenedIssues: false,
        authorsWithAccess: false,
        superplaneLabelAdded: true,
      }),
    );
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it.each([
    "github-issues",
    "dependabot-alerts",
    "sentry-exceptions",
    "jira-issues",
    "productive-tasks",
    "datadog",
    "linear-issues",
  ] as const)("hides connection and unavailable automatic intake controls for a %s intake", (sourceId) => {
    renderPopup({ sourceId });

    expect(screen.queryByTestId("intake-connection")).not.toBeInTheDocument();
    expect(screen.queryByTestId("intake-source-settings-automatic-intake")).not.toBeInTheDocument();
  });

  it.each([
    "github-issues",
    "dependabot-alerts",
    "sentry-exceptions",
    "jira-issues",
    "productive-tasks",
    "datadog",
    "linear-issues",
  ] as const)("deletes a %s intake after confirmation", async (sourceId) => {
    const onDelete = vi.fn();
    const user = userEvent.setup();
    renderPopup({ sourceId, onDelete });

    expect(screen.getByTestId("intake-source-settings-delete")).toHaveTextContent(INTAKE_SETTINGS_COPY.delete);

    await user.click(screen.getByTestId("intake-source-settings-delete"));
    expect(screen.getByTestId("intake-delete-dialog")).toBeInTheDocument();
    expect(onDelete).not.toHaveBeenCalled();
    await user.click(screen.getByTestId("intake-delete-cancel"));
    expect(screen.queryByTestId("intake-delete-dialog")).not.toBeInTheDocument();
    expect(onDelete).not.toHaveBeenCalled();

    await user.click(screen.getByTestId("intake-source-settings-delete"));
    await user.click(screen.getByTestId("intake-delete-confirm"));
    expect(onDelete).toHaveBeenCalledTimes(1);
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

  it("shows the Jira completion column on the General tab", async () => {
    const onSave = vi.fn();
    const user = userEvent.setup();
    renderPopup({
      sourceId: "jira-issues",
      settings: { ...DEFAULT_GITHUB_INTAKE_SETTINGS, name: "Jira issues" },
      organizationId: "org-1",
      integrationId: "jira-1",
      resourceId: "ENG",
      onSave,
    });

    expect(screen.getByRole("heading", { name: "Intake Jira issues" })).toBeInTheDocument();
    expect(screen.getByText(JIRA_COMPLETION_COLUMN_COPY.section)).toBeInTheDocument();
    expect(screen.getByTestId("jira-move-on-complete")).toBeChecked();
    await user.click(screen.getByTestId("jira-completion-column-select"));
    await user.click(screen.getByRole("option", { name: "QA" }));
    await user.click(screen.getByTestId("intake-source-settings-save"));

    expect(onSave).toHaveBeenCalledWith(
      expect.objectContaining({
        jiraMoveOnComplete: true,
        jiraCompletionColumn: "QA",
      }),
    );
  });

  it("does not save a Linear intake with no project", async () => {
    const onSave = vi.fn();
    const user = userEvent.setup();
    renderPopup({
      sourceId: "linear-issues",
      settings: { ...DEFAULT_GITHUB_INTAKE_SETTINGS, name: "Linear issues", linearProjectIds: [] },
      onSave,
    });

    expect(screen.getByText(LINEAR_INTAKE_SETUP_COPY.projectsRequired)).toBeInTheDocument();
    const save = screen.getByTestId("intake-source-settings-save");
    expect(save).toBeDisabled();
    await user.click(save);
    expect(onSave).not.toHaveBeenCalled();
  });

  it("does not save a Sentry intake with no project", async () => {
    const onSave = vi.fn();
    const user = userEvent.setup();
    renderPopup({
      sourceId: "sentry-exceptions",
      settings: { ...DEFAULT_SENTRY_INTAKE_SETTINGS, sentryProjectIds: [] },
      onSave,
    });

    expect(screen.getByText(SENTRY_INTAKE_SETUP_COPY.projectsRequired)).toBeInTheDocument();
    const save = screen.getByTestId("intake-source-settings-save");
    expect(save).toBeDisabled();
    await user.click(save);
    expect(onSave).not.toHaveBeenCalled();
  });
});
