import type { MouseEvent as ReactMouseEvent } from "react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";

export const AGENT_SIDEBAR_WIDTH_STORAGE_KEY = "agentSidebarWidth";

/** Legacy client-side AI builder keys; removed on mount so stale cache cannot linger. */
const LEGACY_AI_BUILDER_STORAGE_KEY_PREFIX = "sp:canvas-ai-builder:";

const DEFAULT_SIDEBAR_WIDTH_PX = 400;
const SIDEBAR_WIDTH_MIN_PX = 280;
const SIDEBAR_WIDTH_MAX_PX = 560;

function readInitialSidebarWidthPx(): number {
  if (typeof window === "undefined") {
    return DEFAULT_SIDEBAR_WIDTH_PX;
  }

  const saved = window.localStorage.getItem(AGENT_SIDEBAR_WIDTH_STORAGE_KEY);
  return saved ? parseInt(saved, 10) : DEFAULT_SIDEBAR_WIDTH_PX;
}

export function useSidebarWidth() {
  const sidebarRef = useRef<HTMLDivElement>(null);
  const [sidebarWidth, setSidebarWidth] = useState(readInitialSidebarWidthPx);
  const [isResizing, setIsResizing] = useState(false);

  useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }

    const keysToRemove: string[] = [];
    for (let index = 0; index < window.localStorage.length; index += 1) {
      const key = window.localStorage.key(index);
      if (key?.startsWith(LEGACY_AI_BUILDER_STORAGE_KEY_PREFIX)) {
        keysToRemove.push(key);
      }
    }

    for (const key of keysToRemove) {
      window.localStorage.removeItem(key);
    }
  }, []);

  useEffect(() => {
    window.localStorage.setItem(AGENT_SIDEBAR_WIDTH_STORAGE_KEY, String(sidebarWidth));
  }, [sidebarWidth]);

  const onResizeMouseDown = useCallback((event: ReactMouseEvent) => {
    event.preventDefault();
    setIsResizing(true);
  }, []);

  const handleResizeMouseMove = useCallback(
    (event: MouseEvent) => {
      if (!isResizing || !sidebarRef.current) {
        return;
      }

      const rect = sidebarRef.current.getBoundingClientRect();
      const newWidth = event.clientX - rect.left;
      const clampedWidth = Math.max(SIDEBAR_WIDTH_MIN_PX, Math.min(SIDEBAR_WIDTH_MAX_PX, newWidth));
      setSidebarWidth(clampedWidth);
    },
    [isResizing],
  );

  const handleResizeMouseUp = useCallback(() => {
    setIsResizing(false);
  }, []);

  useEffect(() => {
    if (isResizing) {
      document.addEventListener("mousemove", handleResizeMouseMove);
      document.addEventListener("mouseup", handleResizeMouseUp);
      document.body.style.cursor = "ew-resize";
      document.body.style.userSelect = "none";

      return () => {
        document.removeEventListener("mousemove", handleResizeMouseMove);
        document.removeEventListener("mouseup", handleResizeMouseUp);
        document.body.style.cursor = "";
        document.body.style.userSelect = "";
      };
    }
  }, [isResizing, handleResizeMouseMove, handleResizeMouseUp]);

  const sidebarStyle = useMemo(
    () => ({
      width: `${sidebarWidth}px`,
      minWidth: `${sidebarWidth}px`,
      maxWidth: `${sidebarWidth}px`,
    }),
    [sidebarWidth],
  );

  return {
    sidebarRef,
    sidebarWidth,
    isResizing,
    onResizeMouseDown,
    sidebarStyle,
  };
}
