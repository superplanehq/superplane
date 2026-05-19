import { useCallback, useEffect, useState } from "react";
import { persistAgentMode, readInitialAgentMode, type AgentMode } from "@/components/AgentSidebar/agentMode";
import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";

// Keep in sync with pkg/features/features.go.
const FEATURE_CLAUDE_MANAGED_AGENTS = "claude_managed_agents";
/** Key unchanged so existing browser state continues to work. */
const CANVAS_TOOL_SIDEBAR_OPEN_STORAGE_KEY = "canvasAgentSidebarOpen";

function readInitialToolSidebarOpen(): boolean {
  if (typeof window === "undefined") return false;
  try {
    return window.localStorage.getItem(CANVAS_TOOL_SIDEBAR_OPEN_STORAGE_KEY) === "true";
  } catch {
    return false;
  }
}

export type UseCanvasToolSidebarStateOptions = {
  isEditing: boolean;
  readOnly: boolean;
  canvasId?: string;
  organizationId?: string;
  /** When true (e.g. template canvas picker), hides the tool sidebar toggle and clears open state. */
  hideCanvasToolSidebar?: boolean;
  /** Keeps the tool sidebar available even when managed agents are disabled (runs/versions flows). */
  forceEnable?: boolean;
};

export function useCanvasToolSidebarState({
  isEditing,
  readOnly,
  canvasId,
  organizationId,
  hideCanvasToolSidebar,
  forceEnable = false,
}: UseCanvasToolSidebarStateOptions) {
  const { has: hasFeature } = useExperimentalFeature(organizationId);
  const featureEnabled = hasFeature(FEATURE_CLAUDE_MANAGED_AGENTS);

  const [isToolSidebarOpen, setIsToolSidebarOpen] = useState(readInitialToolSidebarOpen);
  const [agentMode, setAgentMode] = useState<AgentMode>(readInitialAgentMode);

  const persistOpen = useCallback((open: boolean) => {
    if (typeof window === "undefined") return;
    window.localStorage.setItem(CANVAS_TOOL_SIDEBAR_OPEN_STORAGE_KEY, open ? "true" : "false");
  }, []);

  const closeToolSidebar = useCallback(() => {
    setIsToolSidebarOpen(false);
    persistOpen(false);
  }, [persistOpen]);

  const openToolSidebar = useCallback(() => {
    setIsToolSidebarOpen(true);
    persistOpen(true);
  }, [persistOpen]);

  const handleToolSidebarToggle = useCallback(() => {
    setIsToolSidebarOpen((prev) => {
      const next = !prev;
      persistOpen(next);
      return next;
    });
  }, [persistOpen]);

  const switchAgentMode = useCallback((mode: AgentMode) => {
    setAgentMode(mode);
    persistAgentMode(mode);
  }, []);

  useEffect(() => {
    if ((!featureEnabled && !forceEnable) || hideCanvasToolSidebar) closeToolSidebar();
  }, [featureEnabled, forceEnable, hideCanvasToolSidebar, closeToolSidebar]);

  const showToolSidebarToggle = (featureEnabled || forceEnable) && !hideCanvasToolSidebar;

  return {
    canvasId,
    organizationId,
    isEditing,
    readOnly,
    isToolSidebarOpen,
    showToolSidebarToggle,
    handleToolSidebarToggle,
    openToolSidebar,
    closeToolSidebar,
    agentMode,
    switchAgentMode,
  };
}

export type CanvasToolSidebarState = ReturnType<typeof useCanvasToolSidebarState>;
