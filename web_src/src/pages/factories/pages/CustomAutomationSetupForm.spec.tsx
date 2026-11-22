import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { expect, it, vi } from "bun:test";
import type { FactoriesFactoryCustomAutomation } from "@/api-client";
import { ThemeProvider } from "@/contexts/ThemeProvider";
import { CustomAutomationSetupForm } from "./CustomAutomationSetupForm";
import { useCustomAutomationSetupPage } from "./useCustomAutomationSetupPage";

const saveAutomation = vi.fn();
let automation: FactoriesFactoryCustomAutomation = {
  automation: { id: "automation", name: "Repository review" },
  settings: {
    startMode: "START_MODE_MANUAL",
    agent: { component: "runnerClaudeCode" },
    steps: [
      { name: "Lint", type: "TYPE_BASH", command: "make lint", workingDirectory: "repo", secrets: ["deploy"] },
      { name: "Review", type: "TYPE_PROMPT", prompt: "Review the repository.", workingDirectory: "repo" },
    ],
  },
};

vi.mock("@/hooks/useFactoryCustomAutomationData", () => ({
  useFactoryCustomAutomation: () => ({ data: automation }),
  useCreateFactoryCustomAutomation: () => ({ mutateAsync: saveAutomation, isPending: false }),
  useUpdateFactoryCustomAutomation: () => ({ mutateAsync: saveAutomation, isPending: false }),
}));
vi.mock("@/hooks/useComponentData", () => ({ useComponent: () => ({}) }));
vi.mock("@/hooks/useByokRunnerModelDefault", () => ({ useByokRunnerModelDefault: () => undefined }));
vi.mock("@/hooks/useExperimentalFeature", () => ({ useExperimentalFeature: () => ({ has: () => false }) }));
vi.mock("@/hooks/useFactoryAgentResources", () => ({ useFactoryAgentResources: () => ({}) }));
vi.mock("@/hooks/useSecrets", () => ({ useSecrets: () => ({ data: [] }) }));
vi.mock("@/contexts/usePermissions", () => ({ usePermissions: () => ({ canAct: () => false }) }));
vi.mock("@monaco-editor/react", () => ({ Editor: () => null }));

function Editor({ columnKey, onClose = () => {} }: { columnKey?: "backlog" | "verify"; onClose?: () => void }) {
  const model = useCustomAutomationSetupPage({
    organizationId: "org",
    factoryId: "factory",
    factoryKey: "workspace",
    automationId: "automation",
  });
  return (
    <ThemeProvider>
      <CustomAutomationSetupForm
        name={model.name}
        icon={model.icon}
        triggers={model.triggers}
        runsOnTask={model.runsOnTask}
        columnKey={columnKey}
        draft={model.draft}
        organizationId="org"
        factoryId="factory"
        factoryKey="workspace"
        saving={model.saving}
        deleting={false}
        error={model.saveError}
        onNameChange={model.setName}
        onIconChange={model.setIcon}
        onTriggersChange={model.setTriggers}
        onRunsOnTaskChange={model.setRunsOnTask}
        onDraftChange={model.setDraft}
        onSave={() => void model.save()}
        onClose={onClose}
      />
    </ThemeProvider>
  );
}

it("selects Schedule, recovers from a save error, and reopens the saved schedule without changing steps", async () => {
  const user = userEvent.setup();
  saveAutomation.mockRejectedValueOnce(new Error("Schedule has no future run."));
  saveAutomation.mockResolvedValueOnce({});
  const view = render(<Editor />);
  expect(screen.getByRole("heading", { name: "Agent" })).toBeInTheDocument();
  expect(screen.getByText("The model used for this automation.")).toBeInTheDocument();
  expect(screen.getByText("Do not run workspace setup commands.")).toBeInTheDocument();
  expect(screen.getByText("Do not check out the repository.")).toBeInTheDocument();
  const skipSetup = screen.getByRole("switch", { name: "Skip setup commands" });
  const skipCheckout = screen.getByRole("switch", { name: "Skip repository checkout" });
  expect(skipSetup).not.toBeChecked();
  expect(skipCheckout).not.toBeChecked();
  await user.click(skipSetup);
  await user.click(skipCheckout);
  expect(saveAutomation).not.toHaveBeenCalled();
  await user.click(screen.getByRole("tab", { name: "with no task" }));
  await user.click(screen.getByRole("button", { name: "Add trigger" }));
  await user.click(screen.getByRole("menuitem", { name: /Schedule/ }));
  await user.click(screen.getByRole("button", { name: "Remove Manual" }));
  expect(screen.getByLabelText("Cron expression")).toHaveValue("0 9 * * *");
  fireEvent.change(screen.getByLabelText("Cron expression"), { target: { value: "0 9 31 2 *" } });
  await user.click(screen.getByRole("combobox", { name: "Timezone" }));
  await user.click(screen.getByRole("option", { name: "GMT+5:45 (Nepal)" }));
  await user.click(screen.getByRole("button", { name: "Save" }));
  expect(await screen.findByRole("alert")).toHaveTextContent("Schedule has no future run.");
  expect(screen.getByLabelText("Cron expression")).toHaveValue("0 9 31 2 *");
  fireEvent.change(screen.getByLabelText("Cron expression"), { target: { value: "30 0 9 * * MON-FRI" } });
  await user.click(screen.getByRole("button", { name: "Save" }));
  await waitFor(() => expect(saveAutomation).toHaveBeenCalledTimes(2));
  const saved = saveAutomation.mock.calls[1]![0];
  expect(saved.settings.schedule).toEqual({ cronExpression: "30 0 9 * * MON-FRI", utcOffset: 5.75 });
  expect(saved.settings.runsOnTask).toBe(false);
  expect(saved.settings.skipSetupCommands).toBe(true);
  expect(saved.settings.skipRepositoryCheckout).toBe(true);
  expect(saved.settings.steps).toEqual(automation.settings!.steps);
  view.unmount();
  automation = { automation: automation.automation, settings: saved.settings };
  render(<Editor />);
  expect(screen.getByRole("switch", { name: "Skip setup commands" })).toBeChecked();
  await user.click(screen.getByRole("switch", { name: "Skip setup commands" }));
  expect(screen.queryByLabelText("Cron expression")).not.toBeInTheDocument();
  await user.click(screen.getByRole("button", { name: /^Schedule/ }));
  expect(screen.getByLabelText("Cron expression")).toHaveValue("30 0 9 * * MON-FRI");
  expect(screen.getByRole("combobox", { name: "Timezone" })).toHaveTextContent("GMT+5:45 (Nepal)");
  await user.click(screen.getByRole("combobox", { name: "Timezone" }));
  await user.click(screen.getByRole("option", { name: "GMT-3:30 (Newfoundland)" }));
  await user.click(screen.getByRole("button", { name: "Save" }));
  await waitFor(() => expect(saveAutomation).toHaveBeenCalledTimes(3));
  expect(saveAutomation.mock.calls[2]![0].settings.schedule.utcOffset).toBe(-3.5);
  expect(saveAutomation.mock.calls[2]![0].settings.skipSetupCommands).toBe(false);
  expect(saveAutomation.mock.calls[2]![0].settings.skipRepositoryCheckout).toBe(true);
  expect(screen.getByRole("button", { name: /Bash Lint/ })).toBeInTheDocument();
  expect(screen.getByRole("button", { name: /Prompt Review/ })).toBeInTheDocument();
});

function flowRows() {
  return screen.getAllByTestId("custom-automation-step-row");
}

function startStepDrag(position: number) {
  flowRows().forEach((row, index) => {
    row.getBoundingClientRect = () => new DOMRect(0, index * 100, 300, 100);
  });
  fireEvent.mouseDown(within(flowRows()[position]).getByRole("button", { name: /Reorder/ }), { button: 0 });
}

it("previews non-adjacent moves, preserves selection and step contents, and reopens the saved order", async () => {
  const originalSteps = [
    { name: "Lint", type: "TYPE_BASH" as const, command: "make lint", workingDirectory: "repo", secrets: ["deploy"] },
    { name: "Review", type: "TYPE_PROMPT" as const, prompt: "Review the repository.", workingDirectory: "repo" },
    { name: "Lint", type: "TYPE_BASH" as const, command: "make build", workingDirectory: "build", secrets: ["build"] },
    { name: "Publish", type: "TYPE_PROMPT" as const, prompt: "Publish the report.", workingDirectory: "reports" },
  ];
  automation = {
    automation: { id: "automation", name: "Repository review" },
    settings: { startMode: "START_MODE_MANUAL", agent: { component: "runnerClaudeCode" }, steps: originalSteps },
  };
  saveAutomation.mockClear();
  saveAutomation.mockResolvedValue({});
  const user = userEvent.setup();
  const view = render(<Editor />);
  const flow = within(flowRows()[0].parentElement!);
  expect(flow.queryByRole("button", { name: "Move step up" })).not.toBeInTheDocument();
  await user.click(screen.getByRole("button", { name: /Bash Lint make lint/ }));
  startStepDrag(0);
  fireEvent.mouseMove(window, { clientY: 350 });
  expect(flowRows().map((row) => row.textContent)).toEqual([
    expect.stringContaining("Review"),
    expect.stringContaining("make build"),
    expect.stringContaining("Publish"),
    expect.stringContaining("make lint"),
  ]);
  expect(screen.getByLabelText("Name")).toHaveValue("Lint");
  expect(within(flowRows()[3]).getByRole("button", { pressed: true })).toHaveTextContent("make lint");
  expect(saveAutomation).not.toHaveBeenCalled();
  fireEvent.mouseUp(window);
  fireEvent.change(screen.getByLabelText("Name"), { target: { value: "Check lint" } });
  startStepDrag(0);
  fireEvent.mouseMove(window, { clientY: 350 });
  expect(screen.getByLabelText("Name")).toHaveValue("Check lint");
  expect(within(flowRows()[2]).getByRole("button", { pressed: true })).toHaveTextContent("Check lint");
  fireEvent.mouseUp(window);
  startStepDrag(2);
  fireEvent.mouseMove(window, { clientY: 50 });
  fireEvent.mouseUp(window);
  expect(screen.getByLabelText("Name")).toHaveValue("Check lint");
  await user.click(screen.getByRole("button", { name: "Move step down" }));
  startStepDrag(1);
  fireEvent.mouseUp(window);
  await user.click(screen.getByRole("button", { name: "Save" }));
  await waitFor(() => expect(saveAutomation).toHaveBeenCalledTimes(1));
  const saved = saveAutomation.mock.calls[0]![0];
  const expectedSteps = [
    originalSteps[2],
    { ...originalSteps[0], name: "Check lint" },
    originalSteps[3],
    originalSteps[1],
  ];
  expect(saved.settings.steps).toEqual(expectedSteps);
  view.unmount();
  automation = { automation: automation.automation, settings: saved.settings };
  render(<Editor />);
  expect(flowRows().map((row) => within(row).getByRole("button", { name: /^(Bash|Prompt)/ }).textContent)).toEqual([
    expect.stringContaining("make build"),
    expect.stringContaining("Check lint"),
    expect.stringContaining("Publish"),
    expect.stringContaining("Review"),
  ]);
});

it("discards an unsaved reorder on Cancel and supports adding and removing the last step", async () => {
  automation = {
    automation: { id: "automation", name: "Repository review" },
    settings: {
      startMode: "START_MODE_MANUAL",
      steps: [
        { name: "Lint", type: "TYPE_BASH", command: "make lint" },
        { name: "Review", type: "TYPE_PROMPT", prompt: "Review the repository." },
      ],
    },
  };
  saveAutomation.mockClear();
  const onClose = vi.fn();
  const user = userEvent.setup();
  const view = render(<Editor onClose={onClose} />);
  startStepDrag(0);
  fireEvent.mouseMove(window, { clientY: 150 });
  fireEvent.mouseUp(window);
  await user.click(screen.getByRole("button", { name: "Cancel" }));
  expect(onClose).toHaveBeenCalledTimes(1);
  expect(saveAutomation).not.toHaveBeenCalled();
  view.unmount();
  render(<Editor />);
  expect(flowRows()[0]).toHaveTextContent("make lint");
  while (screen.queryAllByTestId("custom-automation-step-row").length > 1) {
    await user.click(within(flowRows()[0]).getByRole("button", { name: "Remove step" }));
  }
  expect(screen.queryByRole("button", { name: /Reorder/ })).not.toBeInTheDocument();
  await user.click(within(flowRows()[0]).getByRole("button", { name: "Remove step" }));
  expect(screen.queryAllByTestId("custom-automation-step-row")).toHaveLength(0);
  await user.click(screen.getByRole("button", { name: "Add step" }));
  expect(screen.getByLabelText("Name")).toHaveValue("New step");
});

it("saves a schedule beside the existing trigger", async () => {
  automation = {
    automation: { id: "automation", name: "Repository review" },
    settings: {
      startMode: "START_MODE_MANUAL",
      agent: { component: "runnerClaudeCode" },
      steps: automation.settings!.steps,
    },
  };
  const user = userEvent.setup();
  saveAutomation.mockClear();
  saveAutomation.mockResolvedValueOnce({});
  render(<Editor columnKey="backlog" />);
  await user.click(screen.getByRole("tab", { name: "with no task" }));
  await user.click(screen.getByRole("button", { name: "Add trigger" }));
  await user.click(screen.getByRole("menuitem", { name: /Schedule/ }));
  expect(screen.queryByRole("combobox", { name: /Trigger/ })).not.toBeInTheDocument();
  fireEvent.change(screen.getByLabelText("Cron expression"), { target: { value: "0 18 * * *" } });
  await user.click(screen.getByRole("button", { name: "Save" }));
  await waitFor(() => expect(saveAutomation).toHaveBeenCalled());
  const saved = saveAutomation.mock.calls.at(-1)![0];
  expect(saved.settings.triggers).toEqual([
    { startMode: "START_MODE_MANUAL" },
    {
      startMode: "START_MODE_SCHEDULED",
      schedule: { cronExpression: "0 18 * * *", utcOffset: expect.any(Number) },
    },
  ]);
  expect(saved.settings.steps).toEqual(automation.settings!.steps);
  expect(saved.settings.runsOnTask).toBe(false);
});

it("previews Runs on a task above the trigger list without saving the switch", async () => {
  const user = userEvent.setup();
  saveAutomation.mockClear();
  saveAutomation.mockResolvedValueOnce({});
  render(<Editor columnKey="backlog" />);
  expect(screen.getByRole("tab", { name: "on a task" })).toHaveAttribute("aria-selected", "true");
  await user.click(screen.getByRole("button", { name: "Add trigger" }));
  expect(screen.getByRole("menuitem", { name: /When a task enters Backlog/ })).toBeInTheDocument();
  expect(screen.queryByRole("menuitem", { name: /Schedule/ })).not.toBeInTheDocument();
  await user.keyboard("{Escape}");
  await user.click(screen.getByRole("tab", { name: "with no task" }));
  await user.click(screen.getByRole("button", { name: "Add trigger" }));
  expect(screen.getByRole("menuitem", { name: /Schedule/ })).toBeInTheDocument();
  expect(screen.queryByRole("menuitem", { name: /When a task enters Backlog/ })).not.toBeInTheDocument();
  await user.keyboard("{Escape}");
  await user.click(screen.getByRole("button", { name: "Save" }));
  await waitFor(() => expect(saveAutomation).toHaveBeenCalled());
  expect(saveAutomation.mock.calls.at(-1)![0].settings.runsOnTask).toBe(false);
});

it("offers Schedule for a Backlog automation", async () => {
  const user = userEvent.setup();
  render(<Editor columnKey="backlog" />);
  expect(screen.queryByLabelText("Cron expression")).not.toBeInTheDocument();
  await user.click(screen.getByRole("tab", { name: "with no task" }));
  await user.click(screen.getByRole("button", { name: "Add trigger" }));
  await user.click(screen.getByRole("menuitem", { name: /Schedule/ }));
  expect(screen.getByText("A person starts each run from the column.")).toBeInTheDocument();
  expect(screen.getByLabelText("Cron expression")).toBeInTheDocument();
  expect(screen.queryByRole("link", { name: "Build a cron expression" })).not.toBeInTheDocument();
  expect(screen.getByRole("combobox", { name: "Timezone" })).toBeInTheDocument();
  expect(screen.queryByText(/empty result creates no task/)).not.toBeInTheDocument();
  expect(screen.queryByText(/five fields/)).not.toBeInTheDocument();
  expect(screen.queryByText(/daylight saving time/)).not.toBeInTheDocument();
});
