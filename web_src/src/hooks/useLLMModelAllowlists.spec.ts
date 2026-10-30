import { QueryClient, QueryClientProvider, useQueries } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { afterEach, describe, expect, it, vi } from "bun:test";

const { factoriesUpdateFactoryLlmModels, factoriesListFactoryLlmModels, identity } = vi.hoisted(() => ({
  factoriesUpdateFactoryLlmModels: vi.fn(),
  factoriesListFactoryLlmModels: vi.fn(),
  identity: { currentUserId: "member-1" },
}));

vi.mock("@/api-client", () => ({
  factoriesUpdateFactoryLlmModels,
  factoriesListFactoryLlmModels,
}));

vi.mock("@/contexts/usePermissions", () => ({ usePermissions: () => identity }));

import {
  factoryLLMModelsQueryKey,
  useUpdateFactoryLLMModels,
  useSwitchFactoryModelSource,
  useFactoryLLMModels,
} from "./useLLMModelAllowlists";

function createWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

describe("workspace model saves", () => {
  afterEach(() => vi.restoreAllMocks());
  it.each(["hosted", "byok", "subscription", "source switch"])(
    "refreshes mounted settings and pickers for %s",
    async (fundingSource) => {
      vi.spyOn(globalThis, "fetch").mockResolvedValue(new Response(JSON.stringify({}), { status: 200 }));
      factoriesUpdateFactoryLlmModels.mockResolvedValue({ data: {} });
      const queryClient = new QueryClient();
      const keys = (factoryId: string) => [
        factoryLLMModelsQueryKey("org-1", factoryId, "openai", fundingSource),
        ["organizations", "org-1", "selectable-llm-models", factoryId],
        ["factories", "org-1", factoryId, "line-runner-models", "hotfix"],
      ];
      const currentKeys = keys("factory-1");
      const unrelatedKeys = keys("other-factory");
      const { result, unmount } = renderHook(
        () => {
          useQueries({
            queries: [...currentKeys, ...unrelatedKeys].map((queryKey) => ({
              queryKey,
              queryFn: async () => "updated",
              initialData: "cached",
              staleTime: Infinity,
            })),
          });
          const save = useUpdateFactoryLLMModels("org-1", "factory-1");
          const switchSource = useSwitchFactoryModelSource("org-1", "factory-1");
          return { save, switchSource };
        },
        { wrapper: createWrapper(queryClient) },
      );
      await act(async () => {
        if (fundingSource === "source switch") {
          await result.current.switchSource.mutateAsync({ source: "hosted" });
          return;
        }
        await result.current.save.mutateAsync({
          provider: "openai",
          fundingSource,
          allowedModels: [],
          inheritParent: false,
        });
      });
      await waitFor(() => {
        for (const key of currentKeys) expect(queryClient.getQueryData(key)).toBe("updated");
      });
      for (const key of unrelatedKeys) expect(queryClient.getQueryData(key)).toBe("cached");
      if (fundingSource !== "source switch")
        expect(factoriesUpdateFactoryLlmModels).toHaveBeenLastCalledWith(
          expect.objectContaining({
            body: { provider: "openai", fundingSource, allowedModels: [], inheritParent: false },
          }),
        );
      unmount();
    },
  );
});

describe("personal subscription settings catalogs", () => {
  it("loads the new member catalog when the signed-in member changes", async () => {
    identity.currentUserId = "member-1";
    factoriesListFactoryLlmModels
      .mockResolvedValueOnce({ data: { parent: [{ id: "member-one-model" }] } })
      .mockResolvedValueOnce({ data: { parent: [{ id: "member-two-model" }] } });
    const queryClient = new QueryClient();
    const { result, rerender, unmount } = renderHook(
      () => useFactoryLLMModels("org-1", "factory-1", "openai", "subscription", true),
      { wrapper: createWrapper(queryClient) },
    );
    await waitFor(() => expect(result.current.data?.parent?.[0]?.id).toBe("member-one-model"));
    identity.currentUserId = "member-2";
    rerender();
    await waitFor(() => expect(result.current.data?.parent?.[0]?.id).toBe("member-two-model"));
    unmount();
  });
});
