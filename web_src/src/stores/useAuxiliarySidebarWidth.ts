import { useCallback, useEffect, useRef } from "react";
import {
  AUX_SIDEBAR_MIN_WIDTH,
  useAuxLeftSidebarMount,
  useSidebarLayoutStore,
  useSidebarLayoutViewport,
} from "@/stores/sidebarLayoutStore";

export function useAuxiliarySidebarWidth(isOpen: boolean, storageKey: string, defaultWidth: number) {
  const sidebarRef = useRef<HTMLDivElement>(null);
  const dragStartRef = useRef<{ clientX: number; width: number } | null>(null);
  const width = useSidebarLayoutStore((state) => state.auxLeftWidth);
  const isResizing = useSidebarLayoutStore((state) => state.isAuxLeftResizing);
  const setAuxLeftResizing = useSidebarLayoutStore((state) => state.setAuxLeftResizing);
  const resizeAuxLeft = useSidebarLayoutStore((state) => state.resizeAuxLeft);

  useAuxLeftSidebarMount(isOpen, storageKey, defaultWidth);
  useSidebarLayoutViewport();

  const handleMouseDown = useCallback(
    (event: React.MouseEvent) => {
      event.preventDefault();
      dragStartRef.current = { clientX: event.clientX, width };
      setAuxLeftResizing(true);
    },
    [setAuxLeftResizing, width],
  );

  useEffect(() => {
    if (!isResizing) return;

    const handleMouseMove = (event: MouseEvent) => {
      const dragStart = dragStartRef.current;
      if (!dragStart) {
        return;
      }

      const targetWidth = dragStart.width + event.clientX - dragStart.clientX;
      resizeAuxLeft(Math.max(AUX_SIDEBAR_MIN_WIDTH, Math.round(targetWidth)));
    };

    const handleMouseUp = () => {
      dragStartRef.current = null;
      setAuxLeftResizing(false);
    };

    document.addEventListener("mousemove", handleMouseMove);
    document.addEventListener("mouseup", handleMouseUp);
    document.body.style.cursor = "col-resize";
    document.body.style.userSelect = "none";

    return () => {
      document.removeEventListener("mousemove", handleMouseMove);
      document.removeEventListener("mouseup", handleMouseUp);
      dragStartRef.current = null;
      document.body.style.cursor = "";
      document.body.style.userSelect = "";
    };
  }, [isResizing, resizeAuxLeft, setAuxLeftResizing]);

  return { sidebarRef, width, isResizing, handleMouseDown };
}
