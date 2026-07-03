import { useCallback, useState } from "react";

export function useCanvasVersionsSidebarState() {
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
