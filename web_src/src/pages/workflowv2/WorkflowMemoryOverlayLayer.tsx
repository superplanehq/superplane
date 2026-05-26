import type { CanvasMemoryEntry } from "@/hooks/useCanvasData";

import { CanvasMemoryView } from "./CanvasMemoryView";

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
  // Mirrors the workflow page's app-wide read-only state so the trash
  // affordance disappears entirely when the canvas is not in edit mode.
  isReadOnly: boolean;
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
  isReadOnly,
  entries,
  isLoading,
  error,
  deleteCanvasMemoryEntry,
}: WorkflowMemoryOverlayLayerProps) {
  if (!isMemoryMode) return null;

  const canDeleteEntry = !isReadOnly && canUpdateCanvas && isViewingLiveVersion && !isViewingDraftVersion;

  return (
    <CanvasMemoryView
      entries={entries}
      isLoading={isLoading}
      errorMessage={error instanceof Error ? error.message : undefined}
      onDeleteEntry={canDeleteEntry ? (memoryId) => deleteCanvasMemoryEntry.mutate(memoryId) : undefined}
      deletingId={deleteCanvasMemoryEntry.isPending ? deleteCanvasMemoryEntry.variables : undefined}
    />
  );
}
