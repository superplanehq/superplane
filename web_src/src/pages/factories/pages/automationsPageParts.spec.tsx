import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ComponentProps } from "react";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";

import type { FactoriesFactoryLine, FactoriesWorkOrder, FactoryApp } from "@/api-client";

import { AutomationDetail } from "./AutomationDetail";
import { AutomationCard } from "./automationsPageParts";
import { duplicateAutomationName } from "./automationCardActions";
import type { WorkOrderCardContext } from "../workOrders/WorkOrderCard";

const planLine: FactoriesFactoryLine = {
  id: "line-plan",
  name: "plan-and-implement",
  steps: [{ name: "plan", app: { app: "app-refund-planner", entrypoint: "start-plan" } }],
};

const hotfixLine: FactoriesFactoryLine = {
  id: "line-hotfix",
  name: "hotfix",
  steps: [{ name: "verify", app: { app: "app-refund-planner", entrypoint: "start-verification" } }],
};

vi.mock("@/hooks/useCanvasData", () => ({
  useInfiniteCanvasRuns: () => ({
    data: {
      pages: [
        {
          runs: [
            {
              id: "run-c1111111",
              state: "STATE_STARTED",
              updatedAt: "2026-08-13T10:00:00.000Z",
              rootEvent: { customName: "Run c1111111" },
            },
            {
              id: "run-c2222222",
              state: "STATE_STARTED",
              updatedAt: "2026-08-12T10:00:00.000Z",
              rootEvent: { customName: "Run c2222222" },
            },
          ],
        },
      ],
    },
    isLoading: false,
    fetchNextPage: vi.fn(),
    hasNextPage: false,
    isFetchingNextPage: false,
  }),
}));

vi.mock("./LineVelocityPanel", () => ({
  LineVelocityPanel: () => <div data-testid="line-velocity-panel">Velocity</div>,
}));

vi.mock("@/hooks/useOrgUserLookup", () => ({
  useOrgUserLookup: () => ({
    resolveUser: (id: string | undefined, name?: string) =>
      id ? { id, name: name ?? "Unknown member", initials: "U" } : null,
    isLoading: false,
  }),
}));

const app: FactoryApp = {
  id: "app-refund-planner",
  name: "Refund Planner",
  description: "Plans reconciliation work across ledger + payment services.",
};

function renderCard(props: Partial<ComponentProps<typeof AutomationCard>> = {}) {
  return render(
    <MemoryRouter>
      <AutomationCard app={app} tick="passed" statusLabel="Passed" emphasized lines={[planLine]} {...props} />
    </MemoryRouter>,
  );
}

describe("duplicateAutomationName", () => {
  it("appends copy to the trimmed name", () => {
    expect(duplicateAutomationName("Refund Planner")).toBe("Refund Planner copy");
  });

  it("falls back when the name is empty", () => {
    expect(duplicateAutomationName("  ")).toBe("Unnamed automation copy");
  });
});

describe("AutomationCard menu", () => {
  const actions = {
    onEdit: vi.fn(),
    onDuplicate: vi.fn(),
    onDelete: vi.fn(),
    canEdit: true,
    canDuplicate: true,
    canDelete: true,
  };

  it("hides the list-card menu until hover, then opens Edit, Duplicate, and Delete", async () => {
    const user = userEvent.setup();

    render(
      <MemoryRouter>
        <AutomationCard
          app={app}
          tick="passed"
          statusLabel="Passed"
          href="/automations/app-refund-planner"
          actions={actions}
        />
      </MemoryRouter>,
    );

    const trigger = screen.getByTestId("automations-card-menu");
    expect(trigger).toHaveClass("opacity-0");

    await user.click(trigger);
    expect(screen.getByTestId("automations-card-edit")).toHaveTextContent("Edit");
    expect(screen.getByTestId("automations-card-duplicate")).toHaveTextContent("Duplicate");
    expect(screen.getByRole("separator")).toBeInTheDocument();
    expect(screen.getByTestId("automations-card-delete")).toHaveTextContent("Delete");
  });

  it("keeps the detail-card menu visible and opens Edit, Duplicate, and Delete", async () => {
    const user = userEvent.setup();

    renderCard({ actions });

    const trigger = screen.getByTestId("automations-card-menu");
    expect(trigger).not.toHaveClass("opacity-0");

    await user.click(trigger);
    expect(screen.getByTestId("automations-card-edit")).toHaveTextContent("Edit");
    expect(screen.getByTestId("automations-card-duplicate")).toHaveTextContent("Duplicate");
    expect(screen.getByRole("separator")).toBeInTheDocument();
    expect(screen.getByTestId("automations-card-delete")).toHaveTextContent("Delete");
    expect(screen.getByTestId("automations-card-delete")).toHaveClass("text-destructive");

    await user.click(screen.getByTestId("automations-card-edit"));
    expect(actions.onEdit).toHaveBeenCalledTimes(1);
  });
});

describe("AutomationCard line usage", () => {
  it("lists the lines that use the automation", () => {
    renderCard({ lines: [planLine, hotfixLine] });

    const usedBy = screen.getByTestId("automations-used-by");
    expect(usedBy).toHaveTextContent("Used by Plan and Implement, Hotfix");
    expect(usedBy.querySelector("a")).toBeNull();
  });

  it("says when no line uses the automation", () => {
    renderCard({
      app: { ...app, id: "app-unused" },
      lines: [planLine],
    });

    expect(screen.getByTestId("automations-used-by")).toHaveTextContent("Not used by a line");
  });
});

describe("AutomationDetail tabs", () => {
  const actions = {
    onEdit: vi.fn(),
    onDuplicate: vi.fn(),
    onDelete: vi.fn(),
    canEdit: true,
    canDuplicate: true,
    canDelete: true,
  };

  const workOrderCardContext: WorkOrderCardContext = {
    organizationId: "org-1",
    factoryKey: "SP",
    factoryLines: [],
    canDispatch: false,
    canAssign: false,
    isDispatching: false,
    isAssigneesSaving: false,
    onDispatch: vi.fn().mockResolvedValue(undefined),
    onAssigneesSave: vi.fn().mockResolvedValue(undefined),
  };

  const matchingOrder: FactoriesWorkOrder = {
    id: "wo-1",
    number: "103",
    key: "SP-103",
    title: "Add refund reconciliation test",
    state: "STATE_OPEN",
    assignees: [{ id: "user-1", name: "Ada Lovelace" }],
    lineDispatches: [
      {
        id: "dispatch-1",
        line: { id: "line-1", name: "Refunds" },
        state: "STATE_ACTIVE",
        stepExecutions: [
          {
            id: "e1",
            step: "implement",
            state: "STATE_STARTED",
            run: { id: "run-c1111111", appId: "app-refund-planner" },
          },
        ],
      },
    ],
  };

  function renderDetail(workOrders: FactoriesWorkOrder[] = []) {
    return render(
      <MemoryRouter>
        <AutomationDetail
          organizationId="org-1"
          factoryKey="SP"
          app={app}
          actions={actions}
          factory={{ id: "factory-1", name: "Refunds", key: "SP", lines: [planLine, hotfixLine] }}
          workOrders={workOrders}
          workOrderCardContext={workOrderCardContext}
        />
      </MemoryRouter>,
    );
  }

  it("shows Runs and Velocity tabs, with run rows as soft cards", () => {
    renderDetail();

    expect(screen.getByRole("tab", { name: "Runs" })).toHaveAttribute("data-state", "active");
    expect(screen.getByRole("tab", { name: "Velocity" })).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Runs" })).not.toBeInTheDocument();

    const run = screen.getByTestId("automations-run-run-c1111111");
    expect(run).toHaveTextContent("Run c1111111");
    expect(run.className).toMatch(/\bborder\b/);
    expect(run.className).toMatch(/\brounded-md\b/);
    expect(screen.getByTestId("automations-runs-scroll").className).toMatch(/\bgap-2\b/);
    expect(screen.getByTestId("automations-runs-scroll").closest("section")).toBeNull();
    expect(screen.getByTestId("automations-detail-body").className).toMatch(/\bmax-w-none\b/);
  });

  it("renders the work order card when the run belongs to a work order", () => {
    renderDetail([matchingOrder]);

    const card = screen.getByTestId("work-order-card-wo-1");
    expect(card).toHaveTextContent("Add refund reconciliation test");
    expect(card).toHaveTextContent("Running");
    expect(card).toHaveTextContent("SP-103");
    expect(screen.getByTestId("work-order-row-assignees-wo-1")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Open Add refund reconciliation test" })).toHaveAttribute(
      "href",
      expect.stringContaining("run=run-c1111111"),
    );
  });

  it("links each line that uses the automation", () => {
    renderDetail();

    const usedBy = screen.getByTestId("automations-used-by");
    expect(usedBy).toHaveTextContent("Used by Plan and Implement, Hotfix");
    expect(screen.getByRole("link", { name: "Plan and Implement" })).toHaveAttribute(
      "href",
      "/org-1/workspaces/SP/lines/line-plan",
    );
    expect(screen.getByRole("link", { name: "Hotfix" })).toHaveAttribute(
      "href",
      "/org-1/workspaces/SP/lines/line-hotfix",
    );
  });

  it("opens the Velocity tab", async () => {
    const user = userEvent.setup();
    renderDetail();

    await user.click(screen.getByRole("tab", { name: "Velocity" }));
    expect(screen.getByTestId("line-velocity-panel")).toBeInTheDocument();
  });
});
