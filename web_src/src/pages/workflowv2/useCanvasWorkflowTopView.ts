import { useCallback, useState } from "react";

export type CanvasWorkflowTopViewTab = "canvas" | "yaml" | "cli" | "memory";

export function useCanvasWorkflowTopView() {
  const [topViewMode, setTopViewMode] = useState<CanvasWorkflowTopViewTab>("canvas");

  const applyTopViewMode = useCallback((mode: CanvasWorkflowTopViewTab) => {
    setTopViewMode(mode);
  }, []);

  const goToCanvasEditorView = useCallback(() => {
    setTopViewMode("canvas");
  }, []);

  return {
    resolvedTopViewMode: topViewMode,
    applyTopViewMode,
    goToCanvasEditorView,
  };
}
