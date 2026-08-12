import type { CanvasesCanvas } from "@/api-client";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";
import type { Dispatch, MutableRefObject, SetStateAction } from "react";

import { materializeCanvasSpec } from "./lib/workflow-spec-files";

type UpdateCanvasVersionMutation = {
  mutateAsync: (input: { versionId: string; canvasYaml: string }) => Promise<unknown>;
};

export type FactoryConfigureSaveDeps = {
  canStageCanvasVersion: boolean;
  activeCanvasVersionIdRef: MutableRefObject<string>;
  activeCanvasVersionId: string;
  liveCanvasVersionId?: string;
  editSessionActive: boolean;
  setEditSessionActive: Dispatch<SetStateAction<boolean>>;
  getCurrentWorkflowSnapshot: () => CanvasesCanvas | null | undefined;
  setSavePending: (pending: boolean) => void;
  updateCanvasVersionMutation: UpdateCanvasVersionMutation;
  draftCanvasSpecsRef: MutableRefObject<Map<string, CanvasesCanvas["spec"] | null>>;
  setDraftCanvasSpec: Dispatch<SetStateAction<CanvasesCanvas["spec"] | null>>;
  setLastSavedWorkflowSnapshot: (workflow: CanvasesCanvas | null) => void;
  handleCommitStaging: (commitMessage: string, options?: { versionId?: string }) => Promise<boolean | void>;
  onDone?: () => void;
};

export type FactoryConfigureDiscardDeps = {
  setSavePending: (pending: boolean) => void;
  hasStagingChanges: boolean;
  hasUncommittedCanvasDraftChanges: boolean;
  handleResetStaging: () => Promise<void> | Promise<unknown>;
  handleExitEditSession: () => void;
  onDone?: () => void;
};

export async function runFactoryConfigureSave(deps: FactoryConfigureSaveDeps): Promise<void> {
  if (!deps.canStageCanvasVersion) {
    showErrorToast("You don't have permission to edit this canvas.");
    return;
  }

  const savingVersionId =
    deps.activeCanvasVersionIdRef.current || deps.activeCanvasVersionId || deps.liveCanvasVersionId || "";
  if (!savingVersionId) {
    showErrorToast("Edit session is not ready. Try Configure again.");
    return;
  }

  // Align the sync ref before staging. The shared save queue rejects as
  // "stale" when ref !== savingVersionId — that was failing Configure Save.
  deps.activeCanvasVersionIdRef.current = savingVersionId;
  if (!deps.editSessionActive) {
    deps.setEditSessionActive(true);
  }

  const workflow = deps.getCurrentWorkflowSnapshot();
  if (!workflow?.spec) {
    showErrorToast("Nothing to save");
    return;
  }

  deps.setSavePending(true);
  try {
    // Stage canvas.yaml directly — skip enqueueCanvasSave stale/session checks.
    await deps.updateCanvasVersionMutation.mutateAsync({
      versionId: savingVersionId,
      canvasYaml: materializeCanvasSpec(workflow),
    });
    applyStagedWorkflowSnapshot(deps, savingVersionId, workflow);
    const committed = await deps.handleCommitStaging("Update automation", { versionId: savingVersionId });
    if (committed) {
      deps.onDone?.();
    }
  } catch (error) {
    showErrorToast(getApiErrorMessage(error, "Failed to stage canvas changes"));
  } finally {
    deps.setSavePending(false);
  }
}

function applyStagedWorkflowSnapshot(
  deps: FactoryConfigureSaveDeps,
  savingVersionId: string,
  workflow: NonNullable<ReturnType<FactoryConfigureSaveDeps["getCurrentWorkflowSnapshot"]>>,
) {
  if (!workflow.spec) {
    return;
  }
  deps.draftCanvasSpecsRef.current.set(savingVersionId, workflow.spec);
  deps.setDraftCanvasSpec(workflow.spec);
  deps.setLastSavedWorkflowSnapshot(workflow);
}

export async function runFactoryConfigureDiscard(deps: FactoryConfigureDiscardDeps): Promise<void> {
  deps.setSavePending(true);
  try {
    if (deps.hasStagingChanges || deps.hasUncommittedCanvasDraftChanges) {
      await deps.handleResetStaging();
    }
    deps.handleExitEditSession();
    deps.onDone?.();
  } finally {
    deps.setSavePending(false);
  }
}
