import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { expect, it, vi } from "bun:test";
import type { FactoriesFactoryCustomAutomation } from "@/api-client";
import { ThemeProvider } from "@/contexts/ThemeProvider";
import type { CustomAutomationColumn } from "../lib/customAutomation";
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
let current = automation;

vi.mock("@/hooks/useFactoryCustomAutomationData", () => ({
  useFactoryCustomAutomation: () => ({ data: current, isFetchedAfterMount: true }),
  useCreateFactoryCustomAutomation: () => ({ mutateAsync: saveAutomation, isPending: false }),
  useUpdateFactoryCustomAutomation: () => ({ mutateAsync: saveAutomation, isPending: false }),
}));
vi.mock("@/hooks/useComponentData", () => ({ useComponent: () => ({}) }));
vi.mock("@monaco-editor/react", () => ({ Editor: () => null }));
vi.mock("@/hooks/useByokRunnerModelDefault", () => ({ useByokRunnerModelDefault: () => undefined }));
vi.mock("@/hooks/useExperimentalFeature", () => ({ useExperimentalFeature: () => ({ has: () => false }) }));
vi.mock("@/hooks/useFactoryAgentResources", () => ({ useFactoryAgentResources: () => ({}) }));
vi.mock("@/hooks/useIntegrations", () => ({ useConnectedIntegrations: () => ({}) }));
vi.mock("@/contexts/usePermissions", () => ({ usePermissions: () => ({ canAct: () => false }) }));
vi.mock("@/hooks/useSecrets", () => ({ useSecrets: () => ({ data: [] }) }));
vi.mock("@/hooks/useFactoryWebhooks", () => ({
  useFactoryWebhooks: () => ({
    data: [{ id: "webhook-1", name: "Jira issues", url: "https://superplane.test/api/v1/webhooks/webhook-1" }],
  }),
}));

function Editor({ columnKey = "backlog" }: { columnKey?: CustomAutomationColumn }) {
  const model = useCustomAutomationSetupPage({
    organizationId: "org",
    factoryId: "factory",
    factoryKey: "workspace",
    automationId: "automation",
    columnKey,
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

async function selectWebhook(user: ReturnType<typeof userEvent.setup>) {
  await user.click(screen.getByRole("combobox", { name: "Webhook" }));
  await user.click(screen.getByRole("option", { name: "Jira issues" }));
}

it("saves a Backlog webhook trigger with the selected webhook", async () => {
  const user = userEvent.setup();
  current = automation;
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

  await selectWebhook(user);
  expect(screen.getByText("https://superplane.test/api/v1/webhooks/webhook-1")).toBeInTheDocument();
  expect(screen.getByRole("link", { name: "Manage webhooks" })).toHaveAttribute("target", "_blank");
  await user.click(screen.getByRole("button", { name: "Save" }));
  await waitFor(() => expect(saveAutomation).toHaveBeenCalled());
  const saved = saveAutomation.mock.calls.at(-1)![0];
  expect(saved.settings.triggers).toEqual([
    { startMode: "START_MODE_MANUAL" },
    { startMode: "START_MODE_WEBHOOK", webhookId: "webhook-1" },
  ]);
  expect(saved.settings.runsOnTask).toBe(false);
});

it("saves a webhook next to the line trigger on a custom stage", async () => {
  const user = userEvent.setup();
  current = {
    automation: { ...automation.automation, columnKey: "custom-qa" },
    settings: { ...automation.settings, startMode: "START_MODE_AUTOMATIC" },
  };
  saveAutomation.mockClear();
  saveAutomation.mockResolvedValueOnce({});
  render(<Editor columnKey="custom-qa" />);
  expect(screen.getByRole("button", { name: /Remove When a task enters/ })).toBeEnabled();
  await user.click(screen.getByRole("button", { name: "Add trigger" }));
  expect(screen.getAllByRole("menuitem")).toHaveLength(4);
  await user.click(screen.getByRole("menuitem", { name: /Webhook/ }));
  await selectWebhook(user);
  await user.click(screen.getByRole("button", { name: "Save" }));
  await waitFor(() => expect(saveAutomation).toHaveBeenCalled());
  const saved = saveAutomation.mock.calls.at(-1)![0];
  expect(saved.settings.triggers).toEqual([
    { startMode: "START_MODE_AUTOMATIC" },
    { startMode: "START_MODE_WEBHOOK", webhookId: "webhook-1" },
  ]);
  expect(saved.settings.runsOnTask).toBeUndefined();
});

it("saves a custom stage automation that only a webhook starts", async () => {
  const user = userEvent.setup();
  current = {
    automation: { ...automation.automation, columnKey: "custom-qa" },
    settings: { ...automation.settings, startMode: "START_MODE_AUTOMATIC" },
  };
  saveAutomation.mockClear();
  saveAutomation.mockResolvedValueOnce({});
  render(<Editor columnKey="custom-qa" />);
  await user.click(screen.getByRole("button", { name: "Add trigger" }));
  await user.click(screen.getByRole("menuitem", { name: /Webhook/ }));
  await selectWebhook(user);
  await user.click(screen.getByRole("button", { name: /Remove When a task enters/ }));
  await user.click(screen.getByRole("button", { name: "Save" }));
  await waitFor(() => expect(saveAutomation).toHaveBeenCalled());
  const saved = saveAutomation.mock.calls.at(-1)![0];
  expect(saved.settings.triggers).toEqual([{ startMode: "START_MODE_WEBHOOK", webhookId: "webhook-1" }]);
  expect(saved.settings.runsOnTask).toBeUndefined();
});

it("saves the line trigger on a task after the webhook is removed from a custom stage", async () => {
  const user = userEvent.setup();
  current = {
    automation: { ...automation.automation, columnKey: "custom-qa" },
    settings: {
      ...automation.settings,
      startMode: "START_MODE_WEBHOOK",
      runsOnTask: false,
      triggers: [{ startMode: "START_MODE_WEBHOOK", webhookId: "webhook-1" }],
    },
  };
  saveAutomation.mockClear();
  saveAutomation.mockResolvedValueOnce({});
  render(<Editor columnKey="custom-qa" />);
  await user.click(screen.getByRole("button", { name: "Add trigger" }));
  await user.click(screen.getByRole("menuitem", { name: /When a task enters/ }));
  await user.click(screen.getByRole("button", { name: /Remove Webhook/ }));
  await user.click(screen.getByRole("button", { name: "Save" }));
  await waitFor(() => expect(saveAutomation).toHaveBeenCalled());
  const saved = saveAutomation.mock.calls.at(-1)![0];
  expect(saved.settings).toMatchObject({ startMode: "START_MODE_AUTOMATIC", runsOnTask: true });
  expect(saved.settings.triggers ?? []).not.toContainEqual(
    expect.objectContaining({ startMode: "START_MODE_WEBHOOK" }),
  );
});
