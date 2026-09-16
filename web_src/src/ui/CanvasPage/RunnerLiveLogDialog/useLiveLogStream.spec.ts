import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "bun:test";
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

  it("ignores a benign CloudWatch log-stream-not-found broker error without reporting it", async () => {
    pumpMock.mockImplementation(
      (handlers: { onOpen?: () => void; onStreamError: (message: string) => void }) =>
        new Promise<void>((resolve) => {
          handlers.onOpen?.();
          handlers.onStreamError(
            "operation error CloudWatch Logs: GetLogEvents, https response error StatusCode: 400, " +
              "RequestID: bffc49eb-3863-4426-a5e1-dfd28b43bbe3, ResourceNotFoundException: " +
              "The specified log stream does not exist.",
          );
          resolve();
        }),
    );

    const { result } = renderHook(() =>
      useLiveLogStream("execution-1", false, "passed", null, {
        organizationId: "organization-1",
        canvasId: "canvas-1",
      }),
    );

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.error).toBeNull();
    expect(captureExceptionMock).not.toHaveBeenCalled();
  });

  it("reports a CloudWatch log-group-not-found broker error", async () => {
    const message =
      "operation error CloudWatch Logs: GetLogEvents, https response error StatusCode: 400, " +
      "RequestID: bffc49eb-3863-4426-a5e1-dfd28b43bbe3, ResourceNotFoundException: " +
      "The specified log group does not exist.";
    pumpMock.mockImplementation(
      (handlers: { onOpen?: () => void; onStreamError: (message: string) => void }) =>
        new Promise<void>((resolve) => {
          handlers.onOpen?.();
          handlers.onStreamError(message);
          resolve();
        }),
    );

    const { result } = renderHook(() =>
      useLiveLogStream("execution-1", false, "passed", null, {
        organizationId: "organization-1",
        canvasId: "canvas-1",
      }),
    );

    await waitFor(() => expect(result.current.error).toBe(message));
    expect(captureExceptionMock).toHaveBeenCalledOnce();
    expect(captureExceptionMock).toHaveBeenCalledWith(
      expect.objectContaining({ message }),
      expect.objectContaining({
        fingerprint: ["runner-live-logs", "broker"],
      }),
    );
  });

  it("reports a non-benign broker stream error", async () => {
    pumpMock.mockImplementation(
      (handlers: { onOpen?: () => void; onStreamError: (message: string) => void }) =>
        new Promise<void>((resolve) => {
          handlers.onOpen?.();
          handlers.onStreamError("broker connection reset");
          resolve();
        }),
    );

    const { result } = renderHook(() =>
      useLiveLogStream("execution-1", false, "passed", null, {
        organizationId: "organization-1",
        canvasId: "canvas-1",
      }),
    );

    await waitFor(() => expect(result.current.error).toBe("broker connection reset"));
    expect(captureExceptionMock).toHaveBeenCalledOnce();
    expect(captureExceptionMock).toHaveBeenCalledWith(
      expect.objectContaining({ message: "broker connection reset" }),
      expect.objectContaining({
        fingerprint: ["runner-live-logs", "broker"],
      }),
    );
  });

  it("preserves existing logs when a retry opens a healthy stream", async () => {
    pumpMock.mockImplementationOnce(async (handlers: { onOpen?: () => void; onLogLine: (line: string) => void }) => {
      handlers.onOpen?.();
      handlers.onLogLine("existing output");
      throw new Error("Failed to fetch");
    });
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
    expect(result.current.orphanLines).toEqual(["existing output"]);
    act(() => result.current.retry());

    await waitFor(() => expect(result.current.error).toBeNull());
    expect(result.current.isLoading).toBe(false);
    expect(result.current.orphanLines).toEqual(["existing output"]);
  });

  it("resets parsed logs when terminal status starts a new session for the same execution", async () => {
    pumpMock.mockImplementation(
      async (handlers: {
        onOpen?: () => void;
        onCmdStart?: (index: number, text: string, startedAtMs: number | null, kind?: string, preview?: string) => void;
        onToolStart?: (kind: string, text: string, id?: string) => void;
        onLogLine: (line: string, commandIndex?: number) => void;
      }) => {
        handlers.onOpen?.();
        handlers.onToolStart?.("grep", "rootTriggerRenderer", "toolu_grep");
        handlers.onLogLine("Found 1 matches");
        handlers.onLogLine("/home/ubuntu/repo/web_src/.eslint-budget-baseline.json:");
        handlers.onCmdStart?.(5, "Implementation", 1, "prompt", "You are implementing");
      },
    );

    const session = { organizationId: "organization-1", canvasId: "canvas-1" };
    const { result, rerender } = renderHook(
      ({ inFlight, status }: { inFlight: boolean; status: "passed" | "failed" | null }) =>
        useLiveLogStream("execution-1", inFlight, status, status ? 9_000 : null, session),
      { initialProps: { inFlight: true, status: null as "passed" | "failed" | null } },
    );

    await waitFor(() => {
      expect(result.current.sections).toHaveLength(1);
      const tools = result.current.sections[0]?.events[0];
      expect(tools?.kind).toBe("tools");
    });

    rerender({ inFlight: false, status: "passed" });

    await waitFor(() => expect(pumpMock).toHaveBeenCalledTimes(2));
    await waitFor(() => {
      expect(result.current.sections).toHaveLength(1);
      const tools = result.current.sections[0]?.events[0];
      expect(tools?.kind).toBe("tools");
      if (tools?.kind !== "tools") {
        throw new Error("expected tools group");
      }
      expect(tools.tools).toHaveLength(1);
      expect(tools.tools[0]?.lines).toEqual([
        "Found 1 matches",
        "/home/ubuntu/repo/web_src/.eslint-budget-baseline.json:",
      ]);
    });
  });

  it("keeps parsed logs when a terminal session fails to open", async () => {
    pumpMock.mockImplementationOnce(
      async (handlers: {
        onOpen?: () => void;
        onCmdStart?: (index: number, text: string, startedAtMs: number | null, kind?: string, preview?: string) => void;
        onLogLine: (line: string, commandIndex?: number) => void;
      }) => {
        handlers.onOpen?.();
        handlers.onCmdStart?.(0, "Clone Repo", 1, "bash", "git clone");
        handlers.onLogLine("Cloning into 'repo'...", 0);
        return new Promise(() => undefined);
      },
    );
    pumpMock.mockRejectedValueOnce(new Error("Failed to fetch"));

    const session = { organizationId: "organization-1", canvasId: "canvas-1" };
    const { result, rerender } = renderHook(
      ({ inFlight, status }: { inFlight: boolean; status: "passed" | "failed" | null }) =>
        useLiveLogStream("execution-1", inFlight, status, status ? 9_000 : null, session),
      { initialProps: { inFlight: true, status: null as "passed" | "failed" | null } },
    );

    await waitFor(() => expect(result.current.sections[0]?.lines).toEqual(["Cloning into 'repo'..."]));

    rerender({ inFlight: false, status: "passed" });

    await waitFor(() => expect(result.current.error).toBe("Failed to fetch"));
    expect(result.current.sections).toHaveLength(1);
    expect(result.current.sections[0]).toMatchObject({
      text: "Clone Repo",
      lines: ["Cloning into 'repo'..."],
    });
  });

  it("does not duplicate finished section lines on in-flight reconnect", async () => {
    pumpMock.mockImplementationOnce(
      async (handlers: {
        onOpen?: () => void;
        onCmdStart?: (index: number, text: string, startedAtMs: number | null, kind?: string, preview?: string) => void;
        onCmdEnd?: (index: number, status: "passed" | "failed", durationMs: number) => void;
        onLogLine: (line: string, commandIndex?: number) => void;
      }) => {
        handlers.onOpen?.();
        handlers.onCmdStart?.(0, "Clone Repo", 1, "bash", "git clone");
        handlers.onLogLine("Cloning into 'repo'...", 0);
        handlers.onCmdEnd?.(0, "passed", 20);
      },
    );
    pumpMock.mockImplementationOnce(
      async (handlers: {
        onOpen?: () => void;
        onCmdStart?: (index: number, text: string, startedAtMs: number | null, kind?: string, preview?: string) => void;
        onCmdEnd?: (index: number, status: "passed" | "failed", durationMs: number) => void;
        onLogLine: (line: string, commandIndex?: number) => void;
      }) => {
        handlers.onOpen?.();
        handlers.onCmdStart?.(0, "Clone Repo", 1, "bash", "git clone");
        handlers.onLogLine("Cloning into 'repo'...", 0);
        handlers.onCmdEnd?.(0, "passed", 20);
        return new Promise(() => undefined);
      },
    );

    const { result } = renderHook(() =>
      useLiveLogStream("execution-1", true, null, null, {
        organizationId: "organization-1",
        canvasId: "canvas-1",
      }),
    );

    await waitFor(() => expect(result.current.sections[0]?.lines).toEqual(["Cloning into 'repo'..."]));
    await waitFor(() => expect(pumpMock).toHaveBeenCalledTimes(2), { timeout: 5000 });
    expect(result.current.sections).toHaveLength(1);
    expect(result.current.sections[0]?.lines).toEqual(["Cloning into 'repo'..."]);
  });

  it("keeps unindexed records after cmd_end across in-flight reconnect", async () => {
    type LiveLogHandlers = {
      onOpen?: () => void;
      onCmdStart?: (index: number, text: string, startedAtMs: number | null, kind?: string, preview?: string) => void;
      onCmdEnd?: (index: number, status: "passed" | "failed", durationMs: number) => void;
      onToolStart?: (kind: string, text: string, id?: string) => void;
      onLogLine: (line: string, commandIndex?: number) => void;
    };
    const playHistory = (handlers: LiveLogHandlers, includeNextCommand: boolean) => {
      handlers.onOpen?.();
      handlers.onCmdStart?.(0, "Clone Repo", 1, "bash", "git clone");
      handlers.onLogLine("Cloning into 'repo'...", 0);
      handlers.onCmdEnd?.(0, "passed", 20);
      handlers.onToolStart?.("grep", "rootTriggerRenderer", "toolu_grep");
      handlers.onLogLine("Found 1 matches");
      if (includeNextCommand) {
        handlers.onCmdStart?.(5, "Implementation", 2, "prompt", "You are implementing");
      }
    };

    pumpMock.mockImplementationOnce(async (handlers: LiveLogHandlers) => {
      playHistory(handlers, false);
    });
    pumpMock.mockImplementationOnce(async (handlers: LiveLogHandlers) => {
      playHistory(handlers, true);
      return new Promise(() => undefined);
    });

    const { result } = renderHook(() =>
      useLiveLogStream("execution-1", true, null, null, {
        organizationId: "organization-1",
        canvasId: "canvas-1",
      }),
    );

    await waitFor(() => expect(result.current.orphanLines).toEqual(["Found 1 matches"]));
    await waitFor(() => expect(pumpMock).toHaveBeenCalledTimes(2), { timeout: 5000 });
    await waitFor(() => expect(result.current.sections).toHaveLength(2));
    expect(result.current.sections[0]?.lines).toEqual(["Cloning into 'repo'..."]);
    expect(result.current.orphanLines).toEqual([]);
    const tools = result.current.sections[1]?.events[0];
    expect(tools?.kind).toBe("tools");
    if (tools?.kind !== "tools") {
      throw new Error("expected tools group");
    }
    expect(tools.tools).toHaveLength(1);
    expect(tools.tools[0]).toMatchObject({
      kind: "grep",
      text: "rootTriggerRenderer",
      sourceId: "toolu_grep",
      lines: ["Found 1 matches"],
    });
  });
});
