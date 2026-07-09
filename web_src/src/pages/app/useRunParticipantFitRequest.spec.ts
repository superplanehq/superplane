import { act, renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { useRunParticipantFitRequest } from "./useRunParticipantFitRequest";
import type { RunCanvasData } from "./useRunCanvasData";

function buildRunCanvasData(participantNodeIds: string[]): RunCanvasData {
  return {
    nodes: [],
    edges: [],
    participantNodeIds,
  };
}

type HookProps = {
  isRunInspectionMode: boolean;
  selectedRunId: string | null;
  runCanvasLoading: boolean;
  runCanvasData: RunCanvasData | null;
};

describe("useRunParticipantFitRequest", () => {
  it("waits for the selected run participant nodes before emitting a fit request", () => {
    const initialProps: HookProps = {
      isRunInspectionMode: false,
      selectedRunId: null,
      runCanvasLoading: false,
      runCanvasData: null,
    };
    const { result, rerender } = renderHook((props: HookProps) => useRunParticipantFitRequest(props), {
      initialProps,
    });

    act(() => {
      result.current.requestParticipantFit("run-1");
    });

    expect(result.current.fitRequest).toBeNull();

    rerender({
      isRunInspectionMode: true,
      selectedRunId: "run-1",
      runCanvasLoading: true,
      runCanvasData: null,
    });
    expect(result.current.fitRequest).toBeNull();

    rerender({
      isRunInspectionMode: true,
      selectedRunId: "run-1",
      runCanvasLoading: false,
      runCanvasData: buildRunCanvasData(["trigger-1", "action-1"]),
    });

    expect(result.current.fitRequest).toBe(1);
  });

  it("clears pending and emitted fit requests", () => {
    const { result } = renderHook(() =>
      useRunParticipantFitRequest({
        isRunInspectionMode: true,
        selectedRunId: "run-1",
        runCanvasLoading: false,
        runCanvasData: buildRunCanvasData(["trigger-1"]),
      }),
    );

    act(() => {
      result.current.requestParticipantFit("run-1");
    });
    expect(result.current.fitRequest).toBe(1);

    act(() => {
      result.current.clearParticipantFit();
    });

    expect(result.current.fitRequest).toBeNull();
  });
});
