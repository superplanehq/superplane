import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeAll, beforeEach, describe, expect, it, vi } from "bun:test";
import { useSelectableLLMModels } from "@/hooks/useSelectableLLMModels";
import { useFactoryCodexConnection } from "@/pages/factories/pages/onboarding/useFactoryCodexConnection";
import { HostedModelFieldRenderer } from "./HostedModelFieldRenderer";
import {
  createField,
  createSuperPlaneField,
  installPointerCapturePolyfills,
  mockSelectableModels,
  mockWorkspaceUsage,
  renderField,
  selectableModel,
  selectFlyoutOption,
} from "./HostedModelFieldRenderer.testHelpers";

const useCanvasMock = vi.hoisted(() =>
  vi.fn((): { data: { metadata?: { factoryId?: string } } | undefined; isPending: boolean } => ({
    data: undefined,
    isPending: false,
  })),
);

vi.mock("@/hooks/useSelectableLLMModels", () => ({
  useSelectableLLMModels: vi.fn(),
}));

vi.mock("@/hooks/useOrganizationWorkspaceUsage", () => ({
  useOrganizationWorkspaceUsage: vi.fn(),
}));

vi.mock("@/hooks/useCanvasData", () => ({
  useCanvas: useCanvasMock,
}));

vi.mock("@/pages/factories/pages/onboarding/useFactoryCodexConnection", () => ({
  useFactoryCodexConnection: vi.fn(),
}));

describe("subscription model picker", () => {
  it("shows Default for an unset model and saves an explicit subscription selection", async () => {
    vi.mocked(useFactoryCodexConnection).mockReturnValue({
      query: {
        data: {
          status: "ready",
          availableModels: ["gpt-5.4", "gpt-5.3"],
          availableModelOptions: [
            { id: "gpt-5.4", name: "gpt-5.4", thinkingLevels: ["low", "high"] },
            { id: "gpt-5.3", name: "gpt-5.3", thinkingLevels: [] },
          ],
        },
        isLoading: false,
        isError: false,
      },
    } as unknown as ReturnType<typeof useFactoryCodexConnection>);
    mockSelectableModels([selectableModel("byok", "openai", "gpt-4o")]);
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

describe("subscription default model thinking", () => {
  beforeAll(installPointerCapturePolyfills);

  it("shows the thinking levels of the catalog default model and keeps the model unset", async () => {
    vi.mocked(useFactoryCodexConnection).mockReturnValue({
      query: {
        data: {
          status: "ready",
          availableModels: ["gpt-5.4", "gpt-5.3"],
          availableModelOptions: [
            { id: "gpt-5.4", name: "gpt-5.4", thinkingLevels: ["low", "high"], isDefault: true },
            { id: "gpt-5.3", name: "gpt-5.3", thinkingLevels: [] },
          ],
        },
        isLoading: false,
        isError: false,
      },
    } as unknown as ReturnType<typeof useFactoryCodexConnection>);
    mockSelectableModels([]);
    const onValuesChange = vi.fn();

    renderField(
      <HostedModelFieldRenderer
        field={createField("openai")}
        value=""
        onChange={vi.fn()}
        onValuesChange={onValuesChange}
        organizationId="org-1"
        factoryId="factory-1"
        allValues={{ credentials: { source: "subscription" }, thinkingLevel: "high" }}
      />,
    );

    const trigger = screen.getByTestId("field-model-hosted-model");
    expect(trigger).toHaveTextContent("Default");
    expect(trigger).toHaveTextContent("High");

    const user = userEvent.setup();
    await user.click(trigger);
    await selectFlyoutOption(user, "field-model-hosted-thinking", "Low");
    expect(onValuesChange).toHaveBeenCalledWith({ model: undefined, thinkingLevel: "low" });
  });
});

describe("HostedModelFieldRenderer", () => {
  beforeAll(installPointerCapturePolyfills);

  beforeEach(() => {
    useCanvasMock.mockReset();
    useCanvasMock.mockReturnValue({ data: undefined, isPending: false });
    mockSelectableModels([
      selectableModel("byok", "anthropic", "claude-sonnet-4-6"),
      selectableModel("byok", "openai", "gpt-5"),
      selectableModel("byok", "openrouter", "moonshotai/kimi-k2.6"),
      selectableModel("hosted", "anthropic", "claude-sonnet-4-6"),
      selectableModel("hosted", "openai", "gpt-5"),
      selectableModel("hosted", "openrouter", "moonshotai/kimi-k2.6"),
    ]);
    mockWorkspaceUsage({
      data: { defaultHostedProvider: "anthropic", defaultHostedModel: "claude-sonnet-4-6" },
      isLoading: false,
    });
  });

  it("shows organization BYOK models for a provider runner without waiting for credentials", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();

    renderField(<HostedModelFieldRenderer field={createField()} value="" onChange={onChange} organizationId="org-1" />);

    expect(useSelectableLLMModels).toHaveBeenCalledWith("org-1", {
      factoryId: undefined,
      sources: ["byok"],
      enabled: true,
    });
    expect(screen.queryByRole("textbox")).not.toBeInTheDocument();

    await user.click(screen.getByTestId("field-model-hosted-model"));
    expect(screen.getByTestId("field-model-hosted-model-list")).toHaveTextContent("Model");
    expect(screen.getByTestId("field-model-hosted-thinking")).toHaveTextContent("Thinking");
    expect(screen.queryByRole("menuitem", { name: "openai/gpt-5" })).not.toBeInTheDocument();

    await selectFlyoutOption(user, "field-model-hosted-model-list", "sonnet 4-6");
    expect(onChange).toHaveBeenCalledWith("claude-sonnet-4-6");
  });

  it("shows the allowlisted Claude model for an alias without saving it", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    mockSelectableModels([selectableModel("byok", "anthropic", "claude-opus-5-5")]);

    renderField(
      <HostedModelFieldRenderer field={createField()} value="opus" onChange={onChange} organizationId="org-1" />,
    );

    expect(screen.getByTestId("field-model-hosted-model")).toHaveTextContent("opus 5-5");
    expect(onChange).not.toHaveBeenCalled();

    await user.click(screen.getByTestId("field-model-hosted-model"));
    await user.hover(screen.getByTestId("field-model-hosted-model-list"));
    expect(await screen.findByRole("menuitem", { name: "opus 5-5" })).toBeInTheDocument();
    expect(screen.queryByRole("menuitem", { name: "opus" })).not.toBeInTheDocument();
  });

  it("explains when organization BYOK models cannot load", () => {
    mockSelectableModels([], { isError: true });

    renderField(<HostedModelFieldRenderer field={createField()} value="" onChange={vi.fn()} organizationId="org-1" />);

    expect(screen.getByText("Unable to load models. Try again.")).toBeInTheDocument();
  });

  it("explains when no organization BYOK models are selected", () => {
    mockSelectableModels([]);

    renderField(<HostedModelFieldRenderer field={createField()} value="" onChange={vi.fn()} organizationId="org-1" />);

    expect(
      screen.getByText(
        "No models are selected for this provider. Select models on Organization LLM Models, or connect a provider on Integrations.",
      ),
    ).toBeInTheDocument();
  });

  it("waits for the canvas factory before it loads organization BYOK models", () => {
    useCanvasMock.mockReturnValue({ data: undefined, isPending: true });

    renderField(
      <HostedModelFieldRenderer field={createField()} value="" onChange={vi.fn()} organizationId="org-1" />,
      "/org-1/apps/canvas-1",
    );

    expect(screen.getByText("Loading models...")).toBeInTheDocument();
    expect(useSelectableLLMModels).toHaveBeenCalledWith("org-1", {
      factoryId: undefined,
      sources: ["byok"],
      enabled: false,
    });
  });

  it("uses an explicit factory when the route has no canvas", () => {
    useCanvasMock.mockReturnValue({ data: undefined, isPending: true });

    renderField(
      <HostedModelFieldRenderer
        field={createField()}
        value=""
        onChange={vi.fn()}
        organizationId="org-1"
        factoryId="factory-9"
      />,
      "/org-1/apps/canvas-1",
    );

    expect(screen.queryByText("Loading models...")).not.toBeInTheDocument();
    expect(useSelectableLLMModels).toHaveBeenCalledWith("org-1", {
      factoryId: "factory-9",
      sources: ["byok"],
      enabled: true,
    });
  });

  it("loads organization BYOK models for the canvas factory", () => {
    useCanvasMock.mockReturnValue({ data: { metadata: { factoryId: "factory-1" } }, isPending: false });

    renderField(
      <HostedModelFieldRenderer
        field={createField()}
        value="claude-sonnet-4-6"
        onChange={vi.fn()}
        organizationId="org-1"
      />,
      "/org-1/apps/canvas-1",
    );

    expect(screen.getByTestId("field-model-hosted-model")).toBeInTheDocument();
    expect(useSelectableLLMModels).toHaveBeenCalledWith("org-1", {
      factoryId: "factory-1",
      sources: ["byok"],
      enabled: true,
    });
  });

  it("lists every allowlisted SuperPlane model as provider/model", async () => {
    const user = userEvent.setup();

    renderField(
      <HostedModelFieldRenderer field={createSuperPlaneField()} value="" onChange={vi.fn()} organizationId="org-1" />,
    );

    expect(useSelectableLLMModels).toHaveBeenCalledWith("org-1", {
      factoryId: undefined,
      sources: ["hosted"],
      enabled: true,
    });
    await user.click(screen.getByTestId("field-model-hosted-model"));
    await user.hover(screen.getByTestId("field-model-hosted-model-list"));
    expect(await screen.findByRole("menuitem", { name: "anthropic/claude-sonnet-4-6" })).toBeInTheDocument();
    expect(screen.getByRole("menuitem", { name: "openai/gpt-5" })).toBeInTheDocument();
    expect(screen.getByRole("menuitem", { name: "moonshotai/kimi-k2.6" })).toBeInTheDocument();
  });

  it("selects the instance SuperPlane agent model by default", () => {
    mockWorkspaceUsage({
      data: { defaultHostedProvider: "openai", defaultHostedModel: "gpt-5" },
      isLoading: false,
    });

    const onChange = vi.fn();
    renderField(
      <HostedModelFieldRenderer field={createSuperPlaneField()} value="" onChange={onChange} organizationId="org-1" />,
    );

    expect(screen.getByTestId("field-model-hosted-model")).toHaveTextContent("openai/gpt-5");
    expect(onChange).not.toHaveBeenCalled();
  });

  it("loads the hosted union list for the SuperPlane node factory", () => {
    useCanvasMock.mockReturnValue({ data: { metadata: { factoryId: "factory-1" } }, isPending: false });

    renderField(
      <HostedModelFieldRenderer field={createSuperPlaneField()} value="" onChange={vi.fn()} organizationId="org-1" />,
      "/org-1/apps/canvas-1",
    );

    expect(useSelectableLLMModels).toHaveBeenCalledWith("org-1", {
      factoryId: "factory-1",
      sources: ["hosted"],
      enabled: true,
    });
  });

  it("keeps an unset SuperPlane model when only thinking changes", async () => {
    const user = userEvent.setup();
    const onValuesChange = vi.fn();

    renderField(
      <HostedModelFieldRenderer
        field={createSuperPlaneField()}
        value=""
        onChange={vi.fn()}
        onValuesChange={onValuesChange}
        allValues={{}}
        organizationId="org-1"
      />,
    );

    expect(screen.getByTestId("field-model-hosted-model")).toHaveTextContent("anthropic/claude-sonnet-4-6");
    await user.click(screen.getByTestId("field-model-hosted-model"));
    await selectFlyoutOption(user, "field-model-hosted-thinking", "High");
    expect(onValuesChange).toHaveBeenCalledWith({ model: undefined, thinkingLevel: "high" });
    expect(screen.getByTestId("field-model-hosted-model-list")).toBeInTheDocument();
  });

  it("saves the displayed model when only thinking changes", async () => {
    const user = userEvent.setup();
    const onValuesChange = vi.fn();
    mockSelectableModels([selectableModel("byok", "anthropic", "claude-opus-5-5")]);

    renderField(
      <HostedModelFieldRenderer
        field={createField()}
        value="opus"
        onChange={vi.fn()}
        onValuesChange={onValuesChange}
        allValues={{ model: "opus" }}
        organizationId="org-1"
      />,
    );

    expect(screen.getByTestId("field-model-hosted-model")).toHaveTextContent("opus 5-5");
    await user.click(screen.getByTestId("field-model-hosted-model"));
    await selectFlyoutOption(user, "field-model-hosted-thinking", "High");
    expect(onValuesChange).toHaveBeenCalledWith({ model: "claude-opus-5-5", thinkingLevel: "high" });
  });

  it("keeps an empty BYOK model when only thinking changes", async () => {
    const user = userEvent.setup();
    const onValuesChange = vi.fn();

    renderField(
      <HostedModelFieldRenderer
        field={createField()}
        value=""
        onChange={vi.fn()}
        onValuesChange={onValuesChange}
        allValues={{}}
        organizationId="org-1"
      />,
    );

    expect(screen.getByTestId("field-model-hosted-model")).toHaveTextContent("sonnet 4-6");
    await user.click(screen.getByTestId("field-model-hosted-model"));
    await selectFlyoutOption(user, "field-model-hosted-thinking", "High");
    expect(onValuesChange).toHaveBeenCalledWith({ model: undefined, thinkingLevel: "high" });
  });

  it("lists the custom provider allowlist on an OpenRouter agent", async () => {
    const user = userEvent.setup();
    const onValuesChange = vi.fn();
    mockSelectableModels([
      selectableModel("byok", "openrouter", "openrouter-only"),
      selectableModel("byok", "custom", "muse-spark-1.3-contributor"),
    ]);

    renderField(
      <HostedModelFieldRenderer
        field={createField("openrouter")}
        value="deepseek-flash"
        onChange={vi.fn()}
        onValuesChange={onValuesChange}
        allValues={{ llmProvider: "custom", model: "deepseek-flash" }}
        organizationId="org-1"
      />,
    );

    await user.click(screen.getByTestId("field-model-hosted-model"));
    await user.hover(screen.getByTestId("field-model-hosted-model-list"));
    expect(await screen.findByRole("menuitem", { name: "muse-spark-1.3-contributor" })).toBeInTheDocument();
    expect(screen.queryByRole("menuitem", { name: "openrouter-only" })).not.toBeInTheDocument();

    await selectFlyoutOption(user, "field-model-hosted-model-list", "muse-spark-1.3-contributor");
    expect(onValuesChange).toHaveBeenCalledWith({
      model: "muse-spark-1.3-contributor",
      thinkingLevel: undefined,
    });
  });
});
