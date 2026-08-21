import type { CanvasesCanvas } from "@/api-client";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";
import type { Dispatch, MutableRefObject, SetStateAction } from "react";

import { materializeCanvasSpec } from "./lib/workflow-spec-files";

type StagingSummary = {
  hasStaging?: boolean;
  stagedPaths?: string[];
};

type UpdateCanvasVersionResult = {
  data?: {
    stagingSummary?: StagingSummary;
  };
};

type UpdateCanvasVersionMutation = {
  mutateAsync: (input: { versionId: string; canvasYaml: string }) => Promise<UpdateCanvasVersionResult | unknown>;
};

export type FactoryConfigureSaveOptions = {
  /** Overlay metadata.name before staging canvas.yaml (keeps YAML in sync after rename). */
  canvasName?: string;
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
  /** After a real commit tears down the edit session, allow Configure to seed again. */
  onAfterCommit?: () => void;
  canvasName?: string;
};

export type FactoryConfigureDiscardDeps = {
  setSavePending: (pending: boolean) => void;
  hasStagingChanges: boolean;
  hasUncommittedCanvasDraftChanges: boolean;
  handleResetStaging: () => Promise<void> | Promise<unknown>;
  handleExitEditSession: () => void;
  onDone?: () => void;
};

/** Merge a renamed canvas name into the workflow snapshot used for canvas.yaml. */
export function withCanvasMetadataName(workflow: CanvasesCanvas, canvasName?: string): CanvasesCanvas {
  const name = canvasName?.trim();
  if (!name || name === workflow.metadata?.name) {
    return workflow;
  }
  return {
    ...workflow,
    metadata: {
      ...workflow.metadata,
      name,
    },
  };
}

export function stagingSummaryHasChanges(summary: StagingSummary | undefined): boolean {
  if (!summary) {
    return false;
  }
  if (summary.hasStaging) {
    return true;
  }
  return (summary.stagedPaths?.length ?? 0) > 0;
}

function readStagingSummary(result: UpdateCanvasVersionResult | unknown): StagingSummary | undefined {
  if (!result || typeof result !== "object") {
    return undefined;
  }
  const data = (result as UpdateCanvasVersionResult).data;
  return data?.stagingSummary;
}

async function stageAndCommitFactoryConfigure(
  deps: FactoryConfigureSaveDeps,
  savingVersionId: string,
  workflow: CanvasesCanvas,
) {
  const stageResult = await deps.updateCanvasVersionMutation.mutateAsync({
    versionId: savingVersionId,
    canvasYaml: materializeCanvasSpec(workflow),
  });
  applyStagedWorkflowSnapshot(deps, savingVersionId, workflow);

  // Matching YAML discards the staged path. Skip commit so title-only / no-op
  // Saves do not fail with "no staged changes".
  if (!stagingSummaryHasChanges(readStagingSummary(stageResult))) {
    deps.onDone?.();
    return;
  }

  const committed = await deps.handleCommitStaging("Update automation", { versionId: savingVersionId });
  if (committed) {
    deps.onAfterCommit?.();
    deps.onDone?.();
  }
}

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

  const snapshot = deps.getCurrentWorkflowSnapshot();
  if (!snapshot?.spec) {
    showErrorToast("Nothing to save");
    return;
  }
  const workflow = withCanvasMetadataName(snapshot, deps.canvasName);

  deps.setSavePending(true);
  try {
    await stageAndCommitFactoryConfigure(deps, savingVersionId, workflow);
  } catch (error) {
    showErrorToast(getApiErrorMessage(error, "Failed to stage canvas changes"));
  } finally {
    deps.setSavePending(false);
  }
}

function applyStagedWorkflowSnapshot(
  deps: FactoryConfigureSaveDeps,
  savingVersionId: string,
  workflow: CanvasesCanvas,
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
