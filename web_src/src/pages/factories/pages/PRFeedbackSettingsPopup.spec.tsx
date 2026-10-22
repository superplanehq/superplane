import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, useLocation } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import type * as CanvasDataModule from "@/hooks/useCanvasData";
import type * as IntegrationsModule from "@/hooks/useIntegrations";
import { useConnectedIntegrations, useIntegrationResources } from "@/hooks/useIntegrations";
import { unmockedSrc } from "@/test/unmockedModule";
import { organizationIntegrationsPath } from "@/lib/integrationSettingsPaths";
import { prepareData } from "@/pages/app/workflowPageHelpers";
import { TooltipProvider } from "@/ui/tooltip";

import { PRFeedbackSettingsPopup } from "./PRFeedbackSettingsPopup";
import { PLANNING_REVIEW_DRAFT } from "./planningReviewMockup";
import type { PRFeedbackDraftSettings } from "./prFeedbackSettingsModel";
import type { IntakeAutomationGraph } from "./useIntakeAutomationCanvas";

Element.prototype.scrollIntoView ??= () => undefined;

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

vi.mock("@/hooks/useIntegrations", () => {
  const actual = unmockedSrc<typeof IntegrationsModule>("hooks/useIntegrations");
  return {
    ...actual,
    useConnectedIntegrations: vi.fn(() => ({ data: [] })),
    useIntegrationResources: vi.fn(() => ({
      data: [],
      isLoading: false,
      isPending: false,
      isFetching: false,
      isError: false,
    })),
  };
});

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

function mockIntegrationResources(data: unknown[] = [], overrides: Record<string, unknown> = {}) {
  return {
    data,
    isLoading: false,
    isPending: false,
    isFetching: false,
    isError: false,
    ...overrides,
  } as unknown as ReturnType<typeof useIntegrationResources>;
}

function discussionDraft(overrides: Partial<PRFeedbackDraftSettings> = {}): PRFeedbackDraftSettings {
  return {
    source: "discussion",
    name: "Address PR feedback",
    repository: "acme/payments",
    mention: "@superplaneagent",
    ignoreBots: true,
    addressAllBots: false,
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
    addressAllBots: false,
    allowedBots: [],
    checkNames: [],
    maximumAttempts: 3,
    runnerIntegrationIds: [],
    ...overrides,
  };
}

useInfiniteCanvasRuns.mockReturnValue({
  data: {
    pages: [
      {
        runs: [
          {
            id: "run-pr-1",
            canvasId: "app-pr-feedback",
            state: "STATE_FINISHED",
            result: "RESULT_PASSED",
            createdAt: "2026-05-01T12:00:00Z",
            rootEvent: { nodeId: "comment", customName: "Please fix the lint error" },
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

beforeEach(() => {
  vi.mocked(useConnectedIntegrations).mockReturnValue(mockConnectedIntegrations());
  vi.mocked(useIntegrationResources).mockReturnValue(mockIntegrationResources());
});

afterEach(() => {
  localStorage.clear();
});

function renderChecksPopup(
  onSave = vi.fn(),
  settings: PRFeedbackDraftSettings = checksDraft(),
  organizationId?: string,
  githubIntegrationId?: string,
  vcsProvider?: string,
) {
  render(
    <MemoryRouter>
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <PRFeedbackSettingsPopup
          organizationId={organizationId}
          githubIntegrationId={githubIntegrationId}
          vcsProvider={vcsProvider}
          settings={settings}
          onSave={onSave}
          onDelete={vi.fn()}
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
  const { nodes, edges } = prepareData({
    workflow: {
      metadata: { id: "app-pr-feedback", name: "Address PR feedback", factoryId: "factory-1" },
      spec: {
        nodes: [
          { id: "comment", name: "On PR Comment", type: "TYPE_TRIGGER", component: "github.onPRComment" },
          { id: "find", name: "Find Pull Request", type: "TYPE_ACTION", component: "findPullRequest" },
          { id: "runner", name: "Address PR feedback", type: "TYPE_ACTION", component: "runnerClaude" },
        ],
        edges: [
          { channel: "default", sourceId: "comment", targetId: "find" },
          { channel: "found", sourceId: "find", targetId: "runner" },
        ],
      },
    },
    triggers: [{ name: "github.onPRComment", label: "On PR Comment" }],
    components: [
      {
        name: "findPullRequest",
        label: "Find Pull Request",
        outputChannels: [{ name: "found" }, { name: "notFound" }],
      },
      { name: "runnerClaude", label: "Run Claude Code", outputChannels: [{ name: "passed" }, { name: "failed" }] },
    ],
    nodeEventsMap: {},
    nodeExecutionsMap: {},
    nodeQueueItemsMap: {},
    workflowId: "app-pr-feedback",
    queryClient: new QueryClient(),
    user: null,
    canvasMode: "live",
  });

  return {
    nodes,
    edges,
    factoryId: "factory-1",
    specNodes: [{ id: "comment", name: "On PR Comment", type: "TYPE_TRIGGER", component: "github.onPRComment" }],
  };
}

function renderAutomationPopup(props: { canvasId?: string } = {}) {
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <ThemeProvider>
          <TooltipProvider>
            <PRFeedbackSettingsPopup
              settings={discussionDraft()}
              automationGraph={automationGraph}
              onSave={vi.fn()}
              onClose={vi.fn()}
              canvasId={props.canvasId}
              initialTab="automation"
              fixed={false}
            />
          </TooltipProvider>
        </ThemeProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

function SearchProbe() {
  const location = useLocation();
  return <output data-testid="location">{location.search}</output>;
}

function renderAgentSettingsPopup(
  settings: PRFeedbackDraftSettings,
  showVisualEvidenceSetting: boolean,
  initialPath = "/",
) {
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter initialEntries={[initialPath]}>
        <PRFeedbackSettingsPopup
          settings={settings}
          automationGraph={automationGraph}
          agent={{
            draft: PLANNING_REVIEW_DRAFT,
            organizationId: "org-1",
            onSave: vi.fn(),
            showVisualEvidenceSetting,
          }}
          onSave={vi.fn()}
          onClose={vi.fn()}
          initialTab="agent"
          fixed={false}
        />
        <SearchProbe />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("PRFeedbackSettingsPopup general tab", () => {
  it.each([
    ["discussion", discussionDraft()],
    ["checks", checksDraft()],
  ] as const)("starts %s configuration without a Health block", (_source, settings) => {
    renderChecksPopup(vi.fn(), settings, "org-1", "gh-1");

    expect(screen.queryByRole("heading", { name: "Health" })).not.toBeInTheDocument();
    expect(screen.queryByText("Ready")).not.toBeInTheDocument();
    expect(screen.queryByText("Needs repair")).not.toBeInTheDocument();
    expect(screen.queryByText("This automation can receive a mention and address it.")).not.toBeInTheDocument();
    expect(screen.queryByText("This automation can wait for checks and start a fix.")).not.toBeInTheDocument();
    expect(screen.queryByText("Open the Automation tab and repair the steps.")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Delete automation" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Save" })).toBeInTheDocument();

    if (settings.source === "discussion") {
      expect(screen.getByRole("radiogroup", { name: "How to handle human comments?" })).toBeInTheDocument();
      expect(
        screen.getByRole("radiogroup", { name: "Which bot comments should SuperPlane address?" }),
      ).toBeInTheDocument();
      return;
    }

    expect(screen.getByTestId("pr-feedback-check-names-picker")).toBeInTheDocument();
    expect(screen.getByLabelText("Maximum automatic fix attempts")).toBeInTheDocument();
    expect(screen.getByRole("listbox", { name: "Additional integration access" })).toBeInTheDocument();
  });
});

describe("PRFeedbackSettingsPopup discussion", () => {
  beforeEach(() => {
    vi.mocked(useIntegrationResources).mockReturnValue(
      mockIntegrationResources([{ type: "review_bot", id: "coderabbitai", name: "coderabbitai[bot]" }]),
    );
  });

  it("uses the same mention and bot choices as setup", () => {
    renderChecksPopup(vi.fn(), discussionDraft({ mention: "", allowedBots: ["coderabbitai"] }), "org-1", "gh-1");

    expect(screen.queryByTestId("pr-feedback-name")).not.toBeInTheDocument();
    expect(screen.queryByTestId("pr-feedback-repository")).not.toBeInTheDocument();
    expect(screen.queryByTestId("pr-feedback-mention")).not.toBeInTheDocument();
    expect(screen.getByRole("radio", { name: /Start from any human comment/ })).toBeChecked();
    expect(screen.getByRole("radio", { name: /Require @superplaneagent/ })).not.toBeChecked();
    expect(screen.getByRole("radio", { name: /Choose bots/ })).toBeChecked();
    expect(screen.getByTestId("discussion-setup-bot-coderabbitai")).toHaveAttribute("aria-selected", "true");
  });

  it("switches mention and bot modes without free-text fields", async () => {
    const user = userEvent.setup();
    const { onSave } = renderChecksPopup(vi.fn(), discussionDraft(), "org-1", "gh-1");

    expect(screen.getByRole("radio", { name: /Require @superplaneagent/ })).toBeChecked();
    expect(screen.getByRole("radio", { name: /Ignore bots/ })).toBeChecked();
    expect(screen.queryByTestId("discussion-setup-bots-list")).not.toBeInTheDocument();

    await user.click(screen.getByRole("radio", { name: /Start from any human comment/ }));
    await user.click(screen.getByRole("radio", { name: /Choose bots/ }));
    expect(screen.getByTestId("discussion-setup-bot-coderabbitai")).toHaveAttribute("aria-selected", "false");
    await user.click(screen.getByTestId("discussion-setup-bot-coderabbitai"));
    await user.click(screen.getByTestId("pr-feedback-settings-save"));

    expect(onSave).toHaveBeenCalledWith(
      expect.objectContaining({
        mention: "",
        ignoreBots: true,
        allowedBots: ["coderabbitai"],
      }),
    );
  });

  it("shows a bot name field for Bitbucket instead of an empty catalog", async () => {
    const user = userEvent.setup();
    renderChecksPopup(vi.fn(), discussionDraft(), "org-1", "gh-1", "bitbucket");
    await user.click(screen.getByRole("radio", { name: /Choose bots/ }));
    expect(screen.queryByTestId("discussion-setup-bots-empty")).not.toBeInTheDocument();
    expect(screen.queryByTestId("discussion-setup-bots-picker")).not.toBeInTheDocument();
    expect(screen.queryByTestId("discussion-setup-bot-coderabbitai")).not.toBeInTheDocument();
    expect(screen.getByLabelText("Bot name")).toBeInTheDocument();
  });
});

describe("PRFeedbackSettingsPopup check names", () => {
  it("does not offer name or repository fields because those come from setup", () => {
    renderChecksPopup(vi.fn(), checksDraft({ checkNames: ["lint"] }), "org-1", "gh-1");

    expect(screen.queryByTestId("pr-feedback-name")).not.toBeInTheDocument();
    expect(screen.queryByTestId("pr-feedback-repository")).not.toBeInTheDocument();
    expect(screen.getByTestId("pr-feedback-check-names-picker")).toBeInTheDocument();
  });

  it("shows configured checks while other catalog checks are loading", () => {
    vi.mocked(useIntegrationResources).mockReturnValue(
      mockIntegrationResources([], { isLoading: true, isPending: true, isFetching: true }),
    );
    renderChecksPopup(vi.fn(), checksDraft({ checkNames: ["lint", "e2e"] }), "org-1", "gh-1");

    expect(screen.getByTestId("pr-feedback-check-option-lint")).toHaveAttribute("aria-selected", "true");
    expect(screen.getByTestId("pr-feedback-check-option-e2e")).toHaveAttribute("aria-selected", "true");
    expect(screen.getByTestId("pr-feedback-check-names-loading")).toHaveTextContent("Loading other status checks");
  });

  it("selects a check from the repository catalog", async () => {
    vi.mocked(useIntegrationResources).mockReturnValue(
      mockIntegrationResources([
        { type: "status_check", id: "lint", name: "lint" },
        { type: "status_check", id: "e2e", name: "e2e" },
      ]),
    );
    const user = userEvent.setup();
    renderChecksPopup(vi.fn(), checksDraft(), "org-1", "gh-1");

    const e2e = screen.getByTestId("pr-feedback-check-option-e2e");
    expect(e2e).toHaveAttribute("aria-selected", "false");
    await user.click(e2e);

    expect(screen.getByTestId("pr-feedback-check-option-e2e")).toHaveAttribute("aria-selected", "true");
    expect(screen.getByTestId("pr-feedback-check-option-lint")).toHaveAttribute("aria-selected", "false");
  });

  it("deselects a selected check from the catalog list", async () => {
    vi.mocked(useIntegrationResources).mockReturnValue(
      mockIntegrationResources([
        { type: "status_check", id: "lint", name: "lint" },
        { type: "status_check", id: "unit", name: "unit" },
      ]),
    );
    const user = userEvent.setup();
    renderChecksPopup(vi.fn(), checksDraft({ checkNames: ["lint", "unit"] }), "org-1", "gh-1");

    await user.click(screen.getByTestId("pr-feedback-check-option-lint"));

    expect(screen.getByTestId("pr-feedback-check-option-lint")).toHaveAttribute("aria-selected", "false");
    expect(screen.getByTestId("pr-feedback-check-option-unit")).toHaveAttribute("aria-selected", "true");
  });

  it("saves only the selected catalog checks", async () => {
    vi.mocked(useIntegrationResources).mockReturnValue(
      mockIntegrationResources([
        { type: "status_check", id: "lint", name: "lint" },
        { type: "status_check", id: "e2e", name: "e2e" },
      ]),
    );
    const user = userEvent.setup();
    const { onSave } = renderChecksPopup(vi.fn(), checksDraft(), "org-1", "gh-1");

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
    const { onSave } = renderChecksPopup(vi.fn(), checksDraft({ checkNames: ["lint"] }), "org-1", "gh-1");

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

    expect(screen.getAllByRole("tab").map((tab) => tab.textContent)).toEqual(["General", "Runs"]);
    expect(screen.queryByTestId("pr-feedback-settings-tab-agent")).not.toBeInTheDocument();
  });

  it("puts Agent between General and Runs when the canvas has an agent", () => {
    renderAgentSettingsPopup(discussionDraft(), true);

    expect(screen.getAllByRole("tab").map((tab) => tab.textContent)).toEqual(["General", "Agent", "Runs"]);

    expect(screen.getByTestId("planning-review-editor")).toBeInTheDocument();
    expect(screen.getByTestId("planning-review-save")).toHaveTextContent("Save Agent");
    expect(screen.getByRole("switch", { name: "Include visual evidence" })).not.toBeChecked();
    expect(screen.queryByRole("button", { name: "Cancel" })).not.toBeInTheDocument();
  });

  it("opens a checks Agent link on General and hides the Agent tab", () => {
    renderAgentSettingsPopup(checksDraft(), false, "/?prFeedbackSettings=agent");

    expect(screen.getAllByRole("tab").map((tab) => tab.textContent)).toEqual(["General", "Runs"]);
    expect(screen.queryByTestId("pr-feedback-settings-tab-agent")).not.toBeInTheDocument();
    expect(screen.getByTestId("pr-feedback-settings-tab-general")).toHaveAttribute("aria-selected", "true");
    expect(screen.queryByTestId("planning-review-editor")).not.toBeInTheDocument();
    expect(screen.getByTestId("location")).toHaveTextContent("prFeedbackSettings=agent");
  });
});
