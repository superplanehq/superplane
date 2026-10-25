import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "bun:test";
import type { ReactNode } from "react";
import * as apiClient from "@/api-client";
import { FactoryAutomationRunHistory } from "./FactoryAutomationRunHistory";
import { useFactoryWebsocket } from "@/hooks/useFactoryWebsocket";

const api = vi.hoisted(() => ({
  canvasesListRuns: vi.fn(),
}));
const socket = vi.hoisted(() => ({
  onMessage: undefined as ((event: MessageEvent<unknown>) => void) | undefined,
}));
vi.mock("@/api-client/sdk.gen", () => api);
vi.mock("@/hooks/useCanvasWebsocket", () => ({ useCanvasRuntimeWebsocket: () => {} }));
vi.mock("@/lib/reactUseWebsocket", () => ({
  useWebSocket: (_url: string | null, options: { onMessage?: (event: MessageEvent<unknown>) => void }) => {
    socket.onMessage = options.onMessage;
  },
}));

function OpenHistory() {
  useFactoryWebsocket("org", "factory");
  return (
    <FactoryAutomationRunHistory
      canvasId="canvas"
      organizationId="org"
      factoryId="factory"
      selectedRunId={null}
      onSelectRun={() => {}}
    />
  );
}

describe("open run history task labels", () => {
  it("shows a renamed task while history is open", async () => {
    api.canvasesListRuns.mockResolvedValue({
      data: {
        runs: [
          {
            id: "run-1",
            state: "STATE_FINISHED",
            result: "RESULT_PASSED",
            task: { id: "task-1", key: "SP-123", title: "Old title" },
          },
        ],
        totalCount: 1,
      },
    });
    vi.spyOn(apiClient, "factoriesDescribeWorkOrder").mockResolvedValue({
      data: { order: { id: "task-1", key: "SP-123", title: "Renamed task" } },
    } as Awaited<ReturnType<typeof apiClient.factoriesDescribeWorkOrder>>);
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(<OpenHistory />, {
      wrapper: ({ children }: { children: ReactNode }) => (
        <QueryClientProvider client={client}>{children}</QueryClientProvider>
      ),
    });
    await waitFor(() => expect(screen.getByText("Old title")).toBeInTheDocument());
    expect(api.canvasesListRuns).toHaveBeenCalledTimes(1);

    await act(async () => {
      socket.onMessage?.(
        new MessageEvent("message", {
          data: JSON.stringify({
            event: "work_order_updated",
            payload: { factoryId: "factory", orderId: "task-1" },
          }),
        }),
      );
    });

    await waitFor(() => expect(screen.getByText("Renamed task")).toBeInTheDocument());
    expect(screen.queryByText("Old title")).not.toBeInTheDocument();
    expect(api.canvasesListRuns).toHaveBeenCalledTimes(1);
    client.clear();
  });
});
