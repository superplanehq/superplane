import { useCallback, useState } from "react";

export function useCanvasVersionsSidebarState() {
  // The versions/history sidebar starts collapsed; it is never auto-opened as
  // the initial view (see issue #5803) and is only shown via the header toggle.
  const [isVersionsSidebarOpen, setIsVersionsSidebarOpen] = useState(false);

  const handleVersionsSidebarToggle = useCallback(() => {
    setIsVersionsSidebarOpen((current) => !current);
  }, []);

  const openVersionsSidebar = useCallback(() => {
    setIsVersionsSidebarOpen(true);
  }, []);

  const closeVersionsSidebar = useCallback(() => {
    setIsVersionsSidebarOpen(false);
  }, []);

  return {
    isVersionsSidebarOpen,
    handleVersionsSidebarToggle,
    openVersionsSidebar,
    closeVersionsSidebar,
  };
}

export type CanvasVersionsSidebarState = ReturnType<typeof useCanvasVersionsSidebarState> & {
  showVersionsSidebarToggle: boolean;
};
