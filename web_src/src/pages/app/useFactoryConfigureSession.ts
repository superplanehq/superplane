import type { ActionsAction, CanvasesCanvas, CanvasesCanvasVersion } from "@/api-client";
import { useEffect, useRef, useState, type Dispatch, type MutableRefObject, type SetStateAction } from "react";

import type { ResyncStagedOptions } from "@/hooks/useCanvasStagingResync";

import {
  runFactoryConfigureDiscard,
  runFactoryConfigureSave,
  type FactoryConfigureSaveOptions,
} from "./factoryConfigureActions";
import { useFactoryConfigureEnter } from "./useFactoryConfigureEnter";
import { getWorkflowSpecSignature } from "./lib/draft-canvas-sync";
import { applyFactoryCanvasLayout } from "./useTopologyMutationCommit";

export type FactoryConfigureActions = {
  save: (options?: FactoryConfigureSaveOptions) => Promise<void>;
  discard: () => void;
  busy: boolean;
  /** True when Configure has graph/console/files changes to stage+commit. */
  hasUncommittedChanges: boolean;
  /**
   * Loads `spec` into the current edit session as an unsaved draft (merged
   * onto the live canvas snapshot). Leaves the session dirty — the caller
   * still needs Save to persist it.
   */
  applyDraftSpec: (spec: NonNullable<CanvasesCanvas["spec"]>) => Promise<void>;
};

type UpdateCanvasVersionMutation = {
  mutateAsync: (input: { versionId: string; canvasYaml: string }) => Promise<unknown>;
};

type UseFactoryConfigureSessionOptions = {
  factoryConfigure: boolean;
  factoryConfigureActionsRef?: MutableRefObject<FactoryConfigureActions | null>;
  onFactoryConfigureBusyChange?: (busy: boolean) => void;
  onFactoryConfigureDone?: () => void;
  /** Called after Configure Save. Discard still uses onFactoryConfigureDone. */
  onFactoryConfigureSaved?: () => void;
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
  resyncStagedEditorState: (versionId: string, options?: ResyncStagedOptions) => Promise<void>;
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
  applyLocalWorkflowUpdate: (updatedWorkflow: CanvasesCanvas) => void;
  factoryAutoLayout?: boolean;
  components?: ActionsAction[];
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
    onFactoryConfigureSaved,
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
    applyLocalWorkflowUpdate,
    factoryAutoLayout,
    components,
  } = options;

  const onFactoryConfigureDoneRef = useRef(onFactoryConfigureDone);
  onFactoryConfigureDoneRef.current = onFactoryConfigureDone;
  const onFactoryConfigureSavedRef = useRef(onFactoryConfigureSaved);
  onFactoryConfigureSavedRef.current = onFactoryConfigureSaved;
  const editSessionActiveRef = useRef(editSessionActive);
  editSessionActiveRef.current = editSessionActive;
  const factoryAutoLayoutRef = useRef(factoryAutoLayout);
  factoryAutoLayoutRef.current = factoryAutoLayout;
  const componentsRef = useRef(components);
  componentsRef.current = components;
  const [factoryConfigureSavePending, setFactoryConfigureSavePending] = useState(false);

  const { allowNextConfigureEnter, configureVisitIdRef } = useFactoryConfigureEnter(options);

  const factoryConfigureBusy = commitStagingPending || resetStagingPending || factoryConfigureSavePending;
  const hasUncommittedChanges = hasStagingChanges || hasUncommittedCanvasDraftChanges;
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
          hasUncommittedChanges,
          save: async (saveOptions) => {
            if (factoryConfigureBusy) {
              return;
            }
            await runFactoryConfigureSave({
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
              canvasName: saveOptions?.canvasName,
              onAfterCommit: allowNextConfigureEnter,
              onDone: () => (onFactoryConfigureSavedRef.current ?? onFactoryConfigureDoneRef.current)?.(),
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
          applyDraftSpec: async (spec) => {
            const current = getCurrentWorkflowSnapshot();
            if (!current) {
              return;
            }
            const requestedVisitId = configureVisitIdRef.current;
            const requestedVersionId = activeCanvasVersionIdRef.current;
            const requestedEditSessionActive = editSessionActiveRef.current;
            const sessionMatches = () =>
              configureVisitIdRef.current === requestedVisitId &&
              activeCanvasVersionIdRef.current === requestedVersionId &&
              editSessionActiveRef.current === requestedEditSessionActive;
            const merged = { ...current, spec };
            applyLocalWorkflowUpdate(merged);
            if (!factoryAutoLayoutRef.current) {
              return;
            }
            const nextWorkflow = await applyFactoryCanvasLayout(merged, componentsRef.current || []);
            if (!sessionMatches()) {
              return;
            }
            const latestDraftSpec = draftCanvasSpecsRef.current.get(activeCanvasVersionIdRef.current);
            if (
              latestDraftSpec != null &&
              getWorkflowSpecSignature(latestDraftSpec) !== getWorkflowSpecSignature(merged.spec)
            ) {
              return;
            }
            applyLocalWorkflowUpdate(nextWorkflow);
          },
        };
  }

  return { allowNextConfigureEnter };
}
