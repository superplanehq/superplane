import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { useFactoryRepositoryStatusChecks } from "@/hooks/useFactoryPRFeedbackData";
import { useConnectedIntegrations } from "@/hooks/useIntegrations";
import { organizationIntegrationsPath } from "@/lib/integrationSettingsPaths";
import { prepareData } from "@/pages/app/workflowPageHelpers";
import { TooltipProvider } from "@/ui/tooltip";

import { PRFeedbackSettingsPopup } from "./PRFeedbackSettingsPopup";
import { PLANNING_REVIEW_DRAFT } from "./planningReviewMockup";
import type { PRFeedbackDraftSettings } from "./prFeedbackSettingsModel";
import type { IntakeAutomationGraph } from "./useIntakeAutomationCanvas";
import type { PlanningReviewAgentSlot } from "./PlanningReviewEditor";

Element.prototype.scrollIntoView ??= () => undefined;

vi.mock("@monaco-editor/react", () => ({
  Editor: ({ value, onChange }: { value?: string; onChange?: (value: string | undefined) => void }) => (
    <textarea value={value ?? ""} onChange={(event) => onChange?.(event.target.value)} />
  ),
}));

vi.mock("@/hooks/useIntegrations", () => ({
  useConnectedIntegrations: vi.fn(() => ({ data: [] })),
}));

vi.mock("@/hooks/useFactoryPRFeedbackData", () => ({
  useFactoryRepositoryStatusChecks: vi.fn(() => ({
    data: [],
    isLoading: false,
    isPending: false,
    isFetching: false,
    isError: false,
  })),
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
    runnerIntegrationIds: [],
    ...overrides,
  };
}

beforeEach(() => {
  vi.mocked(useConnectedIntegrations).mockReturnValue(mockConnectedIntegrations());
  vi.mocked(useFactoryRepositoryStatusChecks).mockReturnValue({
    data: [],
    isLoading: false,
    isPending: false,
    isFetching: false,
    isError: false,
  } as ReturnType<typeof useFactoryRepositoryStatusChecks>);
});

function renderChecksPopup(
  onSave = vi.fn(),
  settings: PRFeedbackDraftSettings = checksDraft(),
  organizationId?: string,
  factoryId?: string,
) {
  render(
    <MemoryRouter>
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <PRFeedbackSettingsPopup
          organizationId={organizationId}
          factoryId={factoryId}
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

describe("PRFeedbackSettingsPopup discussion", () => {
  it("does not offer name or repository fields and keeps mention and allowed bots", () => {
    renderChecksPopup(vi.fn(), discussionDraft({ mention: "", allowedBots: ["coderabbitai"] }));

    expect(screen.queryByTestId("pr-feedback-name")).not.toBeInTheDocument();
    expect(screen.queryByTestId("pr-feedback-repository")).not.toBeInTheDocument();
    expect(screen.getByTestId("pr-feedback-mention")).toHaveValue("");
    expect(screen.getByTestId("pr-feedback-allowed-bots")).toHaveValue("coderabbitai");
  });
});

describe("PRFeedbackSettingsPopup check names", () => {
  it("does not offer name or repository fields because those come from setup", () => {
    renderChecksPopup(vi.fn(), checksDraft({ checkNames: ["lint"] }), "org-1", "factory-1");

    expect(screen.queryByTestId("pr-feedback-name")).not.toBeInTheDocument();
    expect(screen.queryByTestId("pr-feedback-repository")).not.toBeInTheDocument();
    expect(screen.getByTestId("pr-feedback-check-names-picker")).toBeInTheDocument();
  });

  it("shows configured checks while other catalog checks are loading", () => {
    vi.mocked(useFactoryRepositoryStatusChecks).mockReturnValue({
      data: [],
      isLoading: true,
      isPending: true,
      isFetching: true,
      isError: false,
    } as ReturnType<typeof useFactoryRepositoryStatusChecks>);
    renderChecksPopup(vi.fn(), checksDraft({ checkNames: ["lint", "e2e"] }), "org-1", "factory-1");

    expect(screen.getByTestId("pr-feedback-check-option-lint")).toHaveAttribute("aria-selected", "true");
    expect(screen.getByTestId("pr-feedback-check-option-e2e")).toHaveAttribute("aria-selected", "true");
    expect(screen.getByTestId("pr-feedback-check-names-loading")).toHaveTextContent("Loading other status checks");
  });

  it("selects a check from the repository catalog", async () => {
    vi.mocked(useFactoryRepositoryStatusChecks).mockReturnValue({
      data: [
        { name: "lint", required: true },
        { name: "e2e", required: false },
      ],
      isLoading: false,
      isPending: false,
      isFetching: false,
      isError: false,
    } as ReturnType<typeof useFactoryRepositoryStatusChecks>);
    const user = userEvent.setup();
    renderChecksPopup(vi.fn(), checksDraft(), "org-1", "factory-1");

    const e2e = screen.getByTestId("pr-feedback-check-option-e2e");
    expect(e2e).toHaveAttribute("aria-selected", "false");
    await user.click(e2e);

    expect(screen.getByTestId("pr-feedback-check-option-e2e")).toHaveAttribute("aria-selected", "true");
    expect(screen.getByTestId("pr-feedback-check-option-lint")).toHaveAttribute("aria-selected", "false");
  });

  it("deselects a selected check from the catalog list", async () => {
    vi.mocked(useFactoryRepositoryStatusChecks).mockReturnValue({
      data: [
        { name: "lint", required: true },
        { name: "unit", required: false },
      ],
      isLoading: false,
      isPending: false,
      isFetching: false,
      isError: false,
    } as ReturnType<typeof useFactoryRepositoryStatusChecks>);
    const user = userEvent.setup();
    renderChecksPopup(vi.fn(), checksDraft({ checkNames: ["lint", "unit"] }), "org-1", "factory-1");

    await user.click(screen.getByTestId("pr-feedback-check-option-lint"));

    expect(screen.getByTestId("pr-feedback-check-option-lint")).toHaveAttribute("aria-selected", "false");
    expect(screen.getByTestId("pr-feedback-check-option-unit")).toHaveAttribute("aria-selected", "true");
  });

  it("saves only the selected catalog checks", async () => {
    vi.mocked(useFactoryRepositoryStatusChecks).mockReturnValue({
      data: [
        { name: "lint", required: true },
        { name: "e2e", required: false },
      ],
      isLoading: false,
      isPending: false,
      isFetching: false,
      isError: false,
    } as ReturnType<typeof useFactoryRepositoryStatusChecks>);
    const user = userEvent.setup();
    const { onSave } = renderChecksPopup(vi.fn(), checksDraft(), "org-1", "factory-1");

    await user.click(screen.getByTestId("pr-feedback-check-option-e2e"));
    await user.click(screen.getByTestId("pr-feedback-settings-save"));

    expect(onSave).toHaveBeenCalledWith(
      expect.objectContaining({
        checkNames: ["e2e"],
      }),
    );
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

    const row = screen.getByTestId("pr-feedback-integration-int-circleci");
    expect(row).toHaveAttribute("role", "option");
    expect(row).toHaveAttribute("aria-selected", "false");
    expect(within(row).getByTestId("integration-icon-circleci")).toBeInTheDocument();
    expect(row).toHaveTextContent("circleci-prod");
    expect(screen.queryByRole("checkbox")).not.toBeInTheDocument();
  });

  it("selects an integration with the same picker as status checks", async () => {
    const user = userEvent.setup();
    const { onSave } = renderChecksPopup(vi.fn(), checksDraft({ checkNames: ["lint"] }), "org-1", "factory-1");

    const row = screen.getByTestId("pr-feedback-integration-int-circleci");
    expect(row).toHaveAttribute("aria-selected", "false");
    await user.click(row);
    expect(row).toHaveAttribute("aria-selected", "true");

    await user.click(screen.getByTestId("pr-feedback-settings-save"));
    expect(onSave).toHaveBeenCalledWith(expect.objectContaining({ runnerIntegrationIds: ["int-circleci"] }));
  });

  it("groups integrations of the same type next to each other", () => {
    vi.mocked(useConnectedIntegrations).mockReturnValue(
      mockConnectedIntegrations([
        { metadata: { id: "int-slack-2", name: "slack-eng", integrationName: "slack" }, status: { state: "ready" } },
        {
          metadata: { id: "int-circleci-2", name: "circleci-staging", integrationName: "circleci" },
          status: { state: "ready" },
        },
        {
          metadata: { id: "int-semaphore-1", name: "semaphore-prod", integrationName: "semaphore" },
          status: { state: "ready" },
        },
        {
          metadata: { id: "int-circleci-1", name: "circleci-prod", integrationName: "circleci" },
          status: { state: "ready" },
        },
      ]),
    );

    renderChecksPopup(vi.fn(), checksDraft(), "org-1");

    const list = screen.getByTestId("pr-feedback-integrations");
    const rows = within(list).getAllByRole("listitem");
    expect(rows.map((row) => row.textContent)).toEqual(["circleci-prod", "circleci-staging", "semaphore-prod"]);
  });

  it("hides connected integrations that are not common status-check tools", () => {
    vi.mocked(useConnectedIntegrations).mockReturnValue(
      mockConnectedIntegrations([
        { metadata: { id: "int-slack-1", name: "slack-alerts", integrationName: "slack" }, status: { state: "ready" } },
        {
          metadata: { id: "int-cloudflare", name: "cloudflare-prod", integrationName: "cloudflare" },
          status: { state: "ready" },
        },
      ]),
    );

    renderChecksPopup(vi.fn(), checksDraft(), "org-1");

    const list = screen.getByTestId("pr-feedback-integrations");
    expect(list).toHaveTextContent("cloudflare-prod");
    expect(list).not.toHaveTextContent("slack-alerts");
  });

  it("links to the organization Integrations page", () => {
    renderChecksPopup(vi.fn(), checksDraft(), "org-1");

    expect(
      screen.getByText(
        /Give the agent access to CI logs from Semaphore, CircleCI, Harness, Cloudflare, or Cloudsmith/,
        {
          exact: false,
        },
      ),
    ).toHaveTextContent(
      "Give the agent access to CI logs from Semaphore, CircleCI, Harness, Cloudflare, or Cloudsmith. If this list does not include the integration you need, go to the Integrations page and connect it.",
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
  it("hides the Agent tab when the canvas has no agent", () => {
    renderAutomationPopup();

    expect(screen.getAllByRole("tab").map((tab) => tab.textContent)).toEqual(["General", "Automation"]);
    expect(screen.queryByTestId("pr-feedback-settings-tab-agent")).not.toBeInTheDocument();
  });

  it("puts Agent between General and Automation when the canvas has an agent", async () => {
    const user = userEvent.setup();
    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <MemoryRouter>
          <ThemeProvider>
            <TooltipProvider>
              <PRFeedbackSettingsPopup
                settings={discussionDraft()}
                healthy
                automationGraph={automationGraph}
                agent={
                  {
                    draft: PLANNING_REVIEW_DRAFT,
                    organizationId: "org-1",
                    onSave: vi.fn(),
                  } satisfies PlanningReviewAgentSlot
                }
                onSave={vi.fn()}
                onClose={vi.fn()}
                initialTab="general"
                fixed={false}
              />
            </TooltipProvider>
          </ThemeProvider>
        </MemoryRouter>
      </QueryClientProvider>,
    );

    expect(screen.getAllByRole("tab").map((tab) => tab.textContent)).toEqual(["General", "Agent", "Automation"]);

    await user.click(screen.getByTestId("pr-feedback-settings-tab-agent"));
    expect(screen.getByTestId("planning-review-editor")).toBeInTheDocument();
    expect(screen.getByTestId("planning-review-save")).toHaveTextContent("Save Agent");
    expect(screen.queryByRole("button", { name: "Cancel" })).not.toBeInTheDocument();
  });

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
