import { useCallback, useEffect, useState } from "react";

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
  agentContextEnabled: boolean;
  hideAddControls?: boolean;
  readOnly: boolean;
};

export function useAgentState({ agentContextEnabled, hideAddControls = false, readOnly }: UseAgentStateOptions) {
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
    if (!agentContextEnabled) {
      closeAgentSidebar();
    }
  }, [agentContextEnabled, closeAgentSidebar]);

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

  return {
    isAgentSidebarOpen,
    handleAgentSidebarOpenChange,
    handleAgentSidebarToggle,
  };
}
