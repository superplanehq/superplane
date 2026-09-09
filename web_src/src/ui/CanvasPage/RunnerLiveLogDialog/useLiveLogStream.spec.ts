import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ExecutionInfo } from "../../../pages/app/mappers/types";
import type { LogState } from "./types";
import {
  finalizeRunningCommandSections,
  terminalCommandStatusForExecution,
  terminalTimeMsForExecution,
  useLiveLogStream,
} from "./useLiveLogStream";

const { captureExceptionMock, pumpMock, stopMock } = vi.hoisted(() => ({
  captureExceptionMock: vi.fn(),
  pumpMock: vi.fn(),
  stopMock: vi.fn(),
}));

vi.mock("@/sentry", () => ({
  Sentry: { captureException: captureExceptionMock },
}));

vi.mock("./liveLogStream", () => {
  class LiveLogStreamMock {
    pump = pumpMock;
    stop = stopMock;
  }

  return { LiveLogStream: LiveLogStreamMock };
});

vi.mock("@/hooks/useOrganizationId", () => ({
  useOrganizationId: () => undefined,
}));

vi.mock("@/hooks/useCanvasId", () => ({
  useCanvasId: () => undefined,
}));

beforeEach(() => {
  captureExceptionMock.mockReset();
  pumpMock.mockReset();
  stopMock.mockReset();
});

function baseLogState(): LogState {
  return {
    sections: [
      {
        index: 0,
        text: "completed",
        lines: [],
        events: [],
        status: "passed",
        duration_ms: 100,
        started_at: 1_000,
        collapsed: true,
      },
      {
        index: 1,
        text: "Set up DevEnv",
        lines: ["docker compose up"],
        events: [],
        status: "running",
        duration_ms: null,
        started_at: 2_000,
        collapsed: false,
      },
    ],
    orphanLines: [],
    error: null,
    isLoading: false,
    isStreaming: false,
  };
}

function execution(overrides: Partial<ExecutionInfo>): ExecutionInfo {
  return {
    id: "execution-1",
    createdAt: "2026-07-22T15:46:48.000Z",
    updatedAt: "2026-07-22T15:54:58.000Z",
    state: "STATE_FINISHED",
    result: "RESULT_FAILED",
    resultReason: "RESULT_REASON_ERROR",
    resultMessage: "",
    metadata: {},
    configuration: {},
    rootEvent: undefined,
    ...overrides,
  } as ExecutionInfo;
}

describe("runner live log state", () => {
  it("finalizes unfinished command sections when a terminal execution failed", () => {
    const finalized = finalizeRunningCommandSections(baseLogState(), "failed", 5_000);

    expect(finalized.sections[0]).toMatchObject({ status: "passed", duration_ms: 100 });
    expect(finalized.sections[1]).toMatchObject({
      status: "failed",
      duration_ms: 3_000,
      collapsed: false,
    });
  });

  it("collapses unfinished command sections when a terminal execution passed", () => {
    const finalized = finalizeRunningCommandSections(baseLogState(), "passed", 5_000);

    expect(finalized.sections[1]).toMatchObject({
      status: "passed",
      duration_ms: 3_000,
      collapsed: true,
    });
  });

  it("closes nested running tools when the execution ends", () => {
    const finalized = finalizeRunningCommandSections(
      {
        ...baseLogState(),
        sections: [
          {
            index: 5,
            text: "Implementation",
            kind: "prompt",
            lines: [],
            events: [
              {
                kind: "tools",
                id: "5-tools-0",
                tools: [
                  {
                    id: "5-tool-0",
                    kind: "read",
                    text: "pkg/foo.go",
                    lines: [],
                    status: "running",
                    duration_ms: null,
                  },
                ],
              },
            ],
            status: "running",
            duration_ms: null,
            started_at: 2_000,
            collapsed: false,
          },
        ],
      },
      "failed",
      5_000,
    );

    const tools = finalized.sections[0]?.events[0];
    expect(tools?.kind).toBe("tools");
    if (tools?.kind !== "tools") {
      throw new Error("expected tools group");
    }
    expect(tools.tools[0]).toMatchObject({ status: "failed", duration_ms: 0 });
  });

  it("maps terminal execution result to command status", () => {
    expect(terminalCommandStatusForExecution(execution({ result: "RESULT_PASSED" }))).toBe("passed");
    expect(terminalCommandStatusForExecution(execution({ result: "RESULT_FAILED" }))).toBe("failed");
    expect(terminalCommandStatusForExecution(execution({ result: "RESULT_CANCELLED" }))).toBe("failed");
    expect(
      terminalCommandStatusForExecution(execution({ state: "STATE_STARTED", result: "RESULT_UNKNOWN" })),
    ).toBeNull();
  });

  it("uses the execution update time as the terminal command timestamp", () => {
    expect(terminalTimeMsForExecution(execution({ updatedAt: "2026-07-22T15:54:58.000Z" }))).toBe(
      Date.parse("2026-07-22T15:54:58.000Z"),
    );
  });

  it("does not expose a terminal command timestamp while execution is in flight", () => {
    expect(
      terminalTimeMsForExecution(
        execution({
          state: "STATE_CANCELLING",
          result: "RESULT_UNKNOWN",
          updatedAt: "2026-07-22T15:54:58.000Z",
        }),
      ),
    ).toBeNull();
  });
});

describe("useLiveLogStream", () => {
  it("stops loading after the live log response opens", async () => {
    let openStream: (() => void) | undefined;
    pumpMock.mockImplementation(
      (handlers: { onOpen?: () => void }) =>
        new Promise<void>((resolve) => {
          openStream = () => {
            handlers.onOpen?.();
            resolve();
          };
        }),
    );

    const { result } = renderHook(() =>
      useLiveLogStream("execution-1", false, "passed", null, {
        organizationId: "organization-1",
        canvasId: "canvas-1",
      }),
    );

    expect(result.current.isLoading).toBe(true);
    act(() => openStream?.());
    await waitFor(() => expect(result.current.isLoading).toBe(false));
  });

  it("reports a request error and retries the terminal log session on demand", async () => {
    pumpMock.mockRejectedValue(new Error("Failed to fetch"));
    const { result } = renderHook(() =>
      useLiveLogStream("execution-1", false, "failed", null, {
        organizationId: "organization-1",
        canvasId: "canvas-1",
      }),
    );

    await waitFor(() => expect(result.current.error).toBe("Failed to fetch"));
    expect(captureExceptionMock).toHaveBeenCalledOnce();
    expect(captureExceptionMock).toHaveBeenCalledWith(
      expect.objectContaining({ message: "Failed to fetch" }),
      expect.objectContaining({
        fingerprint: ["runner-live-logs", "request"],
        extra: {
          organizationId: "organization-1",
          canvasId: "canvas-1",
          executionId: "execution-1",
        },
      }),
    );

    act(() => result.current.retry());

    await waitFor(() => expect(pumpMock).toHaveBeenCalledTimes(2));
  });

  it("clears a stale error when a retry opens a healthy stream", async () => {
    pumpMock.mockRejectedValueOnce(new Error("Failed to fetch"));
    pumpMock.mockImplementationOnce((handlers: { onOpen?: () => void }) => {
      handlers.onOpen?.();
      return new Promise<void>(() => undefined);
    });
    const { result } = renderHook(() =>
      useLiveLogStream("execution-1", true, null, null, {
        organizationId: "organization-1",
        canvasId: "canvas-1",
      }),
    );

    await waitFor(() => expect(result.current.error).toBe("Failed to fetch"));
    act(() => result.current.retry());

    await waitFor(() => expect(result.current.error).toBeNull());
    expect(result.current.isLoading).toBe(false);
  });
});
