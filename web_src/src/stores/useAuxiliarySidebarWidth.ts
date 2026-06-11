import { useCallback, useEffect, useRef } from "react";
import {
  AUX_SIDEBAR_MIN_WIDTH,
  useAuxLeftSidebarMount,
  useSidebarLayoutStore,
  useSidebarLayoutViewport,
} from "@/stores/sidebarLayoutStore";

export function useAuxiliarySidebarWidth(isOpen: boolean, storageKey: string, defaultWidth: number) {
  const sidebarRef = useRef<HTMLDivElement>(null);
  const width = useSidebarLayoutStore((state) => state.auxLeftWidth);
  const isResizing = useSidebarLayoutStore((state) => state.isAuxLeftResizing);
  const setAuxLeftResizing = useSidebarLayoutStore((state) => state.setAuxLeftResizing);
  const resizeAuxLeft = useSidebarLayoutStore((state) => state.resizeAuxLeft);

  useAuxLeftSidebarMount(isOpen, storageKey, defaultWidth);
  useSidebarLayoutViewport();

  const handleMouseDown = useCallback(
    (event: React.MouseEvent) => {
      event.preventDefault();
      setAuxLeftResizing(true);
    },
    [setAuxLeftResizing],
  );

  useEffect(() => {
    if (!isResizing) return;

    const handleMouseMove = (event: MouseEvent) => {
      const rect = sidebarRef.current?.getBoundingClientRect();
      const left = rect?.left ?? 0;
      resizeAuxLeft(Math.max(AUX_SIDEBAR_MIN_WIDTH, Math.round(event.clientX - left)));
    };

    const handleMouseUp = () => setAuxLeftResizing(false);

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
  }, [isResizing, resizeAuxLeft, setAuxLeftResizing]);

  return { sidebarRef, width, isResizing, handleMouseDown };
}
