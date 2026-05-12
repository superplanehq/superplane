import { useCallback, useEffect, useRef, useState } from "react";

export const RUNS_SIDEBAR_WIDTH_STORAGE_KEY = "runs-sidebar-width";
export const RUNS_SIDEBAR_MIN_WIDTH = 280;
export const RUNS_SIDEBAR_MAX_WIDTH = 640;
export const RUNS_SIDEBAR_DEFAULT_WIDTH = 340;

export function useResizableSidebar() {
  const sidebarRef = useRef<HTMLDivElement>(null);
  const [width, setWidth] = useState(() => {
    const saved = typeof window !== "undefined" ? localStorage.getItem(RUNS_SIDEBAR_WIDTH_STORAGE_KEY) : null;
    const parsed = saved ? parseInt(saved, 10) : NaN;
    if (!Number.isFinite(parsed)) return RUNS_SIDEBAR_DEFAULT_WIDTH;
    return Math.max(RUNS_SIDEBAR_MIN_WIDTH, Math.min(RUNS_SIDEBAR_MAX_WIDTH, parsed));
  });
  const [isResizing, setIsResizing] = useState(false);

  useEffect(() => {
    localStorage.setItem(RUNS_SIDEBAR_WIDTH_STORAGE_KEY, String(width));
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
      const nextWidth = Math.max(RUNS_SIDEBAR_MIN_WIDTH, Math.min(RUNS_SIDEBAR_MAX_WIDTH, event.clientX - left));
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
