import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "bun:test";
import { MemoryRouter, useLocation } from "react-router";
import { useColumnAutomationRun } from "./useColumnAutomationRun";
import { customAutomationColumnAutomation } from "@/pages/factories/lib/customAutomationColumnAutomation";
import { ColumnAutomationRows } from "@/pages/factories/pages/ColumnAutomationRows";
import { AutomationRowContext } from "@/pages/factories/pages/AutomationRowContext";
import type { CustomAutomationMode } from "@/pages/factories/lib/customAutomation";

const mutateAsync = vi.fn();
const showErrorToast = vi.fn();
const settings = vi.fn();
vi.mock("./useFactoryCustomAutomationData", () => ({ useRunFactoryCustomAutomation: () => ({ mutateAsync }) }));
vi.mock("@/lib/toast", () => ({ showErrorToast }));
beforeEach(() => {
  mutateAsync.mockReset();
  showErrorToast.mockReset();
  settings.mockReset();
});

function Board({ runsOnTask = false, modes = ["manual"] }: { runsOnTask?: boolean; modes?: CustomAutomationMode[] }) {
  const run = useColumnAutomationRun("org", "factory", "workspace", "line");
  const location = useLocation();
  const automation = customAutomationColumnAutomation(
    {
      id: "verify-manual",
      name: "Inspect repository",
      columnKey: "verify",
      customAutomationRunsOnTask: runsOnTask,
      customAutomationMode: modes[0],
      customAutomationTriggers: modes.map((mode) => ({ mode })),
    },
    [],
    () => 0,
  )!;
  return (
    <AutomationRowContext.Provider value={run}>
      <ColumnAutomationRows
        title="Verify"
        automations={[automation]}
        rowCount={1}
        testId="verify"
        onRowAction={settings}
      />
      <output data-testid="location">{location.search}</output>
    </AutomationRowContext.Provider>
  );
}

describe("column manual run", () => {
  it("runs a mixed no-task Verify automation once and opens its Verify history", async () => {
    let finish!: (value: { runId: string }) => void;
    mutateAsync.mockImplementation(
      () =>
        new Promise((resolve) => {
          finish = resolve;
        }),
    );
    render(
      <MemoryRouter>
        <Board modes={["webhook", "manual"]} />
      </MemoryRouter>,
    );
    const play = screen.getByRole("button", { name: "Run Inspect repository" });
    fireEvent.click(play);
    expect(play).toBeDisabled();
    fireEvent.click(play);
    expect(mutateAsync).toHaveBeenCalledTimes(1);
    expect(mutateAsync).toHaveBeenCalledWith({ automationId: "verify-manual", workOrderId: "" });
    expect(settings).not.toHaveBeenCalled();
    await act(async () => finish({ runId: "run-1" }));
    const params = new URLSearchParams(screen.getByTestId("location").textContent!);
    expect(params.get("automationColumn")).toBe("verify");
    expect(params.get("automationRun")).toBe("run-1");
    expect(params.get("automationTab")).toBe("runs");
  });

  it("reports failure and permits retry", async () => {
    mutateAsync.mockRejectedValueOnce(new Error("failed")).mockResolvedValueOnce({ runId: "retry" });
    render(
      <MemoryRouter>
        <Board />
      </MemoryRouter>,
    );
    const play = screen.getByRole("button", { name: "Run Inspect repository" });
    fireEvent.click(play);
    await waitFor(() => expect(showErrorToast).toHaveBeenCalledTimes(1));
    expect(play).toBeEnabled();
    fireEvent.click(play);
    await waitFor(() => expect(screen.getByTestId("location")).toHaveTextContent("automationRun=retry"));
  });

  it.each([
    { runsOnTask: true, modes: ["manual"] },
    { runsOnTask: false, modes: ["webhook"] },
  ] as const)("keeps task-based and webhook-only runs off the column", ({ runsOnTask, modes }) => {
    render(
      <MemoryRouter>
        <Board runsOnTask={runsOnTask} modes={[...modes]} />
      </MemoryRouter>,
    );
    expect(screen.queryByRole("button", { name: "Run Inspect repository" })).not.toBeInTheDocument();
  });
});
