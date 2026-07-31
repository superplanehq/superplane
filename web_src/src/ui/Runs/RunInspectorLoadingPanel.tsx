import { Loader2 } from "lucide-react";
import { RunInspectorChrome } from "./RunInspectorChrome";
import { ResizeHandle } from "./RunInspectorResize";
import { useResizableInspectorWidth } from "./useResizableInspectorWidth";

export function RunInspectorLoadingPanel({ onClose }: { onClose: () => void }) {
  const inspectorWidth = useResizableInspectorWidth();

  return (
    <aside
      className="relative z-20 flex h-full shrink-0 flex-col border-l border-edge-subtle bg-surface-raised text-content-primary"
      style={{ width: inspectorWidth.width }}
      aria-label="Run inspector"
    >
      <ResizeHandle onPointerDown={inspectorWidth.startResize} isResizing={inspectorWidth.isResizing} />
      <RunInspectorChrome onClose={onClose} />
      <div className="flex min-h-0 flex-1 items-center justify-center">
        <Loader2 className="h-5 w-5 animate-spin text-content-secondary" />
      </div>
    </aside>
  );
}
