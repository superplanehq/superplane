import type { CanvasesCanvasRun } from "@/api-client";
import { client } from "@/api-client/client.gen";
import { CUSTOM_AUTOMATION_TEMPLATE_ID } from "@/pages/factories/lib/customAutomation";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, renderHook, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it } from "bun:test";
import { createElement, type ReactNode } from "react";

import { useBacklogCustomRuns } from "./useBacklogCustomRuns";

describe("useBacklogCustomRuns", () => {
  const previousConfig = client.getConfig();

  afterEach(() => {
    cleanup();
    client.setConfig({ ...previousConfig, fetch: previousConfig.fetch });
  });

  it("refreshes a completed task-linked run outside the recent page", async () => {
    const queryClient = new QueryClient();
    const cachedRun: CanvasesCanvasRun = {
      id: "older-run",
      state: "STATE_STARTED",
      task: { id: "wo-1" },
      rootEvent: { data: { work_order: { id: "wo-1" } } },
    };
    queryClient.setQueryData(["backlog-custom-runs", "org-1", "canvas-1", "wo-1"], {
      runs: [{ canvasId: "canvas-1", workOrderId: "wo-1", run: cachedRun }],
      latestRunId: "recent-0",
    });
    const recentRuns = Array.from({ length: 200 }, (_, index) => ({
      id: `recent-${index}`,
      state: "STATE_FINISHED",
      task: { id: "another-task" },
    }));
    client.setConfig({
      baseUrl: "http://localhost",
      fetch: async (input) => {
        const request = input instanceof Request ? input : new Request(input);
        const url = new URL(request.url);
        if (url.pathname === "/api/v1/canvases/canvas-1/runs") {
          return Response.json({ runs: recentRuns, hasNextPage: true, lastTimestamp: new Date().toISOString() });
        }
        if (url.pathname === "/api/v1/canvases/canvas-1/runs/older-run") {
          // The detail API includes the task association only when requested.
          return Response.json({
            run: {
              ...cachedRun,
              state: "STATE_FINISHED",
              result: "RESULT_PASSED",
              task: url.searchParams.get("includeTask") === "true" ? cachedRun.task : undefined,
            },
          });
        }
        throw new Error(`Unexpected request: ${url.pathname}`);
      },
    });
    function Wrapper({ children }: { children: ReactNode }) {
      return createElement(QueryClientProvider, { client: queryClient }, children);
    }

    const { result } = renderHook(
      () =>
        useBacklogCustomRuns("org-1", "wo-1", [
          { id: "canvas-1", columnKey: "custom-review", templateId: CUSTOM_AUTOMATION_TEMPLATE_ID },
        ]),
      { wrapper: Wrapper },
    );

    await waitFor(() => expect(result.current.isFetching).toBe(false));
    expect(result.current.isError).toBe(false);
    expect(result.current.runs).toHaveLength(1);
    expect(result.current.runs[0]).toMatchObject({
      workOrderId: "wo-1",
      run: { id: "older-run", state: "STATE_FINISHED", result: "RESULT_PASSED" },
    });
  });
});
