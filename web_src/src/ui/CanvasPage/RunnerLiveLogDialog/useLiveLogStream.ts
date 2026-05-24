import { useCanvasId } from "@/hooks/useCanvasId";
import { useOrganizationId } from "@/hooks/useOrganizationId";
import { useCallback, useEffect, useMemo, useRef, useState, type Dispatch, type SetStateAction } from "react";
import { LiveLogStream, type LiveLogStreamHandlers } from "./liveLogStream";
import type { CommandSection, LogState } from "./types";
import { useScrollToBottom } from "./useScrollToBottom";

const RECONNECT_DELAY_MS = 2000;

const initialLogState: LogState = {
  sections: [],
  orphanLines: [],
  error: null,
  streamWarning: null,
  isStreaming: false,
};

function hasRunningCommand(state: LogState): boolean {
  return state.sections.some((section) => section.status === "running");
}

function applyStreamFailure(state: LogState, message: string): LogState {
  if (hasRunningCommand(state)) {
    return {
      ...state,
      streamWarning: "Live log stream interrupted. Reconnecting…",
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
    streamWarning: null,
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
    streamWarning: null,
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
    streamWarning: null,
  };
}

function createStreamHandlers(
  reconnecting: boolean,
  replayLineSkip: Map<number, number>,
  setState: Dispatch<SetStateAction<LogState>>,
): LiveLogStreamHandlers {
  return {
    onLogLine: (text) => setState((prev) => appendLineToLatestSection(prev, text, replayLineSkip)),
    onStreamError: (message) => setState((prev) => applyStreamFailure(prev, message)),
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

export function useLiveLogStream(executionId: string) {
  const organizationId = useOrganizationId();
  const canvasId = useCanvasId();
  const [state, setState] = useState<LogState>(initialLogState);
  const stateRef = useRef(state);
  stateRef.current = state;

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

    (async () => {
      let reconnecting = false;

      while (!sessionAbort.signal.aborted) {
        const stream = new LiveLogStream(organizationId, canvasId, executionId);
        activeStream = stream;
        const replayLineSkip = new Map<number, number>();

        try {
          await stream.pump(createStreamHandlers(reconnecting, replayLineSkip, setState));
        } catch (error) {
          if ((error as Error).name === "AbortError") {
            return;
          }
          if (!sessionAbort.signal.aborted) {
            setState((prev) => applyStreamFailure(prev, (error as Error).message));
          }
        } finally {
          stream.stop();
          if (activeStream === stream) {
            activeStream = null;
          }
        }

        if (sessionAbort.signal.aborted) {
          return;
        }

        if (!hasRunningCommand(stateRef.current)) {
          break;
        }

        reconnecting = true;
        setState((prev) => ({
          ...prev,
          isStreaming: false,
          streamWarning: "Live log stream interrupted. Reconnecting…",
        }));

        try {
          await sleep(RECONNECT_DELAY_MS, sessionAbort.signal);
        } catch {
          return;
        }

        setState((prev) => ({ ...prev, isStreaming: true }));
      }

      if (!sessionAbort.signal.aborted) {
        setState((prev) => ({ ...prev, isStreaming: false }));
      }
    })();

    return () => {
      sessionAbort.abort();
      activeStream?.stop();
    };
  }, [organizationId, canvasId, executionId]);

  return { ...state, toggleSection, scrollRef };
}
