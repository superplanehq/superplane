import { useCanvasId } from "@/hooks/useCanvasId";
import { useOrganizationId } from "@/hooks/useOrganizationId";
import { useCallback, useEffect, useState } from "react";
import { LiveLogStream } from "./liveLogStream";
import type { CommandSection, LogState } from "./types";
import { useScrollToBottom } from "./useScrollToBottom";

const initialLogState: LogState = {
  sections: [],
  orphanLines: [],
  error: null,
  isStreaming: false,
};

function appendLineToLatestSection(state: LogState, text: string): LogState {
  if (state.sections.length === 0) {
    return {
      ...state,
      orphanLines: [...state.orphanLines, text],
    };
  }

  const nextSections = [...state.sections];
  const lastSectionIndex = nextSections.length - 1;
  nextSections[lastSectionIndex] = {
    ...nextSections[lastSectionIndex],
    lines: [...nextSections[lastSectionIndex].lines, text],
  };
  return {
    ...state,
    sections: nextSections,
  };
}

function pushCommandSection(state: LogState, index: number, text: string): LogState {
  const section: CommandSection = {
    index,
    text,
    lines: [],
    status: "running",
    duration_ms: null,
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

export function useLiveLogStream(executionId: string) {
  const organizationId = useOrganizationId();
  const canvasId = useCanvasId();
  const [state, setState] = useState<LogState>(initialLogState);

  const { scrollRef } = useScrollToBottom(state);

  const toggleSection = useCallback((index: number) => {
    setState((prev) => ({
      ...prev,
      sections: prev.sections.map((section) => {
        if (section.index !== index || section.status !== "passed") {
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

    setState({ ...initialLogState, isStreaming: true });

    const stream = new LiveLogStream(organizationId, canvasId, executionId);

    (async () => {
      try {
        await stream.pump({
          onLogLine: (t) => setState((prev) => appendLineToLatestSection(prev, t)),
          onStreamError: (m) => setState((prev) => ({ ...prev, error: m })),
          onCmdStart: (index, text) => setState((prev) => pushCommandSection(prev, index, text)),
          onCmdEnd: (index, status, durationMs) => setState((prev) => completeCommandSection(prev, index, status, durationMs)),
        });
      } catch (e) {
        if ((e as Error).name === "AbortError") {
          return;
        }
        setState((prev) => ({ ...prev, error: (e as Error).message }));
      } finally {
        setState((prev) => ({ ...prev, isStreaming: false }));
      }
    })();

    return () => stream.stop();
  }, [organizationId, canvasId, executionId]);

  return { ...state, toggleSection, scrollRef };
}
