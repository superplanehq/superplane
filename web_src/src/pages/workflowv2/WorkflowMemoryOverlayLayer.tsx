import type {
  CanvasMemoryEntry,
  CreateCanvasMemoryBankInput,
  UpdateCanvasMemoryBankInput,
} from "@/hooks/useCanvasData";

import { CanvasMemoryView } from "./CanvasMemoryView";

interface DeleteCanvasMemoryMutation {
  mutate: (memoryId: string) => void;
  isPending: boolean;
  variables: string | undefined;
}

interface CreateCanvasMemoryBankMutation {
  mutateAsync: (input: CreateCanvasMemoryBankInput) => Promise<unknown>;
  isPending: boolean;
}

interface UpdateCanvasMemoryBankMutation {
  mutateAsync: (input: UpdateCanvasMemoryBankInput) => Promise<unknown>;
  isPending: boolean;
}

interface WorkflowMemoryOverlayLayerProps {
  isMemoryMode: boolean;
  // True when the viewer is allowed to mutate memory entries/banks. Computed
  // by the workflow page via `canEditCanvasMemory` so the gating logic stays
  // in one place.
  canEdit: boolean;
  entries: CanvasMemoryEntry[];
  isLoading: boolean;
  error: unknown;
  deleteCanvasMemoryEntry: DeleteCanvasMemoryMutation;
  createCanvasMemoryBank: CreateCanvasMemoryBankMutation;
  updateCanvasMemoryBank: UpdateCanvasMemoryBankMutation;
}

export function WorkflowMemoryOverlayLayer({
  isMemoryMode,
  canEdit,
  entries,
  isLoading,
  error,
  deleteCanvasMemoryEntry,
  createCanvasMemoryBank,
  updateCanvasMemoryBank,
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
      onCreateBank={
        canEdit
          ? async (input) => {
              await createCanvasMemoryBank.mutateAsync(input);
            }
          : undefined
      }
      isCreatingBank={createCanvasMemoryBank.isPending}
      onUpdateBank={
        canEdit
          ? async (input) => {
              await updateCanvasMemoryBank.mutateAsync(input);
            }
          : undefined
      }
      isUpdatingBank={updateCanvasMemoryBank.isPending}
    />
  );
}
