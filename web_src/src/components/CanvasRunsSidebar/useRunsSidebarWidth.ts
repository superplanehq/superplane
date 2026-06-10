import { useCallback, useEffect, useRef, useState } from "react";

export const RUNS_SIDEBAR_WIDTH_STORAGE_KEY = "runs-sidebar-width";
export const RUNS_SIDEBAR_MIN_WIDTH = 300;
export const RUNS_SIDEBAR_DEFAULT_WIDTH = 380;

function readPersistedWidth(): number {
  if (typeof window === "undefined") return RUNS_SIDEBAR_DEFAULT_WIDTH;
  try {
    const saved = window.localStorage.getItem(RUNS_SIDEBAR_WIDTH_STORAGE_KEY);
    const parsed = saved ? Number.parseInt(saved, 10) : Number.NaN;
    if (!Number.isFinite(parsed)) return RUNS_SIDEBAR_DEFAULT_WIDTH;
    return Math.max(RUNS_SIDEBAR_MIN_WIDTH, parsed);
  } catch {
    return RUNS_SIDEBAR_DEFAULT_WIDTH;
  }
}

function persistWidth(value: number): void {
  if (typeof window === "undefined") return;
  try {
    window.localStorage.setItem(RUNS_SIDEBAR_WIDTH_STORAGE_KEY, String(value));
  } catch {
    // ignore storage errors
  }
}

export function useRunsSidebarWidth() {
  const sidebarRef = useRef<HTMLDivElement>(null);
  const [width, setWidth] = useState(readPersistedWidth);
  const [isResizing, setIsResizing] = useState(false);

  const handleMouseDown = useCallback((event: React.MouseEvent) => {
    event.preventDefault();
    setIsResizing(true);
  }, []);

  useEffect(() => {
    if (!isResizing) return;

    const handleMouseMove = (event: MouseEvent) => {
      const rect = sidebarRef.current?.getBoundingClientRect();
      const left = rect?.left ?? 0;
      const next = Math.max(RUNS_SIDEBAR_MIN_WIDTH, Math.round(event.clientX - left));
      setWidth(next);
      persistWidth(next);
    };

    const handleMouseUp = () => setIsResizing(false);

    document.addEventListener("mousemove", handleMouseMove);
    document.addEventListener("mouseup", handleMouseUp);
    document.body.style.cursor = "col-resize";
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
