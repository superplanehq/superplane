import { useCallback, useEffect, useState } from "react";
import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";

// Keep in sync with pkg/features/features.go.
const FEATURE_CLAUDE_MANAGED_AGENTS = "claude_managed_agents";
const CANVAS_AGENT_SIDEBAR_STORAGE_KEY = "canvasAgentSidebarOpen";
const CANVAS_AGENT_MODE_STORAGE_KEY = "canvasAgentMode";

export type AgentMode = "builder" | "operator" | "architect";

function readInitialAgentSidebarOpen(): boolean {
  if (typeof window === "undefined") return false;
  try {
    return window.localStorage.getItem(CANVAS_AGENT_SIDEBAR_STORAGE_KEY) === "true";
  } catch {
    return false;
  }
}

function readInitialAgentMode(): AgentMode {
  if (typeof window === "undefined") return "operator";
  try {
    const stored = window.localStorage.getItem(CANVAS_AGENT_MODE_STORAGE_KEY);
    if (stored === "builder" || stored === "architect") return stored;
    return "operator";
  } catch {
    return "operator";
  }
}

export type UseAgentStateOptions = {
  isEditing: boolean;
  readOnly: boolean;
  canvasId?: string;
  organizationId?: string;
  hideAddControls?: boolean;
};

export function useAgentState({
  isEditing,
  readOnly,
  canvasId,
  organizationId,
  hideAddControls,
}: UseAgentStateOptions) {
  const { has: hasFeature } = useExperimentalFeature(organizationId);
  const featureEnabled = hasFeature(FEATURE_CLAUDE_MANAGED_AGENTS);

  const [isAgentSidebarOpen, setIsAgentSidebarOpen] = useState(readInitialAgentSidebarOpen);
  const [agentMode, setAgentMode] = useState<AgentMode>(readInitialAgentMode);

  const persistOpen = useCallback((open: boolean) => {
    if (typeof window === "undefined") return;
    window.localStorage.setItem(CANVAS_AGENT_SIDEBAR_STORAGE_KEY, open ? "true" : "false");
  }, []);

  const closeSidebar = useCallback(() => {
    setIsAgentSidebarOpen(false);
    persistOpen(false);
  }, [persistOpen]);

  const handleAgentSidebarToggle = useCallback(() => {
    setIsAgentSidebarOpen((prev) => {
      const next = !prev;
      persistOpen(next);
      return next;
    });
  }, [persistOpen]);

  const switchAgentMode = useCallback((mode: AgentMode) => {
    setAgentMode(mode);
    if (typeof window !== "undefined") {
      window.localStorage.setItem(CANVAS_AGENT_MODE_STORAGE_KEY, mode);
    }
  }, []);

  // The agent is read-only-safe: a user can still ask questions about a
  // published canvas without editing it. Only the feature flag and the
  // canvas-creation-mode flag (hideAddControls) gate the sidebar entirely.
  useEffect(() => {
    if (!featureEnabled || hideAddControls) closeSidebar();
  }, [featureEnabled, hideAddControls, closeSidebar]);

  const showAgentSidebarToggle = featureEnabled && !hideAddControls;

  return {
    canvasId,
    organizationId,
    isEditing,
    readOnly,
    isAgentSidebarOpen,
    showAgentSidebarToggle,
    handleAgentSidebarToggle,
    closeSidebar,
    agentMode,
    switchAgentMode,
  };
}

export type AgentState = ReturnType<typeof useAgentState>;
