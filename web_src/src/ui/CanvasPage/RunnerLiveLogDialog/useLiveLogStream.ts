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
import { Sentry } from "@/sentry";
import {
  applyPromptUsageRecord,
  emptyAgentRunTelemetry,
  emptyPromptUsageState,
  promptUsageSeries,
  startPromptUsageSeries,
  type AgentPromptUsageState,
} from "@/lib/agentRunTelemetry";

const RECONNECT_DELAY_MS = 2000;

type LiveLogFailureSource = "broker" | "request";

type LiveLogFailureContext = {
  organizationId: string;
  canvasId: string;
  executionId: string;
};

const initialLogState: LogState = {
  sections: [],
  orphanLines: [],
  error: null,
  isLoading: false,
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

function applyStreamFailure(state: LogState, message: string): LogState {
  return { ...state, error: message, isLoading: false, isStreaming: false };
}

function withClearedError(state: LogState): LogState {
  return { ...state, error: null };
}

// CloudWatch Logs raises ResourceNotFoundException for GetLogEvents when the
// runner hasn't created its log stream yet (e.g. right after an execution
// starts, or during the brief window between reconnect attempts). The
// broker relays that as a stream error, but it's an expected, self-healing
// condition rather than an application bug: the live log session already
// reconnects automatically, and the log stream appears as soon as the
// runner starts writing to it. Surfacing it as a failure (in the UI or in
// Sentry) would just be noise, so it's ignored.
const BENIGN_BROKER_ERROR_PATTERN = /ResourceNotFoundException.*log stream .*(does not exist|not found)/i;

function isBenignBrokerError(message: string): boolean {
  return BENIGN_BROKER_ERROR_PATTERN.test(message);
}

type StreamHandlerContext = {
  reconnecting: boolean;
  replayLineSkip: Map<number, number>;
  setState: Dispatch<SetStateAction<LogState>>;
  setUsage: Dispatch<SetStateAction<AgentPromptUsageState>>;
  commandCursor: { index?: number };
  onFailure: (message: string) => void;
};

function createStreamHandlers(ctx: StreamHandlerContext): LiveLogStreamHandlers {
  const { reconnecting, replayLineSkip, setState, setUsage, commandCursor, onFailure } = ctx;
  return {
    onOpen: () => setState((prev) => ({ ...prev, error: null, isLoading: false, isStreaming: true })),
    onLogLine: (text, commandIndex) => {
      const index = commandIndex ?? commandCursor.index;
      setState((prev) => appendReplayedLogLine(prev, text, replayLineSkip, index, reconnecting));
    },
    onStreamError: (message) => {
      if (isBenignBrokerError(message)) {
        return;
      }
      onFailure(message);
      setState((prev) => applyStreamFailure(prev, message));
    },
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
      setState((prev) => withClearedError(completeCommandSection(prev, index, status, durationMs))),
    onToolStart: (kind, text, id, turn, startedAtMs) => {
      setState((prev) =>
        startReplayedTool(
          prev,
          { kind, text, sourceId: id, commandIndex: commandCursor.index, startedAtMs },
          reconnecting,
        ),
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
      setState((prev) => withClearedError(prev));
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
  return withClearedError(appendLineToLatestSection(state, text, replayLineSkip, commandIndex));
}

function rememberOrStartCommand(
  state: LogState,
  start: CommandStart,
  replayLineSkip: Map<number, number>,
  reconnecting: boolean,
): LogState {
  const existing = state.sections.find((section) => section.index === start.index);
  if (!existing) {
    return withClearedError(startCommandSection(state, start));
  }
  if (reconnecting) {
    replayLineSkip.set(start.index, existing.lines.length);
  }
  return withClearedError(state);
}

function startReplayedTool(
  state: LogState,
  tool: { kind: string; text: string; sourceId?: string; commandIndex?: number; startedAtMs?: number | null },
  reconnecting: boolean,
): LogState {
  if (shouldSkipUnindexedLiveLogReplay(reconnecting, tool.commandIndex, hasFinishedCommandSection(state))) {
    return state;
  }
  return withClearedError(
    startToolOnLatestSection(state, tool.kind, tool.text, tool.sourceId, tool.commandIndex, tool.startedAtMs),
  );
}

function endReplayedTool(
  state: LogState,
  tool: { status: "passed" | "failed"; durationMs: number; sourceId?: string; commandIndex?: number },
  reconnecting: boolean,
): LogState {
  if (shouldSkipUnindexedLiveLogReplay(reconnecting, tool.commandIndex, hasFinishedCommandSection(state))) {
    return state;
  }
  return withClearedError(
    endToolOnLatestSection(state, tool.status, tool.durationMs, tool.sourceId, tool.commandIndex),
  );
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

function createLiveLogFailureReporter(context: LiveLogFailureContext) {
  const reportedFailures = new Set<LiveLogFailureSource>();

  return (source: LiveLogFailureSource, error: Error): void => {
    if (reportedFailures.has(source)) {
      return;
    }
    reportedFailures.add(source);

    Sentry.captureException(error, {
      fingerprint: ["runner-live-logs", source],
      tags: {
        feature: "runner-live-logs",
        source,
      },
      extra: context,
    });
  };
}

function errorFromUnknown(error: unknown): Error {
  return error instanceof Error ? error : new Error(String(error));
}

async function waitForLiveLogReconnect(
  sessionAbort: AbortController,
  setState: Dispatch<SetStateAction<LogState>>,
): Promise<boolean> {
  setState((prev) => ({ ...prev, isLoading: true, isStreaming: false }));
  try {
    await sleep(RECONNECT_DELAY_MS, sessionAbort.signal);
  } catch {
    return false;
  }
  setState((prev) => ({ ...prev, isStreaming: true }));
  return true;
}

async function pumpLiveLogConnection(
  params: LiveLogSessionParams,
  reconnecting: boolean,
  reportFailure: (source: LiveLogFailureSource, error: Error) => void,
): Promise<"aborted" | "open"> {
  const { organizationId, canvasId, executionId, sessionAbort, setState, setUsage, setActiveStream } = params;
  const stream = new LiveLogStream(organizationId, canvasId, executionId);
  setActiveStream(stream);
  try {
    await stream.pump(
      createStreamHandlers({
        reconnecting,
        replayLineSkip: new Map<number, number>(),
        setState,
        setUsage,
        commandCursor: {},
        onFailure: (message) => reportFailure("broker", new Error(message)),
      }),
    );
  } catch (error) {
    const streamError = errorFromUnknown(error);
    if (streamError.name === "AbortError") {
      return "aborted";
    }
    if (!sessionAbort.signal.aborted) {
      reportFailure("request", streamError);
      setState((prev) => applyStreamFailure(prev, streamError.message));
    }
  } finally {
    stream.stop();
    setActiveStream(null);
  }
  return sessionAbort.signal.aborted ? "aborted" : "open";
}

async function runLiveLogSession(params: LiveLogSessionParams): Promise<void> {
  const {
    organizationId,
    canvasId,
    executionId,
    executionInFlight,
    terminalCommandStatus,
    terminalAtMs,
    sessionAbort,
    setState,
  } = params;
  let reconnecting = false;
  const reportFailure = createLiveLogFailureReporter({ organizationId, canvasId, executionId });

  while (!sessionAbort.signal.aborted) {
    const result = await pumpLiveLogConnection(params, reconnecting, reportFailure);
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
    if (!(await waitForLiveLogReconnect(sessionAbort, setState))) {
      return;
    }
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
  const [state, setState] = useState<LogState>(() => ({ ...initialLogState, isLoading: true, isStreaming: true }));
  const [usage, setUsage] = useState(emptyPromptUsageState);
  const [sessionAttempt, setSessionAttempt] = useState(0);

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

  const retry = useCallback(() => {
    setSessionAttempt((attempt) => attempt + 1);
  }, []);

  useEffect(() => {
    const canLoad = Boolean(organizationId && canvasId && executionId);
    setState({ ...initialLogState, isLoading: canLoad, isStreaming: canLoad });
    setUsage(emptyPromptUsageState());
  }, [organizationId, canvasId, executionId]);

  useEffect(() => {
    if (!organizationId || !canvasId || !executionId) {
      return;
    }

    const sessionAbort = new AbortController();
    let activeStream: LiveLogStream | null = null;
    setState((prev) => ({ ...prev, error: null, isLoading: true, isStreaming: true }));

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
        setState((prev) => ({ ...prev, isLoading: false, isStreaming: false }));
      }
    });

    return () => {
      sessionAbort.abort();
      activeStream?.stop();
    };
  }, [organizationId, canvasId, executionId, executionInFlight, terminalCommandStatus, terminalAtMs, sessionAttempt]);

  const usageSeries = useMemo(() => promptUsageSeries(usage), [usage]);
  const telemetry = usageSeries.at(-1)?.telemetry ?? emptyAgentRunTelemetry();
  return { ...state, telemetry, usageSeries, retry, toggleSection, scrollRef };
}
