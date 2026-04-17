import type { CanvasChangesetChange } from "@/api-client";
import { useCallback, useEffect, useState } from "react";

import { useAgentContext } from "./agentChat";

export const CANVAS_AGENT_SIDEBAR_STORAGE_KEY = "canvasAgentSidebarOpen";

function readInitialAgentSidebarOpen(): boolean {
  if (typeof window === "undefined") {
    return false;
  }

  const stored = window.localStorage.getItem(CANVAS_AGENT_SIDEBAR_STORAGE_KEY);
  if (stored === null) {
    return false;
  }

  try {
    return JSON.parse(stored) === true;
  } catch (error) {
    console.warn("Failed to parse agent sidebar state from local storage:", error);
    return false;
  }
}

export type UseAgentStateOptions = {
  isEditing: boolean;
  canvasVersion: string;
  hideAddControls?: boolean;
  readOnly: boolean;
  canvasId?: string;
  organizationId?: string;
  onApplyAiOperations?: (changes: CanvasChangesetChange[]) => Promise<void>;
};

export function useAgentState({
  isEditing,
  canvasVersion,
  hideAddControls = false,
  readOnly,
  canvasId,
  organizationId,
  onApplyAiOperations,
}: UseAgentStateOptions) {
  const agentContext = useAgentContext(isEditing, canvasVersion);
  const [isAgentSidebarOpen, setIsAgentSidebarOpen] = useState(readInitialAgentSidebarOpen);

  const persistAgentSidebarOpen = useCallback((open: boolean) => {
    if (typeof window !== "undefined") {
      window.localStorage.setItem(CANVAS_AGENT_SIDEBAR_STORAGE_KEY, JSON.stringify(open));
    }
  }, []);

  const handleAgentSidebarOpenChange = useCallback(
    (open: boolean) => {
      setIsAgentSidebarOpen(open);
      persistAgentSidebarOpen(open);
    },
    [persistAgentSidebarOpen],
  );

  const handleAgentSidebarToggle = useCallback(() => {
    setIsAgentSidebarOpen((previous) => {
      const next = !previous;
      persistAgentSidebarOpen(next);
      return next;
    });
  }, [persistAgentSidebarOpen]);

  const closeAgentSidebar = useCallback(() => {
    setIsAgentSidebarOpen(false);
    persistAgentSidebarOpen(false);
  }, [persistAgentSidebarOpen]);

  useEffect(() => {
    if (!agentContext.enabled) {
      closeAgentSidebar();
    }
  }, [agentContext.enabled, closeAgentSidebar]);

  useEffect(() => {
    if (hideAddControls) {
      closeAgentSidebar();
    }
  }, [hideAddControls, closeAgentSidebar]);

  useEffect(() => {
    if (readOnly) {
      closeAgentSidebar();
    }
  }, [readOnly, closeAgentSidebar]);

  const showAgentSidebarToggle = agentContext.enabled && !hideAddControls && !readOnly;

  return {
    agentContext,
    isAgentSidebarOpen,
    handleAgentSidebarOpenChange,
    handleAgentSidebarToggle,
    canvasId,
    organizationId,
    showAgentSidebarToggle,
    readOnly,
    onApplyAiOperations: onApplyAiOperations ?? (async () => {}),
    closeSidebar: closeAgentSidebar,
  };
}

export type AgentState = ReturnType<typeof useAgentState>;
