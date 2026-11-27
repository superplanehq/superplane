import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook } from "@testing-library/react";
import { type ReactNode } from "react";
import { describe, expect, it, vi } from "bun:test";

const runAutomation = vi.fn();

vi.mock("@/api-client", () => ({
  factoriesRunFactoryCustomAutomation: runAutomation,
}));

import { factoryQueryKeys } from "./useFactoryData";
import { useRunFactoryCustomAutomation } from "./useFactoryCustomAutomationData";

describe("useRunFactoryCustomAutomation", () => {
  it.each([{ runId: "run-1" }, { queueItemId: "queue-1" }])(
    "accepts admitted work and refreshes task data: %j",
    async (data) => {
      runAutomation.mockResolvedValue({ data });
      const client = new QueryClient();
      const detailKey = factoryQueryKeys.workOrderDetail("org-1", "factory-1", "task-1");
      const listKey = factoryQueryKeys.workOrders("org-1", "factory-1");
      client.setQueryData(detailKey, { id: "task-1" });
      client.setQueryData(listKey, [{ id: "task-1" }]);
      const { result } = renderHook(() => useRunFactoryCustomAutomation("org-1", "factory-1"), {
        wrapper: ({ children }: { children: ReactNode }) => (
          <QueryClientProvider client={client}>{children}</QueryClientProvider>
        ),
      });

      await expect(
        result.current.mutateAsync({ automationId: "automation-1", workOrderId: "task-1" }),
      ).resolves.toEqual(data);
      expect(client.getQueryState(detailKey)?.isInvalidated).toBe(true);
      expect(client.getQueryState(listKey)?.isInvalidated).toBe(true);
    },
  );

  it.each([{ data: {} }, { error: { message: "Start failed" } }])("rejects failed starts: %j", async (response) => {
    runAutomation.mockResolvedValue(response);
    const client = new QueryClient();
    const { result } = renderHook(() => useRunFactoryCustomAutomation("org-1", "factory-1"), {
      wrapper: ({ children }: { children: ReactNode }) => (
        <QueryClientProvider client={client}>{children}</QueryClientProvider>
      ),
    });

    await expect(result.current.mutateAsync({ automationId: "automation-1", workOrderId: "task-1" })).rejects.toThrow(
      "SuperPlane could not start the automation.",
    );
  });
});
