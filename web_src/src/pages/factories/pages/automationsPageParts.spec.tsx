import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ComponentProps } from "react";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";

import type { FactoryApp } from "@/api-client";

import { AutomationCard, AutomationDetail } from "./automationsPageParts";
import { duplicateAutomationName } from "./automationCardActions";

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

const app: FactoryApp = {
  id: "app-refund-planner",
  name: "Refund Planner",
  description: "Plans reconciliation work across ledger + payment services.",
};

function renderCard(props: Partial<ComponentProps<typeof AutomationCard>> = {}) {
  return render(
    <MemoryRouter>
      <AutomationCard app={app} tick="passed" statusLabel="Passed" emphasized {...props} />
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

describe("AutomationDetail tabs", () => {
  const actions = {
    onEdit: vi.fn(),
    onDuplicate: vi.fn(),
    onDelete: vi.fn(),
    canEdit: true,
    canDuplicate: true,
    canDelete: true,
  };

  function renderDetail() {
    return render(
      <MemoryRouter>
        <AutomationDetail organizationId="org-1" factoryKey="SP" app={app} actions={actions} />
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
  });

  it("opens the Velocity tab", async () => {
    const user = userEvent.setup();
    renderDetail();

    await user.click(screen.getByRole("tab", { name: "Velocity" }));
    expect(screen.getByTestId("line-velocity-panel")).toBeInTheDocument();
  });
});
