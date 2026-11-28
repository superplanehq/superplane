import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { expect, it, vi } from "bun:test";

const useWebSocket = vi.fn();
vi.mock("@/lib/reactUseWebsocket", () => ({ useWebSocket }));

import { useCanvasWebsocket } from "./useCanvasWebsocket";

for (const eventName of ["canvas_updated", "canvas_deleted"]) {
  it(`refreshes workspace warnings after ${eventName}`, () => {
    const queryClient = new QueryClient();
    const workspaceKey = ["integrations", "connected", "org-1", "workspace", "factory-1"];
    const otherKey = ["integrations", "connected", "other-org", "workspace", "factory-2"];
    queryClient.setQueryData(workspaceKey, [{ metadata: { id: "jira-1" } }]);
    queryClient.setQueryData(otherKey, []);
    renderHook(() => useCanvasWebsocket({ canvasId: "canvas-1", organizationId: "org-1" }), {
      wrapper: ({ children }: { children: ReactNode }) =>
        createElement(QueryClientProvider, { client: queryClient }, children),
    });

    const options = useWebSocket.mock.calls.at(-1)![1] as { onMessage: (event: MessageEvent) => void };
    act(() => {
      options.onMessage(
        new MessageEvent("message", {
          data: JSON.stringify({ event: eventName, payload: { canvasId: "canvas-1" } }),
        }),
      );
    });

    expect(queryClient.getQueryState(workspaceKey)?.isInvalidated).toBe(true);
    expect(queryClient.getQueryState(otherKey)?.isInvalidated).toBe(false);
  });
}
