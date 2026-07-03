import { useCallback, useEffect, useRef, useState } from "react";
import { persistAgentMode, readInitialAgentMode, type AgentMode } from "@/components/AgentSidebar/agentMode";
import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";

// Keep in sync with pkg/features/features.go.
export const FEATURE_CLAUDE_MANAGED_AGENTS = "claude_managed_agents";
/** Legacy global key kept as a fallback when a canvas id is not available. */
const CANVAS_TOOL_SIDEBAR_OPEN_STORAGE_KEY = "canvasAgentSidebarOpen";

/**
 * localStorage key for the agent sidebar open/closed preference. The preference
 * is stored per canvas so each app remembers its own state; the legacy global
 * key is used only when no canvas id is available.
 */
export function canvasAgentSidebarOpenStorageKey(canvasId?: string): string {
  return canvasId ? `${CANVAS_TOOL_SIDEBAR_OPEN_STORAGE_KEY}:${canvasId}` : CANVAS_TOOL_SIDEBAR_OPEN_STORAGE_KEY;
}

/** Persist the agent sidebar open/closed preference for a specific canvas. */
export function writeCanvasAgentSidebarOpen(canvasId: string, open: boolean): void {
  if (typeof window === "undefined") return;
  try {
    window.localStorage.setItem(canvasAgentSidebarOpenStorageKey(canvasId), open ? "true" : "false");
  } catch {
    // Ignore storage failures (e.g. private mode); preference is best-effort.
  }
}

function readInitialToolSidebarOpen(canvasId?: string): boolean {
  if (typeof window === "undefined") return false;
  try {
    return window.localStorage.getItem(canvasAgentSidebarOpenStorageKey(canvasId)) === "true";
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

  const [isToolSidebarOpen, setIsToolSidebarOpen] = useState(() => readInitialToolSidebarOpen(canvasId));
  const [agentMode, setAgentMode] = useState<AgentMode>(readInitialAgentMode);

  // Tracks the canvas whose managed-agent provider failed to provision a
  // session (e.g. the instance has no agent credentials configured). Keyed by
  // canvas id so navigating to another canvas re-evaluates availability.
  const [agentUnavailableCanvasId, setAgentUnavailableCanvasId] = useState<string | undefined>(undefined);
  const agentUnavailable = Boolean(canvasId) && agentUnavailableCanvasId === canvasId;

  // Re-read the preference when navigating between canvases (the open/closed
  // state is stored per canvas) so each app keeps its own sidebar state.
  const previousCanvasIdRef = useRef(canvasId);
  useEffect(() => {
    if (previousCanvasIdRef.current === canvasId) return;
    previousCanvasIdRef.current = canvasId;
    setIsToolSidebarOpen(readInitialToolSidebarOpen(canvasId));
    setAgentUnavailableCanvasId(undefined);
  }, [canvasId]);

  const persistOpen = useCallback(
    (open: boolean) => {
      if (typeof window === "undefined") return;
      window.localStorage.setItem(canvasAgentSidebarOpenStorageKey(canvasId), open ? "true" : "false");
    },
    [canvasId],
  );

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

  // Called by the agent panel when it cannot set up a chat because the agent
  // provider isn't configured on this instance. Hiding the toggle and closing
  // the panel avoids advertising a chat that can never work (issue #5803).
  const markAgentUnavailable = useCallback(() => {
    setAgentUnavailableCanvasId(canvasId);
  }, [canvasId]);

  const markAgentAvailable = useCallback(() => {
    if (agentUnavailable) {
      setIsToolSidebarOpen(readInitialToolSidebarOpen(canvasId));
    }
    setAgentUnavailableCanvasId((currentCanvasId) => (currentCanvasId === canvasId ? undefined : currentCanvasId));
  }, [agentUnavailable, canvasId]);

  useEffect(() => {
    if ((!featureEnabled && !forceEnable) || hideCanvasToolSidebar) {
      setIsToolSidebarOpen(false);
    }
  }, [featureEnabled, forceEnable, hideCanvasToolSidebar]);

  const showToolSidebarToggle = (featureEnabled || forceEnable) && !hideCanvasToolSidebar;

  useEffect(() => {
    if (!showToolSidebarToggle) return;

    const handleKeyDown = (event: KeyboardEvent) => {
      // Some instrumentation SDKs dispatch synthetic keyboard events where `key` is unset.
      const { key } = event as KeyboardEvent & { key?: unknown };
      const lowerKey = typeof key === "string" ? key.toLowerCase() : "";

      if (lowerKey !== "b" || !(event.metaKey || event.ctrlKey) || event.altKey || event.shiftKey) {
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
    agentUnavailable,
    markAgentUnavailable,
    markAgentAvailable,
    handleToolSidebarToggle,
    openToolSidebar,
    closeToolSidebar,
    agentMode,
    switchAgentMode,
  };
}

export type CanvasToolSidebarState = ReturnType<typeof useCanvasToolSidebarState>;
