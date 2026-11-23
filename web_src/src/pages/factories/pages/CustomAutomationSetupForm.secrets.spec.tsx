import { render, screen, waitFor } from "@testing-library/react";
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
  useFactoryCustomAutomation: () => ({ data: automation, isFetchedAfterMount: true }),
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
vi.mock("@/hooks/useSecrets", () => ({
  useSecrets: () => ({
    data: [{ metadata: { name: "figma-api-key" } }, { metadata: { name: "deploy" } }],
    isLoading: false,
  }),
}));

function Editor() {
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
  );
}

it("searches prompt and bash secrets and preserves selections through save and reopen", async () => {
  const user = userEvent.setup();
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
});
