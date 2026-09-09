import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { prepareData } from "@/pages/app/workflowPageHelpers";
import { TooltipProvider } from "@/ui/tooltip";

import { IntakeSourceSettingsPopup } from "./IntakeSourceSettingsPopup";
import {
  DEFAULT_GITHUB_INTAKE_SETTINGS,
  GITHUB_INTAKE_RUNS,
  type IntakeAutomationRun,
  type IntakeSettingsTab,
} from "./intakeSourceSettingsModel";
import { PLANNING_REVIEW_DRAFT } from "./planningReviewMockup";
import type { IntakeAutomationGraph } from "./useIntakeAutomationCanvas";
import type { PlanningReviewAgentSlot } from "./PlanningReviewEditor";

vi.mock("@monaco-editor/react", () => ({
  Editor: ({ value, onChange }: { value?: string; onChange?: (value: string | undefined) => void }) => (
    <textarea value={value ?? ""} onChange={(event) => onChange?.(event.target.value)} />
  ),
}));

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

  return { nodes, edges, factoryId: "factory-1" };
}

function renderPopup(
  props: {
    onSave?: (next: typeof DEFAULT_GITHUB_INTAKE_SETTINGS) => void;
    onClose?: () => void;
    onOpenRun?: (run: IntakeAutomationRun) => void;
    runs?: IntakeAutomationRun[];
    editAutomationHref?: string;
    agent?: PlanningReviewAgentSlot;
    initialTab?: IntakeSettingsTab;
    labelOptions?: string[];
    labelOptionsLoading?: boolean;
  } = {},
) {
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <ThemeProvider>
          <TooltipProvider>
            <IntakeSourceSettingsPopup
              settings={DEFAULT_GITHUB_INTAKE_SETTINGS}
              labelOptions={props.labelOptions ?? ["bug", "enhancement"]}
              labelOptionsLoading={props.labelOptionsLoading}
              automationGraph={githubAutomationGraph}
              onSave={props.onSave ?? vi.fn()}
              onOpenRun={props.onOpenRun}
              runs={props.runs}
              editAutomationHref={props.editAutomationHref}
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
    expect(screen.getByRole("group", { name: "Create tasks from" })).toBeInTheDocument();
    expect(screen.getByRole("checkbox", { name: "New and re-opened issues" })).toBeChecked();
    expect(screen.getByRole("checkbox", { name: "Issues assigned to @superplaneagent" })).not.toBeChecked();
    expect(screen.getByRole("group", { name: "Filters" })).toBeInTheDocument();
    expect(screen.getByRole("checkbox", { name: "Only issues with any of these labels" })).not.toBeChecked();
    expect(screen.queryByTestId("intake-label-options")).not.toBeInTheDocument();
    expect(screen.getByRole("checkbox", { name: "Only issues from people with repository access" })).not.toBeChecked();
    expect(screen.getByRole("tab", { name: "General" })).toHaveAttribute("data-state", "active");
    expect(screen.getAllByRole("tab").map((tab) => tab.textContent)).toEqual(["General", "Runs", "Automation"]);
    expect(screen.queryByTestId("intake-settings-tab-agent")).not.toBeInTheDocument();
    expect(screen.queryByTestId("intake-source-automation")).not.toBeInTheDocument();
    expect(screen.queryByTestId("intake-source-runs")).not.toBeInTheDocument();
  });

  it("offers the labels that exist in the repository", async () => {
    const user = userEvent.setup();
    renderPopup({ labelOptions: ["bug", "needs design"] });

    await user.click(screen.getByRole("checkbox", { name: "Only issues with any of these labels" }));

    const options = screen.getByTestId("intake-label-options");
    expect(within(options).getByRole("checkbox", { name: "bug" })).not.toBeChecked();
    expect(within(options).getByRole("checkbox", { name: "needs design" })).not.toBeChecked();

    await user.click(within(options).getByRole("checkbox", { name: "bug" }));
    expect(within(options).getByRole("checkbox", { name: "bug" })).toBeChecked();
  });

  it("reports when the repository has no labels", async () => {
    const user = userEvent.setup();
    renderPopup({ labelOptions: [] });

    await user.click(screen.getByRole("checkbox", { name: "Only issues with any of these labels" }));

    expect(screen.getByText("No labels found in the repository. Add a label name.")).toBeInTheDocument();
  });

  it("keeps a label that the user types", async () => {
    const user = userEvent.setup();
    renderPopup();

    await user.click(screen.getByRole("checkbox", { name: "Only issues with any of these labels" }));
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

    await user.click(screen.getByRole("checkbox", { name: "Only issues with any of these labels" }));
    await user.click(screen.getByTestId("intake-label-new"));
    await user.type(screen.getByRole("textbox", { name: "Issue label" }), "needs-triage{Enter}");

    expect(screen.getByRole("checkbox", { name: "needs-triage" })).toBeChecked();
  });

  it("hides the label input when the user cancels", async () => {
    const user = userEvent.setup();
    renderPopup();

    await user.click(screen.getByRole("checkbox", { name: "Only issues with any of these labels" }));
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
    expect(within(automation).getByRole("link", { name: "Edit automation" })).toHaveAttribute(
      "href",
      "/org-1/workspaces/RF/apps/app-github-issues-intake?configure=1&agent=1",
    );
    expect(document.querySelector(".sp-canvas-editing")).toBeNull();
    expect(within(automation).queryByRole("button", { name: /Add next component/ })).not.toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Accepted events go to Backlog" })).not.toBeInTheDocument();
    expect(screen.queryByLabelText("Name")).not.toBeInTheDocument();
    expect(screen.queryByTestId("intake-source-settings-save")).not.toBeInTheDocument();
  });

  it("lists scored intake runs on the Runs tab", async () => {
    const onOpenRun = vi.fn();
    const user = userEvent.setup();
    renderPopup({ onOpenRun, runs: GITHUB_INTAKE_RUNS });

    await user.click(screen.getByRole("tab", { name: "Runs" }));

    const runs = screen.getByTestId("intake-source-runs");
    expect(runs).toHaveAccessibleName("Runs");
    expect(within(runs).getByText("Handle duplicate refunds on retry")).toBeInTheDocument();
    expect(within(runs).queryByText("acme/payments-service")).not.toBeInTheDocument();
    expect(within(runs).queryByText("acme/docs")).not.toBeInTheDocument();

    const implement = within(runs).getByTestId("intake-source-run-gh-issue-1");
    expect(implement).toHaveTextContent("94%");
    expect(implement).toHaveTextContent("3h ago");
    expect(implement).toHaveTextContent("2h ago");
    expect(implement).toHaveTextContent("Implement");
    expect(implement).toHaveTextContent("Writing the retry handler.");
    expect(implement).not.toHaveTextContent("Moved to Backlog");

    expect(within(runs).getByTestId("intake-source-run-gh-issue-2")).toHaveTextContent("Verify");
    expect(within(runs).getByTestId("intake-source-run-gh-issue-3")).toHaveTextContent("In Backlog");
    expect(within(runs).getByTestId("intake-source-run-gh-issue-4")).toHaveTextContent("Rejected");
    expect(within(runs).getByTestId("intake-source-run-gh-issue-5")).toHaveTextContent("Waiting for review.");

    const held = within(runs).getByTestId("intake-source-run-gh-issue-6");
    expect(held).toHaveTextContent("52%");
    expect(held).toHaveTextContent("Not moved to Backlog");
    expect(held).not.toHaveTextContent("Below the confidence score");

    expect(within(runs).getAllByTestId(/^intake-source-run-/)).toHaveLength(6);
    expect(screen.queryByLabelText("Name")).not.toBeInTheDocument();
    expect(screen.queryByTestId("intake-source-settings-save")).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "View run for Handle duplicate refunds on retry" }));
    expect(onOpenRun).toHaveBeenCalledWith(expect.objectContaining({ id: "gh-issue-1", placement: "progressed" }));
  });

  it("puts Agent after General when the intake canvas has an agent", async () => {
    const user = userEvent.setup();
    renderPopup({
      agent: { draft: PLANNING_REVIEW_DRAFT, organizationId: "org-1", onSave: vi.fn() },
    });

    expect(screen.getAllByRole("tab").map((tab) => tab.textContent)).toEqual([
      "General",
      "Agent",
      "Runs",
      "Automation",
    ]);

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

    await user.click(screen.getByRole("checkbox", { name: "Only issues with any of these labels" }));
    await user.click(within(screen.getByTestId("intake-label-options")).getByRole("checkbox", { name: "bug" }));
    await user.click(screen.getByRole("checkbox", { name: "Issues assigned to @superplaneagent" }));
    await user.click(screen.getByRole("checkbox", { name: "Only issues from people with repository access" }));
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
        assignedToAgent: true,
        authorsWithAccess: true,
      }),
    );
    expect(onClose).toHaveBeenCalledTimes(1);
  });
});
