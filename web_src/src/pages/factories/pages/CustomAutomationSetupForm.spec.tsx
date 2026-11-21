import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { expect, it, vi } from "bun:test";
import type { FactoriesFactoryCustomAutomation } from "@/api-client";
import { CustomAutomationSetupForm } from "./CustomAutomationSetupForm";
import { ThemeProvider } from "@/contexts/ThemeProvider";
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
vi.mock("@monaco-editor/react", () => ({ Editor: () => null }));
vi.mock("@/hooks/useByokRunnerModelDefault", () => ({ useByokRunnerModelDefault: () => undefined }));
vi.mock("@/hooks/useExperimentalFeature", () => ({ useExperimentalFeature: () => ({ has: () => false }) }));
vi.mock("@/hooks/useFactoryAgentResources", () => ({ useFactoryAgentResources: () => ({}) }));
vi.mock("@/contexts/usePermissions", () => ({ usePermissions: () => ({ canAct: () => false }) }));
vi.mock("@/hooks/useSecrets", () => ({
  useSecrets: () => ({
    data: [{ metadata: { name: "figma-api-key" } }, { metadata: { name: "deploy" } }],
    isLoading: false,
  }),
}));

function Editor({ columnKey }: { columnKey?: "backlog" | "verify" }) {
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
        onClose={() => {}}
      />
    </ThemeProvider>
  );
}

it("searches prompt and bash secrets and preserves selections through save and reopen", async () => {
  const user = userEvent.setup();
  saveAutomation.mockClear();
  saveAutomation.mockResolvedValue({});
  let view = render(<Editor />);
  await user.click(screen.getByRole("button", { name: /Prompt Review/ }));
  await user.click(screen.getByRole("combobox", { name: "Add secret" }));
  expect(screen.queryByText("Create new secret")).not.toBeInTheDocument();
  await user.type(screen.getByPlaceholderText("Find a secret"), "unmatched");
  expect(screen.getByText("No matching secrets.")).toBeVisible();
  await user.clear(screen.getByPlaceholderText("Find a secret"));
  await user.type(screen.getByPlaceholderText("Find a secret"), "FIGMA");
  await user.keyboard("{ArrowDown}{Enter}");
  expect(screen.getByRole("button", { name: "Remove figma-api-key" })).toBeVisible();
  await user.click(screen.getByRole("combobox", { name: "Type" }));
  await user.click(screen.getByRole("option", { name: "Bash" }));
  await user.click(screen.getByRole("combobox", { name: "Add secret" }));
  expect(screen.getByPlaceholderText("Find a secret")).toHaveValue("");
  expect(screen.queryByRole("option", { name: "figma-api-key" })).not.toBeInTheDocument();
  await user.type(screen.getByPlaceholderText("Find a secret"), "DEPLOY");
  await user.click(screen.getByRole("option", { name: "deploy" }));
  await user.click(screen.getByRole("combobox", { name: "Type" }));
  await user.click(screen.getByRole("option", { name: "Prompt" }));
  await user.click(screen.getByRole("button", { name: "Save" }));
  await waitFor(() => expect(saveAutomation).toHaveBeenCalled());
  const saved = saveAutomation.mock.calls.at(-1)![0];
  expect(saved.settings.steps[1].secrets).toEqual(["figma-api-key", "deploy"]);
  view.unmount();
  automation = { automation: automation.automation, settings: saved.settings };
  view = render(<Editor />);
  await user.click(screen.getByRole("button", { name: /Prompt Review/ }));
  await user.click(screen.getByRole("button", { name: "Remove figma-api-key" }));
  await user.click(screen.getByRole("button", { name: "Save" }));
  await waitFor(() => expect(saveAutomation).toHaveBeenCalledTimes(2));
  expect(saveAutomation.mock.calls.at(-1)![0].settings.steps[1].secrets).toEqual(["deploy"]);
  view.unmount();
  saveAutomation.mockClear();
});

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
