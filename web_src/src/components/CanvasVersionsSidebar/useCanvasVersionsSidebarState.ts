import { useCallback, useState } from "react";

interface CanvasVersionsSidebarStateConfig {
  defaultOpen?: boolean;
}

export function useCanvasVersionsSidebarState(config: CanvasVersionsSidebarStateConfig = {}) {
  const { defaultOpen = true } = config;
  const [isVersionsSidebarOpen, setIsVersionsSidebarOpen] = useState(defaultOpen);

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
