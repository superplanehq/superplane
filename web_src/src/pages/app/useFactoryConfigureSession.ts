import type { CanvasesCanvas, CanvasesCanvasVersion } from "@/api-client";
import { useEffect, useRef, useState, type Dispatch, type MutableRefObject, type SetStateAction } from "react";

import { runFactoryConfigureDiscard, runFactoryConfigureSave } from "./factoryConfigureActions";

export type FactoryConfigureActions = {
  save: () => void;
  discard: () => void;
  busy: boolean;
};

type UpdateCanvasVersionMutation = {
  mutateAsync: (input: { versionId: string; canvasYaml: string }) => Promise<unknown>;
};

type UseFactoryConfigureSessionOptions = {
  factoryConfigure: boolean;
  factoryConfigureActionsRef?: MutableRefObject<FactoryConfigureActions | null>;
  onFactoryConfigureBusyChange?: (busy: boolean) => void;
  onFactoryConfigureDone?: () => void;
  editSessionActive: boolean;
  setEditSessionActive: Dispatch<SetStateAction<boolean>>;
  canStageCanvasVersion: boolean;
  canvasLoading: boolean;
  liveCanvasVersionLoading: boolean;
  liveCanvasVersionId?: string;
  liveCanvasVersion?: CanvasesCanvasVersion | null;
  liveCanvas?: CanvasesCanvas | null;
  previewingCurrentVersionRef: MutableRefObject<boolean>;
  activateCanvasVersionForEditing: (
    versionId: string,
    version: CanvasesCanvasVersion,
    options?: { preserveStagedLayer?: boolean },
  ) => void;
  draftCanvasSpecsRef: MutableRefObject<Map<string, CanvasesCanvas["spec"] | null>>;
  setDraftCanvasSpec: Dispatch<SetStateAction<CanvasesCanvas["spec"] | null>>;
  resyncStagedEditorState: (
    versionId: string,
    options?: { bumpResetNonce?: boolean; preferCachedStagedSpec?: boolean },
  ) => Promise<void>;
  setLastSavedWorkflowSnapshot: (workflow: CanvasesCanvas | null) => void;
  commitStagingPending: boolean;
  resetStagingPending: boolean;
  activeCanvasVersionIdRef: MutableRefObject<string>;
  activeCanvasVersionId: string;
  getCurrentWorkflowSnapshot: () => CanvasesCanvas | null | undefined;
  updateCanvasVersionMutation: UpdateCanvasVersionMutation;
  handleCommitStaging: (commitMessage: string, options?: { versionId?: string }) => Promise<boolean | void>;
  handleResetStaging: () => Promise<void> | Promise<unknown>;
  handleExitEditSession: () => void;
  hasStagingChanges: boolean;
  hasUncommittedCanvasDraftChanges: boolean;
};

/**
 * Factory-shell Configure: seed/resync the edit session and expose Save/Discard
 * for the outer chrome without keeping that logic inside AppPage.
 */
export function useFactoryConfigureSession({
  factoryConfigure,
  factoryConfigureActionsRef,
  onFactoryConfigureBusyChange,
  onFactoryConfigureDone,
  editSessionActive,
  setEditSessionActive,
  canStageCanvasVersion,
  canvasLoading,
  liveCanvasVersionLoading,
  liveCanvasVersionId,
  liveCanvasVersion,
  liveCanvas,
  previewingCurrentVersionRef,
  activateCanvasVersionForEditing,
  draftCanvasSpecsRef,
  setDraftCanvasSpec,
  resyncStagedEditorState,
  setLastSavedWorkflowSnapshot,
  commitStagingPending,
  resetStagingPending,
  activeCanvasVersionIdRef,
  activeCanvasVersionId,
  getCurrentWorkflowSnapshot,
  updateCanvasVersionMutation,
  handleCommitStaging,
  handleResetStaging,
  handleExitEditSession,
  hasStagingChanges,
  hasUncommittedCanvasDraftChanges,
}: UseFactoryConfigureSessionOptions) {
  const onFactoryConfigureDoneRef = useRef(onFactoryConfigureDone);
  onFactoryConfigureDoneRef.current = onFactoryConfigureDone;

  const factoryConfigureEnterStartedRef = useRef(false);
  const [factoryConfigureSavePending, setFactoryConfigureSavePending] = useState(false);

  useEffect(() => {
    if (!factoryConfigure) {
      factoryConfigureEnterStartedRef.current = false;
      return;
    }
    if (factoryConfigureEnterStartedRef.current || editSessionActive) {
      return;
    }
    if (!canStageCanvasVersion || !liveCanvasVersionId || !liveCanvasVersion) {
      return;
    }
    if (canvasLoading || liveCanvasVersionLoading) {
      return;
    }

    factoryConfigureEnterStartedRef.current = true;
    // Seed draft from the live version directly — do not wait on the versions
    // list (handleUseVersion), which often left Configure with no activeVersionId.
    const immediateSpec = liveCanvas?.spec ?? liveCanvasVersion.spec ?? { nodes: [], edges: [] };
    const versionForEdit: CanvasesCanvasVersion = {
      ...liveCanvasVersion,
      metadata: {
        ...liveCanvasVersion.metadata,
        id: liveCanvasVersionId,
      },
      spec: immediateSpec,
    };
    const configureVersionId = liveCanvasVersionId;
    let cancelled = false;
    let editEnabled = false;

    previewingCurrentVersionRef.current = true;
    activateCanvasVersionForEditing(configureVersionId, versionForEdit, { preserveStagedLayer: true });
    draftCanvasSpecsRef.current.set(configureVersionId, immediateSpec);
    setDraftCanvasSpec(immediateSpec);

    // Await staged resync before enabling edit so a late applyStagedSpec cannot
    // wipe edits typed against the immediate seed.
    void (async () => {
      try {
        await resyncStagedEditorState(configureVersionId, { bumpResetNonce: false });
      } catch {
        // Keep the immediate live/committed spec so Configure stays usable.
      }
      if (cancelled) {
        return;
      }
      editEnabled = true;
      setEditSessionActive(true);
      if (liveCanvas) {
        const spec = draftCanvasSpecsRef.current.get(configureVersionId) ?? immediateSpec;
        setLastSavedWorkflowSnapshot({ ...liveCanvas, spec });
      }
    })();

    return () => {
      cancelled = true;
      if (!editEnabled) {
        factoryConfigureEnterStartedRef.current = false;
      }
    };
  }, [
    activateCanvasVersionForEditing,
    canStageCanvasVersion,
    canvasLoading,
    draftCanvasSpecsRef,
    editSessionActive,
    factoryConfigure,
    liveCanvas,
    liveCanvasVersion,
    liveCanvasVersionId,
    liveCanvasVersionLoading,
    previewingCurrentVersionRef,
    resyncStagedEditorState,
    setDraftCanvasSpec,
    setEditSessionActive,
    setLastSavedWorkflowSnapshot,
  ]);

  const factoryConfigureBusy = commitStagingPending || resetStagingPending || factoryConfigureSavePending;
  useEffect(() => {
    if (!factoryConfigure || !onFactoryConfigureBusyChange) {
      return;
    }
    onFactoryConfigureBusyChange(factoryConfigureBusy);
  }, [factoryConfigure, factoryConfigureBusy, onFactoryConfigureBusyChange]);

  if (factoryConfigureActionsRef) {
    factoryConfigureActionsRef.current = !factoryConfigure
      ? null
      : {
          busy: factoryConfigureBusy,
          save: () => {
            if (factoryConfigureBusy) {
              return;
            }
            void runFactoryConfigureSave({
              canStageCanvasVersion,
              activeCanvasVersionIdRef,
              activeCanvasVersionId,
              liveCanvasVersionId,
              editSessionActive,
              setEditSessionActive,
              getCurrentWorkflowSnapshot,
              setSavePending: setFactoryConfigureSavePending,
              updateCanvasVersionMutation,
              draftCanvasSpecsRef,
              setDraftCanvasSpec,
              setLastSavedWorkflowSnapshot,
              handleCommitStaging,
              onDone: () => onFactoryConfigureDoneRef.current?.(),
            });
          },
          discard: () => {
            if (factoryConfigureBusy) {
              return;
            }
            void runFactoryConfigureDiscard({
              setSavePending: setFactoryConfigureSavePending,
              hasStagingChanges,
              hasUncommittedCanvasDraftChanges,
              handleResetStaging,
              handleExitEditSession,
              onDone: () => onFactoryConfigureDoneRef.current?.(),
            });
          },
        };
  }
}
