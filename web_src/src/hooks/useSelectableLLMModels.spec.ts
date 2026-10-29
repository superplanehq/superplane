import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "bun:test";

const { organizationsListSelectableLlmModels, organizationsSyncOrganizationBilling } = vi.hoisted(() => ({
  organizationsListSelectableLlmModels: vi.fn(),
  organizationsSyncOrganizationBilling: vi.fn(),
}));

vi.mock("@/api-client", () => ({
  organizationsListSelectableLlmModels,
  organizationsSyncOrganizationBilling,
}));

import { syncOrganizationBilling, organizationBillingQueryKey } from "./useOrganizationBilling";

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
            thinkingLevels: ["low", "high"],
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
        thinkingLevels: ["low", "high"],
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

it("refreshes an active model catalog when billing synchronization removes BYOK access", async () => {
  const byok = {
    source: { id: "byok", name: "Your keys" },
    provider: { id: "openai", name: "OpenAI" },
    model: { id: "gpt-5", name: "gpt-5" },
    key: "byok::openai::gpt-5",
    label: "openai/gpt-5",
  };
  organizationsListSelectableLlmModels.mockResolvedValue({ data: { models: [byok] } });
  organizationsSyncOrganizationBilling.mockImplementation(async () => {
    organizationsListSelectableLlmModels.mockResolvedValue({ data: { models: [] } });
    return { data: { productFeatures: { byo_key: false } } };
  });
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const { result } = renderHook(() => useSelectableLLMModels("org-1"), { wrapper: createWrapper(queryClient) });
  await waitFor(() => expect(result.current.data).toHaveLength(1));
  await act(async () => syncOrganizationBilling("org-1", queryClient));
  await waitFor(() => expect(result.current.data).toEqual([]));
  expect(queryClient.getQueryData(organizationBillingQueryKey("org-1"))).toEqual({
    productFeatures: { byo_key: false },
  });
});
