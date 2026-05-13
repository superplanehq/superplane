import { useCallback, useEffect, useRef, useState } from "react";

export const AGENT_SIDEBAR_WIDTH_STORAGE_KEY = "agent-sidebar-width";
export const AGENT_SIDEBAR_MIN_WIDTH = 320;
export const AGENT_SIDEBAR_MAX_WIDTH = 720;
export const AGENT_SIDEBAR_DEFAULT_WIDTH = 380;

export function useSidebarWidth() {
  const sidebarRef = useRef<HTMLDivElement>(null);
  const [width, setWidth] = useState(() => {
    const saved = typeof window !== "undefined" ? localStorage.getItem(AGENT_SIDEBAR_WIDTH_STORAGE_KEY) : null;
    const parsed = saved ? parseInt(saved, 10) : NaN;
    if (!Number.isFinite(parsed)) return AGENT_SIDEBAR_DEFAULT_WIDTH;
    return Math.max(AGENT_SIDEBAR_MIN_WIDTH, Math.min(AGENT_SIDEBAR_MAX_WIDTH, parsed));
  });
  const [isResizing, setIsResizing] = useState(false);

  useEffect(() => {
    localStorage.setItem(AGENT_SIDEBAR_WIDTH_STORAGE_KEY, String(width));
  }, [width]);

  const handleMouseDown = useCallback((event: React.MouseEvent) => {
    event.preventDefault();
    setIsResizing(true);
  }, []);

  useEffect(() => {
    if (!isResizing) return;

    const handleMouseMove = (event: MouseEvent) => {
      const rect = sidebarRef.current?.getBoundingClientRect();
      const left = rect?.left ?? 0;
      const nextWidth = Math.max(AGENT_SIDEBAR_MIN_WIDTH, Math.min(AGENT_SIDEBAR_MAX_WIDTH, event.clientX - left));
      setWidth(nextWidth);
    };

    const handleMouseUp = () => setIsResizing(false);

    document.addEventListener("mousemove", handleMouseMove);
    document.addEventListener("mouseup", handleMouseUp);
    document.body.style.cursor = "ew-resize";
    document.body.style.userSelect = "none";

    return () => {
      document.removeEventListener("mousemove", handleMouseMove);
      document.removeEventListener("mouseup", handleMouseUp);
      document.body.style.cursor = "";
      document.body.style.userSelect = "";
    };
  }, [isResizing]);

  return { sidebarRef, width, isResizing, handleMouseDown };
}
