import { render, screen } from "@testing-library/react";
import { beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import type { ConfigurationField } from "@/api-client";
import { useHostedLLMModels } from "@/hooks/useHostedLLMModels";
import { HostedModelFieldRenderer } from "./HostedModelFieldRenderer";

vi.mock("@/hooks/useHostedLLMModels", () => ({
  useHostedLLMModels: vi.fn(),
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

describe("HostedModelFieldRenderer", () => {
  beforeAll(() => {
    Element.prototype.hasPointerCapture ??= () => false;
    Element.prototype.setPointerCapture ??= () => {};
    Element.prototype.releasePointerCapture ??= () => {};
    Element.prototype.scrollIntoView ??= () => {};
  });

  beforeEach(() => {
    mockHostedModels({
      data: {
        enabled: true,
        models: [{ id: "claude-sonnet-4-6", name: "claude-sonnet-4-6" }],
      },
      isLoading: false,
    });
  });

  it("shows a free-text model field for secret credentials", () => {
    render(
      <HostedModelFieldRenderer
        field={createField()}
        value="sonnet"
        onChange={vi.fn()}
        organizationId="org-1"
        allValues={{ credentials: { source: "secret" } }}
      />,
    );

    expect(screen.getByDisplayValue("sonnet")).toBeInTheDocument();
    expect(useHostedLLMModels).toHaveBeenCalledWith("org-1", "anthropic", false);
  });

  it("shows the SuperPlane-hosted allowlist when credentials are hosted", () => {
    render(
      <HostedModelFieldRenderer
        field={createField()}
        value="claude-sonnet-4-6"
        onChange={vi.fn()}
        organizationId="org-1"
        allValues={{ credentials: { source: "hosted" } }}
      />,
    );

    expect(screen.getByTestId("field-model-hosted-model")).toBeInTheDocument();
    expect(useHostedLLMModels).toHaveBeenCalledWith("org-1", "anthropic", true);
  });

  it("explains when SuperPlane-hosted models are not configured", () => {
    mockHostedModels({
      data: { enabled: false, models: [] },
      isLoading: false,
    });

    render(
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
});
