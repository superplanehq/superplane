import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, useLocation, useNavigate } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";
import type { CanvasesCanvasRun, FactoriesFactoryCustomAutomation } from "@/api-client";
import { ThemeProvider } from "@/contexts/ThemeProvider";
import { FactoriesLayoutContext } from "../layout/factoriesLayoutContext";
import { CustomAutomationSetupModal } from "./CustomAutomationSetupModal";

const hooks = vi.hoisted(() => ({
  describe: vi.fn(),
  create: vi.fn(),
  update: vi.fn(),
  delete: vi.fn(),
  runs: vi.fn(),
  run: vi.fn(),
  runtime: vi.fn(),
}));
vi.mock("@/hooks/useFactoryCustomAutomationData", () => ({
  useFactoryCustomAutomation: (...args: unknown[]) => hooks.describe(...args),
  useCreateFactoryCustomAutomation: () => ({ mutateAsync: hooks.create }),
  useUpdateFactoryCustomAutomation: () => ({ mutateAsync: hooks.update }),
}));
vi.mock("@/hooks/useFactoryData", () => ({
  useDeleteFactoryAutomation: () => ({ mutateAsync: hooks.delete }),
}));
vi.mock("@/hooks/useCanvasData", () => ({
  useInfiniteCanvasRuns: (...args: unknown[]) => hooks.runs(...args),
  useDescribeRun: (...args: unknown[]) => hooks.run(...args),
  useEventExecutions: () => ({ data: { executions: [] } }),
  useCanvasVersion: () => ({ data: { spec: { nodes: [] } } }),
}));
vi.mock("@/hooks/useCanvasWebsocket", () => ({ useCanvasRuntimeWebsocket: hooks.runtime }));
// Agent model controls have separate coverage; retain the real setup form and step selection.
vi.mock("./PlanningReviewForm", () => ({ PlanningReviewForm: () => null }));

const automation: FactoriesFactoryCustomAutomation = {
  automation: { id: "custom-1", name: "Deploy preview" },
  settings: {
    startMode: "START_MODE_MANUAL",
    steps: [{ name: "Deploy", type: "TYPE_PROMPT", prompt: "Deploy the preview." }],
  },
};
const run: CanvasesCanvasRun = {
  id: "run-1",
  canvasId: "custom-1",
  state: "STATE_FINISHED",
  result: "RESULT_PASSED",
  task: { id: "task-1", key: "SP-123", title: "Preview settings" },
};

function BrowserControls() {
  const location = useLocation();
  const navigate = useNavigate();
  return (
    <>
      <output data-testid="url">{location.search}</output>
      <button onClick={() => navigate(-1)}>Browser Back</button>
      <button onClick={() => navigate(1)}>Browser Forward</button>
    </>
  );
}

function renderModal(
  search = "?customAutomation=custom-1",
  automationId: string | undefined = search.includes("customAutomation=new") ? undefined : "custom-1",
) {
  const onClose = vi.fn();
  render(
    <MemoryRouter initialEntries={[`/org/workspaces/sp/lines/line${search}`]}>
      <BrowserControls />
      <FactoriesLayoutContext.Provider
        value={{
          organizationId: "org",
          factoryId: "factory",
          factoryKey: "sp",
          factory: null,
          factories: [],
          openCreateWorkOrder: vi.fn(),
        }}
      >
        <ThemeProvider>
          <CustomAutomationSetupModal lineId="line" automationId={automationId} onClose={onClose} />
        </ThemeProvider>
      </FactoriesLayoutContext.Provider>
    </MemoryRouter>,
  );
  return onClose;
}

describe("custom automation configuration run history", () => {
  beforeEach(() => {
    Object.values(hooks).forEach((hook) => hook.mockReset());
    hooks.describe.mockReturnValue({ data: automation });
    hooks.runs.mockImplementation((canvasId: string) => ({
      data: {
        pages: [
          {
            runs:
              canvasId === "custom-1" ? [run] : [{ ...run, id: "other-run", task: { title: "Other automation task" } }],
          },
        ],
      },
    }));
    hooks.run.mockReturnValue({ data: { run } });
  });

  it.each(["START_MODE_MANUAL", "START_MODE_AUTOMATIC"] as const)(
    "shows only the saved automation's runs in %s mode and restores run navigation",
    async (startMode) => {
      hooks.describe.mockReturnValue({ data: { ...automation, settings: { ...automation.settings, startMode } } });
      const onClose = renderModal();
      const dialog = within(screen.getByRole("dialog"));
      expect(dialog.getByRole("tab", { name: "Configuration" })).toHaveAttribute("aria-selected", "true");
      expect(hooks.runs).not.toHaveBeenCalled();
      const user = userEvent.setup();
      await user.click(dialog.getByRole("tab", { name: "Runs" }));
      expect(dialog.getByText("SP-123")).toBeVisible();
      expect(dialog.getByText("Preview settings")).toBeVisible();
      expect(dialog.queryByText("Other automation task")).not.toBeInTheDocument();
      expect(hooks.runs).toHaveBeenCalledWith("custom-1", { includeTask: true });
      expect(hooks.runtime).toHaveBeenCalledWith("custom-1", "org", true);
      await user.click(dialog.getByRole("button", { name: "Open run run-1" }));
      expect(screen.getByTestId("url")).toHaveTextContent("customAutomation=custom-1");
      expect(screen.getByTestId("url")).toHaveTextContent("automationRun=run-1");
      expect(dialog.getByRole("button", { name: "Run history" })).toBeVisible();
      fireEvent.click(screen.getByRole("button", { name: "Browser Back", hidden: true }));
      await waitFor(() => expect(dialog.getByRole("table")).toBeVisible());
      fireEvent.click(screen.getByRole("button", { name: "Browser Back", hidden: true }));
      await waitFor(() =>
        expect(dialog.getByRole("tab", { name: "Configuration" })).toHaveAttribute("aria-selected", "true"),
      );
      fireEvent.click(screen.getByRole("button", { name: "Browser Forward", hidden: true }));
      await waitFor(() => expect(dialog.getByRole("table")).toBeVisible());
      fireEvent.click(screen.getByRole("button", { name: "Browser Forward", hidden: true }));
      await waitFor(() => expect(dialog.getByRole("button", { name: "Run history" })).toBeVisible());
      await user.click(dialog.getByRole("button", { name: "Run history" }));
      expect(dialog.getByRole("table")).toBeVisible();
      expect(screen.getByTestId("url")).not.toHaveTextContent("automationRun=");
      expect(hooks.create).not.toHaveBeenCalled();
      expect(hooks.update).not.toHaveBeenCalled();
      expect(hooks.delete).not.toHaveBeenCalled();
      expect(onClose).not.toHaveBeenCalled();
    },
  );

  it.each(["automationTab=runs", "automationRun=run-1"])("opens explicit history links: %s", (selection) => {
    renderModal(`?customAutomation=custom-1&${selection}`);
    expect(screen.getByRole("tab", { name: "Runs" })).toHaveAttribute("aria-selected", "true");
    expect(
      selection.includes("automationRun")
        ? screen.getByRole("button", { name: "Run history" })
        : screen.getByRole("table"),
    ).toBeVisible();
  });

  it("retains unsaved fields and the selected step when switching tabs", async () => {
    renderModal();
    const user = userEvent.setup();
    await user.clear(screen.getByLabelText("Automation name"));
    await user.type(screen.getByLabelText("Automation name"), "Draft automation");
    await user.click(screen.getByRole("button", { name: /Deploy.*Deploy the preview/ }));
    await user.clear(screen.getByLabelText("Name"));
    await user.type(screen.getByLabelText("Name"), "Draft step");
    await user.click(screen.getByRole("tab", { name: "Runs" }));
    expect(screen.queryByRole("button", { name: "Save" })).not.toBeInTheDocument();
    await user.click(screen.getByRole("tab", { name: "Configuration" }));
    expect(screen.getByLabelText("Name")).toHaveValue("Draft step");
    await user.click(screen.getByRole("button", { name: /Manual.*Runs when/ }));
    expect(screen.getByLabelText("Automation name")).toHaveValue("Draft automation");
    expect(hooks.update).not.toHaveBeenCalled();
  });

  it("keeps new automation configuration available without mounting history", async () => {
    hooks.describe.mockReturnValue({});
    const onClose = renderModal("?customAutomation=new&automationTab=runs&automationRun=run-1");
    expect(screen.queryByRole("tab", { name: "Runs" })).not.toBeInTheDocument();
    expect(screen.getByLabelText("Automation name")).toBeVisible();
    expect(screen.getByRole("button", { name: "Save" })).toBeVisible();
    expect(screen.queryByRole("button", { name: "Delete" })).not.toBeInTheDocument();
    expect(hooks.runs).not.toHaveBeenCalled();
    expect(hooks.runtime).not.toHaveBeenCalled();
    await userEvent.setup().click(screen.getByRole("button", { name: "Cancel" }));
    expect(onClose).toHaveBeenCalledTimes(1);
    expect(hooks.create).not.toHaveBeenCalled();
  });

  it("saves configuration through the existing mutation and closes", async () => {
    const onClose = renderModal();
    await userEvent.setup().click(screen.getByRole("button", { name: "Save" }));
    await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1));
    expect(hooks.update).toHaveBeenCalledWith(
      expect.objectContaining({
        name: "Deploy preview",
        settings: expect.objectContaining({ startMode: "START_MODE_MANUAL" }),
      }),
    );
    expect(hooks.runs).not.toHaveBeenCalled();
  });

  it("keeps deletion confirmation with configuration and closes after deletion", async () => {
    const onClose = renderModal();
    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: "Delete" }));
    expect(hooks.delete).not.toHaveBeenCalled();
    await user.click(screen.getByRole("button", { name: "Delete" }));
    await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1));
    expect(hooks.delete).toHaveBeenCalledWith("custom-1");
  });
});
