import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const { organizationsListSelectableLlmModels } = vi.hoisted(() => ({
  organizationsListSelectableLlmModels: vi.fn(),
}));

vi.mock("@/api-client", async (importOriginal) => {
  const actual = await importOriginal<Record<string, unknown>>();
  return {
    ...actual,
    organizationsListSelectableLlmModels,
  };
});

import { useSelectableLLMModels } from "./useSelectableLLMModels";

function createWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

describe("useSelectableLLMModels", () => {
  beforeEach(() => {
    organizationsListSelectableLlmModels.mockReset();
  });

  it("maps the union list and filters by source", async () => {
    organizationsListSelectableLlmModels.mockResolvedValue({
      data: {
        models: [
          {
            source: { id: "hosted", name: "SuperPlane" },
            provider: { id: "anthropic", name: "Anthropic" },
            model: { id: "claude-sonnet-4-6", name: "claude-sonnet-4-6" },
            key: "hosted::anthropic::claude-sonnet-4-6",
            label: "anthropic/claude-sonnet-4-6",
          },
          {
            source: { id: "byok", name: "Your keys" },
            provider: { id: "anthropic", name: "Anthropic" },
            model: { id: "claude-sonnet-4-6", name: "claude-sonnet-4-6" },
            key: "byok::anthropic::claude-sonnet-4-6",
            label: "anthropic/claude-sonnet-4-6",
          },
        ],
      },
    });
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const { result } = renderHook(
      () => useSelectableLLMModels("org-1", { factoryId: "factory-1", sources: ["hosted"] }),
      { wrapper: createWrapper(queryClient) },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual([
      {
        source: { id: "hosted", name: "SuperPlane" },
        provider: { id: "anthropic", name: "Anthropic" },
        model: { id: "claude-sonnet-4-6", name: "claude-sonnet-4-6" },
        key: "hosted::anthropic::claude-sonnet-4-6",
        label: "anthropic/claude-sonnet-4-6",
      },
    ]);
    expect(organizationsListSelectableLlmModels).toHaveBeenCalledWith(
      expect.objectContaining({
        path: { id: "org-1" },
        query: { factoryId: "factory-1" },
      }),
    );
  });
});
