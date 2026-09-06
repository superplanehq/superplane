import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactElement } from "react";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import type { ConfigurationField } from "@/api-client";
import { useOrganizationWorkspaceUsage } from "@/hooks/useOrganizationWorkspaceUsage";
import { useSelectableLLMModels } from "@/hooks/useSelectableLLMModels";
import { HostedModelFieldRenderer } from "./HostedModelFieldRenderer";

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

function createField(provider = "anthropic"): ConfigurationField {
  return {
    name: "model",
    label: "Model",
    type: "hosted-model",
    placeholder: "Select a model",
    typeOptions: { hostedModel: { provider } },
  };
}

function createSuperPlaneField(): ConfigurationField {
  return {
    name: "model",
    label: "Model",
    type: "hosted-model",
    placeholder: "Instance SuperPlane agent model",
    typeOptions: { hostedModel: { provider: "all" } },
  };
}

function mockSelectableModels(
  data: Array<{
    source: { id: string; name: string };
    provider: { id: string; name: string };
    model: { id: string; name: string };
    key: string;
    label: string;
  }>,
  isLoading = false,
) {
  vi.mocked(useSelectableLLMModels).mockImplementation((_organizationId, options) => {
    const sources = options?.sources;
    const listed = sources ? data.filter((item) => sources.includes(item.source.id as "hosted" | "byok")) : data;
    return { data: listed, isLoading } as unknown as ReturnType<typeof useSelectableLLMModels>;
  });
}

function selectableModel(
  source: "hosted" | "byok",
  provider: string,
  id: string,
): {
  source: { id: string; name: string };
  provider: { id: string; name: string };
  model: { id: string; name: string };
  key: string;
  label: string;
} {
  const label = provider === "openrouter" ? id : `${provider}/${id}`;
  return {
    source: { id: source, name: source === "hosted" ? "SuperPlane" : "Your keys" },
    provider: { id: provider, name: provider },
    model: { id, name: id },
    key: `${source}::${provider}::${id}`,
    label,
  };
}

function mockWorkspaceUsage(value: {
  data: { defaultHostedProvider?: string; defaultHostedModel?: string };
  isLoading: boolean;
}) {
  vi.mocked(useOrganizationWorkspaceUsage).mockReturnValue(
    value as unknown as ReturnType<typeof useOrganizationWorkspaceUsage>,
  );
}

function renderField(ui: ReactElement, path = "/") {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route path="/:organizationId/apps/:appId" element={ui} />
        <Route path="*" element={ui} />
      </Routes>
    </MemoryRouter>,
  );
}

describe("HostedModelFieldRenderer", () => {
  beforeAll(() => {
    Element.prototype.hasPointerCapture ??= () => false;
    Element.prototype.setPointerCapture ??= () => {};
    Element.prototype.releasePointerCapture ??= () => {};
    Element.prototype.scrollIntoView ??= () => {};
  });

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
    expect(screen.getByRole("option", { name: "anthropic/claude-sonnet-4-6" })).toBeInTheDocument();
    expect(screen.queryByRole("option", { name: "openai/gpt-5" })).not.toBeInTheDocument();

    await user.click(screen.getByRole("option", { name: "anthropic/claude-sonnet-4-6" }));
    expect(onChange).toHaveBeenCalledWith("claude-sonnet-4-6");
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
    expect(screen.getByRole("option", { name: "anthropic/claude-sonnet-4-6" })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "openai/gpt-5" })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "moonshotai/kimi-k2.6" })).toBeInTheDocument();
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
});
