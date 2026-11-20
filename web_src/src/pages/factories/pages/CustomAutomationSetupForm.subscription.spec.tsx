import type { FactoriesFactory, FactoriesFactoryCustomAutomation } from "@/api-client";
import type { useSelectableLLMModels } from "@/hooks/useSelectableLLMModels";
import { selectableLLMModelsFromResponse } from "@/lib/selectableLLMModels";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, expect, it, vi } from "bun:test";
import { MemoryRouter } from "react-router";

import { CustomAutomationSetupForm } from "./CustomAutomationSetupForm";
import { useCustomAutomationSetupPage } from "./useCustomAutomationSetupPage";

let automation: FactoriesFactoryCustomAutomation | undefined;
type SaveInput = { name: string; settings: FactoriesFactoryCustomAutomation["settings"] };
const saveAutomation = async (input: SaveInput) => {
  automation = { automation: { id: "automation", name: input.name }, settings: input.settings };
  return automation;
};
const createAutomation = vi.fn(saveAutomation);
const updateAutomation = vi.fn(saveAutomation);

vi.mock("@/hooks/useFactoryCustomAutomationData", () => ({
  useFactoryCustomAutomation: (_org: string, _factory: string, id?: string) => ({ data: id ? automation : undefined }),
  useCreateFactoryCustomAutomation: () => ({ mutateAsync: createAutomation, isPending: false }),
  useUpdateFactoryCustomAutomation: () => ({ mutateAsync: updateAutomation, isPending: false }),
}));
vi.mock("@/hooks/useComponentData", () => ({
  useComponent: (_org: string, component: string) => ({
    data: {
      configuration: [
        {
          name: "model",
          type: "hosted-model",
          typeOptions: { hostedModel: { provider: component === "runnerSuperPlane" ? "all" : "openai" } },
        },
      ],
    },
  }),
}));
vi.mock("@/hooks/useSelectableLLMModels", () => ({
  useSelectableLLMModels: (_org: string, options?: Parameters<typeof useSelectableLLMModels>[1]) => ({
    data:
      options?.factoryId === "factory"
        ? selectableLLMModelsFromResponse(
            [
              {
                source: { id: "subscription" },
                provider: { id: "openai" },
                model: { id: "gpt-5.4" },
                key: "subscription::openai::gpt-5.4",
              },
              {
                source: { id: "subscription" },
                provider: { id: "openai" },
                model: { id: "gpt-5.3" },
                key: "subscription::openai::gpt-5.3",
              },
              {
                source: { id: "byok" },
                provider: { id: "openai" },
                model: { id: "gpt-4o" },
                key: "byok::openai::gpt-4o",
              },
            ],
            options.sources,
          )
        : [],
    isLoading: false,
    isError: false,
  }),
}));
vi.mock("@/hooks/useOrganizationWorkspaceUsage", () => ({ useOrganizationWorkspaceUsage: () => ({}) }));
vi.mock("@/hooks/useCanvasData", () => ({ useCanvas: () => ({ data: undefined, isPending: false }) }));
vi.mock("@/hooks/useExperimentalFeature", () => ({ useExperimentalFeature: () => ({ has: () => false }) }));
vi.mock("@/hooks/useFactoryAgentResources", () => ({ useFactoryAgentResources: () => ({}) }));
vi.mock("@/hooks/useSecrets", () => ({ useSecrets: () => ({ data: [] }) }));
vi.mock("@/contexts/usePermissions", () => ({ usePermissions: () => ({ canAct: () => false }) }));
vi.mock("@/contexts/useTheme", () => ({ useTheme: () => ({ resolvedTheme: "light" }) }));
vi.mock("@monaco-editor/react", () => ({
  Editor: ({ value, onChange }: { value: string; onChange: (value: string) => void }) => (
    <textarea aria-label="Commands" value={value} onChange={(event) => onChange(event.target.value)} />
  ),
}));

beforeEach(() => {
  automation = undefined;
  createAutomation.mockClear();
  updateAutomation.mockClear();
});

function Editor({ factory, editing = false }: { factory?: FactoriesFactory; editing?: boolean }) {
  const model = useCustomAutomationSetupPage({
    organizationId: "org",
    factoryId: "factory",
    factoryKey: "workspace",
    factory,
    automationId: editing ? "automation" : undefined,
  });
  return (
    <MemoryRouter>
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
        onRunsOnTaskChange={model.setRunsOnTask}
        onDraftChange={model.setDraft}
        onSave={() => void model.save()}
        onClose={() => {}}
      />
    </MemoryRouter>
  );
}

const subscriptionFactory: FactoriesFactory = {
  onboarding: { agentHarness: "AGENT_HARNESS_SUPERPLANE", codexSubscription: true },
};

for (const delayedSubscription of [false, true]) {
  it(`creates and reopens a subscription automation (${delayedSubscription ? "updated" : "initial"} onboarding)`, async () => {
    const user = userEvent.setup();
    const view = render(
      <Editor
        factory={
          delayedSubscription
            ? { onboarding: { ...subscriptionFactory.onboarding, codexSubscription: false } }
            : subscriptionFactory
        }
      />,
    );
    if (delayedSubscription) {
      view.rerender(<Editor factory={subscriptionFactory} />);
    }
    expect(screen.getByTestId("field-model-hosted-model")).toHaveTextContent("Default");
    await user.click(screen.getByRole("button", { name: "Add step" }));
    fireEvent.change(screen.getByRole("textbox", { name: "Commands" }), { target: { value: "make lint" } });
    await user.click(screen.getByRole("button", { name: /Trigger Manual/ }));
    await user.click(screen.getByRole("button", { name: "Save" }));
    await waitFor(() => expect(createAutomation).toHaveBeenCalledTimes(1));
    expect(createAutomation.mock.calls[0]![0].settings?.agent).toEqual({
      component: "runnerCodex",
      model: "",
      credentialSource: "subscription",
    });

    await user.click(screen.getByTestId("field-model-hosted-model"));
    expect(screen.getByRole("menuitem", { name: "gpt-5.4" })).toBeInTheDocument();
    expect(screen.queryByRole("menuitem", { name: /gpt-4o/ })).not.toBeInTheDocument();
    await user.click(screen.getByRole("menuitem", { name: "gpt-5.3" }));
    await user.click(screen.getByRole("button", { name: "Save" }));
    await waitFor(() => expect(createAutomation).toHaveBeenCalledTimes(2));
    expect(createAutomation.mock.calls[1]![0].settings?.agent).toEqual({
      component: "runnerCodex",
      model: "gpt-5.3",
      credentialSource: "subscription",
    });
    view.unmount();
    render(<Editor factory={subscriptionFactory} editing />);
    expect(screen.getByTestId("field-model-hosted-model")).toHaveTextContent("gpt-5.3");
    expect(updateAutomation).not.toHaveBeenCalled();
  });
}

it("changes a saved subscription model without applying workspace agent defaults", async () => {
  automation = {
    automation: { id: "automation", name: "Review" },
    settings: {
      agent: { component: "runnerCodex", model: "gpt-5.3", credentialSource: "subscription" },
      steps: [{ name: "Review", type: "TYPE_PROMPT", prompt: "Review the change." }],
    },
  };
  const user = userEvent.setup();
  const view = render(<Editor factory={subscriptionFactory} editing />);
  view.rerender(
    <Editor factory={{ onboarding: { agentHarness: "AGENT_HARNESS_SUPERPLANE", codexSubscription: false } }} editing />,
  );
  expect(screen.getByTestId("field-model-hosted-model")).toHaveTextContent("gpt-5.3");
  await user.click(screen.getByTestId("field-model-hosted-model"));
  expect(screen.queryByRole("menuitem", { name: /gpt-4o/ })).not.toBeInTheDocument();
  await user.click(screen.getByRole("menuitem", { name: "gpt-5.4" }));
  await user.click(screen.getByRole("button", { name: "Save" }));
  await waitFor(() => expect(updateAutomation).toHaveBeenCalledTimes(1));
  expect(updateAutomation.mock.calls[0]![0].settings?.agent).toEqual({
    component: "runnerCodex",
    model: "gpt-5.4",
    credentialSource: "subscription",
  });
  view.unmount();
  render(<Editor factory={subscriptionFactory} editing />);
  expect(screen.getByTestId("field-model-hosted-model")).toHaveTextContent("gpt-5.4");
  expect(createAutomation).not.toHaveBeenCalled();
});
