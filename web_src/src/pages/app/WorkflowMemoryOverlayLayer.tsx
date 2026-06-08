import type {
  CanvasMemoryEntry,
  CreateCanvasMemoryNamespaceInput,
  UpdateCanvasMemoryNamespaceInput,
} from "@/hooks/useCanvasData";

import { CanvasMemoryView } from "./CanvasMemoryView";

interface DeleteCanvasMemoryMutation {
  mutate: (memoryId: string) => void;
  isPending: boolean;
  variables: string | undefined;
}

interface CreateCanvasMemoryNamespaceMutation {
  mutateAsync: (input: CreateCanvasMemoryNamespaceInput) => Promise<unknown>;
  isPending: boolean;
}

interface UpdateCanvasMemoryNamespaceMutation {
  mutateAsync: (input: UpdateCanvasMemoryNamespaceInput) => Promise<unknown>;
  isPending: boolean;
}

interface WorkflowMemoryOverlayLayerProps {
  isMemoryMode: boolean;
  // True when the viewer is allowed to mutate memory entries/namespaces.
  // Computed by the workflow page via `canEditCanvasMemory` so the gating
  // logic stays in one place.
  canEdit: boolean;
  entries: CanvasMemoryEntry[];
  isLoading: boolean;
  error: unknown;
  deleteCanvasMemoryEntry: DeleteCanvasMemoryMutation;
  createCanvasMemoryNamespace: CreateCanvasMemoryNamespaceMutation;
  updateCanvasMemoryNamespace: UpdateCanvasMemoryNamespaceMutation;
}

export function WorkflowMemoryOverlayLayer({
  isMemoryMode,
  canEdit,
  entries,
  isLoading,
  error,
  deleteCanvasMemoryEntry,
  createCanvasMemoryNamespace,
  updateCanvasMemoryNamespace,
}: WorkflowMemoryOverlayLayerProps) {
  if (!isMemoryMode) return null;

  return (
    <CanvasMemoryView
      entries={entries}
      isLoading={isLoading}
      errorMessage={error instanceof Error ? error.message : undefined}
      canEdit={canEdit}
      onDeleteEntry={canEdit ? (memoryId) => deleteCanvasMemoryEntry.mutate(memoryId) : undefined}
      deletingId={deleteCanvasMemoryEntry.isPending ? deleteCanvasMemoryEntry.variables : undefined}
      onCreateNamespace={
        canEdit
          ? async (input) => {
              await createCanvasMemoryNamespace.mutateAsync(input);
            }
          : undefined
      }
      isCreatingNamespace={createCanvasMemoryNamespace.isPending}
      onUpdateNamespace={
        canEdit
          ? async (input) => {
              await updateCanvasMemoryNamespace.mutateAsync(input);
            }
          : undefined
      }
      isUpdatingNamespace={updateCanvasMemoryNamespace.isPending}
    />
  );
}
