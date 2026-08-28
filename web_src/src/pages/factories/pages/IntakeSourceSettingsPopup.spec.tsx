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
} from "./intakeSourceSettingsModel";
import type { IntakeAutomationGraph } from "./useIntakeAutomationCanvas";

const githubAutomationGraph = githubIntakeGraph();

/** Same pipeline the canvas editor uses, so the popup renders editor nodes. */
function githubIntakeGraph(): IntakeAutomationGraph {
  const { nodes, edges } = prepareData(
    {
      metadata: { id: "app-github-issues-intake", name: "GitHub issues", factoryId: "factory-1" },
      spec: {
        nodes: [
          { id: "trigger", name: "On Issue", type: "TYPE_TRIGGER", component: "github.onIssue" },
          { id: "analysis", name: "Analyze intake", type: "TYPE_ACTION", component: "runnerClaudeCode" },
          { id: "create", name: "Create Work Order", type: "TYPE_ACTION", component: "createWorkOrder" },
        ],
        edges: [
          { channel: "default", sourceId: "trigger", targetId: "analysis" },
          { channel: "passed", sourceId: "analysis", targetId: "create" },
        ],
      },
    },
    [{ name: "github.onIssue", label: "On Issue" }],
    [
      { name: "runnerClaudeCode", label: "Run Claude Code" },
      { name: "createWorkOrder", label: "Create Work Order" },
    ],
    {},
    {},
    {},
    "app-github-issues-intake",
    new QueryClient(),
    null,
    "edit",
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
    initialTab?: "general" | "runs" | "automation";
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
              onOpenRun={props.onOpenRun}
              runs={props.runs}
              editAutomationHref={props.editAutomationHref}
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
    expect(screen.getByLabelText("Name")).toHaveValue("GitHub issues");
    expect(screen.getByRole("radio", { name: /Listen for new issues/ })).toBeChecked();
    expect(screen.getByRole("radio", { name: /Run on a schedule/ })).not.toBeChecked();
    expect(screen.getByTestId("intake-confidence-value")).toHaveTextContent("65%");
    expect(screen.getByRole("radio", { name: "Include these labels" })).toBeChecked();
    expect(screen.getByRole("radio", { name: "Exclude these labels" })).not.toBeChecked();
    expect(screen.getByRole("checkbox", { name: "bug" })).not.toBeChecked();
    expect(screen.getByRole("radio", { name: "Any assignment" })).toBeChecked();
    expect(screen.getByRole("tab", { name: "General" })).toHaveAttribute("data-state", "active");
    expect(screen.getAllByRole("tab").map((tab) => tab.textContent)).toEqual(["General", "Runs", "Automation"]);
    expect(screen.queryByTestId("intake-source-automation")).not.toBeInTheDocument();
    expect(screen.queryByTestId("intake-source-runs")).not.toBeInTheDocument();
  });

  it("shows the intake automation on the Automation tab", async () => {
    const user = userEvent.setup();
    renderPopup({ editAutomationHref: "/org-1/workspaces/RF/apps/app-github-issues-intake?configure=1&agent=1" });

    await user.click(screen.getByRole("tab", { name: "Automation" }));

    const automation = screen.getByTestId("intake-source-automation");
    expect(automation).toHaveAccessibleName("Automation");
    expect(within(automation).getByTestId("rf__node-trigger")).toBeInTheDocument();
    expect(within(automation).getAllByText("On Issue").length).toBeGreaterThan(0);
    expect(within(automation).getByText("Analyze intake")).toBeInTheDocument();
    expect(within(automation).getAllByText("Create Work Order").length).toBeGreaterThan(0);
    expect(within(automation).getByRole("link", { name: "Edit automation" })).toHaveAttribute(
      "href",
      "/org-1/workspaces/RF/apps/app-github-issues-intake?configure=1&agent=1",
    );
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
    await user.click(screen.getByTestId("intake-source-settings-save"));

    await waitFor(() =>
      expect(onSave).toHaveBeenCalledWith({
        name: "Acme GitHub issues",
        listenMode: "listen",
        confidencePct: 65,
        labelFilterMode: "exclude",
        labels: ["bug"],
        assignment: "unassigned",
      }),
    );
    expect(onClose).toHaveBeenCalledTimes(1);
  });
});
