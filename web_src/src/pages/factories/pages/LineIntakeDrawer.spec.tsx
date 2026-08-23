import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";

import { LineIntakeDrawer } from "./LineIntakeDrawer";
import { LINE_INTAKE_SOURCES, type LineIntakeAnalyzingTicket, type LineIntakeSource } from "./lineIntakeModel";

function renderDrawer(
  props: {
    onClose?: () => void;
    initialSourceId?: "github-issues" | "sentry-exceptions" | "pagerduty-incidents";
    initialSettingsOpen?: boolean;
    initialSettingsTab?: "general" | "runs" | "automation";
    sources?: LineIntakeSource[];
    onOpenTicket?: (ticket: LineIntakeAnalyzingTicket) => void;
    editAutomationHref?: string;
  } = {},
) {
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <ThemeProvider>
          <TooltipProvider>
            <LineIntakeDrawer
              onClose={props.onClose ?? vi.fn()}
              initialSourceId={props.initialSourceId}
              initialSettingsOpen={props.initialSettingsOpen}
              initialSettingsTab={props.initialSettingsTab}
              sources={props.sources}
              onOpenTicket={props.onOpenTicket}
              editAutomationHref={props.editAutomationHref}
            />
          </TooltipProvider>
        </ThemeProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("LineIntakeDrawer", () => {
  it("lists GitHub, Sentry, and PagerDuty as intake sources", () => {
    renderDrawer();

    const drawer = screen.getByTestId("line-intake-drawer");
    expect(drawer).toHaveAccessibleName("Intake");
    expect(screen.getByRole("heading", { name: "Intake" })).toBeInTheDocument();
    expect(screen.getByTestId("line-intake-source-github-issues")).toHaveTextContent("GitHub issues");
    expect(screen.getByTestId("line-intake-source-sentry-exceptions")).toHaveTextContent("Sentry exceptions");
    expect(screen.getByTestId("line-intake-source-pagerduty-incidents")).toHaveTextContent("PagerDuty incidents");
  });

  it("can show GitHub issues without Sentry or PagerDuty", () => {
    renderDrawer({
      sources: LINE_INTAKE_SOURCES.filter((source) => source.id === "github-issues"),
    });

    expect(screen.getByTestId("line-intake-source-github-issues")).toBeInTheDocument();
    expect(screen.queryByTestId("line-intake-source-sentry-exceptions")).not.toBeInTheDocument();
    expect(screen.queryByTestId("line-intake-source-pagerduty-incidents")).not.toBeInTheDocument();
  });

  it("closes the drawer from the header control", async () => {
    const onClose = vi.fn();
    const user = userEvent.setup();
    renderDrawer({ onClose });

    await user.click(screen.getByTestId("line-intake-close"));
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("opens a searchable picker with six intake templates", async () => {
    const user = userEvent.setup();
    renderDrawer();

    await user.click(screen.getByTestId("line-intake-add"));

    const picker = screen.getByTestId("add-intake-picker");
    expect(within(picker).getByRole("heading", { name: "Add intake" })).toBeInTheDocument();
    expect(within(picker).getByTestId("add-intake-search")).toBeInTheDocument();
    expect(within(picker).getByTestId("add-intake-template-improve-ci-runtime")).toHaveTextContent(
      "Improve CI runtime",
    );
    expect(within(picker).getByTestId("add-intake-template-improve-page-performance")).toHaveTextContent(
      "Improve page performance",
    );
    expect(within(picker).getAllByTestId(/^add-intake-template-/)).toHaveLength(6);
  });

  it("filters templates from the picker search", async () => {
    const user = userEvent.setup();
    renderDrawer();

    await user.click(screen.getByTestId("line-intake-add"));
    await user.type(screen.getByTestId("add-intake-search"), "runtime");

    expect(screen.getByTestId("add-intake-template-improve-ci-runtime")).toBeInTheDocument();
    expect(screen.queryByTestId("add-intake-template-flaky-tests")).not.toBeInTheDocument();
  });

  it("lets each source expand and collapse on its own", async () => {
    const user = userEvent.setup();
    renderDrawer();

    expect(screen.getByRole("button", { name: "Expand GitHub issues" })).toHaveAttribute("aria-expanded", "false");
    expect(screen.getByRole("button", { name: "Expand Sentry exceptions" })).toHaveAttribute("aria-expanded", "false");
    expect(screen.getByRole("button", { name: "Expand PagerDuty incidents" })).toHaveAttribute(
      "aria-expanded",
      "false",
    );

    await user.click(screen.getByRole("button", { name: "Expand GitHub issues" }));

    const github = screen.getByTestId("line-intake-source-github-issues");
    const analyzing = within(github).getByTestId("line-intake-analyzing");
    expect(within(analyzing).getAllByTestId("line-intake-analyzing-spinner")).toHaveLength(5);
    expect(within(analyzing).getByText("Handle duplicate refunds on retry")).toBeInTheDocument();
    expect(within(analyzing).queryByText("acme/api")).not.toBeInTheDocument();
    expect(within(analyzing).queryByText("Analyzing")).not.toBeInTheDocument();
    expect(within(github).getByText("Analyzing")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Collapse GitHub issues" })).toHaveAttribute("aria-expanded", "true");
    expect(screen.queryByTestId("work-order-split-run")).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Expand Sentry exceptions" }));

    const sentry = screen.getByTestId("line-intake-source-sentry-exceptions");
    expect(within(sentry).getByText("No tickets in analysis.")).toBeInTheDocument();
    expect(within(sentry).queryByText("Handle duplicate refunds on retry")).not.toBeInTheDocument();
    expect(within(github).getByTestId("line-intake-analyzing")).toBeInTheDocument();
  });

  it("collapses GitHub issues on a second header click", async () => {
    const user = userEvent.setup();
    renderDrawer({ initialSourceId: "github-issues" });

    expect(screen.getByTestId("line-intake-analyzing")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Collapse GitHub issues" }));

    expect(screen.queryByTestId("line-intake-analyzing")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Expand GitHub issues" })).toHaveAttribute("aria-expanded", "false");
    expect(screen.queryByTestId("work-order-split-run")).not.toBeInTheDocument();
  });

  it("expands GitHub issues when that source is chosen from Add intake", async () => {
    const user = userEvent.setup();
    renderDrawer();

    await user.click(screen.getByTestId("line-intake-add"));
    await user.click(screen.getByTestId("add-intake-template-github-issues"));

    expect(
      within(screen.getByTestId("line-intake-source-github-issues")).getByTestId("line-intake-analyzing"),
    ).toBeInTheDocument();
    expect(screen.queryByTestId("add-intake-picker")).not.toBeInTheDocument();
  });

  it("expands GitHub issues when it is the initial source", () => {
    renderDrawer({ initialSourceId: "github-issues" });

    const source = screen.getByTestId("line-intake-source-github-issues");
    expect(within(source).getByTestId("line-intake-analyzing")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Collapse GitHub issues" })).toHaveAttribute("aria-expanded", "true");
  });

  it("opens the analysis popup from a nested GitHub issues ticket", async () => {
    const onOpenTicket = vi.fn();
    const user = userEvent.setup();
    renderDrawer({ initialSourceId: "github-issues", onOpenTicket });

    await user.click(screen.getByRole("button", { name: "Open Handle duplicate refunds on retry" }));

    expect(onOpenTicket).toHaveBeenCalledWith(
      expect.objectContaining({ id: "gh-issue-1", title: "Handle duplicate refunds on retry" }),
    );
    const dialog = screen.getByTestId("work-order-split-run");
    expect(within(dialog).getByRole("heading", { name: "Handle duplicate refunds on retry" })).toBeInTheDocument();
    expect(within(dialog).getByTestId("split-run-phase-ingest")).toBeInTheDocument();
    expect(within(dialog).getByTestId("split-run-phase-analyze")).toBeInTheDocument();
    expect(within(dialog).getByTestId("split-run-phase-plan")).toBeInTheDocument();
    expect(within(dialog).getByTestId("split-run-phase-score")).toBeInTheDocument();
    expect(within(dialog).getByTestId("split-run-canvas-node-ticket-ingest")).toBeInTheDocument();
    expect(within(dialog).getByTestId("split-run-canvas-node-ticket-analyze")).toBeInTheDocument();
    expect(within(dialog).getByTestId("split-run-canvas-node-ticket-plan")).toBeInTheDocument();
    expect(within(dialog).getByTestId("split-run-canvas-node-ticket-score")).toBeInTheDocument();
    expect(within(dialog).getByRole("heading", { name: "SuperPlane is analyzing this ticket" })).toBeInTheDocument();
  });

  it("opens GitHub issues settings from the source gear", async () => {
    const user = userEvent.setup();
    renderDrawer();

    await user.click(screen.getByRole("button", { name: "Open GitHub issues settings" }));

    const dialog = screen.getByTestId("intake-source-settings");
    expect(within(dialog).getByRole("heading", { name: "Intake GitHub issues" })).toBeInTheDocument();
    expect(within(dialog).getByLabelText("Name")).toHaveValue("GitHub issues");
    expect(within(dialog).getByRole("radio", { name: /Listen for new issues/ })).toBeChecked();
    expect(within(dialog).getByTestId("intake-confidence-value")).toHaveTextContent("65%");
    expect(within(dialog).getByRole("tab", { name: "General" })).toHaveAttribute("data-state", "active");
    expect(
      within(dialog)
        .getAllByRole("tab")
        .map((tab) => tab.textContent),
    ).toEqual(["General", "Runs", "Automation"]);
    expect(screen.queryByTestId("work-order-split-run")).not.toBeInTheDocument();
    expect(screen.queryByTestId("intake-source-automation")).not.toBeInTheDocument();
  });

  it("shows the GitHub issues automation from the settings Automation tab", async () => {
    const user = userEvent.setup();
    renderDrawer({
      editAutomationHref: "/org-1/workspaces/RF/apps/app-github-issues-intake?configure=1&from=lines",
    });

    await user.click(screen.getByRole("button", { name: "Open GitHub issues settings" }));
    await user.click(screen.getByRole("tab", { name: "Automation" }));

    const automation = within(screen.getByTestId("intake-source-settings")).getByTestId("intake-source-automation");
    expect(within(automation).getByTestId("run-overlay-compact-canvas")).toBeInTheDocument();
    expect(within(automation).getByTestId("split-run-canvas-node-github-issues-trigger")).toBeInTheDocument();
    expect(within(automation).getByRole("link", { name: "Edit automation" })).toHaveAttribute(
      "href",
      "/org-1/workspaces/RF/apps/app-github-issues-intake?configure=1&from=lines",
    );
    expect(within(automation).queryByTestId("split-run-phase-listen")).not.toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Accepted events go to Backlog" })).not.toBeInTheDocument();
    expect(screen.queryByTestId("work-order-split-run")).not.toBeInTheDocument();
  });

  it("keeps the analysis popup when a ticket is opened after settings", async () => {
    const user = userEvent.setup();
    renderDrawer({ initialSourceId: "github-issues" });

    await user.click(screen.getByRole("button", { name: "Open GitHub issues settings" }));
    expect(screen.getByTestId("intake-source-settings")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Open Handle duplicate refunds on retry" }));

    expect(screen.queryByTestId("intake-source-settings")).not.toBeInTheDocument();
    expect(
      within(screen.getByTestId("work-order-split-run")).getByRole("heading", {
        name: "Handle duplicate refunds on retry",
      }),
    ).toBeInTheDocument();
  });

  it("opens an intake run from the Runs tab and keeps settings open", async () => {
    const onOpenTicket = vi.fn();
    const user = userEvent.setup();
    renderDrawer({
      initialSourceId: "github-issues",
      initialSettingsOpen: true,
      initialSettingsTab: "runs",
      onOpenTicket,
    });

    const settings = screen.getByTestId("intake-source-settings");
    expect(within(settings).getByTestId("intake-source-run-gh-issue-1")).toHaveTextContent("Implement");
    expect(within(settings).queryByText("acme/payments-service")).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "View run for Handle duplicate refunds on retry" }));

    expect(onOpenTicket).toHaveBeenCalledWith(
      expect.objectContaining({ id: "gh-issue-1", title: "Handle duplicate refunds on retry" }),
    );
    expect(screen.getByTestId("intake-source-settings")).toBeInTheDocument();
    const dialog = screen.getByTestId("work-order-split-run");
    expect(within(dialog).getByRole("heading", { name: "Handle duplicate refunds on retry" })).toBeInTheDocument();
    expect(within(dialog).getByTestId("split-run-phase-analyze")).toBeInTheDocument();
  });

  it("updates the GitHub issues name after save", async () => {
    const user = userEvent.setup();
    renderDrawer();

    await user.click(screen.getByRole("button", { name: "Open GitHub issues settings" }));
    await user.clear(screen.getByLabelText("Name"));
    await user.type(screen.getByLabelText("Name"), "Acme issues");
    await user.click(screen.getByTestId("intake-source-settings-save"));

    expect(screen.queryByTestId("intake-source-settings")).not.toBeInTheDocument();
    expect(screen.getByTestId("line-intake-source-github-issues")).toHaveTextContent("Acme issues");
  });
});
