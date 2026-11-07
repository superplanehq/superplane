import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "bun:test";
import { MemoryRouter, Route, Routes, useLocation } from "react-router";

import type { ColumnAutomation } from "../lib/columnAutomations";
import type { WorkOrderCardContext } from "../workOrders/WorkOrderCard";
import { BacklogColumn } from "./BacklogColumn";

const runAutomation = vi.fn();

vi.mock("@/hooks/useFactoryCustomAutomationData", () => ({
  useRunFactoryCustomAutomation: () => ({ mutateAsync: runAutomation, isPending: false }),
}));

vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({ canAct: () => true }),
}));

vi.mock("@/hooks/useFactoryIntakeData", () => ({
  useFactoryIntakes: () => ({ data: [] }),
  useRefreshBacklog: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

const AUTOMATION: ColumnAutomation = {
  id: "eslint",
  kind: "custom-verification",
  name: "ESLint tasks",
  trigger: "On schedule",
  action: "Run automation",
  iconSrc: "",
  iconAlt: "Automation",
  health: "healthy",
  runningCount: 0,
  catalogId: "custom",
  canvasId: "canvas-1",
  canRunNow: true,
};

function LocationProbe() {
  const location = useLocation();
  return <div data-testid="url">{`${location.pathname}${location.search}`}</div>;
}

function renderColumn() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={["/org-1/workspaces/newws/lines/line-1"]}>
        <Routes>
          <Route
            path="/org-1/workspaces/newws/lines/:lineId"
            element={
              <>
                <BacklogColumn
                  organizationId="org-1"
                  factoryId="factory-1"
                  factoryKey="newws"
                  lineId="line-1"
                  orders={[]}
                  title="Backlog"
                  size={null}
                  settingsOpen={false}
                  onOpenSettings={() => undefined}
                  onCloseSettings={() => undefined}
                  onSaveSettings={() => undefined}
                  colorId={null}
                  onColorChange={() => undefined}
                  canCreateWorkOrder={false}
                  canRename={false}
                  onRename={() => undefined}
                  onCreateWorkOrder={() => undefined}
                  workOrderCardContext={{ organizationId: "org-1", factoryKey: "newws" } as WorkOrderCardContext}
                  onOpenWorkOrder={() => undefined}
                  automations={[AUTOMATION]}
                  automationRowCount={1}
                />
                <LocationProbe />
              </>
            }
          />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("BacklogColumn schedule start", () => {
  beforeEach(() => {
    runAutomation.mockReset();
  });

  it("opens the created run after the start request returns", async () => {
    const user = userEvent.setup();
    let resolveRun: (value: { runId: string }) => void = () => undefined;
    runAutomation.mockReturnValue(
      new Promise((resolve) => {
        resolveRun = resolve;
      }),
    );
    renderColumn();

    const click = user.click(screen.getByRole("button", { name: "Run ESLint tasks" }));
    await waitFor(() => expect(screen.getByRole("button", { name: "Run ESLint tasks" })).toBeDisabled());
    expect(screen.getByTestId("url")).not.toHaveTextContent("automationRun=");

    resolveRun({ runId: "run-42" });
    await click;

    await waitFor(() => expect(screen.getByTestId("url")).toHaveTextContent("automationRun=run-42"));
    const url = screen.getByTestId("url").textContent ?? "";
    expect(url).toContain("customAutomation=canvas-1");
    expect(url).toContain("automationColumn=backlog");
    expect(url).toContain("automationTab=runs");
    expect(runAutomation).toHaveBeenCalledWith({ automationId: "canvas-1", workOrderId: "" });
  });

  it("stays on the board when the start request fails", async () => {
    const user = userEvent.setup();
    runAutomation.mockRejectedValue(new Error("SuperPlane could not start the automation."));
    renderColumn();

    await user.click(screen.getByRole("button", { name: "Run ESLint tasks" }));

    expect(screen.getByTestId("url")).not.toHaveTextContent("automationRun=");
    expect(screen.getByTestId("url")).not.toHaveTextContent("customAutomation=");
  });
});
