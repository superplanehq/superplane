import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";

import { clearBacklogAnalysisPending, pendingBacklogAnalysisIds } from "@/pages/factories/lib/backlogAnalysis";

const { factoriesCreateWorkOrder, factoriesSelectFactoryVcsProviderRepository, organizationsListIntegrations } =
  vi.hoisted(() => ({
    factoriesCreateWorkOrder: vi.fn(),
    factoriesSelectFactoryVcsProviderRepository: vi.fn(),
    organizationsListIntegrations: vi.fn(),
  }));

vi.mock("@/api-client", () => ({
  factoriesCreateWorkOrder,
  factoriesSelectFactoryVcsProviderRepository,
  organizationsListIntegrations,
}));

import {
  factoryQueryKeys,
  mergeFactoryBoardWorkOrders,
  useCreateWorkOrder,
  useSelectFactoryVcsProviderRepository,
} from "./useFactoryData";
import { useConnectedIntegrations } from "./useIntegrations";

function createWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

describe("useSelectFactoryVcsProviderRepository", () => {
  it("refreshes connections created by repository selection before the save completes", async () => {
    const integration = {
      metadata: { id: "bitbucket-1", name: "acme-bitbucket", integrationName: "bitbucket" },
      status: { state: "ready" },
    };
    organizationsListIntegrations.mockResolvedValueOnce({ data: { integrations: [] } });
    organizationsListIntegrations.mockResolvedValue({ data: { integrations: [integration] } });
    factoriesSelectFactoryVcsProviderRepository.mockResolvedValue({
      data: { factory: { id: "factory-1", onboarding: { vcsIntegrationId: "bitbucket-1" } } },
    });
    const queryClient = new QueryClient();
    const { result } = renderHook(
      () => ({
        connections: useConnectedIntegrations("org-1"),
        selectRepository: useSelectFactoryVcsProviderRepository("org-1", "factory-1", "bitbucket"),
      }),
      { wrapper: createWrapper(queryClient) },
    );

    await waitFor(() => expect(result.current.connections.isSuccess).toBe(true));
    expect(result.current.connections.data).toEqual([]);

    await act(async () => {
      await result.current.selectRepository.mutateAsync({ repository: "acme/api" });
      expect(queryClient.getQueryData(["integrations", "connected", "org-1"])).toEqual([integration]);
    });
  });
});

describe("mergeFactoryBoardWorkOrders", () => {
  it("keeps one row when backlog, open, and done share an id", () => {
    const shared = { id: "wo-1", title: "first" };
    expect(
      mergeFactoryBoardWorkOrders([shared], [{ id: "wo-1", title: "open copy" }], [{ id: "wo-1", title: "done copy" }]),
    ).toEqual([shared]);
  });
});

describe("useCreateWorkOrder", () => {
  const createdOrderIds = ["wo-created-1", "wo-planning-on", "wo-planning-off"];

  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    for (const orderId of createdOrderIds) {
      clearBacklogAnalysisPending(orderId);
    }
  });

  function seedFactory(queryClient: QueryClient, enabled: boolean) {
    queryClient.setQueryData(factoryQueryKeys.detail("org-1", "factory-1"), {
      id: "factory-1",
      planning: { enabled },
    });
  }

  async function createOrder(queryClient: QueryClient, orderId: string, model?: string) {
    factoriesCreateWorkOrder.mockResolvedValue({ data: { order: { id: orderId } } });
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries");
    const { result } = renderHook(() => useCreateWorkOrder("org-1", "factory-1"), {
      wrapper: createWrapper(queryClient),
    });

    await act(async () => {
      await result.current.mutateAsync({ title: "New task", description: "", model });
    });

    return invalidateSpy;
  }

  function expectListAndRunInvalidation(invalidateSpy: ReturnType<typeof vi.spyOn>) {
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: factoryQueryKeys.workOrders("org-1", "factory-1"),
    });
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: factoryQueryKeys.workOrdersPagePrefix("org-1", "factory-1"),
    });
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: ["backlog-analysis-runs", "org-1"],
    });
  }

  it("marks the new order pending analysis when Planning is on", async () => {
    const queryClient = new QueryClient();
    seedFactory(queryClient, true);

    const invalidateSpy = await createOrder(queryClient, "wo-planning-on");

    await waitFor(() => expect(pendingBacklogAnalysisIds().has("wo-planning-on")).toBe(true));
    expectListAndRunInvalidation(invalidateSpy);
  });

  it.each([undefined, "claude-opus-4-6"])("sends the optional model through the create API (%s)", async (model) => {
    await createOrder(new QueryClient(), "wo-created-1", model);
    const request = factoriesCreateWorkOrder.mock.calls.at(-1)?.[0];
    expect(request.body).toEqual({
      title: "New task",
      description: "",
      assigneeIds: undefined,
      ...(model ? { model } : {}),
    });
    if (!model) {
      expect(request.body).not.toHaveProperty("model");
    }
  });

  it("marks the new order pending analysis when the factory cache is missing", async () => {
    const queryClient = new QueryClient();

    const invalidateSpy = await createOrder(queryClient, "wo-created-1");

    await waitFor(() => expect(pendingBacklogAnalysisIds().has("wo-created-1")).toBe(true));
    expectListAndRunInvalidation(invalidateSpy);
  });

  it("does not mark analysis pending when Planning is off", async () => {
    const queryClient = new QueryClient();
    seedFactory(queryClient, false);

    const invalidateSpy = await createOrder(queryClient, "wo-planning-off");

    expect(pendingBacklogAnalysisIds().has("wo-planning-off")).toBe(false);
    expectListAndRunInvalidation(invalidateSpy);
  });
});
