import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const { organizationsUpdateByokllmModels } = vi.hoisted(() => ({
  organizationsUpdateByokllmModels: vi.fn(),
}));

vi.mock("@/api-client", async (importOriginal) => {
  const actual = await importOriginal<Record<string, unknown>>();
  return {
    ...actual,
    organizationsUpdateByokllmModels,
  };
});

import { factoryLLMModelsQueryKey, isFactoryBYOKModelsQuery, useUpdateBYOKLLMModels } from "./useLLMModelAllowlists";

function createWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

describe("isFactoryBYOKModelsQuery", () => {
  it("matches factory BYOK model list keys for the organization and provider", () => {
    expect(
      isFactoryBYOKModelsQuery(
        factoryLLMModelsQueryKey("org-1", "factory-1", "anthropic", "byok"),
        "org-1",
        "anthropic",
      ),
    ).toBe(true);
  });

  it("rejects hosted factory keys and other organizations", () => {
    expect(
      isFactoryBYOKModelsQuery(
        factoryLLMModelsQueryKey("org-1", "factory-1", "anthropic", "hosted"),
        "org-1",
        "anthropic",
      ),
    ).toBe(false);
    expect(
      isFactoryBYOKModelsQuery(
        factoryLLMModelsQueryKey("org-2", "factory-1", "anthropic", "byok"),
        "org-1",
        "anthropic",
      ),
    ).toBe(false);
  });
});

describe("useUpdateBYOKLLMModels", () => {
  beforeEach(() => {
    organizationsUpdateByokllmModels.mockReset();
  });

  it("invalidates BYOK lists and selectable models after a save", async () => {
    organizationsUpdateByokllmModels.mockResolvedValue({
      data: { selected: [{ id: "anthropic/claude-sonnet-4-6", name: "anthropic/claude-sonnet-4-6" }] },
    });
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
    const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");
    const { result } = renderHook(() => useUpdateBYOKLLMModels("org-1"), { wrapper: createWrapper(queryClient) });

    await result.current.mutateAsync({
      provider: "openrouter",
      allowedModels: ["anthropic/claude-sonnet-4-6"],
    });

    await waitFor(() => {
      expect(invalidateQueries).toHaveBeenCalled();
    });
    expect(invalidateQueries).toHaveBeenCalledWith({
      queryKey: ["organizations", "org-1", "byok-models", "openrouter"],
    });
    expect(invalidateQueries).toHaveBeenCalledWith({
      queryKey: ["organizations", "org-1", "selectable-llm-models"],
    });
    expect(invalidateQueries).toHaveBeenCalledWith({
      predicate: expect.any(Function),
    });
  });
});
