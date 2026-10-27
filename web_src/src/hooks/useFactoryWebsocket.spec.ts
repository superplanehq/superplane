import * as apiClient from "@/api-client";
import type { FactoriesWorkOrder } from "@/api-client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "bun:test";
import { createElement, type ReactNode } from "react";
import { canvasKeys } from "@/hooks/useCanvasData";
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
  it.each(["factory-1", "other-factory"])("refreshes capacity as a board batch scoped to %s", async (factoryId) => {
    const describeWorkOrder = vi.spyOn(apiClient, "factoriesDescribeWorkOrder");
    const { queryClient } = renderFactoryWebsocket();
    const listKey = factoryQueryKeys.workOrders("org-1", "factory-1");
    const sessionKey = factoryQueryKeys.planningSession("org-1", "factory-1", "order-1");
    const detailKey = factoryQueryKeys.workOrderDetail("org-1", "factory-1", "order-1");
    const otherKey = factoryQueryKeys.workOrders("org-1", "other-factory");
    queryClient.setQueryData(listKey, [{ id: "order-1" }, { id: "order-2" }]);
    queryClient.setQueryData(sessionKey, { runnerState: "queued", queueReason: "parallelism_limit" });
    queryClient.setQueryData(detailKey, { id: "order-1" });
    queryClient.setQueryData(otherKey, []);

    await emit({ event: "work_order_updated", payload: { factoryId, reason: "runner_capacity" } });

    expect(queryClient.getQueryState(listKey)?.isInvalidated).toBe(factoryId === "factory-1");
    expect(queryClient.getQueryState(sessionKey)?.isInvalidated).toBe(factoryId === "factory-1");
    expect(queryClient.getQueryState(detailKey)?.isInvalidated).toBe(false);
    expect(queryClient.getQueryState(otherKey)?.isInvalidated).toBe(false);
    expect(describeWorkOrder).not.toHaveBeenCalled();
  });
  it.each(["existing", "new", "archived"])(
    "updates historical tasks without reloading all pages: %s",
    async (state) => {
      const describe = vi.spyOn(apiClient, "factoriesDescribeWorkOrder");
      if (state === "archived") describe.mockRejectedValue(new Error("404 not found"));
      else describe.mockResolvedValue(describeResponse({ id: "order-1", key: "SP-42", title: "Updated" }));
      const { queryClient } = renderFactoryWebsocket();
      const key = [...factoryQueryKeys.workOrders("org-1", "factory-1"), "including-archived"];
      const otherKey = [...factoryQueryKeys.workOrders("org-1", "other-factory"), "including-archived"];
      const other = { id: "order-2", title: "Other", key: "SP-43" };
      queryClient.setQueryData(key, state === "new" ? [other] : [{ id: "order-1", title: "Old" }, other]);
      queryClient.setQueryData(otherKey, []);
      await emit({ event: "work_order_updated", payload: { factoryId: "factory-1", orderId: "order-1" } });
      expect(queryClient.getQueryState(key)?.isInvalidated).toBe(state === "archived");
      if (state !== "archived") {
        expect(queryClient.getQueryData(key)).toEqual([
          expect.objectContaining({ id: "order-1", key: "SP-42", title: "Updated" }),
          other,
        ]);
      }
      expect(queryClient.getQueryState(otherKey)?.isInvalidated).toBe(false);
    },
  );
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
      queryKey: factoryQueryKeys.planningSession("org-1", "factory-1", "order-1"),
    });
    expect(invalidateSpy).not.toHaveBeenCalledWith({
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

  it("refetches the page total when an updated task is not in the loaded pages", async () => {
    vi.spyOn(apiClient, "factoriesDescribeWorkOrder").mockResolvedValue(
      describeResponse({ id: "order-old", title: "Edited", state: "STATE_DRAFT", checks: [] }),
    );
    const { queryClient, invalidateSpy } = renderFactoryWebsocket();
    const pageKey = factoryQueryKeys.workOrdersPage("org-1", "factory-1", ["STATE_DRAFT"]);
    queryClient.setQueryData(pageKey, {
      pageParams: [undefined],
      pages: [
        {
          orders: [{ id: "order-2", title: "Loaded", state: "STATE_DRAFT" }],
          hasNextPage: true,
          totalCount: 80,
        },
      ],
    });

    await emit({
      event: "work_order_updated",
      payload: { factoryId: "factory-1", orderId: "order-old" },
    });

    expect(queryClient.getQueryData<{ pages: Array<{ totalCount?: number }> }>(pageKey)?.pages[0]?.totalCount).toBe(80);
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: pageKey });
  });

  it("keeps the local total when a new task joins a fully loaded page", async () => {
    vi.spyOn(apiClient, "factoriesDescribeWorkOrder").mockResolvedValue(
      describeResponse({ id: "order-new", title: "New", state: "STATE_DRAFT", checks: [] }),
    );
    const { queryClient, invalidateSpy } = renderFactoryWebsocket();
    const pageKey = factoryQueryKeys.workOrdersPage("org-1", "factory-1", ["STATE_DRAFT"]);
    queryClient.setQueryData(pageKey, {
      pageParams: [undefined],
      pages: [
        {
          orders: [{ id: "order-2", title: "Loaded", state: "STATE_DRAFT" }],
          hasNextPage: false,
          totalCount: 1,
        },
      ],
    });

    await emit({
      event: "work_order_updated",
      payload: { factoryId: "factory-1", orderId: "order-new" },
    });

    expect(queryClient.getQueryData<{ pages: Array<{ totalCount?: number }> }>(pageKey)?.pages[0]?.totalCount).toBe(2);
    expect(invalidateSpy).not.toHaveBeenCalledWith({ queryKey: pageKey });
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

  it("drops a missing task from the board without a warning", async () => {
    vi.spyOn(apiClient, "factoriesDescribeWorkOrder").mockRejectedValue(new Error("Not Found"));
    const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
    const { queryClient } = renderFactoryWebsocket();
    queryClient.setQueryData(factoryQueryKeys.workOrders("org-1", "factory-1"), [
      { id: "order-1", title: "Gone" },
      { id: "order-2", title: "Other" },
    ]);
    queryClient.setQueryData(factoryQueryKeys.workOrdersPage("org-1", "factory-1", ["STATE_OPEN"]), {
      pageParams: [undefined],
      pages: [
        {
          orders: [
            { id: "order-1", title: "Gone" },
            { id: "order-2", title: "Other" },
          ],
          hasNextPage: false,
        },
      ],
    });
    queryClient.setQueryData(factoryQueryKeys.workOrderDetail("org-1", "factory-1", "order-1"), {
      id: "order-1",
      title: "Gone",
    });

    await emit({
      event: "work_order_updated",
      payload: { factoryId: "factory-1", orderId: "order-1" },
    });

    expect(warn).not.toHaveBeenCalled();
    expect(queryClient.getQueryData(factoryQueryKeys.workOrders("org-1", "factory-1"))).toEqual([
      { id: "order-2", title: "Other" },
    ]);
    expect(queryClient.getQueryData(factoryQueryKeys.workOrdersPage("org-1", "factory-1", ["STATE_OPEN"]))).toEqual({
      pageParams: [undefined],
      pages: [{ orders: [{ id: "order-2", title: "Other" }], hasNextPage: false }],
    });
    expect(queryClient.getQueryData(factoryQueryKeys.workOrderDetail("org-1", "factory-1", "order-1"))).toBeUndefined();
  });

  it("stays quiet when a live refresh hits an expired session", async () => {
    vi.spyOn(apiClient, "factoriesDescribeWorkOrder").mockRejectedValue(new Error("Unauthorized"));
    const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
    const { queryClient } = renderFactoryWebsocket();
    queryClient.setQueryData(factoryQueryKeys.workOrders("org-1", "factory-1"), [
      { id: "order-1", title: "Still here" },
    ]);
    queryClient.setQueryData(factoryQueryKeys.workOrderDetail("org-1", "factory-1", "order-1"), {
      id: "order-1",
      title: "Still here",
    });

    await emit({
      event: "work_order_updated",
      payload: { factoryId: "factory-1", orderId: "order-1" },
    });

    expect(warn).not.toHaveBeenCalled();
    expect(queryClient.getQueryData(factoryQueryKeys.workOrders("org-1", "factory-1"))).toEqual([
      { id: "order-1", title: "Still here" },
    ]);
    expect(queryClient.getQueryData(factoryQueryKeys.workOrderDetail("org-1", "factory-1", "order-1"))).toEqual({
      id: "order-1",
      title: "Still here",
    });
  });

  it("warns when a live refresh fails for another reason", async () => {
    const failure = new Error("network down");
    vi.spyOn(apiClient, "factoriesDescribeWorkOrder").mockRejectedValue(failure);
    const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
    const { queryClient } = renderFactoryWebsocket();
    queryClient.setQueryData(factoryQueryKeys.workOrders("org-1", "factory-1"), [
      { id: "order-1", title: "Still here" },
    ]);

    await emit({
      event: "work_order_updated",
      payload: { factoryId: "factory-1", orderId: "order-1" },
    });

    expect(warn).toHaveBeenCalledWith("factory ws: failed to refresh work order", failure);
    expect(queryClient.getQueryData(factoryQueryKeys.workOrders("org-1", "factory-1"))).toEqual([
      { id: "order-1", title: "Still here" },
    ]);
  });

  it("keeps a newer task when an older refresh is not found", async () => {
    let rejectOlder: (reason?: unknown) => void = () => {};
    const older = new Promise<DescribeWorkOrderResult>((_, reject) => {
      rejectOlder = reject;
    });
    const describeWorkOrder = vi.spyOn(apiClient, "factoriesDescribeWorkOrder");
    describeWorkOrder.mockImplementationOnce(() => older as ReturnType<typeof apiClient.factoriesDescribeWorkOrder>);
    describeWorkOrder.mockResolvedValueOnce(describeResponse({ id: "order-1", title: "Newer", checks: [] }));
    const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
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
    rejectOlder(new Error("Not Found"));
    await act(async () => {
      await older.catch(() => undefined);
    });

    expect(warn).not.toHaveBeenCalled();
    expect(
      queryClient.getQueryData<Array<{ title?: string }>>(factoryQueryKeys.workOrders("org-1", "factory-1"))?.[0]
        ?.title,
    ).toBe("Newer");
  });

  it("updates open run history labels when a linked task is renamed", async () => {
    vi.spyOn(apiClient, "factoriesDescribeWorkOrder").mockResolvedValue(
      describeResponse({ id: "order-1", key: "SP-42", title: "Renamed task" }),
    );
    const { queryClient } = renderFactoryWebsocket();
    const historyKey = canvasKeys.infiniteRuns("canvas-1", { includeTask: true });
    const otherKey = canvasKeys.infiniteRuns("canvas-2", { includeTask: true });
    const detailKey = canvasKeys.run("canvas-1", "run-1", true);
    queryClient.setQueryData(historyKey, {
      pageParams: [undefined, "cursor"],
      pages: [
        { runs: [{ id: "run-1", task: { id: "order-1", key: "SP-42", title: "Old title" } }] },
        { runs: [{ id: "run-2", task: { id: "order-1", key: "SP-42", title: "Old title" } }] },
      ],
    });
    queryClient.setQueryData(detailKey, {
      run: { id: "run-1", task: { id: "order-1", key: "SP-42", title: "Old title" } },
    });
    queryClient.setQueryData(otherKey, {
      pageParams: [undefined],
      pages: [{ runs: [{ id: "run-3", task: { id: "order-9", key: "SP-9", title: "Other task" } }] }],
    });

    await emit({
      event: "work_order_updated",
      payload: { factoryId: "factory-1", orderId: "order-1" },
    });

    expect(queryClient.getQueryData(historyKey)).toEqual({
      pageParams: [undefined, "cursor"],
      pages: [
        { runs: [{ id: "run-1", task: { id: "order-1", key: "SP-42", title: "Renamed task" } }] },
        { runs: [{ id: "run-2", task: { id: "order-1", key: "SP-42", title: "Renamed task" } }] },
      ],
    });
    expect(queryClient.getQueryData(detailKey)).toEqual({
      run: { id: "run-1", task: { id: "order-1", key: "SP-42", title: "Renamed task" } },
    });
    expect(queryClient.getQueryData(otherKey)).toEqual({
      pageParams: [undefined],
      pages: [{ runs: [{ id: "run-3", task: { id: "order-9", key: "SP-9", title: "Other task" } }] }],
    });
    expect(queryClient.getQueryState(historyKey)?.isInvalidated).toBe(false);
  });

  it.each([
    ["source run", { sourceRunId: "run-1" }],
    ["line execution", { lineDispatches: [{ stepExecutions: [{ run: { id: "run-1" } }] }] }],
    ["pull request activity", { pullRequests: [{ activities: [{ run: { id: "run-1" } }] }] }],
  ])("links an open history run when a task gains a %s", async (_link, order) => {
    vi.spyOn(apiClient, "factoriesDescribeWorkOrder").mockResolvedValue(
      describeResponse({ id: "order-1", key: "SP-42", title: "Created task", ...order }),
    );
    const { queryClient } = renderFactoryWebsocket();
    const historyKey = canvasKeys.infiniteRuns("canvas-1", { includeTask: true });
    const otherKey = canvasKeys.infiniteRuns("canvas-2", { includeTask: true });
    const detailKey = canvasKeys.run("canvas-1", "run-1", true);
    queryClient.setQueryData(historyKey, {
      pageParams: [undefined, "cursor"],
      pages: [
        { runs: [{ id: "run-1" }, { id: "run-child", parent: { id: "run-1" } }] },
        { runs: [{ id: "run-kept", task: { id: "order-9", key: "SP-9", title: "Other task" } }] },
      ],
    });
    queryClient.setQueryData(detailKey, { run: { id: "run-1" } });
    queryClient.setQueryData(otherKey, {
      pageParams: [undefined],
      pages: [{ runs: [{ id: "run-3" }] }],
    });

    await emit({
      event: "work_order_updated",
      payload: { factoryId: "factory-1", orderId: "order-1" },
    });

    const label = { id: "order-1", key: "SP-42", title: "Created task" };
    expect(queryClient.getQueryData(historyKey)).toEqual({
      pageParams: [undefined, "cursor"],
      pages: [
        {
          runs: [
            { id: "run-1", task: label },
            { id: "run-child", parent: { id: "run-1" }, task: label },
          ],
        },
        { runs: [{ id: "run-kept", task: { id: "order-9", key: "SP-9", title: "Other task" } }] },
      ],
    });
    expect(queryClient.getQueryData(detailKey)).toEqual({ run: { id: "run-1", task: label } });
    expect(queryClient.getQueryState(historyKey)?.isInvalidated).toBe(true);
    expect(queryClient.getQueryState(detailKey)?.isInvalidated).toBe(true);
    expect(queryClient.getQueryState(otherKey)?.isInvalidated).toBe(false);
    expect(queryClient.getQueryData(otherKey)).toEqual({
      pageParams: [undefined],
      pages: [{ runs: [{ id: "run-3" }] }],
    });
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
    const { queryClient, invalidateSpy } = renderFactoryWebsocket();
    const historyKey = [...factoryQueryKeys.workOrders("org-1", "factory-1"), "including-archived"];
    queryClient.setQueryData(historyKey, []);
    invalidateSpy.mockClear();
    const [, options] = lastCall();
    const onOpen = options.onOpen as () => void;

    act(() => {
      onOpen();
    });
    expect(invalidateSpy).not.toHaveBeenCalled();
    expect(queryClient.getQueryState(historyKey)?.isInvalidated).toBe(false);

    act(() => {
      onOpen();
    });
    expect(queryClient.getQueryState(historyKey)?.isInvalidated).toBe(true);
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: ["factories", "org-1", "factory-1", "work-orders"],
      exact: true,
    });
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: ["factories", "org-1", "factory-1", "work-orders-page"],
    });
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: factoryQueryKeys.planningSessions("org-1", "factory-1"),
    });
    expect(invalidateSpy).not.toHaveBeenCalledWith({
      queryKey: ["factories", "org-1", "factory-1"],
    });
  });
});
