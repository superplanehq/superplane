import { renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "bun:test";
import { useLiveLogStream } from "./useLiveLogStream";

const { pumpMock, stopMock } = vi.hoisted(() => ({
  pumpMock: vi.fn(),
  stopMock: vi.fn(),
}));

vi.mock("@/sentry", () => ({
  Sentry: { captureException: vi.fn() },
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

const liveLogSession = { organizationId: "organization-1", canvasId: "canvas-1" };

beforeEach(() => {
  pumpMock.mockReset();
  stopMock.mockReset();
});

describe("useLiveLogStream replay", () => {
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

    const { result, rerender } = renderHook(
      ({ inFlight, status }: { inFlight: boolean; status: "passed" | "failed" | null }) =>
        useLiveLogStream("execution-1", inFlight, status, status ? 9_000 : null, liveLogSession),
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

    const { result, rerender } = renderHook(
      ({ inFlight, status }: { inFlight: boolean; status: "passed" | "failed" | null }) =>
        useLiveLogStream("execution-1", inFlight, status, status ? 9_000 : null, liveLogSession),
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

    const { result } = renderHook(() => useLiveLogStream("execution-1", true, null, null, liveLogSession));

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

    const { result } = renderHook(() => useLiveLogStream("execution-1", true, null, null, liveLogSession));

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
