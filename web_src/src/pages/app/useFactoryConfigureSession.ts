import type { CanvasesCanvas, CanvasesCanvasVersion } from "@/api-client";
import { useEffect, useRef, useState, type Dispatch, type MutableRefObject, type SetStateAction } from "react";

import { runFactoryConfigureDiscard, runFactoryConfigureSave } from "./factoryConfigureActions";
import { useFactoryConfigureEnter } from "./useFactoryConfigureEnter";

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
export function useFactoryConfigureSession(options: UseFactoryConfigureSessionOptions) {
  const {
    factoryConfigure,
    factoryConfigureActionsRef,
    onFactoryConfigureBusyChange,
    onFactoryConfigureDone,
    editSessionActive,
    setEditSessionActive,
    canStageCanvasVersion,
    commitStagingPending,
    resetStagingPending,
    activeCanvasVersionIdRef,
    activeCanvasVersionId,
    liveCanvasVersionId,
    getCurrentWorkflowSnapshot,
    updateCanvasVersionMutation,
    draftCanvasSpecsRef,
    setDraftCanvasSpec,
    setLastSavedWorkflowSnapshot,
    handleCommitStaging,
    handleResetStaging,
    handleExitEditSession,
    hasStagingChanges,
    hasUncommittedCanvasDraftChanges,
  } = options;

  const onFactoryConfigureDoneRef = useRef(onFactoryConfigureDone);
  onFactoryConfigureDoneRef.current = onFactoryConfigureDone;
  const [factoryConfigureSavePending, setFactoryConfigureSavePending] = useState(false);

  useFactoryConfigureEnter(options);

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
