import type { CanvasMemoryEntry } from "@/hooks/useCanvasData";

import { CanvasMemoryView } from "./CanvasMemoryView";

interface DeleteCanvasMemoryMutation {
  mutate: (memoryId: string) => void;
  isPending: boolean;
  variables: string | undefined;
}

interface WorkflowMemoryOverlayLayerProps {
  isMemoryMode: boolean;
  // True when the viewer is allowed to delete memory entries. Computed by the
  // workflow page via `canEditCanvasMemory` so the gating logic stays in one
  // place.
  canDelete: boolean;
  entries: CanvasMemoryEntry[];
  isLoading: boolean;
  error: unknown;
  deleteCanvasMemoryEntry: DeleteCanvasMemoryMutation;
}

export function WorkflowMemoryOverlayLayer({
  isMemoryMode,
  canDelete,
  entries,
  isLoading,
  error,
  deleteCanvasMemoryEntry,
}: WorkflowMemoryOverlayLayerProps) {
  if (!isMemoryMode) return null;

  return (
    <CanvasMemoryView
      entries={entries}
      isLoading={isLoading}
      errorMessage={error instanceof Error ? error.message : undefined}
      onDeleteEntry={canDelete ? (memoryId) => deleteCanvasMemoryEntry.mutate(memoryId) : undefined}
      deletingId={deleteCanvasMemoryEntry.isPending ? deleteCanvasMemoryEntry.variables : undefined}
    />
  );
}
