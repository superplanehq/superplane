import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { FactoriesFactory, FactoriesFactoryIntake, FactoriesWorkOrder, FactoryApp } from "@/api-client";
import type * as canvasData from "@/hooks/useCanvasData";
import {
  ACME_ONBOARDING_FACTORY,
  ACME_ONBOARDING_FACTORY_KEY,
  ACME_ONBOARDING_LINE_ID,
  GITHUB_ISSUES_INTAKE,
  GITHUB_ISSUES_INTAKE_ID,
  PRIMARY_FACTORY_KEY,
  REFUND_FACTORY,
  REFUND_LINE_PLAN_ID,
} from "../__fixtures__/factoryPageResponses";
import { LinesBoardSpecHarness } from "./linesPageSpecRender";
import { REVIEW_CANDIDATE_WORK_ORDERS } from "./onboarding/first-run/reviewCandidates";

function renderLinesBoard(
  path = `/org-1/workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_PLAN_ID}`,
  openCreateWorkOrder = vi.fn(),
  factory: FactoriesFactory = REFUND_FACTORY,
) {
  return render(<LinesBoardSpecHarness path={path} openCreateWorkOrder={openCreateWorkOrder} factory={factory} />);
}

const useFactoryWorkOrders = vi.fn(() => ({ data: [] as FactoriesWorkOrder[] }));
const useFactoryApps = vi.fn(() => ({ data: [] as FactoryApp[] }));
const useFactoryIntakes = vi.fn(() => ({ data: [] as FactoriesFactoryIntake[] }));
const searchFactoryIntakeItems = vi.fn(() => ({
  data: [] as { id: string; key: string; title: string; body: string; url: string }[],
  isLoading: false,
  isError: false,
}));
const importFactoryIntakeItem = vi.fn();

const REFUND_INTAKE_SEARCH = {
  data: [
    {
      id: "12",
      key: "#12",
      title: "Handle duplicate refunds",
      body: "Retrying a refund posts twice.",
      url: "https://github.com/acme/payments/issues/12",
    },
  ],
  isLoading: false,
  isError: false,
};

async function importRefundIssue(user: ReturnType<typeof userEvent.setup>) {
  await user.click(screen.getByTestId("lines-backlog-create"));
  await user.click(screen.getByPlaceholderText("Import from GitHub issue"));
  await user.click(screen.getByTestId("lines-backlog-create-item-12"));
}

vi.mock("@/hooks/useFactoryData", () => ({
  useFactoryWorkOrders: () => useFactoryWorkOrders(),
  useFactoryApps: () => useFactoryApps(),
  useCreateFactoryLine: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateFactoryLine: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useWorkOrderEvents: () => ({ data: { pages: [] } }),
  useWorkOrderArtifacts: () => ({ data: [] }),
  useFactoryPullRequests: () => ({ data: [] }),
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
  useCreateFactoryIntake: () => ({ mutateAsync: vi.fn(), isPending: false }),
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

vi.mock("@/hooks/useWorkOrderChecks", () => ({
  useWorkOrderChecks: () => ({ data: [] }),
}));

vi.mock("@/hooks/useFactoryPRFeedbackData", () => ({
  useFactoryPRFeedbackHandlers: () => ({ data: [] }),
}));

vi.mock("@/hooks/useCanvasData", async (importOriginal) => {
  const actual = await importOriginal<typeof canvasData>();
  return {
    ...actual,
    useCanvas: () => ({ data: { spec: { nodes: [] } }, isPending: false, isError: false }),
    useUpdateCanvasVersion: () => ({ mutateAsync: vi.fn(), isPending: false }),
    useCommitCanvasStaging: () => ({ mutateAsync: vi.fn(), isPending: false }),
  };
});

describe("LinesPage backlog create", () => {
  beforeEach(() => {
    window.localStorage.clear();
    useFactoryWorkOrders.mockReturnValue({ data: [] });
    useFactoryApps.mockReturnValue({ data: [] });
    useFactoryIntakes.mockReturnValue({ data: [] });
    searchFactoryIntakeItems.mockReturnValue({ data: [], isLoading: false, isError: false });
    importFactoryIntakeItem.mockReset();
  });

  it("keeps a create ghost card at the bottom of the backlog", async () => {
    Element.prototype.scrollIntoView = vi.fn();
    useFactoryWorkOrders.mockReturnValue({ data: REVIEW_CANDIDATE_WORK_ORDERS });
    useFactoryIntakes.mockReturnValue({ data: [GITHUB_ISSUES_INTAKE] });
    const user = userEvent.setup();
    renderLinesBoard();

    const list = screen.getByTestId("lines-backlog-column-scroll");
    const items = within(list).getAllByRole("listitem");
    expect(items.at(-1)).toHaveAttribute("data-testid", "lines-backlog-create-ghost-item");
    expect(within(items.at(-1)!).getByRole("button", { name: "Create task" })).toBeInTheDocument();
    expect(screen.queryByText("No tasks in the backlog.")).not.toBeInTheDocument();

    await user.click(screen.getByTestId("lines-backlog-create-ghost"));
    expect(screen.getByTestId("lines-backlog-create-menu")).toBeInTheDocument();
  });

  it("shows the create ghost card when the backlog is empty", () => {
    renderLinesBoard();

    expect(screen.getByTestId("lines-backlog-create-ghost")).toBeInTheDocument();
    expect(screen.queryByText("No tasks in the backlog.")).not.toBeInTheDocument();
  });

  it("hides the create ghost card when the backlog is at capacity", async () => {
    useFactoryWorkOrders.mockReturnValue({ data: REVIEW_CANDIDATE_WORK_ORDERS });
    const user = userEvent.setup();
    renderLinesBoard();

    expect(screen.getByTestId("lines-backlog-create-ghost")).toBeInTheDocument();

    await user.click(screen.getByTestId("lines-backlog-menu"));
    await user.click(screen.getByTestId("lines-backlog-menu-edit"));
    await user.type(screen.getByLabelText("Size"), "1");
    await user.click(screen.getByTestId("lines-backlog-settings-save"));

    expect(screen.queryByTestId("lines-backlog-create-ghost")).not.toBeInTheDocument();
    expect(screen.getByTestId("lines-backlog-create")).toBeDisabled();
  });

  it("creates tasks from the backlog header plus", async () => {
    Element.prototype.scrollIntoView = vi.fn();
    const openCreateWorkOrder = vi.fn();
    const user = userEvent.setup();
    renderLinesBoard(`/org-1/workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_PLAN_ID}`, openCreateWorkOrder);

    const backlog = screen.getByTestId("lines-backlog-column");
    expect(within(backlog).queryByRole("button", { name: "Add task" })).not.toBeInTheDocument();

    await user.click(within(backlog).getByTestId("lines-backlog-create"));
    expect(screen.queryByTestId("lines-backlog-create-menu")).not.toBeInTheDocument();
    expect(openCreateWorkOrder).toHaveBeenCalledTimes(1);
  });

  it("imports an intake item and opens the task popup", async () => {
    Element.prototype.scrollIntoView = vi.fn();
    useFactoryIntakes.mockReturnValue({ data: [GITHUB_ISSUES_INTAKE] });
    searchFactoryIntakeItems.mockReturnValue(REFUND_INTAKE_SEARCH);
    importFactoryIntakeItem.mockResolvedValue({
      id: "wo-imported-12",
      title: "Handle duplicate refunds",
      state: "STATE_DRAFT",
    });
    const user = userEvent.setup();
    renderLinesBoard();

    await importRefundIssue(user);

    expect(importFactoryIntakeItem).toHaveBeenCalledWith({
      intakeId: GITHUB_ISSUES_INTAKE_ID,
      itemId: "12",
    });
    await waitFor(() => {
      expect(screen.getByTestId("work-order-split-run")).toBeInTheDocument();
    });
    expect(screen.getByRole("tab", { name: "Description" })).toHaveAttribute("data-state", "active");
  });

  it("opens the popup from a just-imported order that already has a number", async () => {
    Element.prototype.scrollIntoView = vi.fn();
    useFactoryIntakes.mockReturnValue({ data: [GITHUB_ISSUES_INTAKE] });
    searchFactoryIntakeItems.mockReturnValue(REFUND_INTAKE_SEARCH);
    importFactoryIntakeItem.mockResolvedValue({
      id: "wo-imported-12",
      number: "12",
      title: "Handle duplicate refunds",
      state: "STATE_DRAFT",
    });
    const user = userEvent.setup();
    renderLinesBoard();

    await importRefundIssue(user);

    await waitFor(() => {
      expect(screen.getByTestId("work-order-split-run")).toBeInTheDocument();
    });
  });

  it("shows a backlog onboarding card on Acme when the backlog is empty", () => {
    renderLinesBoard(
      `/org-1/workspaces/${ACME_ONBOARDING_FACTORY_KEY}/lines/${ACME_ONBOARDING_LINE_ID}`,
      vi.fn(),
      ACME_ONBOARDING_FACTORY,
    );

    expect(screen.getByTestId("backlog-onboarding-card")).toBeInTheDocument();
    expect(screen.queryByTestId("lines-backlog-create-ghost")).not.toBeInTheDocument();
    expect(screen.queryByText("No tasks in the backlog.")).not.toBeInTheDocument();
  });
});
