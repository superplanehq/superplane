import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeAll, describe, expect, it, vi } from "bun:test";
import type { ConfigurationField } from "@/api-client";
import { useSelectableLLMModels } from "@/hooks/useSelectableLLMModels";
import type { SelectableLLMModel, SelectableLLMSourceID } from "@/lib/selectableLLMModels";
import { HostedModelFieldRenderer } from "./HostedModelFieldRenderer";

vi.mock("@/hooks/useSelectableLLMModels", () => ({ useSelectableLLMModels: vi.fn() }));
vi.mock("@/hooks/useOrganizationWorkspaceUsage", () => ({
  useOrganizationWorkspaceUsage: () => ({
    data: { defaultHostedProvider: "openai", defaultHostedModel: "permitted" },
    isLoading: false,
  }),
}));
vi.mock("@/hooks/useCanvasData", () => ({ useCanvas: () => ({ data: undefined, isPending: false }) }));

beforeAll(() => {
  Element.prototype.hasPointerCapture ??= () => false;
  Element.prototype.setPointerCapture ??= () => {};
  Element.prototype.releasePointerCapture ??= () => {};
  Element.prototype.scrollIntoView ??= () => {};
});

function createField(provider = "anthropic"): ConfigurationField {
  return {
    name: "model",
    label: "Model",
    type: "hosted-model",
    placeholder: "Select a model",
    typeOptions: { hostedModel: { provider } },
  };
}

function createSuperPlaneField() {
  return createField("all");
}

function selectableModel(source: SelectableLLMSourceID, provider: string, id: string): SelectableLLMModel {
  return {
    source: { id: source, name: source },
    provider: { id: provider, name: provider },
    model: { id, name: id },
    key: `${source}::${provider}::${id}`,
    label: source === "subscription" ? id : `${provider}/${id}`,
  };
}

function mockSelectableModels(models: SelectableLLMModel[]) {
  vi.mocked(useSelectableLLMModels).mockImplementation(
    (_organizationId, options) =>
      ({
        data: models.filter((model) => options?.sources?.includes(model.source.id as SelectableLLMSourceID)),
        isLoading: false,
        isError: false,
      }) as unknown as ReturnType<typeof useSelectableLLMModels>,
  );
}

function renderField(element: React.ReactElement) {
  return render(<MemoryRouter>{element}</MemoryRouter>);
}

describe("subscription model picker", () => {
  it("shows Default for an unset model and saves an explicit subscription selection", async () => {
    mockSelectableModels([
      selectableModel("subscription", "openai", "gpt-5.4"),
      selectableModel("subscription", "openai", "gpt-5.3"),
      selectableModel("byok", "openai", "gpt-4o"),
    ]);
    const onChange = vi.fn();
    const field = createField("openai");
    const picker = (value: string) => (
      <MemoryRouter>
        <HostedModelFieldRenderer
          field={field}
          value={value}
          onChange={onChange}
          organizationId="org-1"
          factoryId="factory-1"
          modelOnly
          allValues={{ credentials: { source: "subscription" } }}
        />
      </MemoryRouter>
    );
    const view = render(picker(""));
    expect(useSelectableLLMModels).toHaveBeenCalledWith("org-1", {
      factoryId: "factory-1",
      sources: ["subscription"],
      enabled: true,
    });
    expect(screen.getByTestId("field-model-hosted-model")).toHaveTextContent("Default");
    expect(screen.getByTestId("field-model-hosted-model")).not.toHaveTextContent("gpt-5.4");
    expect(onChange).not.toHaveBeenCalled();
    const user = userEvent.setup();
    await user.click(screen.getByTestId("field-model-hosted-model"));
    expect(screen.getByRole("menuitem", { name: "gpt-5.4" })).toBeInTheDocument();
    expect(screen.getByRole("menuitem", { name: "gpt-5.3" })).toBeInTheDocument();
    expect(screen.queryByRole("menuitem", { name: /gpt-4o/ })).not.toBeInTheDocument();
    await user.click(screen.getByRole("menuitem", { name: "gpt-5.3" }));
    expect(onChange).toHaveBeenCalledWith("gpt-5.3");
    view.rerender(picker("gpt-5.3"));
    expect(screen.getByTestId("field-model-hosted-model")).toHaveTextContent("gpt-5.3");
  });
});

describe("workspace policy replacement", () => {
  for (const scenario of [
    {
      source: "hosted",
      field: createSuperPlaneField(),
      excluded: "hosted::openai::excluded",
      allowed: "hosted::openai::permitted",
      provider: "openai",
      id: "permitted",
      label: "openai/permitted",
    },
    {
      source: "byok",
      field: createField(),
      excluded: "claude-sonnet-4-6",
      allowed: "claude-opus-5-5",
      provider: "anthropic",
      id: "claude-opus-5-5",
      label: "opus 5-5",
    },
    {
      source: "subscription",
      field: createField("openai"),
      excluded: "native-excluded",
      allowed: "native-permitted",
      provider: "openai",
      id: "native-permitted",
      label: "native-permitted",
    },
  ] as const) {
    it(`requires replacement of an excluded saved ${scenario.source} model`, async () => {
      mockSelectableModels([selectableModel(scenario.source, scenario.provider, scenario.id)]);
      const onChange = vi.fn();
      const user = userEvent.setup();
      renderField(
        <HostedModelFieldRenderer
          field={scenario.field}
          value={scenario.excluded}
          onChange={onChange}
          organizationId="org-1"
          modelOnly
          allValues={{ credentials: { source: scenario.source } }}
        />,
      );
      expect(screen.getByTestId("field-model-hosted-model")).toHaveTextContent("unavailable");
      expect(screen.getByRole("alert")).toHaveTextContent("Select an available model to replace the saved model.");
      expect(onChange).not.toHaveBeenCalled();
      await user.click(screen.getByTestId("field-model-hosted-model"));
      expect(screen.queryByRole("menuitem", { name: scenario.excluded })).not.toBeInTheDocument();
      await user.click(screen.getByRole("menuitem", { name: scenario.label }));
      expect(onChange).toHaveBeenCalledWith(scenario.allowed);
    });
  }
});
