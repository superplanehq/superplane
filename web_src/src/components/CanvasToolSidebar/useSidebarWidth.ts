import { useCallback, useEffect, useRef } from "react";
import {
  SIDEBAR_MIN_WIDTH,
  useSidebarLayoutStore,
  useSidebarLayoutViewport,
  useSidebarMount,
} from "@/stores/sidebarLayoutStore";

/**
 * Backward-compat exports. The actual width state and constraint logic now
 * live in {@link useSidebarLayoutStore} so the left and right sidebars can
 * coordinate.
 */
export const CANVAS_TOOL_SIDEBAR_WIDTH_STORAGE_KEY = "agent-sidebar-width";
export const CANVAS_TOOL_SIDEBAR_MIN_WIDTH = SIDEBAR_MIN_WIDTH;
export const CANVAS_TOOL_SIDEBAR_DEFAULT_WIDTH = 380;

export function useSidebarWidth() {
  const sidebarRef = useRef<HTMLDivElement>(null);
  const width = useSidebarLayoutStore((state) => state.leftWidth);
  const isResizing = useSidebarLayoutStore((state) => state.isLeftResizing);
  const setLeftResizing = useSidebarLayoutStore((state) => state.setLeftResizing);
  const resizeLeft = useSidebarLayoutStore((state) => state.resizeLeft);

  useSidebarMount("left");
  useSidebarLayoutViewport();

  const handleMouseDown = useCallback(
    (event: React.MouseEvent) => {
      event.preventDefault();
      setLeftResizing(true);
    },
    [setLeftResizing],
  );

  useEffect(() => {
    if (!isResizing) return;

    const handleMouseMove = (event: MouseEvent) => {
      const rect = sidebarRef.current?.getBoundingClientRect();
      const left = rect?.left ?? 0;
      resizeLeft(event.clientX - left);
    };

    const handleMouseUp = () => setLeftResizing(false);

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
  }, [isResizing, resizeLeft, setLeftResizing]);

  return { sidebarRef, width, isResizing, handleMouseDown };
}
