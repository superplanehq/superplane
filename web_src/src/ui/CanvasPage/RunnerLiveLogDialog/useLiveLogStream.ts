import { useCanvasId } from "@/hooks/useCanvasId";
import { useOrganizationId } from "@/hooks/useOrganizationId";
import { useCallback, useEffect, useMemo, useState, type Dispatch, type SetStateAction } from "react";
import { LiveLogStream, type LiveLogStreamHandlers } from "./liveLogStream";
import {
  appendLineToLatestSection,
  closeOpenTool,
  completeCommandSection,
  endToolOnLatestSection,
  startCommandSection,
  startToolOnLatestSection,
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
  isStreaming: false,
};

function hasRunningCommand(state: LogState): boolean {
  return state.sections.some((section) => section.status === "running");
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
  return { ...state, error: message, isStreaming: false };
}

function createStreamHandlers(
  reconnecting: boolean,
  replayLineSkip: Map<number, number>,
  setState: Dispatch<SetStateAction<LogState>>,
  setUsage: Dispatch<SetStateAction<AgentPromptUsageState>>,
  onFailure: (message: string) => void,
): LiveLogStreamHandlers {
  return {
    onLogLine: (text) =>
      setState((prev) => ({ ...appendLineToLatestSection(prev, text, replayLineSkip), error: null })),
    onStreamError: (message) => {
      onFailure(message);
      setState((prev) => applyStreamFailure(prev, message));
    },
    onCmdStart: (index, text, startedAtMs, kind, preview) => {
      setState((prev) => {
        const existing = prev.sections.find((section) => section.index === index);
        if (existing) {
          if (reconnecting) {
            replayLineSkip.set(index, existing.lines.length);
          }
          return { ...prev, error: null };
        }
        return { ...startCommandSection(prev, { index, text, startedAtMs, kind, preview }), error: null };
      });
      if (kind === "prompt") {
        setUsage((prev) => startPromptUsageSeries(prev, text, index));
      }
    },
    onCmdEnd: (index, status, durationMs) =>
      setState((prev) => ({ ...completeCommandSection(prev, index, status, durationMs), error: null })),
    onToolStart: (kind, text, id, turn) => {
      setState((prev) => ({ ...startToolOnLatestSection(prev, kind, text, id), error: null }));
      setUsage((prev) => applyPromptUsageRecord(prev, { type: "tool_start", kind, text, id, turn }));
    },
    onToolEnd: (status, durationMs, id, turn) => {
      setState((prev) => ({ ...endToolOnLatestSection(prev, status, durationMs, id), error: null }));
      setUsage((prev) => applyPromptUsageRecord(prev, { type: "tool_end", status, duration_ms: durationMs, id, turn }));
    },
    onTurn: (turn, usage, message) => {
      setState((prev) => ({ ...prev, error: null }));
      setUsage((prev) => applyPromptUsageRecord(prev, { type: "turn", turn, usage, message }));
    },
  };
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
  setState((prev) => ({ ...prev, isStreaming: false }));
  try {
    await sleep(RECONNECT_DELAY_MS, sessionAbort.signal);
  } catch {
    return false;
  }
  setState((prev) => ({ ...prev, isStreaming: true }));
  return true;
}

async function runLiveLogSession({
  organizationId,
  canvasId,
  executionId,
  executionInFlight,
  terminalCommandStatus,
  terminalAtMs,
  sessionAbort,
  setState,
  setUsage,
  setActiveStream,
}: LiveLogSessionParams): Promise<void> {
  let reconnecting = false;
  const reportFailure = createLiveLogFailureReporter({ organizationId, canvasId, executionId });

  while (!sessionAbort.signal.aborted) {
    const stream = new LiveLogStream(organizationId, canvasId, executionId);
    setActiveStream(stream);
    const replayLineSkip = new Map<number, number>();

    try {
      await stream.pump(
        createStreamHandlers(reconnecting, replayLineSkip, setState, setUsage, (message) => {
          reportFailure("broker", new Error(message));
        }),
      );
    } catch (error) {
      const streamError = errorFromUnknown(error);
      if (streamError.name === "AbortError") {
        return;
      }
      if (!sessionAbort.signal.aborted) {
        reportFailure("request", streamError);
        setState((prev) => applyStreamFailure(prev, streamError.message));
      }
    } finally {
      stream.stop();
      setActiveStream(null);
    }

    if (sessionAbort.signal.aborted) {
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
  const [state, setState] = useState<LogState>(() => ({ ...initialLogState, isStreaming: true }));
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
  }, [organizationId, canvasId, executionId, executionInFlight, terminalCommandStatus, terminalAtMs, sessionAttempt]);

  const usageSeries = useMemo(() => promptUsageSeries(usage), [usage]);
  const telemetry = usageSeries.at(-1)?.telemetry ?? emptyAgentRunTelemetry();
  return { ...state, telemetry, usageSeries, retry, toggleSection, scrollRef };
}
