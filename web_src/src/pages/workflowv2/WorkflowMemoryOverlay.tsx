import type { CanvasMemoryEntry } from "@/hooks/useCanvasData";
import { CanvasMemoryView } from "./CanvasMemoryView";

export type WorkflowMemoryOverlayProps = {
  isMemoryMode: boolean;
  entries: CanvasMemoryEntry[];
  isLoading?: boolean;
  errorMessage?: string;
  onDeleteEntry?: (memoryId: string) => void;
  deletingId?: string;
};

export function WorkflowMemoryOverlay({
  isMemoryMode,
  entries,
  isLoading,
  errorMessage,
  onDeleteEntry,
  deletingId,
}: WorkflowMemoryOverlayProps) {
  if (!isMemoryMode) return null;

  return (
    <div
      className="absolute inset-x-0 bottom-0 top-20 z-10 flex flex-col bg-slate-50"
      data-testid="memory-overlay"
    >
      <CanvasMemoryView
        entries={entries}
        isLoading={isLoading}
        errorMessage={errorMessage}
        onDeleteEntry={onDeleteEntry}
        deletingId={deletingId}
      />
    </div>
  );
}
