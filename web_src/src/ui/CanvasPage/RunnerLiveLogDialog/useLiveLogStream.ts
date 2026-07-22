import { useCanvasId } from "@/hooks/useCanvasId";
import { useOrganizationId } from "@/hooks/useOrganizationId";
import { useCallback, useEffect, useMemo, useState, type Dispatch, type SetStateAction } from "react";
import { LiveLogStream, type LiveLogStreamHandlers } from "./liveLogStream";
import type { CommandSection, LogState } from "./types";
import { useScrollToBottom } from "./useScrollToBottom";
import type { ExecutionInfo } from "../../../pages/app/mappers/types";

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

export function shouldReconnectLiveLogSession(executionInFlight: boolean): boolean {
  return executionInFlight;
}

export function terminalCommandStatusForExecution(execution: ExecutionInfo): "passed" | "failed" | null {
  if (execution.state !== "STATE_FINISHED") {
    return null;
  }

  return execution.result === "RESULT_PASSED" ? "passed" : "failed";
}

export function terminalTimeMsForExecution(execution: ExecutionInfo): number | null {
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
        ...section,
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

function appendLineToLatestSection(state: LogState, text: string, replayLineSkip?: Map<number, number>): LogState {
  if (state.sections.length === 0) {
    return {
      ...state,
      orphanLines: [...state.orphanLines, text],
    };
  }

  const lastSectionIndex = state.sections.length - 1;
  const section = state.sections[lastSectionIndex];
  const skipLeft = replayLineSkip?.get(section.index) ?? 0;
  if (skipLeft > 0) {
    replayLineSkip?.set(section.index, skipLeft - 1);
    return state;
  }

  const nextSections = [...state.sections];
  nextSections[lastSectionIndex] = {
    ...section,
    lines: [...section.lines, text],
  };
  return {
    ...state,
    sections: nextSections,
  };
}

function pushCommandSection(state: LogState, index: number, text: string, startedAtMs: number | null): LogState {
  if (state.sections.some((section) => section.index === index)) {
    return state;
  }

  const section: CommandSection = {
    index,
    text,
    lines: [],
    status: "running",
    duration_ms: null,
    started_at: startedAtMs ?? Date.now(),
    collapsed: false,
  };

  return {
    ...state,
    sections: [...state.sections, section],
  };
}

function completeCommandSection(
  state: LogState,
  index: number,
  status: "passed" | "failed",
  durationMs: number,
): LogState {
  const existing = state.sections.find((section) => section.index === index);
  if (!existing || existing.status !== "running") {
    return state;
  }

  const nextSections = state.sections.map((section) => {
    if (section.index !== index) {
      return section;
    }
    return {
      ...section,
      status,
      duration_ms: durationMs,
      collapsed: status === "passed",
    };
  });

  return {
    ...state,
    sections: nextSections,
  };
}

function createStreamHandlers(
  reconnecting: boolean,
  replayLineSkip: Map<number, number>,
  executionInFlight: boolean,
  setState: Dispatch<SetStateAction<LogState>>,
): LiveLogStreamHandlers {
  return {
    onLogLine: (text) => setState((prev) => appendLineToLatestSection(prev, text, replayLineSkip)),
    onStreamError: (message) => setState((prev) => applyStreamFailure(prev, message, executionInFlight)),
    onCmdStart: (index, text, startedAtMs) => {
      setState((prev) => {
        const existing = prev.sections.find((section) => section.index === index);
        if (existing) {
          if (reconnecting) {
            replayLineSkip.set(index, existing.lines.length);
          }
          return prev;
        }
        return pushCommandSection(prev, index, text, startedAtMs);
      });
    },
    onCmdEnd: (index, status, durationMs) =>
      setState((prev) => completeCommandSection(prev, index, status, durationMs)),
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
  setActiveStream: (stream: LiveLogStream | null) => void;
};

async function runLiveLogSession({
  organizationId,
  canvasId,
  executionId,
  executionInFlight,
  terminalCommandStatus,
  terminalAtMs,
  sessionAbort,
  setState,
  setActiveStream,
}: LiveLogSessionParams): Promise<void> {
  let reconnecting = false;

  while (!sessionAbort.signal.aborted) {
    const stream = new LiveLogStream(organizationId, canvasId, executionId);
    setActiveStream(stream);
    const replayLineSkip = new Map<number, number>();

    try {
      await stream.pump(createStreamHandlers(reconnecting, replayLineSkip, executionInFlight, setState));
    } catch (error) {
      if ((error as Error).name === "AbortError") {
        return;
      }
      if (!sessionAbort.signal.aborted) {
        setState((prev) => applyStreamFailure(prev, (error as Error).message, executionInFlight));
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

    if (!shouldReconnectLiveLogSession(executionInFlight)) {
      return;
    }

    reconnecting = true;
    setState((prev) => ({
      ...prev,
      isStreaming: false,
    }));

    try {
      await sleep(RECONNECT_DELAY_MS, sessionAbort.signal);
    } catch {
      return;
    }

    setState((prev) => ({ ...prev, isStreaming: true }));
  }
}

export function useLiveLogStream(
  executionId: string,
  executionInFlight: boolean,
  terminalCommandStatus: "passed" | "failed" | null,
  terminalAtMs: number | null,
) {
  const organizationId = useOrganizationId();
  const canvasId = useCanvasId();
  const [state, setState] = useState<LogState>(initialLogState);

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
      return;
    }

    const sessionAbort = new AbortController();
    let activeStream: LiveLogStream | null = null;
    setState({ ...initialLogState, isStreaming: true });

    void runLiveLogSession({
      organizationId,
      canvasId,
      executionId,
      executionInFlight,
      terminalCommandStatus,
      terminalAtMs,
      sessionAbort,
      setState,
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

  return { ...state, toggleSection, scrollRef };
}
