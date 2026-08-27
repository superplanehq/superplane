import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { FactoriesFactory, FactoriesFactoryIntake, FactoriesWorkOrder, FactoryApp } from "@/api-client";
import { editFactoryLinePath, factoryAppConfigurePath } from "../lib/factoryPagePaths";
import {
  ACME_ONBOARDING_FACTORY,
  ACME_ONBOARDING_FACTORY_KEY,
  ACME_ONBOARDING_LINE_ID,
  GITHUB_ISSUES_INTAKE,
  GITHUB_ISSUES_INTAKE_APP,
  GITHUB_ISSUES_INTAKE_ID,
  PRIMARY_FACTORY_KEY,
  REFUND_FACTORY,
  REFUND_LINE_PLAN_ID,
} from "../__fixtures__/factoryPageResponses";
import { BOARD_DONE_REJECTED_ORDER, BOARD_IMPLEMENT_FAILED_ORDER } from "../__fixtures__/lineMetricsBoardOrders";
import type { FactoryPreviewFlags } from "./factoryPreviewFlagsContext";
import { LinesBoardSpecHarness } from "./linesPageSpecRender";
import { REVIEW_CANDIDATE_WORK_ORDERS } from "./onboarding/first-run/reviewCandidates";

function renderLinesBoard(
  path = `/org-1/workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_PLAN_ID}`,
  openCreateWorkOrder = vi.fn(),
  factory: FactoriesFactory = REFUND_FACTORY,
  previewFlags: FactoryPreviewFlags | null = null,
) {
  return render(
    <LinesBoardSpecHarness
      path={path}
      openCreateWorkOrder={openCreateWorkOrder}
      factory={factory}
      previewFlags={previewFlags}
    />,
  );
}

const createFactoryLineMutateAsync = vi.fn();
const updateFactoryLineMutateAsync = vi.fn();
const useFactoryWorkOrders = vi.fn(() => ({ data: [] as FactoriesWorkOrder[] }));
const useFactoryApps = vi.fn(() => ({ data: [] as FactoryApp[] }));
const useFactoryIntakes = vi.fn(() => ({ data: [] as FactoriesFactoryIntake[] }));
const createFactoryIntakeMutateAsync = vi.fn();
const searchFactoryIntakeItems = vi.fn(() => ({
  data: [] as { id: string; key: string; title: string; body: string; url: string }[],
  isLoading: false,
  isError: false,
}));
const importFactoryIntakeItem = vi.fn();

const SENTRY_INTAKE_ID = "intake-sentry";
const PAGERDUTY_INTAKE_ID = "intake-pagerduty";

const CONFIGURED_INTAKES: FactoriesFactoryIntake[] = [
  GITHUB_ISSUES_INTAKE,
  {
    id: SENTRY_INTAKE_ID,
    canvasId: "app-sentry-intake",
    name: "Sentry exceptions",
    source: "SOURCE_SENTRY_EXCEPTIONS",
    healthy: true,
  },
  {
    id: PAGERDUTY_INTAKE_ID,
    canvasId: "app-pagerduty-intake",
    name: "PagerDuty incidents",
    source: "SOURCE_PAGERDUTY_INCIDENTS",
    healthy: true,
  },
];

vi.mock("@/hooks/useFactoryData", () => ({
  useFactoryWorkOrders: () => useFactoryWorkOrders(),
  useFactoryApps: () => useFactoryApps(),
  useCreateFactoryLine: () => ({ mutateAsync: createFactoryLineMutateAsync, isPending: false }),
  useUpdateFactoryLine: () => ({ mutateAsync: updateFactoryLineMutateAsync, isPending: false }),
  useWorkOrderEvents: () => ({ data: { pages: [] } }),
  useWorkOrderArtifacts: () => ({ data: [] }),
  useCloseWorkOrder: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useDispatchWorkOrder: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateWorkOrder: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateWorkOrderAssignees: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateWorkOrderStatus: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useCreateWorkOrder: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

vi.mock("@/hooks/useFactoryIntakeData", () => ({
  useFactoryIntakes: () => useFactoryIntakes(),
  useFactoryIntakeRuns: () => ({ data: [], isLoading: false, isError: false, refetch: vi.fn() }),
  useCreateFactoryIntake: () => ({ mutateAsync: createFactoryIntakeMutateAsync, isPending: false }),
  useUpdateFactoryIntake: () => ({ mutateAsync: vi.fn(), isPending: false, error: null }),
  useSearchFactoryIntakeItems: () => searchFactoryIntakeItems(),
  useImportFactoryIntakeItem: () => ({ mutateAsync: importFactoryIntakeItem, isPending: false }),
}));

vi.mock("@/hooks/useWorkOrderCardActions", () => ({
  useWorkOrderCardActions: () => ({
    dispatchingOrderIds: new Set<string>(),
    isAssigneesSaving: false,
    onDispatch: vi.fn(),
    onAssigneesSave: vi.fn(),
  }),
}));

vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({ canAct: () => true, isLoading: false }),
}));

vi.mock("@/hooks/usePageTitle", () => ({
  usePageTitle: () => undefined,
}));

vi.mock("@/hooks/useMe", () => ({
  useMe: () => ({ data: { id: "storybook-user" } }),
}));

const useWorkOrderChecks = vi.hoisted(() =>
  vi.fn((_organizationId: string, _factoryId: string, _orderId: string, _options?: { enabled?: boolean }) => ({
    data: [] as unknown[],
  })),
);

const useCanvasMock = vi.hoisted(() => vi.fn());
const updateCanvasVersionMutateAsync = vi.hoisted(() => vi.fn());
const commitCanvasStagingMutateAsync = vi.hoisted(() => vi.fn());

const IMPLEMENTER_AGENT_STEPS = [
  { name: "Clone Repo", type: "bash" as const, command: "git clone $REPO repo" },
  { name: "Implement", type: "prompt" as const, prompt: "Implement the plan.", workingDirectory: "repo" },
];

const implementerCanvas = {
  metadata: { id: "app-refund-implementer", liveVersionId: "version-live" },
  spec: {
    nodes: [
      { id: "on-run", name: "On run", type: "TYPE_TRIGGER", component: "onRun", configuration: {} },
      {
        id: "implementation-agent",
        name: "Agent - Implement from order description",
        type: "TYPE_ACTION",
        component: "runnerClaudeCode",
        concurrency: { max: 5 },
        configuration: {
          machineType: "e1-large-amd64",
          model: "sonnet",
          credentials: { source: "integration", integration: { name: "claude" } },
          steps: IMPLEMENTER_AGENT_STEPS,
        },
      },
    ],
    edges: [],
  },
};

const canvasWithoutAgent = {
  metadata: { id: "app-no-agent", liveVersionId: "version-live" },
  spec: {
    nodes: [{ id: "on-run", name: "On run", type: "TYPE_TRIGGER", component: "onRun", configuration: {} }],
    edges: [],
  },
};

function canvasQuery(data: typeof implementerCanvas | typeof canvasWithoutAgent) {
  return { data, isPending: false, isError: false };
}

vi.mock("@/hooks/useCanvasData", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/hooks/useCanvasData")>();
  return {
    ...actual,
    useCanvas: (organizationId: string, canvasId: string, options?: { enabled?: boolean }) =>
      useCanvasMock(organizationId, canvasId, options),
    useUpdateCanvasVersion: () => ({ mutateAsync: updateCanvasVersionMutateAsync, isPending: false }),
    useCommitCanvasStaging: () => ({ mutateAsync: commitCanvasStagingMutateAsync, isPending: false }),
  };
});

vi.mock("@/lib/toast", () => ({
  showErrorToast: vi.fn(),
  showSuccessToast: vi.fn(),
}));

vi.mock("@/hooks/useWorkOrderChecks", () => ({
  useWorkOrderChecks,
}));

describe("LinesPage board", () => {
  beforeEach(async () => {
    const { DEFAULT_CHECKS_BY_ORDER_ID } = await import("../__fixtures__/workOrderCheckFixtures");
    window.localStorage.clear();
    updateFactoryLineMutateAsync.mockReset();
    useFactoryWorkOrders.mockReturnValue({ data: [] });
    useFactoryApps.mockReturnValue({ data: [] });
    useFactoryIntakes.mockReturnValue({ data: [] });
    createFactoryIntakeMutateAsync.mockReset();
    searchFactoryIntakeItems.mockReturnValue({ data: [], isLoading: false, isError: false });
    importFactoryIntakeItem.mockReset();
    useWorkOrderChecks.mockReset();
    useWorkOrderChecks.mockImplementation(
      (_organizationId: string, _factoryId: string, orderId: string, options?: { enabled?: boolean }) => ({
        data: options?.enabled === false ? [] : (DEFAULT_CHECKS_BY_ORDER_ID[orderId] ?? []),
      }),
    );
    useCanvasMock.mockImplementation((_organizationId: string, canvasId: string, options?: { enabled?: boolean }) => {
      if (options?.enabled === false) {
        return { data: undefined, isPending: false, isError: false };
      }
      if (canvasId === "app-refund-implementer") {
        return canvasQuery(implementerCanvas);
      }
      return canvasQuery(canvasWithoutAgent);
    });
    updateCanvasVersionMutateAsync.mockReset().mockResolvedValue({});
    commitCanvasStagingMutateAsync.mockReset().mockResolvedValue({});
  });

  it("does not show a back link to the lines list", () => {
    renderLinesBoard();

    expect(screen.getByTestId("lines-detail-page")).toBeInTheDocument();
    expect(screen.queryByTestId("lines-back-to-list")).not.toBeInTheDocument();
    expect(screen.queryByTestId("line-intake-drawer")).not.toBeInTheDocument();
  });

  it("sets a pastel colour on the backlog from circular swatches", async () => {
    const user = userEvent.setup();
    renderLinesBoard(`/org-1/workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_PLAN_ID}`);

    await user.click(screen.getByTestId("lines-backlog-menu"));
    await user.click(screen.getByTestId("lines-backlog-menu-color-lime"));

    expect(screen.getByTestId("lines-backlog-column").className).toContain("bg-lime-300");
  });

  it("loads checks only for draft cards that can show a score", () => {
    useFactoryWorkOrders.mockReturnValue({
      data: [...REVIEW_CANDIDATE_WORK_ORDERS, BOARD_IMPLEMENT_FAILED_ORDER],
    });
    renderLinesBoard();

    const fetchedIds = useWorkOrderChecks.mock.calls
      .filter(([, , orderId, options]) => Boolean(orderId) && options?.enabled !== false)
      .map(([, , orderId]) => orderId);

    expect(fetchedIds).toContain("wo-review-pay-842");
    expect(fetchedIds).not.toContain("wo-board-implement-failed");
    expect(screen.queryByTestId("work-order-card-score-wo-board-implement-failed")).not.toBeInTheDocument();
  });

  it("shows a score on a review-candidate backlog card and opens the split run", async () => {
    useFactoryWorkOrders.mockReturnValue({ data: REVIEW_CANDIDATE_WORK_ORDERS });
    const user = userEvent.setup();
    renderLinesBoard();

    const card = screen.getByTestId("work-order-card-wo-review-pay-842");
    const cardScore = within(card).getByTestId("work-order-card-score-wo-review-pay-842");
    expect(cardScore).toHaveAttribute("role", "meter");
    expect(cardScore).toHaveAttribute("aria-valuenow", "5");
    expect(cardScore).toHaveAttribute("aria-valuemax", "5");
    expect(cardScore.querySelectorAll("[data-filled='true']")).toHaveLength(5);
    const start = within(card).getByRole("button", { name: "Start" });
    expect(cardScore.compareDocumentPosition(start) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();

    await user.click(screen.getByRole("button", { name: "Open Add retry handling to webhook delivery" }));

    const dialog = screen.getByTestId("work-order-split-run");
    expect(within(dialog).getByRole("heading", { name: "Add retry handling to webhook delivery" })).toBeInTheDocument();
    expect(within(dialog).queryByRole("tab", { name: "Plan" })).not.toBeInTheDocument();
    expect(within(dialog).queryByRole("tab", { name: "Ticket" })).not.toBeInTheDocument();
    expect(within(dialog).getByRole("tab", { name: "Description" })).toHaveAttribute("data-state", "active");
    expect(within(dialog).getByTestId("split-run-work-order-tab")).toBeInTheDocument();
    expect(within(dialog).getByTestId("split-run-overview-checks")).toHaveTextContent("Confidence score");
    expect(within(dialog).queryByTestId("split-run-review")).not.toBeInTheDocument();

    await user.click(within(dialog).getByRole("tab", { name: "Automations" }));
    expect(within(dialog).getByRole("heading", { name: "Automations" })).toBeInTheDocument();
    expect(within(dialog).getByTestId("split-run-phase-backlog")).toBeInTheDocument();
    expect(within(dialog).queryByTestId("split-run-phase-ingest")).not.toBeInTheDocument();
    expect(within(dialog).queryByTestId("split-run-phase-analyze")).not.toBeInTheDocument();
    expect(within(dialog).queryByTestId("split-run-phase-plan")).not.toBeInTheDocument();
    expect(screen.queryByTestId("review-candidate-modal")).not.toBeInTheDocument();
  });

  it("opens the Intake drawer beside the board when the intake query is set", () => {
    useFactoryIntakes.mockReturnValue({ data: CONFIGURED_INTAKES });
    renderLinesBoard(`/org-1/workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_PLAN_ID}?intake=1`);

    expect(screen.getByTestId("line-intake-drawer")).toBeInTheDocument();
    expect(screen.getByTestId("lines-detail-page")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Plan and Implement" })).toBeInTheDocument();
    expect(screen.getByTestId("line-intake-close")).toBeInTheDocument();
    expect(screen.queryByTestId("line-intake-add")).not.toBeInTheDocument();
    expect(screen.getByTestId(`line-intake-source-${SENTRY_INTAKE_ID}`)).toBeInTheDocument();
    expect(screen.getByTestId(`line-intake-source-${PAGERDUTY_INTAKE_ID}`)).toBeInTheDocument();
    expect(screen.queryByTestId("line-intake-analyzing")).not.toBeInTheDocument();
  });

  it("lists two intakes on the same source", () => {
    useFactoryIntakes.mockReturnValue({
      data: [
        GITHUB_ISSUES_INTAKE,
        { id: "intake-triage", canvasId: "app-triage", name: "Triage issues", source: "SOURCE_GITHUB_ISSUES" },
      ],
    });
    renderLinesBoard(`/org-1/workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_PLAN_ID}?intake=1`);

    expect(screen.getByTestId(`line-intake-source-${GITHUB_ISSUES_INTAKE_ID}`)).toHaveTextContent("GitHub issues");
    expect(screen.getByTestId("line-intake-source-intake-triage")).toHaveTextContent("Triage issues");
  });

  it("creates an intake from the picker and opens its canvas", async () => {
    createFactoryIntakeMutateAsync.mockResolvedValueOnce({ id: "intake-new", canvasId: "canvas-new" });
    const user = userEvent.setup();
    renderLinesBoard(
      `/org-1/workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_PLAN_ID}?intake=1`,
      vi.fn(),
      REFUND_FACTORY,
      { addIntakeControl: true },
    );

    await user.click(screen.getByTestId("line-intake-add"));
    await user.click(screen.getByTestId("add-intake-template-github-issues"));

    await waitFor(() => {
      expect(createFactoryIntakeMutateAsync).toHaveBeenCalledWith({ source: "SOURCE_GITHUB_ISSUES" });
    });
    await waitFor(() => {
      expect(screen.getByTestId("lines-test-location")).toHaveTextContent(
        `/org-1/workspaces/${PRIMARY_FACTORY_KEY}/apps/canvas-new`,
      );
    });
  });

  it("shows GitHub issues only on Acme onboarding intake", () => {
    useFactoryIntakes.mockReturnValue({ data: [GITHUB_ISSUES_INTAKE] });
    renderLinesBoard(
      `/org-1/workspaces/${ACME_ONBOARDING_FACTORY_KEY}/lines/${ACME_ONBOARDING_LINE_ID}?intake=1&intakeId=${GITHUB_ISSUES_INTAKE_ID}`,
      vi.fn(),
      ACME_ONBOARDING_FACTORY,
    );

    expect(screen.getByTestId(`line-intake-source-${GITHUB_ISSUES_INTAKE_ID}`)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Collapse GitHub issues" })).toHaveAttribute("aria-expanded", "true");
    expect(screen.queryByTestId(`line-intake-source-${SENTRY_INTAKE_ID}`)).not.toBeInTheDocument();
    expect(screen.queryByTestId(`line-intake-source-${PAGERDUTY_INTAKE_ID}`)).not.toBeInTheDocument();
  });

  it("opens the factory canvas editor from Edit automation", async () => {
    useFactoryIntakes.mockReturnValue({ data: [GITHUB_ISSUES_INTAKE] });
    const user = userEvent.setup();
    renderLinesBoard(
      `/org-1/workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_PLAN_ID}?intake=1&intakeId=${GITHUB_ISSUES_INTAKE_ID}`,
    );

    await user.click(screen.getByRole("button", { name: `Open ${GITHUB_ISSUES_INTAKE.name} settings` }));
    await user.click(screen.getByRole("tab", { name: "Automation" }));

    expect(screen.getByRole("link", { name: "Edit automation" })).toHaveAttribute(
      "href",
      factoryAppConfigurePath("org-1", PRIMARY_FACTORY_KEY, GITHUB_ISSUES_INTAKE_APP.id!, {
        from: "lines",
        lineId: REFUND_LINE_PLAN_ID,
      }),
    );
  });

  it("loads analyzing tickets from the configured GitHub intake", () => {
    useFactoryIntakes.mockReturnValue({ data: [GITHUB_ISSUES_INTAKE] });
    renderLinesBoard(
      `/org-1/workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_PLAN_ID}?intake=1&intakeId=${GITHUB_ISSUES_INTAKE_ID}`,
    );

    expect(screen.getByTestId(`line-intake-source-${GITHUB_ISSUES_INTAKE_ID}`)).toBeInTheDocument();
    expect(screen.queryByText("Handle duplicate refunds on retry")).not.toBeInTheDocument();
  });

  it("renames the board title on Enter", async () => {
    updateFactoryLineMutateAsync.mockResolvedValueOnce({});
    const user = userEvent.setup();
    renderLinesBoard();

    await user.click(screen.getByTestId("lines-board-title"));
    const input = await screen.findByTestId("lines-board-title-input");
    await waitFor(() => expect(input).toHaveFocus());
    await user.clear(input);
    await user.type(input, "Refund line");
    await user.keyboard("{Enter}");

    await waitFor(() => {
      expect(updateFactoryLineMutateAsync).toHaveBeenCalledWith({
        lineId: REFUND_LINE_PLAN_ID,
        name: "Refund line",
      });
    });
  });

  it("renames a column title on Enter", async () => {
    const user = userEvent.setup();
    renderLinesBoard();

    await user.click(screen.getByTestId("lines-column-title-backlog"));
    const input = await screen.findByTestId("lines-column-title-backlog-input");
    await waitFor(() => expect(input).toHaveFocus());
    await user.clear(input);
    await user.type(input, "Inbox");
    await user.keyboard("{Enter}");

    expect(screen.getByTestId("lines-column-title-backlog")).toHaveTextContent("Inbox");
  });

  it("opens backlog settings in a modal and does not open a canvas", async () => {
    const user = userEvent.setup();
    const linePath = `/org-1/workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_PLAN_ID}`;
    renderLinesBoard();

    await user.click(screen.getByTestId("lines-backlog-menu"));
    await user.click(screen.getByTestId("lines-backlog-menu-edit"));

    expect(screen.getByTestId("lines-backlog-settings")).toBeInTheDocument();
    expect(screen.getByLabelText("Name")).toHaveValue("Backlog");
    expect(screen.getByLabelText("Size")).toBeInTheDocument();
    expect(screen.getByTestId("lines-test-location")).toHaveTextContent(linePath);
    expect(screen.getByTestId("lines-test-location")).not.toHaveTextContent("configure=1");
  });

  it("saves the backlog name from the settings modal", async () => {
    const user = userEvent.setup();
    renderLinesBoard();

    await user.click(screen.getByTestId("lines-backlog-menu"));
    await user.click(screen.getByTestId("lines-backlog-menu-edit"));
    const name = screen.getByLabelText("Name");
    await user.clear(name);
    await user.type(name, "Inbox");
    await user.click(screen.getByTestId("lines-backlog-settings-save"));

    expect(screen.queryByTestId("lines-backlog-settings")).not.toBeInTheDocument();
    expect(screen.getByTestId("lines-column-title-backlog")).toHaveTextContent("Inbox");
  });

  it("offers full-screen automation editing and inline agent editing", async () => {
    const user = userEvent.setup();
    renderLinesBoard();

    await user.click(screen.getByTestId("lines-phase-menu-0"));
    expect(screen.getByTestId("lines-phase-menu-0-edit")).toHaveTextContent("Edit Automation");
    expect(screen.getByTestId("lines-phase-menu-0-edit-agent")).toHaveTextContent("Edit Agent");
    await user.click(screen.getByTestId("lines-phase-menu-0-edit"));

    expect(screen.getByTestId("lines-test-location")).toHaveTextContent(
      factoryAppConfigurePath("org-1", PRIMARY_FACTORY_KEY, "app-refund-implementer", {
        from: "lines",
        lineId: REFUND_LINE_PLAN_ID,
      }),
    );
  });

  it("opens the agent editor with the canvas agent name and steps", async () => {
    const user = userEvent.setup();
    renderLinesBoard();

    await user.click(screen.getByTestId("lines-phase-menu-0"));
    await user.click(screen.getByTestId("lines-phase-menu-0-edit-agent"));

    expect(
      screen.getByRole("heading", { level: 2, name: "Agent - Implement from order description" }),
    ).toBeInTheDocument();
    expect(screen.getByTestId("planning-review-component-toggle-implementation-agent")).toHaveAttribute(
      "aria-expanded",
      "true",
    );
    expect(screen.getByTestId("planning-review-step-name-0")).toHaveValue("Clone Repo");
    expect(screen.getByTestId("planning-review-step-body-0")).toHaveValue("git clone $REPO repo");
  });

  it("hides Edit Agent when the column canvas has no agent", async () => {
    useCanvasMock.mockReturnValue(canvasQuery(canvasWithoutAgent));
    const user = userEvent.setup();
    renderLinesBoard();

    await user.click(screen.getByTestId("lines-phase-menu-0"));
    expect(screen.getByTestId("lines-phase-menu-0-edit")).toHaveTextContent("Edit Automation");
    expect(screen.queryByTestId("lines-phase-menu-0-edit-agent")).not.toBeInTheDocument();
  });

  it("stages and commits the canvas agent on Save Agent", async () => {
    const user = userEvent.setup();
    renderLinesBoard();

    await user.click(screen.getByTestId("lines-phase-menu-0"));
    await user.click(screen.getByTestId("lines-phase-menu-0-edit-agent"));
    const stepName = screen.getByTestId("planning-review-step-name-0");
    await user.clear(stepName);
    await user.type(stepName, "Clone repository");
    await user.click(screen.getByTestId("planning-review-save"));

    await waitFor(() => {
      expect(updateCanvasVersionMutateAsync).toHaveBeenCalledWith(
        expect.objectContaining({
          versionId: "version-live",
          canvasYaml: expect.stringContaining("Clone repository"),
        }),
      );
    });
    expect(commitCanvasStagingMutateAsync).toHaveBeenCalledWith("Update agent");
    expect(screen.queryByTestId("lines-planning-review")).not.toBeInTheDocument();
  });

  it("keeps Backlog Edit as the only edit action", async () => {
    const user = userEvent.setup();
    renderLinesBoard();

    await user.click(screen.getByTestId("lines-backlog-menu"));
    expect(screen.getByTestId("lines-backlog-menu-edit")).toHaveTextContent("Edit");
    expect(screen.queryByTestId("lines-backlog-menu-edit-agent")).not.toBeInTheDocument();
    expect(screen.queryByTestId("lines-backlog-menu-parallelism")).not.toBeInTheDocument();
  });

  it("opens Set parallelism and saves a new cap", async () => {
    updateFactoryLineMutateAsync.mockResolvedValueOnce({});
    const user = userEvent.setup();
    renderLinesBoard();

    await user.click(screen.getByTestId("lines-phase-menu-0"));
    expect(screen.getByTestId("lines-phase-menu-0-parallelism")).toHaveTextContent("Set parallelism (10)");
    await user.click(screen.getByTestId("lines-phase-menu-0-parallelism"));

    const input = screen.getByTestId("lines-parallelism-input");
    await user.clear(input);
    await user.type(input, "20");
    await user.click(screen.getByTestId("lines-parallelism-save"));

    await waitFor(() => {
      expect(updateFactoryLineMutateAsync).toHaveBeenCalledWith(
        expect.objectContaining({
          lineId: REFUND_LINE_PLAN_ID,
          steps: expect.arrayContaining([expect.objectContaining({ maxParallelism: 20 })]),
        }),
      );
    });
  });

  it("hides Edit on the Done column", async () => {
    const user = userEvent.setup();
    renderLinesBoard();

    await user.click(screen.getByTestId("lines-done-menu"));
    expect(screen.queryByTestId("lines-done-menu-edit")).not.toBeInTheDocument();
    expect(screen.queryByTestId("lines-done-create")).not.toBeInTheDocument();
  });

  it("hides the phase path and shows work-order filters", async () => {
    const user = userEvent.setup();
    renderLinesBoard();

    const header = screen.getByTestId("lines-detail-header");
    expect(within(header).queryByText(/→/)).not.toBeInTheDocument();
    expect(within(header).getByTestId("work-orders-scope-all")).toBeInTheDocument();
    expect(within(header).getByTestId("work-orders-scope-active")).toHaveTextContent("Needs attention");
    expect(within(header).getByTestId("work-orders-scope-my")).toBeInTheDocument();
    expect(within(header).getByTestId("work-orders-filter-trigger")).toBeInTheDocument();
    expect(within(header).getByTestId("work-orders-search-trigger")).toBeInTheDocument();
    expect(within(header).queryByTestId("work-order-list-create-button")).not.toBeInTheDocument();

    await user.click(within(header).getByTestId("work-orders-filter-trigger"));
    expect(screen.getByTestId("work-orders-filter-statuses")).toBeInTheDocument();
    expect(screen.queryByTestId("work-orders-filter-lineIds")).not.toBeInTheDocument();
    expect(screen.getByTestId("work-orders-filter-assigneeIds")).toBeInTheDocument();
  });

  it("narrows the board when the search query changes", async () => {
    const user = userEvent.setup();
    useFactoryWorkOrders.mockReturnValue({
      data: [BOARD_IMPLEMENT_FAILED_ORDER, BOARD_DONE_REJECTED_ORDER],
    });
    renderLinesBoard();

    expect(screen.getByText("Fix refund dispatcher timeout loop")).toBeInTheDocument();
    expect(screen.getByText("Replace the refund batch exporter")).toBeInTheDocument();

    await user.click(screen.getByTestId("work-orders-search-trigger"));
    await user.type(screen.getByTestId("work-orders-search-input"), "timeout");

    expect(screen.getByText("Fix refund dispatcher timeout loop")).toBeInTheDocument();
    expect(screen.queryByText("Replace the refund batch exporter")).not.toBeInTheDocument();
  });

  it("opens Edit from the line overflow menu", async () => {
    const user = userEvent.setup();
    renderLinesBoard();

    expect(screen.queryByRole("link", { name: "Edit" })).not.toBeInTheDocument();
    await user.click(screen.getByTestId("lines-edit-menu"));
    await user.click(screen.getByTestId("lines-edit-menu-edit"));

    expect(screen.getByTestId("lines-test-location")).toHaveTextContent(
      editFactoryLinePath("org-1", PRIMARY_FACTORY_KEY, REFUND_LINE_PLAN_ID),
    );
  });
});
