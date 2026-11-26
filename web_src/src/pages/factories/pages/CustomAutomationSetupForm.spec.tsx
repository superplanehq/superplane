import { load } from "js-yaml";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { expect, it, vi } from "bun:test";
import type { FactoriesFactoryCustomAutomation, FactoriesFactoryOnboarding } from "@/api-client";
import type { SelectableLLMSourceID } from "@/lib/selectableLLMModels";
import { Editor, SourceEditor } from "./customAutomationEditorTestSupport";

const saveAutomation = vi.fn();
let sharedHostedId = false;
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
  useFactoryCustomAutomation: () => ({ data: automation, isFetchedAfterMount: true }),
  useCreateFactoryCustomAutomation: () => ({ mutateAsync: saveAutomation, isPending: false }),
  useUpdateFactoryCustomAutomation: () => ({ mutateAsync: saveAutomation, isPending: false }),
}));
vi.mock("@/hooks/useComponentData", () => ({
  useComponent: (_organization: string, component: string) => {
    const providers: Record<string, string> = {
      runnerSuperPlane: "all",
      runnerCodex: "openai",
      runnerClaudeCode: "anthropic",
    };
    const provider = providers[component];
    const model = { name: "model", type: "hosted-model", typeOptions: { hostedModel: { provider } } };
    return { data: { configuration: [model] } };
  },
}));
vi.mock("@/hooks/useCanvasData", () => ({ useCanvas: () => ({}) }));
vi.mock("@/hooks/useOrganizationWorkspaceUsage", () => ({ useOrganizationWorkspaceUsage: () => ({}) }));
vi.mock("@/hooks/useSelectableLLMModels", () => ({
  useSelectableLLMModels: (_organization: string, { sources }: { sources: SelectableLLMSourceID[] }) => ({
    data: sources.flatMap((source) => {
      const id = source === "hosted" ? "anthropic/claude-test" : source + "-model";
      const providers = source === "hosted" ? (sharedHostedId ? ["openai", "openrouter"] : ["openrouter"]) : ["openai"];
      return providers.map((provider) => ({
        provider: { id: provider },
        model: { id, name: id },
        key: source + "::" + provider + "::" + id,
        label: provider + "/" + id,
      }));
    }),
  }),
}));
vi.mock("@/hooks/useByokRunnerModelDefault", () => ({ useByokRunnerModelDefault: () => undefined }));
vi.mock("@/hooks/useExperimentalFeature", () => ({ useExperimentalFeature: () => ({ has: () => false }) }));
vi.mock("@/hooks/useFactoryAgentResources", () => ({ useFactoryAgentResources: () => ({}) }));
vi.mock("@/hooks/useSecrets", () => ({ useSecrets: () => ({ data: [] }) }));
vi.mock("@/contexts/usePermissions", () => ({ usePermissions: () => ({ canAct: () => false }) }));
vi.mock("@monaco-editor/react", () => ({ Editor: () => null }));

vi.mock("@/hooks/useIntegrations", () => ({ useConnectedIntegrations: () => ({}) }));
let onboarding: FactoriesFactoryOnboarding | undefined;
vi.mock("../layout/factoriesLayoutContext", () => ({
  useFactoriesLayout: () => ({
    organizationId: "org",
    factoryId: "factory",
    routeSegment: "workspace",
    factory: { onboarding },
  }),
}));

it.each([
  { source: "hosted", initialModel: "", model: "anthropic/claude-test", label: "openrouter/anthropic/claude-test" },
  {
    source: "hosted",
    initialModel: "hosted::openrouter::anthropic/claude-test",
    model: "anthropic/claude-test",
    label: "openrouter/anthropic/claude-test",
  },
  { source: "subscription", initialModel: "", model: "subscription-model", label: "subscription-model" },
] as const)("saves and clears a $source model from '$initialModel'", async ({ source, initialModel, model, label }) => {
  onboarding = source === "hosted" ? { agentHarness: "AGENT_HARNESS_SUPERPLANE" } : { codexSubscription: true };
  const yaml = `apiVersion: v2
kind: Automation
name: Review
triggers: [{use: manual}]
steps: [{name: Review, prompt: Review the changes.}]
`;
  const save = vi.fn();
  const initial = initialModel ? yaml + "agent: {model: '" + initialModel + "'}\n" : yaml;
  const view = render(<SourceEditor source={initial} save={save} />);
  const user = userEvent.setup();
  const picker = screen.getByTestId("field-model-hosted-model");
  await user.click(picker);
  expect(screen.getAllByRole("menuitem")).toHaveLength(initialModel ? 2 : 1);
  await user.click(screen.getByRole("menuitem", { name: label }));
  expect(picker).toHaveAccessibleName(label);
  await user.click(screen.getByRole("button", { name: "Save" }));
  expect(save).toHaveBeenLastCalledWith({ ...(load(yaml) as object), agent: { model } });
  view.unmount();
  render(<SourceEditor source={yaml + "agent: {model: '" + model + "'}\n"} save={save} />);
  await user.click(screen.getByRole("button", { name: label }));
  await user.click(screen.getByRole("menuitem", { name: "Use default model" }));
  await user.click(screen.getByRole("button", { name: "Save" }));
  expect(save).toHaveBeenLastCalledWith(load(yaml));
  onboarding = undefined;
});

it("keeps the selected provider when two hosted providers offer the same model ID", async () => {
  const previous = automation;
  sharedHostedId = true;
  automation = {
    settings: { agent: { component: "runnerSuperPlane", model: "hosted::openai::anthropic/claude-test" } },
  };
  const yaml = `apiVersion: v2
kind: Automation
name: Review
agent: {model: anthropic/claude-test}
triggers: [{use: manual}]
steps: [{name: Review, prompt: Review the changes.}]
`;
  const save = vi.fn();
  try {
    render(<SourceEditor source={yaml} save={save} automationId="saved" />);
    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: "openai/anthropic/claude-test" }));
    await user.click(screen.getByRole("menuitem", { name: "openrouter/anthropic/claude-test" }));
    await user.click(screen.getByRole("button", { name: "Save" }));
    expect(save).toHaveBeenLastCalledWith(
      expect.objectContaining({ agent: { model: "openrouter::anthropic/claude-test" } }),
    );
  } finally {
    automation = previous;
    sharedHostedId = false;
  }
});

it("keeps the saved automation's subscription when the workspace uses hosted models", async () => {
  onboarding = { agentHarness: "AGENT_HARNESS_SUPERPLANE" };
  const previous = automation;
  automation = { settings: { agent: { component: "runnerCodex", credentialSource: "subscription" } } };
  const yaml = `apiVersion: v2
kind: Automation
name: Review
agent: {harness: codex, credentials: {source: subscription}}
triggers: [{use: manual}]
steps: [{name: Review, prompt: Review the changes.}]
`;
  const save = vi.fn();
  render(<SourceEditor source={yaml} save={save} automationId="saved" />);
  const user = userEvent.setup();
  await user.click(screen.getByTestId("field-model-hosted-model"));
  await user.click(screen.getByRole("menuitem", { name: "subscription-model" }));
  await user.click(screen.getByRole("button", { name: "Save" }));
  expect(save).toHaveBeenLastCalledWith(
    expect.objectContaining({
      agent: {
        harness: "codex",
        credentials: { source: "subscription" },
        model: "subscription-model",
      },
    }),
  );
  onboarding = undefined;
  automation = previous;
});

it("preserves the original schedule when changing run context and editing workspace settings", async () => {
  const source = `apiVersion: v2
kind: Automation
name: Daily build
triggers:
  - use: schedule
    config: { cron: "0 9 * * MON-FRI", utc_offset: 5.5 }
  - use: schedule
    id: weekend
    config: { cron: "0 10 * * SAT", utc_offset: 0 }
steps:
  - name: Inspect
    prompt: Review the build.
skip_setup_commands: true
`;
  const save = vi.fn();
  render(<SourceEditor source={source} save={save} />);
  const user = userEvent.setup();
  await user.click(screen.getByRole("tab", { name: "on a task" }));
  await user.click(screen.getByRole("tab", { name: "with no task" }));
  const skipSetup = screen.getByRole("switch", { name: "Skip setup commands" });
  expect(skipSetup).toBeChecked();
  await user.click(skipSetup);
  await user.click(screen.getByRole("switch", { name: "Skip repository checkout" }));
  await user.click(screen.getByRole("button", { name: "Save" }));
  expect(save).toHaveBeenCalledWith(
    expect.objectContaining({
      runs_on_task: false,
      triggers: [
        { use: "schedule", config: { cron: "0 9 * * MON-FRI", utc_offset: 5.5 } },
        { id: "weekend", use: "schedule", config: { cron: "0 10 * * SAT", utc_offset: 0 } },
      ],
      skip_setup_commands: false,
      skip_repository_checkout: true,
    }),
  );
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

it("replaces custom-stage entry with Manual and reopens the saved trigger", async () => {
  const previous = automation;
  automation = {
    automation: { id: "automation", name: "Update Docs", columnKey: "custom-qa" },
    settings: { ...previous.settings, startMode: "START_MODE_AUTOMATIC", triggers: undefined, runsOnTask: true },
  };
  saveAutomation.mockClear();
  saveAutomation.mockResolvedValue({});
  const user = userEvent.setup();
  try {
    const view = render(<Editor columnKey="custom-qa" />);
    await user.click(screen.getByRole("button", { name: "Add trigger" }));
    expect(screen.getAllByRole("menuitem")).toHaveLength(2);
    await user.click(screen.getByRole("menuitem", { name: /Manual/ }));
    await user.click(screen.getByRole("button", { name: /Remove When a task enters/ }));
    await user.click(screen.getByRole("button", { name: "Save" }));
    await waitFor(() => expect(saveAutomation).toHaveBeenCalledTimes(1));
    const saved = saveAutomation.mock.calls[0]![0];
    expect(saved.settings.startMode).toBe("START_MODE_MANUAL");
    expect(saved.settings.runsOnTask).toBe(true);
    expect(saved.settings.steps).toEqual(automation.settings!.steps);
    expect(saved.settings.agent).toMatchObject(automation.settings!.agent!);
    view.unmount();
    automation = { automation: automation.automation, settings: saved.settings };
    render(<Editor columnKey="custom-qa" />);
    expect(screen.getByRole("button", { name: /^Trigger Manual/ })).toBeInTheDocument();
    expect(screen.queryByText(/When a task enters/)).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Remove Manual" })).not.toBeInTheDocument();
  } finally {
    automation = previous;
  }
});

it("saves custom-stage Manual through the source-backed visual editor", async () => {
  const source = `apiVersion: v2
kind: Automation
name: Update Docs
runs_on_task: true
triggers: [{use: task_enters_stage}]
steps: [{name: Review, prompt: Review the changes.}]
`;
  const save = vi.fn();
  const user = userEvent.setup();
  const view = render(<SourceEditor source={source} stage="custom-qa" save={save} />);
  await user.click(screen.getByRole("button", { name: "Add trigger" }));
  await user.click(screen.getByRole("menuitem", { name: /Manual/ }));
  await user.click(screen.getByRole("button", { name: /Remove When a task enters/ }));
  await user.click(screen.getByRole("button", { name: "Save" }));
  expect(save).toHaveBeenCalledWith(expect.objectContaining({ triggers: [{ use: "manual" }], runs_on_task: true }));
  view.unmount();
  render(<SourceEditor source={source.replace("task_enters_stage", "manual")} stage="custom-qa" save={save} />);
  expect(screen.getByRole("button", { name: /^Trigger Manual/ })).toBeInTheDocument();
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
