import * as apiClient from "@/api-client";
import type { FactoriesWorkOrder } from "@/api-client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "bun:test";
import { createElement, type ReactNode } from "react";
import { factoryQueryKeys } from "@/hooks/useFactoryData";

type DescribeWorkOrderResult = Awaited<ReturnType<typeof apiClient.factoriesDescribeWorkOrder>>;

function describeResponse(order: FactoriesWorkOrder): DescribeWorkOrderResult {
  return { data: { order }, error: undefined } as unknown as DescribeWorkOrderResult;
}

const { useWebSocketMock } = vi.hoisted(() => ({
  useWebSocketMock: vi.fn(),
}));

vi.mock("@/lib/reactUseWebsocket", () => ({
  useWebSocket: useWebSocketMock,
}));

import { useFactoryWebsocket } from "@/hooks/useFactoryWebsocket";

afterEach(() => {
  vi.restoreAllMocks();
});

function lastCall() {
  const call = useWebSocketMock.mock.calls.at(-1);
  if (!call) throw new Error("useWebSocket was not invoked");
  return call;
}

function renderFactoryWebsocket(organizationId = "org-1", factoryId = "factory-1") {
  const queryClient = new QueryClient();
  const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries");
  renderHook(() => useFactoryWebsocket(organizationId, factoryId), {
    wrapper: ({ children }: { children: ReactNode }) =>
      createElement(QueryClientProvider, { client: queryClient }, children),
  });
  return { queryClient, invalidateSpy };
}

async function emit(payload: unknown) {
  const [, options] = lastCall();
  const onMessage = options.onMessage as (e: MessageEvent<unknown>) => void;
  await act(async () => {
    onMessage(
      new MessageEvent("message", {
        data: JSON.stringify(payload),
      }),
    );
  });
}

describe("useFactoryWebsocket", () => {
  it("connects to the per-factory WebSocket URL with org id", () => {
    renderFactoryWebsocket();
    const [url, , enabled] = lastCall();
    expect(url).toContain("/ws/factories/factory-1");
    expect(url).toContain("organization_id=org-1");
    expect(enabled).toBe(true);
  });

  it("disables the connection when factory id is missing", () => {
    const queryClient = new QueryClient();
    renderHook(() => useFactoryWebsocket("org-1", ""), {
      wrapper: ({ children }: { children: ReactNode }) =>
        createElement(QueryClientProvider, { client: queryClient }, children),
    });
    const [url, , enabled] = lastCall();
    expect(url).toBeNull();
    expect(enabled).toBe(false);
  });

  it("loads that task and patches the cached row", async () => {
    const describeWorkOrder = vi.spyOn(apiClient, "factoriesDescribeWorkOrder").mockResolvedValue(
      describeResponse({
        id: "order-1",
        title: "New",
        checks: [{ key: "confidence", name: "Confidence score", score: 4, maxScore: 5, analysis: "long" }],
        planningSession: {
          id: "ps-1",
          state: "running",
          waitState: "pending",
          executionId: "exec-1",
          survey: { id: "survey-1", questions: [{ prompt: "Which?", options: ["A"] }] },
        },
      }),
    );
    const { queryClient, invalidateSpy } = renderFactoryWebsocket();
    queryClient.setQueryData(factoryQueryKeys.workOrders("org-1", "factory-1"), [
      { id: "order-1", title: "Old" },
      { id: "order-2", title: "Other" },
    ]);

    await emit({
      event: "work_order_updated",
      payload: { factoryId: "factory-1", orderId: "order-1", reason: "order.agent_question" },
    });

    expect(describeWorkOrder).toHaveBeenCalledTimes(1);
    const orders = queryClient.getQueryData<Array<{ id?: string; planningSession?: { activities?: unknown } }>>(
      factoryQueryKeys.workOrders("org-1", "factory-1"),
    );
    expect(orders?.[0]).toMatchObject({
      id: "order-1",
      title: "New",
      checkScores: [{ key: "confidence", name: "Confidence score", score: 4, maxScore: 5 }],
      planningSession: {
        id: "ps-1",
        state: "running",
        waitState: "pending",
        executionId: "exec-1",
        survey: { id: "survey-1", questions: [{ prompt: "Which?", options: ["A"] }] },
      },
    });
    expect(orders?.[1]).toEqual({ id: "order-2", title: "Other" });
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: factoryQueryKeys.workOrderEvents("org-1", "factory-1", "order-1"),
    });
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: factoryQueryKeys.workOrderArtifacts("org-1", "factory-1", "order-1"),
    });
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: ["factories", "org-1", "factory-1", "pull-requests"],
    });
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: ["backlog-analysis-runs", "org-1"],
    });
    expect(invalidateSpy).not.toHaveBeenCalledWith({
      queryKey: factoryQueryKeys.detail("org-1", "factory-1"),
    });
  });

  it("adds a task that is not in the loaded list", async () => {
    vi.spyOn(apiClient, "factoriesDescribeWorkOrder").mockResolvedValue(
      describeResponse({ id: "order-1", title: "New", checks: [] }),
    );
    const { queryClient } = renderFactoryWebsocket();
    queryClient.setQueryData(factoryQueryKeys.workOrders("org-1", "factory-1"), [{ id: "order-2", title: "Other" }]);

    await emit({
      event: "work_order_updated",
      payload: { factoryId: "factory-1", orderId: "order-1" },
    });

    expect(queryClient.getQueryData(factoryQueryKeys.workOrders("org-1", "factory-1"))).toEqual([
      expect.objectContaining({ id: "order-1", title: "New" }),
      { id: "order-2", title: "Other" },
    ]);
  });

  it("ignores an older describe that finishes after a newer one", async () => {
    let releaseOlder: (value: DescribeWorkOrderResult) => void = () => {};
    const older = new Promise<DescribeWorkOrderResult>((resolve) => {
      releaseOlder = resolve;
    });
    const describeWorkOrder = vi.spyOn(apiClient, "factoriesDescribeWorkOrder");
    describeWorkOrder.mockImplementationOnce(() => older as ReturnType<typeof apiClient.factoriesDescribeWorkOrder>);
    describeWorkOrder.mockResolvedValueOnce(describeResponse({ id: "order-1", title: "Newer", checks: [] }));
    const { queryClient } = renderFactoryWebsocket();
    queryClient.setQueryData(factoryQueryKeys.workOrders("org-1", "factory-1"), [{ id: "order-1", title: "Old" }]);

    await emit({
      event: "work_order_updated",
      payload: { factoryId: "factory-1", orderId: "order-1" },
    });
    await emit({
      event: "work_order_updated",
      payload: { factoryId: "factory-1", orderId: "order-1" },
    });
    releaseOlder(describeResponse({ id: "order-1", title: "Older", checks: [] }));
    await act(async () => {
      await older;
    });

    expect(
      queryClient.getQueryData<Array<{ title?: string }>>(factoryQueryKeys.workOrders("org-1", "factory-1"))?.[0]
        ?.title,
    ).toBe("Newer");
  });

  it("ignores events for a different factory", async () => {
    const describeWorkOrder = vi.spyOn(apiClient, "factoriesDescribeWorkOrder");
    const { invalidateSpy } = renderFactoryWebsocket();
    await emit({
      event: "work_order_updated",
      payload: { factoryId: "other-factory", orderId: "order-1" },
    });
    expect(describeWorkOrder).not.toHaveBeenCalled();
    expect(invalidateSpy).not.toHaveBeenCalled();
  });

  it("skips invalidation on the first open and refreshes the orders list on reconnect", () => {
    const { invalidateSpy } = renderFactoryWebsocket();
    invalidateSpy.mockClear();
    const [, options] = lastCall();
    const onOpen = options.onOpen as () => void;

    act(() => {
      onOpen();
    });
    expect(invalidateSpy).not.toHaveBeenCalled();

    act(() => {
      onOpen();
    });
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: ["factories", "org-1", "factory-1", "work-orders"],
      exact: true,
    });
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: ["factories", "org-1", "factory-1", "work-orders-page"],
    });
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: ["factories", "org-1", "factory-1", "pull-requests"],
    });
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: ["backlog-analysis-runs", "org-1"],
    });
    expect(invalidateSpy).not.toHaveBeenCalledWith({
      queryKey: ["factories", "org-1", "factory-1"],
    });
  });
});
