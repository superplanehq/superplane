import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { useConnectedIntegrations } from "@/hooks/useIntegrations";
import { organizationIntegrationsPath } from "@/lib/integrationSettingsPaths";
import { prepareData } from "@/pages/app/workflowPageHelpers";
import { TooltipProvider } from "@/ui/tooltip";

import { PRFeedbackSettingsPopup } from "./PRFeedbackSettingsPopup";
import type { PRFeedbackDraftSettings } from "./prFeedbackSettingsModel";
import type { IntakeAutomationGraph } from "./useIntakeAutomationCanvas";

vi.mock("@/hooks/useIntegrations", () => ({
  useConnectedIntegrations: vi.fn(() => ({ data: [] })),
}));

vi.mock("@/ui/componentSidebar/integrationIcons", () => ({
  IntegrationIcon: ({ integrationName }: { integrationName?: string }) => (
    <span data-testid={`integration-icon-${integrationName ?? "unknown"}`} />
  ),
}));

function mockConnectedIntegrations(data: unknown[] = []) {
  return {
    data,
    isLoading: false,
    error: null,
  } as unknown as ReturnType<typeof useConnectedIntegrations>;
}

function discussionDraft(overrides: Partial<PRFeedbackDraftSettings> = {}): PRFeedbackDraftSettings {
  return {
    source: "discussion",
    name: "Address PR feedback",
    repository: "acme/payments",
    mention: "@superplaneagent",
    ignoreBots: true,
    allowedBots: [],
    checkNames: [],
    maximumAttempts: 3,
    baseBranch: "main",
    runnerIntegrationIds: [],
    ...overrides,
  };
}

function checksDraft(overrides: Partial<PRFeedbackDraftSettings> = {}): PRFeedbackDraftSettings {
  return {
    source: "checks",
    name: "Fix pull request checks",
    repository: "acme/app",
    mention: "",
    ignoreBots: false,
    allowedBots: [],
    checkNames: [],
    maximumAttempts: 3,
    baseBranch: "main",
    runnerIntegrationIds: [],
    ...overrides,
  };
}

function conflictsDraft(overrides: Partial<PRFeedbackDraftSettings> = {}): PRFeedbackDraftSettings {
  return {
    source: "conflicts",
    name: "Resolve pull request conflicts",
    repository: "acme/app",
    mention: "",
    ignoreBots: false,
    allowedBots: [],
    checkNames: [],
    maximumAttempts: 3,
    baseBranch: "main",
    runnerIntegrationIds: [],
    ...overrides,
  };
}

beforeEach(() => {
  vi.mocked(useConnectedIntegrations).mockReturnValue(mockConnectedIntegrations());
});

function renderChecksPopup(
  onSave = vi.fn(),
  settings: PRFeedbackDraftSettings = checksDraft(),
  organizationId?: string,
) {
  render(
    <MemoryRouter>
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <PRFeedbackSettingsPopup
          organizationId={organizationId}
          settings={settings}
          healthy
          onSave={onSave}
          onClose={vi.fn()}
          fixed={false}
        />
      </QueryClientProvider>
    </MemoryRouter>,
  );
  return { onSave };
}

function renderConflictsPopup(
  onSave = vi.fn(),
  settings: PRFeedbackDraftSettings = conflictsDraft(),
  healthy = true,
) {
  render(
    <MemoryRouter>
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <PRFeedbackSettingsPopup settings={settings} healthy={healthy} onSave={onSave} onClose={vi.fn()} fixed={false} />
      </QueryClientProvider>
    </MemoryRouter>,
  );
  return { onSave };
}

const automationGraph = prFeedbackGraph();

function prFeedbackGraph(): IntakeAutomationGraph {
  const { nodes, edges } = prepareData(
    {
      metadata: { id: "app-pr-feedback", name: "Address PR feedback", factoryId: "factory-1" },
      spec: {
        nodes: [
          { id: "comment", name: "On PR Comment", type: "TYPE_TRIGGER", component: "github.onPRComment" },
          { id: "find", name: "Find Pull Request", type: "TYPE_ACTION", component: "findPullRequest" },
          {
            id: "runner",
            name: "Address PR feedback",
            type: "TYPE_ACTION",
            component: "runnerClaude",
          },
        ],
        edges: [
          { channel: "default", sourceId: "comment", targetId: "find" },
          { channel: "found", sourceId: "find", targetId: "runner" },
        ],
      },
    },
    [{ name: "github.onPRComment", label: "On PR Comment" }],
    [
      {
        name: "findPullRequest",
        label: "Find Pull Request",
        outputChannels: [{ name: "found" }, { name: "notFound" }],
      },
      { name: "runnerClaude", label: "Run Claude Code", outputChannels: [{ name: "passed" }, { name: "failed" }] },
    ],
    {},
    {},
    {},
    "app-pr-feedback",
    new QueryClient(),
    null,
    "live",
  );

  return { nodes, edges, factoryId: "factory-1" };
}

function renderAutomationPopup() {
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <ThemeProvider>
          <TooltipProvider>
            <PRFeedbackSettingsPopup
              settings={discussionDraft()}
              healthy
              automationGraph={automationGraph}
              onSave={vi.fn()}
              onClose={vi.fn()}
              editAutomationHref="/org-1/workspaces/RF/apps/app-pr-feedback?configure=1&agent=1"
              initialTab="automation"
              fixed={false}
            />
          </TooltipProvider>
        </ThemeProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("PRFeedbackSettingsPopup check names", () => {
  it("adds a check name that contains a comma as one value", async () => {
    const user = userEvent.setup();
    renderChecksPopup();

    const input = screen.getByTestId("pr-feedback-check-names");
    await user.type(input, "lint, typecheck");
    await user.keyboard("{Enter}");

    const names = screen.getByTestId("pr-feedback-check-names-list");
    expect(within(names).getAllByRole("listitem")).toHaveLength(1);
    expect(names).toHaveTextContent("lint, typecheck");
    expect(input).toHaveValue("");
  });

  it("adds a second name with Add and keeps the first comma-containing name", async () => {
    const user = userEvent.setup();
    renderChecksPopup();

    await user.type(screen.getByTestId("pr-feedback-check-names"), "lint, typecheck");
    await user.keyboard("{Enter}");
    await user.type(screen.getByTestId("pr-feedback-check-names"), "unit");
    await user.click(screen.getByTestId("pr-feedback-check-names-add"));

    const names = screen.getByTestId("pr-feedback-check-names-list");
    expect(within(names).getAllByRole("listitem")).toHaveLength(2);
    expect(names).toHaveTextContent("lint, typecheck");
    expect(names).toHaveTextContent("unit");
  });

  it("includes a pending name that contains a comma when the user saves", async () => {
    const user = userEvent.setup();
    const { onSave } = renderChecksPopup();

    await user.type(screen.getByTestId("pr-feedback-check-names"), "lint, typecheck");
    await user.click(screen.getByTestId("pr-feedback-settings-save"));

    expect(onSave).toHaveBeenCalledWith(
      expect.objectContaining({
        checkNames: ["lint, typecheck"],
      }),
    );
  });

  it("removes a selected check name", async () => {
    const user = userEvent.setup();
    renderChecksPopup(vi.fn(), checksDraft({ checkNames: ["lint, typecheck", "unit"] }));

    await user.click(screen.getByRole("button", { name: "Remove check lint, typecheck" }));

    const names = screen.getByTestId("pr-feedback-check-names-list");
    expect(within(names).getAllByRole("listitem")).toHaveLength(1);
    expect(names).toHaveTextContent("unit");
    expect(names).not.toHaveTextContent("lint, typecheck");
  });
});

describe("PRFeedbackSettingsPopup conflicts", () => {
  it("shows the base branch field and the conflict-specific health copy", () => {
    renderConflictsPopup();

    expect(screen.getByTestId("pr-feedback-base-branch")).toHaveValue("main");
    expect(screen.getByText("This automation can wait for merge conflicts and start a fix.")).toBeInTheDocument();
    expect(
      screen.getByText("SuperPlane pauses automatic conflict fixes after this many consecutive attempts."),
    ).toBeInTheDocument();
    expect(screen.queryByTestId("pr-feedback-mention")).not.toBeInTheDocument();
    expect(screen.queryByTestId("pr-feedback-check-names")).not.toBeInTheDocument();
  });

  it("saves the base branch and maximum attempts", async () => {
    const user = userEvent.setup();
    const { onSave } = renderConflictsPopup();

    const baseBranch = screen.getByTestId("pr-feedback-base-branch");
    await user.clear(baseBranch);
    await user.type(baseBranch, "develop");
    await user.click(screen.getByTestId("pr-feedback-settings-save"));

    expect(onSave).toHaveBeenCalledWith(
      expect.objectContaining({
        source: "conflicts",
        baseBranch: "develop",
        maximumAttempts: 3,
      }),
    );
  });

  it("disables save when the base branch is empty", async () => {
    const user = userEvent.setup();
    renderConflictsPopup();

    const baseBranch = screen.getByTestId("pr-feedback-base-branch");
    await user.clear(baseBranch);

    expect(screen.getByTestId("pr-feedback-settings-save")).toBeDisabled();
  });
});

describe("PRFeedbackSettingsPopup additional integrations", () => {
  beforeEach(() => {
    vi.mocked(useConnectedIntegrations).mockReturnValue(
      mockConnectedIntegrations([
        {
          metadata: { id: "int-circleci", name: "circleci-prod", integrationName: "circleci" },
          status: { state: "ready" },
        },
      ]),
    );
  });

  it("shows the integration icon next to the integration name", () => {
    renderChecksPopup(vi.fn(), checksDraft(), "org-1");

    const row = screen.getByTestId("pr-feedback-integrations");
    expect(within(row).getByTestId("integration-icon-circleci")).toBeInTheDocument();
    expect(row).toHaveTextContent("circleci-prod");
    expect(within(row).getByRole("listitem").className).toContain("items-center");
  });

  it("groups integrations of the same type next to each other", () => {
    vi.mocked(useConnectedIntegrations).mockReturnValue(
      mockConnectedIntegrations([
        { metadata: { id: "int-slack-2", name: "slack-eng", integrationName: "slack" }, status: { state: "ready" } },
        {
          metadata: { id: "int-circleci-2", name: "circleci-staging", integrationName: "circleci" },
          status: { state: "ready" },
        },
        { metadata: { id: "int-slack-1", name: "slack-alerts", integrationName: "slack" }, status: { state: "ready" } },
        {
          metadata: { id: "int-circleci-1", name: "circleci-prod", integrationName: "circleci" },
          status: { state: "ready" },
        },
      ]),
    );

    renderChecksPopup(vi.fn(), checksDraft(), "org-1");

    const list = screen.getByTestId("pr-feedback-integrations");
    const rows = within(list).getAllByRole("listitem");
    expect(rows.map((row) => row.textContent)).toEqual([
      "circleci-prod",
      "circleci-staging",
      "slack-alerts",
      "slack-eng",
    ]);
  });

  it("links to the organization Integrations page", () => {
    renderChecksPopup(vi.fn(), checksDraft(), "org-1");

    expect(
      screen.getByText(/Give the agent access to CI logs from other connected integrations/, { exact: false }),
    ).toHaveTextContent(
      "Give the agent access to CI logs from other connected integrations. If this list does not include the integration you need, go to the Integrations page and connect it.",
    );
    const link = screen.getByRole("link", { name: "Integrations page" });
    expect(link).toHaveAttribute("href", organizationIntegrationsPath("org-1"));
    expect(link).toHaveAttribute("target", "_blank");
    expect(link).toHaveAttribute("rel", "noreferrer");
  });

  it("hides the Integrations link when the organization id is missing", () => {
    renderChecksPopup();

    expect(screen.queryByTestId("pr-feedback-integrations-page")).not.toBeInTheDocument();
  });

  it("keeps the Integrations link when no extra integrations are connected", () => {
    vi.mocked(useConnectedIntegrations).mockReturnValue(mockConnectedIntegrations());
    renderChecksPopup(vi.fn(), checksDraft(), "org-1");

    expect(screen.getByTestId("pr-feedback-integrations-empty")).toBeInTheDocument();
    expect(screen.getByTestId("pr-feedback-integrations-page")).toHaveAttribute(
      "href",
      organizationIntegrationsPath("org-1"),
    );
  });
});

describe("PRFeedbackSettingsPopup automation", () => {
  it("shows the automation in display mode at native zoom", () => {
    renderAutomationPopup();

    const automation = screen.getByTestId("pr-feedback-automation");
    expect(automation).toHaveAccessibleName("Automation");
    expect(within(automation).getAllByText("Find Pull Request").length).toBeGreaterThan(0);
    expect(within(automation).getByRole("link", { name: "Edit automation" })).toHaveAttribute(
      "href",
      "/org-1/workspaces/RF/apps/app-pr-feedback?configure=1&agent=1",
    );
    expect(document.querySelector(".sp-canvas-editing")).toBeNull();
    expect(within(automation).queryByRole("button", { name: /Add next component/ })).not.toBeInTheDocument();
    expect(within(automation).queryByText("passed")).not.toBeInTheDocument();
    expect(within(automation).queryByText("failed")).not.toBeInTheDocument();
    expect(within(automation).queryByText("notFound")).not.toBeInTheDocument();
  });
});
