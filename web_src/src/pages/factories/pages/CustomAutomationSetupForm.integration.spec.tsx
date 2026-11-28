import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { expect, it, vi } from "bun:test";
import type { FactoriesFactoryCustomAutomation } from "@/api-client";
import { ThemeProvider } from "@/contexts/ThemeProvider";
import { CustomAutomationSetupForm } from "./CustomAutomationSetupForm";
import { useCustomAutomationSetupPage } from "./useCustomAutomationSetupPage";

const saveAutomation = vi.fn();
const automation: FactoriesFactoryCustomAutomation = {
  automation: { id: "automation", name: "Move on merge", columnKey: "custom-qa" },
  settings: { startMode: "START_MODE_AUTOMATIC", agent: { component: "runnerClaudeCode" }, steps: [] },
};

vi.mock("@/hooks/useFactoryCustomAutomationData", () => ({
  useFactoryCustomAutomation: () => ({ data: automation, isFetchedAfterMount: true }),
  useCreateFactoryCustomAutomation: () => ({ mutateAsync: saveAutomation, isPending: false }),
  useUpdateFactoryCustomAutomation: () => ({ mutateAsync: saveAutomation, isPending: false }),
}));
vi.mock("@/hooks/useComponentData", () => ({ useComponent: () => ({}) }));
vi.mock("@monaco-editor/react", () => ({ Editor: () => null }));
vi.mock("@/hooks/useByokRunnerModelDefault", () => ({ useByokRunnerModelDefault: () => undefined }));
vi.mock("@/hooks/useExperimentalFeature", () => ({ useExperimentalFeature: () => ({ has: () => false }) }));
vi.mock("@/hooks/useFactoryAgentResources", () => ({ useFactoryAgentResources: () => ({}) }));
vi.mock("@/contexts/usePermissions", () => ({ usePermissions: () => ({ canAct: () => false }) }));
vi.mock("@/hooks/useSecrets", () => ({ useSecrets: () => ({ data: [] }) }));
vi.mock("@/hooks/useFactoryWebhooks", () => ({ useFactoryWebhooks: () => ({ data: [] }) }));
vi.mock("@/hooks/useIntegrations", () => ({
  useConnectedIntegrations: () => ({
    isLoading: false,
    data: [
      { metadata: { id: "github-1", name: "Acme GitHub", integrationName: "github" }, status: { state: "ready" } },
    ],
  }),
  useAvailableIntegrations: () => ({
    isLoading: false,
    data: [
      {
        name: "github",
        capabilities: [
          {
            type: "TYPE_TRIGGER",
            name: "github.onPullRequest",
            label: "On Pull Request",
            configuration: [
              {
                name: "actions",
                label: "Actions",
                type: "multi-select",
                defaultValue: '["closed"]',
                typeOptions: { multiSelect: { options: [{ label: "Closed", value: "closed" }] } },
              },
              {
                name: "onlyFactoryPullRequests",
                label: "Only factory pull requests",
                type: "boolean",
                defaultValue: "false",
              },
            ],
          },
        ],
      },
    ],
  }),
}));

function Editor() {
  const model = useCustomAutomationSetupPage({
    organizationId: "org",
    factoryId: "factory",
    factoryKey: "workspace",
    automationId: "automation",
    columnKey: "custom-qa",
  });
  return (
    <MemoryRouter>
      <ThemeProvider>
        <CustomAutomationSetupForm
          name={model.name}
          icon={model.icon}
          triggers={model.triggers}
          runsOnTask={model.runsOnTask}
          columnKey="custom-qa"
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

it("saves a merge integration trigger with a step that moves the task", async () => {
  const user = userEvent.setup();
  saveAutomation.mockResolvedValueOnce({});
  render(<Editor />);
  await user.click(screen.getByRole("button", { name: "Add trigger" }));
  await user.click(screen.getByRole("menuitem", { name: /Integration event/ }));
  await user.click(screen.getByRole("button", { name: "Save" }));
  expect(
    await screen.findByText("Select an integration and an event for each integration trigger."),
  ).toBeInTheDocument();
  expect(saveAutomation).not.toHaveBeenCalled();

  await user.click(screen.getByRole("combobox", { name: "Integration" }));
  await user.click(screen.getByRole("option", { name: "Acme GitHub" }));
  await user.click(screen.getByRole("combobox", { name: "Event" }));
  await user.click(screen.getByRole("option", { name: "On Pull Request" }));
  expect(screen.getByTestId("custom-automation-integration-merge-notice")).toBeInTheDocument();

  await user.click(screen.getByRole("button", { name: "Save" }));
  await waitFor(() => expect(saveAutomation).toHaveBeenCalled());
  const saved = saveAutomation.mock.calls.at(-1)![0];
  expect(saved.settings.triggers).toContainEqual({
    startMode: "START_MODE_INTEGRATION",
    integration: {
      integrationId: "github-1",
      trigger: "github.onPullRequest",
      configuration: { actions: ["closed"], onlyFactoryPullRequests: true },
    },
  });
  expect(saved.settings.steps).toHaveLength(1);
  expect(saved.settings.steps[0]).toMatchObject({ type: "TYPE_BASH" });
  expect(saved.settings.steps[0].command).toContain("{{ root().data.pull_request.merged }}");
  expect(saved.settings.steps[0].command).toContain(
    'superplane task move --task "{{ root().data.workOrder.key }}" --next',
  );
});
