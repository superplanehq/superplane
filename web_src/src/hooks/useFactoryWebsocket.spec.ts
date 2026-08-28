import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { createElement, type ReactNode } from "react";
import { canvasKeys } from "@/hooks/useCanvasData";
import { factoryQueryKeys } from "@/hooks/useFactoryData";

const { useWebSocketMock } = vi.hoisted(() => ({
  useWebSocketMock: vi.fn(),
}));

vi.mock("react-use-websocket", () => ({
  default: useWebSocketMock,
}));

import { useFactoryWebsocket } from "@/hooks/useFactoryWebsocket";

afterEach(() => {
  vi.clearAllMocks();
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

function emit(payload: unknown) {
  const [, options] = lastCall();
  const onMessage = options.onMessage as (e: MessageEvent<unknown>) => void;
  act(() => {
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

  it("invalidates work-order list on work_order_updated", () => {
    const { invalidateSpy } = renderFactoryWebsocket();
    emit({
      event: "work_order_updated",
      payload: { factoryId: "factory-1", orderId: "order-1", reason: "order.opened" },
    });

    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: factoryQueryKeys.workOrders("org-1", "factory-1"),
    });
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: factoryQueryKeys.detail("org-1", "factory-1"),
    });
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: factoryQueryKeys.workOrderDetail("org-1", "factory-1", "order-1"),
    });
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: factoryQueryKeys.workOrderEvents("org-1", "factory-1", "order-1"),
    });
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: factoryQueryKeys.workOrderArtifacts("org-1", "factory-1", "order-1"),
    });
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: ["factories", "org-1", "factory-1", "pull-requests"],
    });
  });

  it("invalidates described canvas runs for the updated work order", () => {
    const { queryClient, invalidateSpy } = renderFactoryWebsocket();
    queryClient.setQueryData(factoryQueryKeys.workOrders("org-1", "factory-1"), [
      {
        id: "order-1",
        lineDispatches: [
          {
            stepExecutions: [{ run: { id: "run-1", appId: "app-1" } }],
          },
        ],
      },
    ]);
    invalidateSpy.mockClear();

    emit({
      event: "work_order_updated",
      payload: { factoryId: "factory-1", orderId: "order-1", reason: "run_started" },
    });

    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: canvasKeys.run("app-1", "run-1"),
    });
  });

  it("ignores events for a different factory", () => {
    const { invalidateSpy } = renderFactoryWebsocket();
    invalidateSpy.mockClear();
    emit({
      event: "work_order_updated",
      payload: { factoryId: "other-factory", orderId: "order-1" },
    });
    expect(invalidateSpy).not.toHaveBeenCalled();
  });

  it("skips invalidation on the first open and invalidates on reconnect", () => {
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
      queryKey: factoryQueryKeys.workOrders("org-1", "factory-1"),
    });
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: factoryQueryKeys.detail("org-1", "factory-1"),
    });
  });
});
