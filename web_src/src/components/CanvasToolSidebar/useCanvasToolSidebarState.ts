import { useCallback, useEffect, useState } from "react";
import { persistAgentMode, readInitialAgentMode, type AgentMode } from "@/components/AgentSidebar/agentMode";
import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";

// Keep in sync with pkg/features/features.go.
export const FEATURE_CLAUDE_MANAGED_AGENTS = "claude_managed_agents";
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
  /** Called before the user closes the tool sidebar via the header toggle. */
  onBeforeClose?: () => void;
};

export function useCanvasToolSidebarState({
  isEditing,
  readOnly,
  canvasId,
  organizationId,
  hideCanvasToolSidebar,
  forceEnable = false,
  onBeforeClose,
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
    if (isToolSidebarOpen) onBeforeClose?.();

    const next = !isToolSidebarOpen;
    setIsToolSidebarOpen(next);
    persistOpen(next);
  }, [isToolSidebarOpen, onBeforeClose, persistOpen]);

  const switchAgentMode = useCallback((mode: AgentMode) => {
    setAgentMode(mode);
    persistAgentMode(mode);
  }, []);

  useEffect(() => {
    if ((!featureEnabled && !forceEnable) || hideCanvasToolSidebar) closeToolSidebar();
  }, [featureEnabled, forceEnable, hideCanvasToolSidebar, closeToolSidebar]);

  const showToolSidebarToggle = (featureEnabled || forceEnable) && !hideCanvasToolSidebar;

  useEffect(() => {
    if (!showToolSidebarToggle) return;

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key.toLowerCase() !== "b" || !(event.metaKey || event.ctrlKey) || event.altKey || event.shiftKey) {
        return;
      }

      const target = event.target;
      if (
        target instanceof Element &&
        target.closest('input, textarea, select, [contenteditable="true"], .monaco-editor')
      ) {
        return;
      }

      event.preventDefault();
      handleToolSidebarToggle();
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [showToolSidebarToggle, handleToolSidebarToggle]);

  return {
    canvasId,
    organizationId,
    isEditing,
    readOnly,
    isToolSidebarOpen,
    showToolSidebarToggle,
    isAgentEnabled: featureEnabled,
    handleToolSidebarToggle,
    openToolSidebar,
    closeToolSidebar,
    agentMode,
    switchAgentMode,
  };
}

export type CanvasToolSidebarState = ReturnType<typeof useCanvasToolSidebarState>;
