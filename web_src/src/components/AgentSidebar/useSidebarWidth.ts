import type { MouseEvent as ReactMouseEvent } from "react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";

export const AGENT_SIDEBAR_WIDTH_STORAGE_KEY = "agentSidebarWidth";

/** Legacy client-side AI builder keys; removed on mount so stale cache cannot linger. */
const LEGACY_AI_BUILDER_STORAGE_KEY_PREFIX = "sp:canvas-ai-builder:";

const DEFAULT_SIDEBAR_WIDTH_PX = 400;
const SIDEBAR_WIDTH_MIN_PX = 280;
const SIDEBAR_WIDTH_MAX_PX = 560;

type SidebarWidthOptions = {
  storageKey?: string;
  defaultWidthPx?: number;
  minWidthPx?: number;
  maxWidthPx?: number;
};

function readInitialSidebarWidthPx(storageKey: string, defaultWidthPx: number): number {
  if (typeof window === "undefined") {
    return defaultWidthPx;
  }

  const saved = window.localStorage.getItem(storageKey);
  const parsed = saved ? parseInt(saved, 10) : NaN;

  return Number.isFinite(parsed) ? parsed : defaultWidthPx;
}

export function useSidebarWidth({
  storageKey = AGENT_SIDEBAR_WIDTH_STORAGE_KEY,
  defaultWidthPx = DEFAULT_SIDEBAR_WIDTH_PX,
  minWidthPx = SIDEBAR_WIDTH_MIN_PX,
  maxWidthPx = SIDEBAR_WIDTH_MAX_PX,
}: SidebarWidthOptions = {}) {
  const sidebarRef = useRef<HTMLDivElement>(null);
  const [sidebarWidth, setSidebarWidth] = useState(() => readInitialSidebarWidthPx(storageKey, defaultWidthPx));
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
    window.localStorage.setItem(storageKey, String(sidebarWidth));
  }, [sidebarWidth, storageKey]);

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
      const clampedWidth = Math.max(minWidthPx, Math.min(maxWidthPx, newWidth));
      setSidebarWidth(clampedWidth);
    },
    [isResizing, minWidthPx, maxWidthPx],
  );

  const handleResizeMouseUp = useCallback(() => {
    setIsResizing(false);
  }, []);

  useEffect(() => {
    if (isResizing) {
      document.addEventListener("mousemove", handleResizeMouseMove);
      document.addEventListener("mouseup", handleResizeMouseUp);
      document.body.style.cursor = "col-resize";
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
