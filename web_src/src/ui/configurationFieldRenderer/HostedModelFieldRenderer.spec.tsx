import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactElement } from "react";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import type { ConfigurationField } from "@/api-client";
import { useBYOKLLMModels } from "@/hooks/useLLMModelAllowlists";
import { useHostedLLMModels } from "@/hooks/useHostedLLMModels";
import { useOrganizationWorkspaceUsage } from "@/hooks/useOrganizationWorkspaceUsage";
import { useSelectableLLMModels } from "@/hooks/useSelectableLLMModels";
import { HostedModelFieldRenderer } from "./HostedModelFieldRenderer";

const useCanvasMock = vi.hoisted(() =>
  vi.fn((): { data: { metadata?: { factoryId?: string } } | undefined; isPending: boolean } => ({
    data: undefined,
    isPending: false,
  })),
);

vi.mock("@/hooks/useHostedLLMModels", () => ({
  useHostedLLMModels: vi.fn(),
}));

vi.mock("@/hooks/useSelectableLLMModels", () => ({
  useSelectableLLMModels: vi.fn(),
}));

vi.mock("@/hooks/useLLMModelAllowlists", () => ({
  useBYOKLLMModels: vi.fn(),
}));

vi.mock("@/hooks/useOrganizationWorkspaceUsage", () => ({
  useOrganizationWorkspaceUsage: vi.fn(),
}));

vi.mock("@/hooks/useCanvasData", () => ({
  useCanvas: useCanvasMock,
}));

function createField(): ConfigurationField {
  return {
    name: "model",
    label: "Model",
    type: "hosted-model",
    placeholder: "sonnet",
    typeOptions: { hostedModel: { provider: "anthropic" } },
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

function mockHostedModels(value: {
  data: { enabled: boolean; models: Array<{ id: string; name: string }> };
  isLoading: boolean;
}) {
  vi.mocked(useHostedLLMModels).mockReturnValue(value as unknown as ReturnType<typeof useHostedLLMModels>);
}

function mockBYOKModels(value: { data: { selected: Array<{ id: string; name: string }> }; isLoading: boolean }) {
  vi.mocked(useBYOKLLMModels).mockReturnValue(value as unknown as ReturnType<typeof useBYOKLLMModels>);
}

function mockSelectableModels(value: {
  data: Array<{
    source: { id: string; name: string };
    provider: { id: string; name: string };
    model: { id: string; name: string };
    key: string;
    label: string;
  }>;
  isLoading: boolean;
}) {
  vi.mocked(useSelectableLLMModels).mockReturnValue(value as unknown as ReturnType<typeof useSelectableLLMModels>);
}

function hostedModel(
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
    source: { id: "hosted", name: "SuperPlane" },
    provider: { id: provider, name: provider },
    model: { id, name: id },
    key: `hosted::${provider}::${id}`,
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
    mockHostedModels({
      data: {
        enabled: true,
        models: [{ id: "claude-sonnet-4-6", name: "claude-sonnet-4-6" }],
      },
      isLoading: false,
    });
    mockBYOKModels({
      data: { selected: [{ id: "claude-sonnet-4-6", name: "claude-sonnet-4-6" }] },
      isLoading: false,
    });
    mockSelectableModels({
      data: [
        hostedModel("anthropic", "claude-sonnet-4-6"),
        hostedModel("openai", "gpt-5"),
        hostedModel("openrouter", "moonshotai/kimi-k2.6"),
      ],
      isLoading: false,
    });
    mockWorkspaceUsage({
      data: { defaultHostedProvider: "anthropic", defaultHostedModel: "claude-sonnet-4-6" },
      isLoading: false,
    });
  });

  it("shows the selected-model list for secret credentials", () => {
    renderField(
      <HostedModelFieldRenderer
        field={createField()}
        value="claude-sonnet-4-6"
        onChange={vi.fn()}
        organizationId="org-1"
        allValues={{ credentials: { source: "secret" } }}
      />,
    );

    expect(screen.getByTestId("field-model-hosted-model")).toBeInTheDocument();
    expect(useBYOKLLMModels).toHaveBeenCalledWith("org-1", "anthropic", true, undefined);
  });

  it("shows the SuperPlane-hosted allowlist when credentials are hosted", () => {
    renderField(
      <HostedModelFieldRenderer
        field={createField()}
        value="claude-sonnet-4-6"
        onChange={vi.fn()}
        organizationId="org-1"
        allValues={{ credentials: { source: "hosted" } }}
      />,
    );

    expect(screen.getByTestId("field-model-hosted-model")).toBeInTheDocument();
    expect(useHostedLLMModels).toHaveBeenCalledWith("org-1", "anthropic", true, undefined);
  });

  it("explains when SuperPlane-hosted models are not configured", () => {
    mockHostedModels({
      data: { enabled: false, models: [] },
      isLoading: false,
    });

    renderField(
      <HostedModelFieldRenderer
        field={createField()}
        value=""
        onChange={vi.fn()}
        organizationId="org-1"
        allValues={{ credentials: { source: "hosted" } }}
      />,
    );

    expect(
      screen.getByText(
        "SuperPlane-hosted models are not configured for this provider. Ask an installation admin to add a key and allowlist.",
      ),
    ).toBeInTheDocument();
  });

  it("explains when no BYOK models are selected", () => {
    mockBYOKModels({
      data: { selected: [] },
      isLoading: false,
    });

    renderField(
      <HostedModelFieldRenderer
        field={createField()}
        value=""
        onChange={vi.fn()}
        organizationId="org-1"
        allValues={{ credentials: { source: "secret" } }}
      />,
    );

    expect(
      screen.getByText(
        "No models are selected for this provider. Select models on Organization LLM Models, or connect a provider on Integrations.",
      ),
    ).toBeInTheDocument();
  });

  it("waits for the canvas factory before it loads SuperPlane-hosted models", () => {
    useCanvasMock.mockReturnValue({ data: undefined, isPending: true });

    renderField(
      <HostedModelFieldRenderer
        field={createField()}
        value=""
        onChange={vi.fn()}
        organizationId="org-1"
        allValues={{ credentials: { source: "hosted" } }}
      />,
      "/org-1/apps/canvas-1",
    );

    expect(screen.getByText("Loading models...")).toBeInTheDocument();
    expect(useHostedLLMModels).toHaveBeenCalledWith("org-1", "anthropic", false, undefined);
  });

  it("loads SuperPlane-hosted models for the canvas factory", () => {
    useCanvasMock.mockReturnValue({ data: { metadata: { factoryId: "factory-1" } }, isPending: false });

    renderField(
      <HostedModelFieldRenderer
        field={createField()}
        value="claude-sonnet-4-6"
        onChange={vi.fn()}
        organizationId="org-1"
        allValues={{ credentials: { source: "hosted" } }}
      />,
      "/org-1/apps/canvas-1",
    );

    expect(screen.getByTestId("field-model-hosted-model")).toBeInTheDocument();
    expect(useHostedLLMModels).toHaveBeenCalledWith("org-1", "anthropic", true, "factory-1");
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
