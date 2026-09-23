import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "bun:test";

import type * as CanvasDataModule from "@/hooks/useCanvasData";
import { unmockedSrc } from "@/test/unmockedModule";

import { DEFAULT_GITHUB_INTAKE_SETTINGS, INTAKE_SETTINGS_COPY, intakeSettingsTitle } from "./intakeSourceSettingsModel";
import { PLANNING_REVIEW_DRAFT } from "./planningReviewMockup";
import { lineIntakeSourceById } from "./lineIntakeModel";
import { intakeConnection, renderPopup } from "./intakeSourceSettingsPopupTestSupport";

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

afterEach(() => {
  localStorage.clear();
});

describe("IntakeSourceSettingsPopup", () => {
  it("shows a settings table of contents on the Settings tab", () => {
    renderPopup();

    const toc = screen.getByTestId("intake-settings-toc");
    expect(
      within(toc)
        .getAllByRole("button")
        .map((button) => button.textContent),
    ).toEqual(["Events that create tasks", "Labels", "Filters", "Pause or delete"]);
    expect(screen.getByTestId("intake-settings-toc-triggers")).toHaveAttribute("aria-current", "true");
    expect(document.getElementById("intake-settings-triggers")).toBeInTheDocument();
    expect(document.getElementById("intake-settings-labels")).toBeInTheDocument();
    expect(document.getElementById("intake-settings-filters")).toBeInTheDocument();
    expect(document.getElementById("intake-settings-danger")).toBeInTheDocument();
  });

  it("scrolls to a section from the table of contents without changing tabs", async () => {
    const user = userEvent.setup();
    renderPopup();

    await user.click(screen.getByTestId("intake-settings-toc-filters"));

    expect(screen.getByTestId("intake-settings-tab-general")).toHaveAttribute("aria-current", "page");
    expect(screen.getByTestId("intake-settings-toc-filters")).toHaveAttribute("aria-current", "true");
    expect(screen.queryByTestId("intake-source-automation")).not.toBeInTheDocument();
  });

  it("hides the settings table of contents on the Canvas tab", async () => {
    const user = userEvent.setup();
    renderPopup({ editAutomationHref: "/org-1/workspaces/RF/apps/app-github-issues-intake?configure=1&agent=1" });

    expect(screen.getByTestId("intake-settings-toc")).toBeInTheDocument();

    await user.click(screen.getByTestId("intake-settings-tab-automation"));

    expect(screen.queryByTestId("intake-settings-toc")).not.toBeInTheDocument();
  });

  it("lists Jira settings sections in the table of contents when a connection is shown", () => {
    renderPopup({
      sourceId: "jira-issues",
      organizationId: "org-1",
      connection: intakeConnection({ health: "HEALTH_MISSING_INTEGRATION" }),
    });

    const toc = screen.getByTestId("intake-settings-toc");
    expect(
      within(toc)
        .getAllByRole("button")
        .map((button) => button.textContent),
    ).toEqual(["Connection", "Events that create tasks", "Labels", "When task completes", "Pause or delete"]);
    expect(screen.getByTestId("intake-settings-toc-connection")).toHaveAttribute("aria-current", "true");
    expect(screen.getByTestId("jira-intake-labels")).toBeInTheDocument();
  });

  it("lists Sentry settings sections in the table of contents when a connection is shown", () => {
    renderPopup({
      sourceId: "sentry-exceptions",
      organizationId: "org-1",
      connection: intakeConnection({
        binding: { integrationId: "sentry-1", resourceId: "proj-1" },
        integrations: [
          {
            metadata: { id: "sentry-1", name: "Sentry org", integrationName: "sentry" },
            status: { state: "ready" },
          },
        ],
        projects: [{ id: "proj-1", name: "Frontend" }],
      }),
    });

    const toc = screen.getByTestId("intake-settings-toc");
    expect(
      within(toc)
        .getAllByRole("button")
        .map((button) => button.textContent),
    ).toEqual(["Connection", "Events that create tasks", "Pause or delete"]);
    expect(screen.getByRole("heading", { name: "Events that create tasks" })).toBeInTheDocument();
    expect(screen.getByRole("group", { name: "Events that create tasks" })).toBeInTheDocument();
  });

  it("uses one settings surface for the top bar title and content", () => {
    renderPopup();

    const dialog = screen.getByTestId("intake-source-settings");
    const topbar = screen.getByTestId("intake-settings-topbar");
    expect(dialog.className).toContain("bg-sidebar!");
    expect(topbar.className).toContain("bg-sidebar");
    expect(within(topbar).getByRole("heading", { name: intakeSettingsTitle("github-issues") })).toBeInTheDocument();
    expect(within(topbar).getByTestId("intake-settings-source-icon")).toHaveAttribute(
      "src",
      lineIntakeSourceById("github-issues")?.iconSrc,
    );
    expect(within(topbar).getByRole("button", { name: "Close" })).toBeInTheDocument();
    expect(within(topbar).getByTestId("intake-source-settings-save")).toBeInTheDocument();
    expect(within(dialog).queryByRole("contentinfo")).not.toBeInTheDocument();
  });

  it.each(["github-issues", "jira-issues", "sentry-exceptions", "productive-tasks", "pagerduty-incidents"] as const)(
    "shows the %s icon beside the intake title",
    (sourceId) => {
      renderPopup({ sourceId });

      const topbar = screen.getByTestId("intake-settings-topbar");
      const icon = within(topbar).getByTestId("intake-settings-source-icon");
      expect(icon).toHaveAttribute("src", lineIntakeSourceById(sourceId)?.iconSrc);
      expect(within(topbar).getByRole("heading", { name: intakeSettingsTitle(sourceId) })).toBeInTheDocument();
    },
  );

  it("shows the GitHub issues configuration fields", () => {
    renderPopup();

    const dialog = screen.getByTestId("intake-source-settings");
    expect(dialog).toHaveAttribute("role", "dialog");
    expect(
      within(screen.getByTestId("intake-settings-topbar")).getByRole("heading", {
        name: intakeSettingsTitle("github-issues"),
      }),
    ).toBeInTheDocument();
    expect(screen.queryByLabelText("Name")).not.toBeInTheDocument();
    expect(screen.queryByRole("radio", { name: /Listen for new issues/ })).not.toBeInTheDocument();
    expect(screen.queryByRole("radio", { name: /Run on a schedule/ })).not.toBeInTheDocument();
    expect(screen.queryByTestId("intake-confidence-value")).not.toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Events that create tasks" })).toBeInTheDocument();
    expect(screen.getByText("The events you select here create tasks in the factory.")).toBeInTheDocument();
    expect(screen.getByRole("group", { name: "Events that create tasks" })).toBeInTheDocument();
    expect(screen.getByRole("switch", { name: "A new issue is opened" })).toBeChecked();
    expect(screen.getByText("GitHub opens an issue.")).toBeInTheDocument();
    expect(screen.getByRole("switch", { name: "A closed issue is re-opened" })).toBeChecked();
    expect(screen.getByText("GitHub opens a closed issue again.")).toBeInTheDocument();
    expect(screen.getByRole("switch", { name: 'The "superplane" label is added to the issue' })).toBeChecked();
    expect(screen.getByText("A person adds that label to an open issue.")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Only intake issues with these labels" })).toBeInTheDocument();
    expect(screen.getByText("Leave empty to intake every issue.")).toBeInTheDocument();
    const nav = screen.getByRole("navigation", { name: "Intake settings" });
    expect(
      within(nav)
        .getAllByRole("button")
        .map((button) => button.textContent),
    ).toEqual(["Settings", "Canvas"]);
    expect(screen.getByTestId("intake-settings-tab-general")).toHaveAttribute("aria-current", "page");
    expect(screen.queryByTestId("intake-settings-tab-agent")).not.toBeInTheDocument();
    expect(screen.queryByRole("tab")).not.toBeInTheDocument();
    expect(screen.queryByTestId("intake-source-automation")).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Edit automation" })).not.toBeInTheDocument();

    expect(screen.getByRole("group", { name: "Filters" })).toBeInTheDocument();
    expect(screen.queryByRole("checkbox", { name: "Issue has one of these labels" })).not.toBeInTheDocument();
    expect(screen.queryByRole("checkbox", { name: "bug" })).not.toBeInTheDocument();
    expect(screen.queryByRole("listbox", { name: "Labels in the repository" })).not.toBeInTheDocument();
    expect(screen.getByRole("switch", { name: "Author is a repository collaborator" })).not.toBeChecked();
    expect(screen.getByText("The issue author has access to the repository.")).toBeInTheDocument();
  });

  it("offers repository labels after the user starts typing", async () => {
    const user = userEvent.setup();
    renderPopup({ labelOptions: ["bug", "needs design"] });

    await user.click(screen.getByTestId("intake-label-new"));
    expect(screen.queryByRole("listbox", { name: "Labels in the repository" })).not.toBeInTheDocument();

    await user.type(screen.getByRole("textbox", { name: "Issue label" }), "des");

    const suggestions = screen.getByRole("listbox", { name: "Labels in the repository" });
    expect(within(suggestions).getByRole("option", { name: "needs design" })).toBeInTheDocument();
    expect(within(suggestions).queryByRole("option", { name: "bug" })).not.toBeInTheDocument();

    await user.click(within(suggestions).getByRole("option", { name: "needs design" }));
    expect(screen.getByRole("button", { name: "Remove needs design" })).toBeInTheDocument();
    expect(screen.queryByRole("textbox", { name: "Issue label" })).not.toBeInTheDocument();
  });

  it("hides recommendations when the repository has no labels", async () => {
    const user = userEvent.setup();
    renderPopup({ labelOptions: [] });

    await user.click(screen.getByTestId("intake-label-new"));
    await user.type(screen.getByRole("textbox", { name: "Issue label" }), "bug");

    expect(screen.queryByRole("listbox", { name: "Labels in the repository" })).not.toBeInTheDocument();
  });

  it("keeps a label that the user types", async () => {
    const user = userEvent.setup();
    renderPopup();

    expect(screen.queryByRole("textbox", { name: "Issue label" })).not.toBeInTheDocument();

    await user.click(screen.getByTestId("intake-label-new"));
    expect(screen.getByTestId("intake-label-add")).toBeDisabled();

    await user.type(screen.getByRole("textbox", { name: "Issue label" }), "needs-triage");
    await user.click(screen.getByTestId("intake-label-add"));

    expect(screen.getByRole("button", { name: "Remove needs-triage" })).toBeInTheDocument();
    expect(screen.queryByRole("textbox", { name: "Issue label" })).not.toBeInTheDocument();
  });

  it("adds a label when the user presses Enter", async () => {
    const user = userEvent.setup();
    renderPopup();

    await user.click(screen.getByTestId("intake-label-new"));
    await user.type(screen.getByRole("textbox", { name: "Issue label" }), "needs-triage{Enter}");

    expect(screen.getByRole("button", { name: "Remove needs-triage" })).toBeInTheDocument();
  });

  it("hides the label input when the user cancels", async () => {
    const user = userEvent.setup();
    renderPopup();

    await user.click(screen.getByTestId("intake-label-new"));
    await user.type(screen.getByRole("textbox", { name: "Issue label" }), "needs-triage");
    await user.click(screen.getByTestId("intake-label-cancel"));

    expect(screen.queryByRole("textbox", { name: "Issue label" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Remove needs-triage" })).not.toBeInTheDocument();
  });

  it("shows the intake automation on the Automation tab", async () => {
    const user = userEvent.setup();
    renderPopup({ editAutomationHref: "/org-1/workspaces/RF/apps/app-github-issues-intake?configure=1&agent=1" });

    await user.click(screen.getByTestId("intake-settings-tab-automation"));

    const automation = screen.getByTestId("intake-source-automation");
    expect(automation).toHaveAccessibleName("Automation");
    expect(within(automation).getByTestId("rf__node-trigger")).toBeInTheDocument();
    expect(within(automation).getAllByText("On Issue").length).toBeGreaterThan(0);
    expect(within(automation).getByText("Matches filters?")).toBeInTheDocument();
    expect(within(automation).getAllByText("Create Task").length).toBeGreaterThan(0);
    expect(screen.queryByTestId("settings-automation-header-row")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Automation menu" })).not.toBeInTheDocument();
    const edit = within(automation).getByRole("link", { name: "Edit automation" });
    expect(edit).toHaveAttribute("href", "/org-1/workspaces/RF/apps/app-github-issues-intake?configure=1&agent=1");
    expect(document.querySelector(".sp-canvas-editing")).toBeNull();
    expect(within(automation).queryByRole("button", { name: /Add next component/ })).not.toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Accepted events go to Backlog" })).not.toBeInTheDocument();
    expect(screen.queryByLabelText("Name")).not.toBeInTheDocument();
    expect(
      within(screen.getByTestId("intake-settings-topbar")).queryByTestId("intake-source-settings-save"),
    ).not.toBeInTheDocument();
    expect(screen.queryByTestId("factory-automation-runs-sidebar")).not.toBeInTheDocument();
  });

  it("lists canvas runs beside the automation when a canvas id is given", async () => {
    const user = userEvent.setup();
    renderPopup({
      canvasId: "app-github-issues-intake",
      runHrefFor: (runId) => `/org-1/workspaces/RF/apps/app-github-issues-intake?run=${runId}`,
    });

    await user.click(screen.getByTestId("intake-settings-tab-automation"));

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

    const nav = screen.getByRole("navigation", { name: "Intake settings" });
    expect(
      within(nav)
        .getAllByRole("button")
        .map((button) => button.textContent),
    ).toEqual(["Settings", "Agent", "Canvas"]);

    await user.click(screen.getByTestId("intake-settings-tab-agent"));
    expect(screen.getByTestId("planning-review-editor")).toBeInTheDocument();
    expect(screen.getByTestId("planning-review-save")).toHaveTextContent("Save Agent");
    expect(screen.queryByTestId("planning-review-automation-note")).not.toBeInTheDocument();
    expect(
      within(screen.getByTestId("intake-settings-topbar")).queryByTestId("intake-source-settings-save"),
    ).not.toBeInTheDocument();
  });

  it("saves the edited configuration", async () => {
    const onSave = vi.fn();
    const onClose = vi.fn();
    const user = userEvent.setup();
    renderPopup({ onSave, onClose });

    await user.click(screen.getByTestId("intake-label-new"));
    await user.type(screen.getByRole("textbox", { name: "Issue label" }), "bug");
    await user.click(screen.getByRole("option", { name: "bug" }));
    await user.click(screen.getByRole("switch", { name: "Author is a repository collaborator" }));
    await user.click(screen.getByRole("switch", { name: "A closed issue is re-opened" }));
    await user.click(screen.getByTestId("intake-source-settings-save"));

    await waitFor(() =>
      expect(onSave).toHaveBeenCalledWith({
        ...DEFAULT_GITHUB_INTAKE_SETTINGS,
        labels: ["bug"],
        filterByLabel: true,
        reopenedIssues: false,
        authorsWithAccess: true,
      }),
    );
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("shows a save error beside the top bar Save button", () => {
    renderPopup({ saveError: INTAKE_SETTINGS_COPY.saveError });

    const topbar = screen.getByTestId("intake-settings-topbar");
    expect(within(topbar).getByRole("alert")).toHaveTextContent(INTAKE_SETTINGS_COPY.saveError);
    expect(within(topbar).getByTestId("intake-source-settings-save")).toBeInTheDocument();
  });

  it("shows Jira intake triggers and completion settings on the Settings page", async () => {
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

    const nav = screen.getByRole("navigation", { name: "Intake settings" });
    expect(
      within(nav)
        .getAllByRole("button")
        .map((button) => button.textContent),
    ).toEqual(["Settings", "Canvas"]);
    expect(
      within(screen.getByTestId("intake-settings-topbar")).getByRole("heading", {
        name: intakeSettingsTitle("jira-issues"),
      }),
    ).toBeInTheDocument();
    expect(screen.getByTestId("jira-intake-triggers")).toBeInTheDocument();
    expect(screen.getByTestId("jira-intake-factory")).toBeInTheDocument();
    expect(screen.getByTestId("intake-danger-zone")).toBeInTheDocument();
    expect(screen.getByTestId("jira-intake-new-issues")).toHaveAttribute("data-active", "true");
    expect(screen.getByTestId("jira-move-on-complete")).toHaveAttribute("data-active", "true");
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
});
