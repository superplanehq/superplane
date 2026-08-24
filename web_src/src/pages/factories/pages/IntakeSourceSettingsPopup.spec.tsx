import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";

import { IntakeSourceSettingsPopup } from "./IntakeSourceSettingsPopup";
import { DEFAULT_GITHUB_INTAKE_SETTINGS, type IntakeAutomationRun } from "./intakeSourceSettingsModel";
import { intakeAutomationCanvas, lineIntakeSourceById } from "./lineIntakeModel";

const githubAutomationCanvas = intakeAutomationCanvas(lineIntakeSourceById("github-issues")!);

function renderPopup(
  props: {
    onSave?: (next: typeof DEFAULT_GITHUB_INTAKE_SETTINGS) => void;
    onClose?: () => void;
    onOpenRun?: (run: IntakeAutomationRun) => void;
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
              automationCanvas={githubAutomationCanvas}
              onSave={props.onSave ?? vi.fn()}
              onOpenRun={props.onOpenRun}
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
    renderPopup({ editAutomationHref: "/org-1/workspaces/RF/apps/app-github-issues-intake?configure=1" });

    await user.click(screen.getByRole("tab", { name: "Automation" }));

    const automation = screen.getByTestId("intake-source-automation");
    expect(automation).toHaveAccessibleName("Automation");
    expect(within(automation).getByText("GitHub issue intake")).toBeInTheDocument();
    expect(within(automation).getByTestId("run-overlay-compact-canvas")).toBeInTheDocument();
    expect(within(automation).getByTestId("split-run-canvas-node-github-issues-trigger")).toBeInTheDocument();
    expect(within(automation).getByTestId("split-run-canvas-node-github-issues-classify")).toBeInTheDocument();
    expect(within(automation).getByTestId("split-run-canvas-node-github-issues-create")).toBeInTheDocument();
    expect(within(automation).getByRole("link", { name: "Edit automation" })).toHaveAttribute(
      "href",
      "/org-1/workspaces/RF/apps/app-github-issues-intake?configure=1",
    );
    expect(within(automation).queryByTestId("split-run-canvas-menu")).not.toBeInTheDocument();
    expect(within(automation).queryByTestId("split-run-phase-listen")).not.toBeInTheDocument();
    expect(within(automation).queryByRole("heading", { name: "Log" })).not.toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Accepted events go to Backlog" })).not.toBeInTheDocument();
    expect(screen.queryByLabelText("Name")).not.toBeInTheDocument();
    expect(screen.queryByTestId("intake-source-settings-save")).not.toBeInTheDocument();
  });

  it("lists scored intake runs on the Runs tab", async () => {
    const onOpenRun = vi.fn();
    const user = userEvent.setup();
    renderPopup({ onOpenRun });

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
    await user.click(screen.getByRole("radio", { name: /Run on a schedule/ }));
    await user.click(screen.getByRole("radio", { name: "Exclude these labels" }));
    await user.click(screen.getByRole("checkbox", { name: "bug" }));
    await user.click(screen.getByRole("radio", { name: "Unassigned" }));
    await user.click(screen.getByTestId("intake-source-settings-save"));

    expect(onSave).toHaveBeenCalledWith({
      name: "Acme GitHub issues",
      listenMode: "schedule",
      confidencePct: 65,
      labelFilterMode: "exclude",
      labels: ["bug"],
      assignment: "unassigned",
    });
    expect(onClose).toHaveBeenCalledTimes(1);
  });
});
