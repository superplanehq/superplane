import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it } from "vitest";
import type { CanvasesCanvasRun } from "@/api-client";
import type { RunInspectorNodeSection } from "./types";
import { useRunInspectorActions } from "./useRunInspectorActions";

const run: CanvasesCanvasRun = {
  rootEvent: {
    id: "root-event-1",
    nodeId: "trigger-1",
  },
};

function renderActions(
  sections: RunInspectorNodeSection[],
  {
    executionsLoading = false,
    runOverride = run,
  }: { executionsLoading?: boolean; runOverride?: CanvasesCanvasRun } = {},
) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  return renderHook(
    () =>
      useRunInspectorActions({
        canvasId: "canvas-1",
        run: runOverride,
        sections,
        executionsLoading,
      }),
    {
      wrapper: ({ children }: { children: ReactNode }) => (
        <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
      ),
    },
  );
}

function actionSection(overrides: Partial<RunInspectorNodeSection> = {}): RunInspectorNodeSection {
  return {
    sectionValue: "action-1",
    nodeId: "action-1",
    nodeName: "Action 1",
    isTrigger: false,
    isQueued: false,
    badge: null,
    tabData: null,
    upstreamSections: [],
    outputSections: [],
    actions: {
      canStop: false,
      canPushThrough: false,
      approvalRecords: [],
    },
    configurationFields: [],
    ...overrides,
  };
}

describe("useRunInspectorActions", () => {
  it("allows Stop when action sections only have lightweight running execution refs", () => {
    const { result } = renderActions([
      actionSection({
        executionRef: {
          id: "execution-ref-1",
          nodeId: "action-1",
          state: "STATE_STARTED",
        },
      }),
    ]);

    expect(result.current.stopDisabled).toBe(false);
  });

  it("allows Stop for loaded action execution details so queued steps can be cancelled", () => {
    const { result } = renderActions([
      actionSection({
        execution: {
          id: "execution-1",
          nodeId: "action-1",
          state: "STATE_FINISHED",
          result: "RESULT_PASSED",
        },
      }),
    ]);

    expect(result.current.stopDisabled).toBe(false);
  });

  it("allows Stop for queued items while executions are loading", () => {
    const { result } = renderActions([], {
      executionsLoading: true,
      runOverride: {
        ...run,
        queueItems: [{ id: "queue-1", nodeId: "action-1" }],
      },
    });

    expect(result.current.stopDisabled).toBe(false);
  });
});
