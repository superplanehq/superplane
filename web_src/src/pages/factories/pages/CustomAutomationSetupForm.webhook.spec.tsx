import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { expect, it, vi } from "bun:test";
import type { FactoriesFactoryCustomAutomation } from "@/api-client";
import { FEATURE_FACTORY_TASK_WEBHOOK_TRIGGERS } from "@/lib/experimentalFeatures";
import { ThemeProvider } from "@/contexts/ThemeProvider";
import { CustomAutomationSetupForm } from "./CustomAutomationSetupForm";
import { useCustomAutomationSetupPage } from "./useCustomAutomationSetupPage";

const saveAutomation = vi.fn();
const automation: FactoriesFactoryCustomAutomation = {
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
vi.mock("@/hooks/useComponentData", () => ({ useComponent: () => ({}) }));
vi.mock("@monaco-editor/react", () => ({ Editor: () => null }));
vi.mock("@/hooks/useByokRunnerModelDefault", () => ({ useByokRunnerModelDefault: () => undefined }));
const enabledFeatures = new Set<string>();
vi.mock("@/hooks/useExperimentalFeature", () => ({
  useExperimentalFeature: () => ({ has: (feature: string) => enabledFeatures.has(feature) }),
}));
vi.mock("@/hooks/useFactoryAgentResources", () => ({ useFactoryAgentResources: () => ({}) }));
vi.mock("@/hooks/useIntegrations", () => ({ useConnectedIntegrations: () => ({}) }));
vi.mock("@/contexts/usePermissions", () => ({ usePermissions: () => ({ canAct: () => false }) }));
vi.mock("@/hooks/useSecrets", () => ({ useSecrets: () => ({ data: [] }) }));
vi.mock("@/hooks/useFactoryWebhooks", () => ({
  useFactoryWebhooks: () => ({
    data: [{ id: "webhook-1", name: "Jira issues", url: "https://superplane.test/api/v1/webhooks/webhook-1" }],
  }),
}));

function Editor({ columnKey = "backlog" }: { columnKey?: "backlog" | "implement" }) {
  const model = useCustomAutomationSetupPage({
    organizationId: "org",
    factoryId: "factory",
    factoryKey: "workspace",
    automationId: "automation",
  });
  return (
    <MemoryRouter>
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
          onRunsOnTaskChange={(runsOnTask) => model.setRunsOnTask(runsOnTask ?? true)}
          onDraftChange={model.setDraft}
          onSave={() => void model.save()}
          onClose={() => {}}
        />
      </ThemeProvider>
    </MemoryRouter>
  );
}

it("saves a Backlog webhook trigger with the selected webhook", async () => {
  const user = userEvent.setup();
  saveAutomation.mockClear();
  saveAutomation.mockResolvedValueOnce({});
  render(<Editor />);
  await user.click(screen.getByRole("button", { name: "Add trigger" }));
  expect(screen.queryByRole("menuitem", { name: /Webhook/ })).not.toBeInTheDocument();
  await user.keyboard("{Escape}");
  await user.click(screen.getByRole("tab", { name: "with no task" }));
  await user.click(screen.getByRole("button", { name: "Add trigger" }));
  await user.click(screen.getByRole("menuitem", { name: /Webhook/ }));
  await user.click(screen.getByRole("button", { name: "Save" }));
  expect(await screen.findByText("Select a webhook for each webhook trigger.")).toBeInTheDocument();
  expect(saveAutomation).not.toHaveBeenCalled();

  await user.click(screen.getByRole("combobox", { name: "Webhook" }));
  await user.click(screen.getByRole("option", { name: "Jira issues" }));
  expect(screen.getByText("https://superplane.test/api/v1/webhooks/webhook-1")).toBeInTheDocument();
  await user.click(screen.getByRole("button", { name: "Save" }));
  await waitFor(() => expect(saveAutomation).toHaveBeenCalled());
  const saved = saveAutomation.mock.calls.at(-1)![0];
  expect(saved.settings.triggers).toEqual([
    { startMode: "START_MODE_MANUAL" },
    { startMode: "START_MODE_WEBHOOK", webhookId: "webhook-1" },
  ]);
  expect(saved.settings.runsOnTask).toBe(false);
});

it("shows a task webhook on Implement when the flag is on", async () => {
  const user = userEvent.setup();
  enabledFeatures.add(FEATURE_FACTORY_TASK_WEBHOOK_TRIGGERS);
  saveAutomation.mockClear();
  saveAutomation.mockResolvedValueOnce({});
  render(<Editor columnKey="implement" />);
  await user.click(screen.getByRole("button", { name: "Add trigger" }));
  await user.click(screen.getByRole("menuitem", { name: /Webhook/ }));
  expect(screen.getByText("Runs for each matched task.")).toBeInTheDocument();
  expect(screen.getByRole("combobox", { name: "Find task by" })).toBeInTheDocument();
  expect(screen.getByLabelText("Field path")).toHaveValue("task");

  await user.click(screen.getByRole("combobox", { name: "Webhook" }));
  await user.click(screen.getByRole("option", { name: "Jira issues" }));
  await user.click(screen.getByRole("button", { name: "Save" }));
  await waitFor(() => expect(saveAutomation).toHaveBeenCalled());
  const saved = saveAutomation.mock.calls.at(-1)![0];
  expect(saved.settings.runsOnTask).toBe(true);
  expect(saved.settings.triggers).toEqual(
    expect.arrayContaining([
      expect.objectContaining({
        startMode: "START_MODE_WEBHOOK",
        webhookId: "webhook-1",
        taskMatch: { kind: "KIND_TASK", fieldPath: "task" },
      }),
    ]),
  );
  enabledFeatures.delete(FEATURE_FACTORY_TASK_WEBHOOK_TRIGGERS);
});
