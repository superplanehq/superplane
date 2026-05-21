import type { CanvasMemoryEntry } from "@/hooks/useCanvasData";

import { MemoryOverlay } from "./MemoryOverlay";

interface DeleteCanvasMemoryMutation {
  mutate: (memoryId: string) => void;
  isPending: boolean;
  variables: string | undefined;
}

interface WorkflowMemoryOverlayLayerProps {
  isMemoryMode: boolean;
  isViewingDraftVersion: boolean;
  isViewingLiveVersion: boolean;
  canUpdateCanvas: boolean;
  entries: CanvasMemoryEntry[];
  isLoading: boolean;
  error: unknown;
  deleteCanvasMemoryEntry: DeleteCanvasMemoryMutation;
}

export function WorkflowMemoryOverlayLayer({
  isMemoryMode,
  isViewingDraftVersion,
  isViewingLiveVersion,
  canUpdateCanvas,
  entries,
  isLoading,
  error,
  deleteCanvasMemoryEntry,
}: WorkflowMemoryOverlayLayerProps) {
  if (!isMemoryMode) return null;

  return (
    <MemoryOverlay
      entries={isViewingDraftVersion ? [] : entries}
      isLoading={isViewingDraftVersion ? false : isLoading}
      errorMessage={isViewingDraftVersion ? undefined : error instanceof Error ? error.message : undefined}
      onDeleteEntry={
        canUpdateCanvas && isViewingLiveVersion ? (memoryId) => deleteCanvasMemoryEntry.mutate(memoryId) : undefined
      }
      deletingId={deleteCanvasMemoryEntry.isPending ? deleteCanvasMemoryEntry.variables : undefined}
    />
  );
}
