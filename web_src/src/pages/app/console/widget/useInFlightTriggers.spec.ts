import { describe, expect, it, vi } from "vitest";
import { renderHook } from "@testing-library/react";

import { useInFlightTriggers } from "./useInFlightTriggers";

vi.mock("@/hooks/useCanvasData", () => ({
  useInfiniteCanvasRuns: vi.fn(),
}));

import { useInfiniteCanvasRuns } from "@/hooks/useCanvasData";

const useInfiniteCanvasRunsMock = vi.mocked(useInfiniteCanvasRuns);

function makeQueryResult(pages: Array<{ runs: Array<{ state: string; rootEvent?: { nodeId?: string } }> }>) {
  return {
    data: { pages },
    isLoading: false,
  } as ReturnType<typeof useInfiniteCanvasRuns>;
}

describe("useInFlightTriggers", () => {
  it("returns the same Set reference when content is unchanged across renders", () => {
    const triggerNodeIds = ["trigger-1", "trigger-2"];
    const queryResult = makeQueryResult([
      {
        runs: [
          { state: "STATE_STARTED", rootEvent: { nodeId: "trigger-1" } },
          { state: "STATE_FINISHED", rootEvent: { nodeId: "trigger-2" } },
        ],
      },
    ]);
    useInfiniteCanvasRunsMock.mockReturnValue(queryResult);

    const { result, rerender } = renderHook(({ ids }) => useInFlightTriggers("canvas-1", ids), {
      initialProps: { ids: triggerNodeIds },
    });

    const firstInFlight = result.current.inFlight;
    expect(Array.from(firstInFlight)).toEqual(["trigger-1"]);

    // Re-render with the same trigger ids reference and the same query data;
    // a stable reference avoids cascading useEffect work in downstream widgets.
    rerender({ ids: triggerNodeIds });
    expect(result.current.inFlight).toBe(firstInFlight);
  });

  it("returns a new Set reference when content changes", () => {
    const triggerNodeIds = ["trigger-1", "trigger-2"];
    useInfiniteCanvasRunsMock.mockReturnValue(
      makeQueryResult([
        {
          runs: [{ state: "STATE_STARTED", rootEvent: { nodeId: "trigger-1" } }],
        },
      ]),
    );

    const { result, rerender } = renderHook(() => useInFlightTriggers("canvas-1", triggerNodeIds));
    const firstInFlight = result.current.inFlight;
    expect(Array.from(firstInFlight)).toEqual(["trigger-1"]);

    useInfiniteCanvasRunsMock.mockReturnValue(
      makeQueryResult([
        {
          runs: [
            { state: "STATE_STARTED", rootEvent: { nodeId: "trigger-1" } },
            { state: "STATE_STARTED", rootEvent: { nodeId: "trigger-2" } },
          ],
        },
      ]),
    );

    rerender();
    expect(result.current.inFlight).not.toBe(firstInFlight);
    expect(Array.from(result.current.inFlight).sort()).toEqual(["trigger-1", "trigger-2"]);
  });

  it("returns the same Set reference when query data updates but content is unchanged", () => {
    const triggerNodeIds = ["trigger-1"];
    useInfiniteCanvasRunsMock.mockReturnValue(
      makeQueryResult([
        {
          runs: [{ state: "STATE_STARTED", rootEvent: { nodeId: "trigger-1" } }],
        },
      ]),
    );

    const { result, rerender } = renderHook(() => useInFlightTriggers("canvas-1", triggerNodeIds));
    const firstInFlight = result.current.inFlight;

    // Simulate a websocket-driven refetch that returns a new response object
    // but with the same set of in-flight trigger node ids. The reference must
    // remain stable so downstream `useEffect` deps don't churn.
    useInfiniteCanvasRunsMock.mockReturnValue(
      makeQueryResult([
        {
          runs: [{ state: "STATE_STARTED", rootEvent: { nodeId: "trigger-1" } }],
        },
      ]),
    );

    rerender();
    expect(result.current.inFlight).toBe(firstInFlight);
  });
});
