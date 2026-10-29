import { fireEvent, render, screen } from "@testing-library/react";
import type userEvent from "@testing-library/user-event";
import type { ReactElement } from "react";
import { MemoryRouter, Route, Routes } from "react-router";
import { expect, vi } from "bun:test";
import type { ConfigurationField } from "@/api-client";
import { useOrganizationWorkspaceUsage } from "@/hooks/useOrganizationWorkspaceUsage";
import { useSelectableLLMModels } from "@/hooks/useSelectableLLMModels";

/**
 * Shared fixtures for HostedModelFieldRenderer specs. Each spec must mock
 * `@/hooks/useSelectableLLMModels` and `@/hooks/useOrganizationWorkspaceUsage`.
 */
export function createField(provider = "anthropic"): ConfigurationField {
  return {
    name: "model",
    label: "Model",
    type: "hosted-model",
    placeholder: "Select a model",
    typeOptions: { hostedModel: { provider } },
  };
}

export function createSuperPlaneField(): ConfigurationField {
  return {
    name: "model",
    label: "Model",
    type: "hosted-model",
    placeholder: "Instance SuperPlane agent model",
    typeOptions: { hostedModel: { provider: "all" } },
  };
}

export function mockSelectableModels(
  data: Array<{
    source: { id: string; name: string };
    provider: { id: string; name: string };
    model: { id: string; name: string };
    key: string;
    label: string;
    thinkingLevels: string[];
  }>,
  options: { isLoading?: boolean; isError?: boolean } = {},
) {
  vi.mocked(useSelectableLLMModels).mockImplementation((_organizationId, queryOptions) => {
    const sources = queryOptions?.sources;
    const listed = sources ? data.filter((item) => sources.includes(item.source.id as "hosted" | "byok")) : data;
    return {
      data: listed,
      isLoading: options.isLoading ?? false,
      isError: options.isError ?? false,
    } as unknown as ReturnType<typeof useSelectableLLMModels>;
  });
}

export function selectableModel(
  source: "hosted" | "byok",
  provider: string,
  id: string,
  thinkingLevels: string[] = ["low", "medium", "high"],
): {
  source: { id: string; name: string };
  provider: { id: string; name: string };
  model: { id: string; name: string };
  key: string;
  label: string;
  thinkingLevels: string[];
} {
  const label = provider === "openrouter" || provider === "custom" ? id : `${provider}/${id}`;
  return {
    source: { id: source, name: source === "hosted" ? "SuperPlane" : "Your keys" },
    provider: { id: provider, name: provider },
    model: { id, name: id },
    key: `${source}::${provider}::${id}`,
    label,
    thinkingLevels,
  };
}

export function mockWorkspaceUsage(value: {
  data: { defaultHostedProvider?: string; defaultHostedModel?: string };
  isLoading: boolean;
}) {
  vi.mocked(useOrganizationWorkspaceUsage).mockReturnValue(
    value as unknown as ReturnType<typeof useOrganizationWorkspaceUsage>,
  );
}

export async function selectFlyoutOption(
  user: ReturnType<typeof userEvent.setup>,
  parentTestId: string,
  optionName: string,
) {
  await user.hover(screen.getByTestId(parentTestId));
  fireEvent.click(await screen.findByRole("menuitem", { name: optionName }));
}

export function expectClosedThinkingWord(modelName: string, thinkingWord: string) {
  const control = screen.getByTestId("field-model-hosted-model");
  const name = control.querySelector(".truncate");
  const word = control.querySelector(".text-muted-foreground");

  expect(name).toHaveTextContent(modelName);
  expect(name).not.toHaveTextContent(thinkingWord);
  expect(word).toHaveTextContent(thinkingWord);
  expect(word).toHaveClass("ml-auto", "shrink-0", "text-[11px]", "text-muted-foreground");
  expect(word?.nextElementSibling?.tagName).toBe("svg");
}

export function renderField(ui: ReactElement, path = "/") {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route path="/:organizationId/apps/:appId" element={ui} />
        <Route path="*" element={ui} />
      </Routes>
    </MemoryRouter>,
  );
}

export function installPointerCapturePolyfills() {
  Element.prototype.hasPointerCapture ??= () => false;
  Element.prototype.setPointerCapture ??= () => {};
  Element.prototype.releasePointerCapture ??= () => {};
  Element.prototype.scrollIntoView ??= () => {};
}
