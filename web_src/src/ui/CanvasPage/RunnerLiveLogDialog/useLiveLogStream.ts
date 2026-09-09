import { useCanvasId } from "@/hooks/useCanvasId";
import { useOrganizationId } from "@/hooks/useOrganizationId";
import { useCallback, useEffect, useMemo, useState, type Dispatch, type SetStateAction } from "react";
import { LiveLogStream, type LiveLogStreamHandlers } from "./liveLogStream";
import {
  appendLineToLatestSection,
  closeOpenTool,
  completeCommandSection,
  endToolOnLatestSection,
  shouldSkipUnindexedLiveLogReplay,
  startCommandSection,
  startToolOnLatestSection,
  type CommandStart,
} from "./liveLogSections";
import type { CommandSection, LogState } from "./types";
import { useScrollToBottom } from "./useScrollToBottom";
import type { ExecutionInfo } from "../../../pages/app/mappers/types";
import {
  applyPromptUsageRecord,
  emptyAgentRunTelemetry,
  emptyPromptUsageState,
  promptUsageSeries,
  startPromptUsageSeries,
  type AgentPromptUsageState,
} from "@/lib/agentRunTelemetry";

const RECONNECT_DELAY_MS = 2000;

const initialLogState: LogState = {
  sections: [],
  orphanLines: [],
  error: null,
  isStreaming: false,
};

function hasRunningCommand(state: LogState): boolean {
  return state.sections.some((section) => section.status === "running");
}

function hasFinishedCommandSection(state: LogState): boolean {
  return state.sections.some((section) => section.status !== "running");
}

export function terminalCommandStatusForExecution(execution: ExecutionInfo): "passed" | "failed" | null {
  if (execution.state !== "STATE_FINISHED") {
    return null;
  }

  return execution.result === "RESULT_PASSED" ? "passed" : "failed";
}

export function terminalTimeMsForExecution(execution: ExecutionInfo): number | null {
  if (execution.state !== "STATE_FINISHED") {
    return null;
  }

  const timestamp = execution.updatedAt || execution.createdAt;
  const parsed = Date.parse(timestamp);
  return Number.isFinite(parsed) ? parsed : null;
}

export function finalizeRunningCommandSections(
  state: LogState,
  status: "passed" | "failed",
  endedAtMs: number | null,
): LogState {
  if (!hasRunningCommand(state)) {
    return state;
  }

  return {
    ...state,
    sections: state.sections.map((section) => {
      if (section.status !== "running") {
        return section;
      }

      return {
        ...closeOpenTool(section, status),
        status,
        duration_ms: commandSectionFinalDuration(section, endedAtMs),
        collapsed: status === "passed",
      };
    }),
  };
}

function commandSectionFinalDuration(section: CommandSection, endedAtMs: number | null): number {
  if (section.started_at === null || endedAtMs === null) {
    return section.duration_ms ?? 0;
  }

  return Math.max(0, endedAtMs - section.started_at);
}

function applyStreamFailure(state: LogState, message: string, executionInFlight: boolean): LogState {
  if (
    hasRunningCommand(state) ||
    (executionInFlight && state.sections.length === 0 && state.orphanLines.length === 0)
  ) {
    return {
      ...state,
      error: null,
    };
  }

  if (state.sections.length === 0 && state.orphanLines.length === 0) {
    return { ...state, error: message };
  }

  return state;
}

type StreamHandlerContext = {
  reconnecting: boolean;
  replayLineSkip: Map<number, number>;
  executionInFlight: boolean;
  setState: Dispatch<SetStateAction<LogState>>;
  setUsage: Dispatch<SetStateAction<AgentPromptUsageState>>;
  commandCursor: { index?: number };
};

function createStreamHandlers(ctx: StreamHandlerContext): LiveLogStreamHandlers {
  const { reconnecting, replayLineSkip, executionInFlight, setState, setUsage, commandCursor } = ctx;
  return {
    onLogLine: (text, commandIndex) => {
      const index = commandIndex ?? commandCursor.index;
      setState((prev) => appendReplayedLogLine(prev, text, replayLineSkip, index, reconnecting));
    },
    onStreamError: (message) => setState((prev) => applyStreamFailure(prev, message, executionInFlight)),
    onCmdStart: (index, text, startedAtMs, kind, preview) => {
      commandCursor.index = index;
      setState((prev) =>
        rememberOrStartCommand(prev, { index, text, startedAtMs, kind, preview }, replayLineSkip, reconnecting),
      );
      if (kind === "prompt") {
        setUsage((prev) => startPromptUsageSeries(prev, text, index));
      }
    },
    onCmdEnd: (index, status, durationMs) =>
      setState((prev) => completeCommandSection(prev, index, status, durationMs)),
    onToolStart: (kind, text, id, turn) => {
      setState((prev) =>
        startReplayedTool(prev, { kind, text, sourceId: id, commandIndex: commandCursor.index }, reconnecting),
      );
      setUsage((prev) => applyPromptUsageRecord(prev, { type: "tool_start", kind, text, id, turn }));
    },
    onToolEnd: (status, durationMs, id, turn) => {
      setState((prev) =>
        endReplayedTool(prev, { status, durationMs, sourceId: id, commandIndex: commandCursor.index }, reconnecting),
      );
      setUsage((prev) => applyPromptUsageRecord(prev, { type: "tool_end", status, duration_ms: durationMs, id, turn }));
    },
    onTurn: (turn, usage, message) => {
      setUsage((prev) => applyPromptUsageRecord(prev, { type: "turn", turn, usage, message }));
    },
  };
}

function appendReplayedLogLine(
  state: LogState,
  text: string,
  replayLineSkip: Map<number, number>,
  commandIndex: number | undefined,
  reconnecting: boolean,
): LogState {
  if (shouldSkipUnindexedLiveLogReplay(reconnecting, commandIndex, hasFinishedCommandSection(state))) {
    return state;
  }
  return appendLineToLatestSection(state, text, replayLineSkip, commandIndex);
}

function rememberOrStartCommand(
  state: LogState,
  start: CommandStart,
  replayLineSkip: Map<number, number>,
  reconnecting: boolean,
): LogState {
  const existing = state.sections.find((section) => section.index === start.index);
  if (!existing) {
    return startCommandSection(state, start);
  }
  if (reconnecting) {
    replayLineSkip.set(start.index, existing.lines.length);
  }
  return state;
}

function startReplayedTool(
  state: LogState,
  tool: { kind: string; text: string; sourceId?: string; commandIndex?: number },
  reconnecting: boolean,
): LogState {
  if (shouldSkipUnindexedLiveLogReplay(reconnecting, tool.commandIndex, hasFinishedCommandSection(state))) {
    return state;
  }
  return startToolOnLatestSection(state, tool.kind, tool.text, tool.sourceId, tool.commandIndex);
}

function endReplayedTool(
  state: LogState,
  tool: { status: "passed" | "failed"; durationMs: number; sourceId?: string; commandIndex?: number },
  reconnecting: boolean,
): LogState {
  if (shouldSkipUnindexedLiveLogReplay(reconnecting, tool.commandIndex, hasFinishedCommandSection(state))) {
    return state;
  }
  return endToolOnLatestSection(state, tool.status, tool.durationMs, tool.sourceId, tool.commandIndex);
}

function sleep(ms: number, signal: AbortSignal): Promise<void> {
  return new Promise((resolve, reject) => {
    if (signal.aborted) {
      reject(new DOMException("Aborted", "AbortError"));
      return;
    }

    const timeout = window.setTimeout(() => {
      signal.removeEventListener("abort", onAbort);
      resolve();
    }, ms);

    const onAbort = () => {
      window.clearTimeout(timeout);
      reject(new DOMException("Aborted", "AbortError"));
    };

    signal.addEventListener("abort", onAbort, { once: true });
  });
}

type LiveLogSessionParams = {
  organizationId: string;
  canvasId: string;
  executionId: string;
  executionInFlight: boolean;
  terminalCommandStatus: "passed" | "failed" | null;
  terminalAtMs: number | null;
  sessionAbort: AbortController;
  setState: Dispatch<SetStateAction<LogState>>;
  setUsage: Dispatch<SetStateAction<AgentPromptUsageState>>;
  setActiveStream: (stream: LiveLogStream | null) => void;
};

async function pumpLiveLogConnection(params: LiveLogSessionParams, reconnecting: boolean): Promise<"aborted" | "open"> {
  const {
    organizationId,
    canvasId,
    executionId,
    executionInFlight,
    sessionAbort,
    setState,
    setUsage,
    setActiveStream,
  } = params;
  const stream = new LiveLogStream(organizationId, canvasId, executionId);
  setActiveStream(stream);
  try {
    await stream.pump(
      createStreamHandlers({
        reconnecting,
        replayLineSkip: new Map<number, number>(),
        executionInFlight,
        setState,
        setUsage,
        commandCursor: {},
      }),
    );
  } catch (error) {
    if ((error as Error).name === "AbortError") {
      return "aborted";
    }
    if (!sessionAbort.signal.aborted) {
      setState((prev) => applyStreamFailure(prev, (error as Error).message, executionInFlight));
    }
  } finally {
    stream.stop();
    setActiveStream(null);
  }
  return sessionAbort.signal.aborted ? "aborted" : "open";
}

async function runLiveLogSession(params: LiveLogSessionParams): Promise<void> {
  const { executionInFlight, terminalCommandStatus, terminalAtMs, sessionAbort, setState } = params;
  let reconnecting = false;

  while (!sessionAbort.signal.aborted) {
    const result = await pumpLiveLogConnection(params, reconnecting);
    if (result === "aborted") {
      return;
    }
    if (!executionInFlight) {
      if (terminalCommandStatus) {
        setState((prev) => finalizeRunningCommandSections(prev, terminalCommandStatus, terminalAtMs));
      }
      return;
    }

    reconnecting = true;
    setState((prev) => ({ ...prev, isStreaming: false }));
    try {
      await sleep(RECONNECT_DELAY_MS, sessionAbort.signal);
    } catch {
      return;
    }
    setState((prev) => ({ ...prev, isStreaming: true }));
  }
}

export type LiveLogStreamSession = {
  organizationId?: string;
  canvasId?: string;
};

export function useLiveLogStream(
  executionId: string,
  executionInFlight: boolean,
  terminalCommandStatus: "passed" | "failed" | null,
  terminalAtMs: number | null,
  session?: LiveLogStreamSession,
) {
  const routeOrganizationId = useOrganizationId();
  const routeCanvasId = useCanvasId();
  const organizationId = session?.organizationId || routeOrganizationId;
  const canvasId = session?.canvasId || routeCanvasId;
  const [state, setState] = useState<LogState>(() => ({ ...initialLogState, isStreaming: true }));
  const [usage, setUsage] = useState(emptyPromptUsageState);

  const scrollTrigger = useMemo(() => {
    const lineCount = state.sections.reduce((count, section) => count + section.lines.length, 0);
    return `${state.sections.length}:${state.orphanLines.length}:${lineCount}`;
  }, [state.sections, state.orphanLines]);

  const { scrollRef } = useScrollToBottom(scrollTrigger);

  const toggleSection = useCallback((index: number) => {
    setState((prev) => ({
      ...prev,
      sections: prev.sections.map((section) => {
        if (section.index !== index) {
          return section;
        }
        return {
          ...section,
          collapsed: !section.collapsed,
        };
      }),
    }));
  }, []);

  useEffect(() => {
    if (!organizationId || !canvasId || !executionId) {
      setState((prev) => ({ ...prev, isStreaming: false }));
      return;
    }

    const sessionAbort = new AbortController();
    let activeStream: LiveLogStream | null = null;
    setState({ ...initialLogState, isStreaming: true });
    setUsage(emptyPromptUsageState());

    void runLiveLogSession({
      organizationId,
      canvasId,
      executionId,
      executionInFlight,
      terminalCommandStatus,
      terminalAtMs,
      sessionAbort,
      setState,
      setUsage,
      setActiveStream: (stream) => {
        activeStream = stream;
      },
    }).finally(() => {
      if (!sessionAbort.signal.aborted) {
        setState((prev) => ({ ...prev, isStreaming: false }));
      }
    });

    return () => {
      sessionAbort.abort();
      activeStream?.stop();
    };
  }, [organizationId, canvasId, executionId, executionInFlight, terminalCommandStatus, terminalAtMs]);

  const usageSeries = useMemo(() => promptUsageSeries(usage), [usage]);
  const telemetry = usageSeries.at(-1)?.telemetry ?? emptyAgentRunTelemetry();
  return { ...state, telemetry, usageSeries, toggleSection, scrollRef };
}
