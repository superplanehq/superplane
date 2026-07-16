import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it } from "vitest";
import type { CanvasesCanvasRun } from "@/api-client";
import { useRunInspectorActions } from "./useRunInspectorActions";

const startedRun: CanvasesCanvasRun = {
  id: "run-1",
  state: "STATE_STARTED",
  rootEvent: {
    id: "root-event-1",
    nodeId: "trigger-1",
  },
};

function renderActions(run: CanvasesCanvasRun) {
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
        run,
      }),
    {
      wrapper: ({ children }: { children: ReactNode }) => (
        <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
      ),
    },
  );
}

describe("useRunInspectorActions", () => {
  it("enables Stop while the run is started", () => {
    const { result } = renderActions(startedRun);

    expect(result.current.stopDisabled).toBe(false);
  });

  it("disables Stop once the run has finished", () => {
    const { result } = renderActions({ ...startedRun, state: "STATE_FINISHED" });

    expect(result.current.stopDisabled).toBe(true);
  });

  it("disables Stop when the run has no id", () => {
    const { result } = renderActions({ ...startedRun, id: undefined });

    expect(result.current.stopDisabled).toBe(true);
  });
});
