import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { expect, it, vi } from "bun:test";
import type { FactoriesFactoryCustomAutomation } from "@/api-client";
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
vi.mock("./PlanningReviewForm", () => ({ PlanningReviewForm: () => null }));

function Editor({ columnKey }: { columnKey?: "backlog" | "verify" }) {
  const model = useCustomAutomationSetupPage({
    organizationId: "org",
    factoryId: "factory",
    factoryKey: "workspace",
    automationId: "automation",
  });
  return (
    <CustomAutomationSetupForm
      name={model.name}
      icon={model.icon}
      triggers={model.triggers}
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
      onDraftChange={model.setDraft}
      onSave={() => void model.save()}
      onClose={() => {}}
    />
  );
}

it("selects Schedule, recovers from a save error, and reopens the saved schedule without changing steps", async () => {
  const user = userEvent.setup();
  saveAutomation.mockRejectedValueOnce(new Error("Schedule has no future run."));
  saveAutomation.mockResolvedValueOnce({});
  const view = render(<Editor />);
  await user.click(screen.getByRole("button", { name: "Add trigger" }));
  await user.click(screen.getByRole("menuitem", { name: /Schedule/ }));
  await user.click(screen.getByRole("button", { name: "Remove Manual" }));
  expect(screen.getByLabelText("Cron expression")).toHaveValue("0 9 * * *");
  fireEvent.change(screen.getByLabelText("Cron expression"), { target: { value: "0 9 31 2 *" } });
  await user.click(screen.getByRole("combobox", { name: "Timezone" }));
  await user.click(screen.getByRole("option", { name: "GMT+5:30 (Mumbai, Delhi)" }));
  await user.click(screen.getByRole("button", { name: "Save" }));
  expect(await screen.findByRole("alert")).toHaveTextContent("Schedule has no future run.");
  expect(screen.getByLabelText("Cron expression")).toHaveValue("0 9 31 2 *");
  fireEvent.change(screen.getByLabelText("Cron expression"), { target: { value: "30 0 9 * * MON-FRI" } });
  await user.click(screen.getByRole("button", { name: "Save" }));
  await waitFor(() => expect(saveAutomation).toHaveBeenCalledTimes(2));
  const saved = saveAutomation.mock.calls[1]![0];
  expect(saved.settings.schedule).toEqual({ cronExpression: "30 0 9 * * MON-FRI", utcOffset: 5.5 });
  expect(saved.settings.steps).toEqual(automation.settings!.steps);
  view.unmount();
  automation = { automation: automation.automation, settings: saved.settings };
  render(<Editor />);
  expect(screen.queryByLabelText("Cron expression")).not.toBeInTheDocument();
  await user.click(screen.getByRole("button", { name: /^Schedule/ }));
  expect(screen.getByLabelText("Cron expression")).toHaveValue("30 0 9 * * MON-FRI");
  expect(screen.getByRole("combobox", { name: "Timezone" })).toHaveTextContent("GMT+5:30 (Mumbai, Delhi)");
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
});

it("offers Schedule for a Backlog automation", async () => {
  const user = userEvent.setup();
  render(<Editor columnKey="backlog" />);
  expect(screen.queryByLabelText("Cron expression")).not.toBeInTheDocument();
  await user.click(screen.getByRole("button", { name: "Add trigger" }));
  await user.click(screen.getByRole("menuitem", { name: /Schedule/ }));
  expect(screen.getByText("A person starts each run.")).toBeInTheDocument();
  expect(screen.getByLabelText("Cron expression")).toBeInTheDocument();
  expect(screen.queryByRole("link", { name: "Build a cron expression" })).not.toBeInTheDocument();
  expect(screen.getByRole("combobox", { name: "Timezone" })).toBeInTheDocument();
  expect(screen.queryByText(/empty result creates no task/)).not.toBeInTheDocument();
  expect(screen.queryByText(/five fields/)).not.toBeInTheDocument();
  expect(screen.queryByText(/daylight saving time/)).not.toBeInTheDocument();
});
