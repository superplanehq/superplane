import { render, screen } from "@testing-library/react";
import type { ReactElement } from "react";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import type { ConfigurationField } from "@/api-client";
import { useBYOKLLMModels } from "@/hooks/useLLMModelAllowlists";
import { useHostedLLMModels } from "@/hooks/useHostedLLMModels";
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

vi.mock("@/hooks/useLLMModelAllowlists", () => ({
  useBYOKLLMModels: vi.fn(),
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

function mockHostedModels(value: {
  data: { enabled: boolean; models: Array<{ id: string; name: string }> };
  isLoading: boolean;
}) {
  vi.mocked(useHostedLLMModels).mockReturnValue(value as unknown as ReturnType<typeof useHostedLLMModels>);
}

function mockBYOKModels(value: { data: { selected: Array<{ id: string; name: string }> }; isLoading: boolean }) {
  vi.mocked(useBYOKLLMModels).mockReturnValue(value as unknown as ReturnType<typeof useBYOKLLMModels>);
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
});
