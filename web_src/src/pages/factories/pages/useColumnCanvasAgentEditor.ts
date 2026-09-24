import type { CanvasesCanvas } from "@/api-client";
import {
  canvasKeys,
  useCanvas,
  useCanvasStaging,
  useCommitCanvasStaging,
  useDiscardCanvasStaging,
  useUpdateCanvasVersion,
} from "@/hooks/useCanvasData";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";

import {
  findAgentNodes,
  planningReviewDraftFromCanvas,
  primaryAgentNode,
  serializeColumnAgentCanvasNodes,
} from "../lib/columnCanvasAgent";
import { resolveFactoryAppTemplate } from "../lib/factoryAppTemplate";
import type { PlanningReviewDraft } from "./planningReviewMockup";

const UPDATE_AGENT_COMMIT_MESSAGE = "Update agent";
const AGENT_SAVED_NOTICE = "Agent saved.";
const AGENT_SAVED_AFTER_DISCARD_NOTICE =
  "Agent saved. Earlier canvas edits were discarded because the live canvas changed.";
const STALE_STAGING_UPDATE_MESSAGE = "stale staging cannot be updated";
const REFINEMENT_AGENT_NODE_ID = "refine-task";

type CanvasDraftSummary = {
  hasStaging?: boolean;
  stale?: boolean;
};

type StageCanvasYaml = (input: { versionId: string; canvasYaml: string }) => Promise<unknown>;

interface ColumnCanvasAgentEditorOptions {
  showVisualEvidenceSetting?: boolean;
  synchronizedAgentNodeIds?: readonly string[];
  preferredAgentNodeId?: string;
}

function editableAgentNodeIds(
  canvas: CanvasesCanvas | undefined,
  primaryNodeId: string | undefined,
  synchronizedNodeIds: readonly string[] | undefined,
): string[] {
  if (!synchronizedNodeIds) {
    return primaryNodeId ? [primaryNodeId] : [];
  }
  const synchronizedNodeIdSet = new Set(synchronizedNodeIds);
  return findAgentNodes(canvas?.spec)
    .map((node) => node.id)
    .filter((id): id is string => Boolean(id && synchronizedNodeIdSet.has(id)));
}

export function useColumnCanvasAgentEditor(
  organizationId: string,
  appId: string | undefined,
  options: ColumnCanvasAgentEditorOptions = {},
) {
  const enabled = Boolean(appId);
  const canvasId = appId ?? "";
  const canvasQuery = useCanvas(organizationId, canvasId, { enabled });
  const canvasStagingQuery = useCanvasStaging(appId, enabled);
  const updateVersion = useUpdateCanvasVersion(canvasId);
  const commitStaging = useCommitCanvasStaging(canvasId);
  const discardCanvasStaging = useDiscardCanvasStaging(canvasId);
  const queryClient = useQueryClient();
  const [editorOpen, setEditorOpen] = useState(false);

  const canvas = canvasQuery.data;
  const preferredAgentNodeId = options.preferredAgentNodeId ?? REFINEMENT_AGENT_NODE_ID;
  const agentNode = primaryAgentNode(canvas?.spec, preferredAgentNodeId);
  const agentNodeIds = editableAgentNodeIds(canvas, agentNode?.id, options.synchronizedAgentNodeIds);
  const draft = canvas && agentNode?.id ? planningReviewDraftFromCanvas(canvas, agentNode.id) : null;
  const showVisualEvidenceSetting =
    options.showVisualEvidenceSetting ?? resolveFactoryAppTemplate(canvas)?.id === "line-implementation";

  const save = async (nextDraft: PlanningReviewDraft) => {
    await persistColumnAgent({
      appId: canvasId,
      canvas,
      agentNodeIds,
      draft: nextDraft,
      stageYaml: (input) => updateVersion.mutateAsync(input),
      commit: (message) => commitStaging.mutateAsync(message),
      invalidate: () => queryClient.invalidateQueries({ queryKey: canvasKeys.detail(organizationId, canvasId) }),
      readStagingSummary: async () => {
        const result = await canvasStagingQuery.refetch();
        if (result.error) {
          throw result.error;
        }
        return result.data;
      },
      discardStaging: () => discardCanvasStaging.mutateAsync(undefined),
      refreshCanvas: async () => {
        const result = await canvasQuery.refetch();
        if (result.error) {
          throw result.error;
        }
        return result.data;
      },
    });
  };

  return {
    agentNode,
    agentNodeIds,
    isLoading: enabled && canvasQuery.isPending,
    draft,
    showVisualEvidenceSetting,
    editorOpen,
    openEditor: agentNode ? () => setEditorOpen(true) : undefined,
    closeEditor: () => setEditorOpen(false),
    save,
  };
}

export async function persistColumnAgent(args: {
  appId: string;
  canvas: CanvasesCanvas | undefined;
  agentNodeId?: string;
  agentNodeIds?: string[];
  draft: PlanningReviewDraft;
  stageYaml: StageCanvasYaml;
  commit: (message: string) => Promise<unknown>;
  invalidate: () => Promise<unknown> | unknown;
  readStagingSummary: () => Promise<CanvasDraftSummary | undefined>;
  discardStaging: () => Promise<unknown>;
  refreshCanvas: () => Promise<CanvasesCanvas | undefined>;
}) {
  const {
    canvas,
    agentNodeId,
    agentNodeIds,
    appId,
    draft,
    stageYaml,
    commit,
    invalidate,
    readStagingSummary,
    discardStaging,
    refreshCanvas,
  } = args;
  const targetNodeIds = agentNodeIds?.length ? agentNodeIds : agentNodeId ? [agentNodeId] : [];
  if (!canvas || targetNodeIds.length === 0 || !appId) {
    throw new Error("Agent canvas is not loaded");
  }

  const editFromLiveCanvas = () =>
    agentEditFromLiveCanvas({
      refreshCanvas,
      agentNodeIds: targetNodeIds,
      draft,
    });

  try {
    const summary = await readStagingSummary();
    let discardedEarlierEdits = false;
    let stagedEdit = agentEditFromCanvas(canvas, targetNodeIds, draft);
    if (canvasDraftIsStale(summary)) {
      await discardStaging();
      discardedEarlierEdits = true;
      stagedEdit = await editFromLiveCanvas();
    }

    const discardedDuringStage = await stageColumnAgentDiscardingStaleDraft({
      stageYaml,
      discardStaging,
      versionId: stagedEdit.versionId,
      canvasYaml: stagedEdit.canvasYaml,
      rebuildFromLiveCanvas: editFromLiveCanvas,
    });
    await commit(UPDATE_AGENT_COMMIT_MESSAGE);
    await invalidate();
    showSuccessToast(
      discardedEarlierEdits || discardedDuringStage ? AGENT_SAVED_AFTER_DISCARD_NOTICE : AGENT_SAVED_NOTICE,
    );
  } catch (error) {
    showErrorToast(getApiErrorMessage(error, "Failed to save agent"));
    throw error;
  }
}

function agentEditFromCanvas(
  canvas: CanvasesCanvas,
  agentNodeIds: string[],
  draft: PlanningReviewDraft,
): { versionId: string; canvasYaml: string } {
  const versionId = canvas.metadata?.liveVersionId;
  if (!versionId) {
    throw new Error("Canvas has no live version");
  }
  return {
    versionId,
    canvasYaml: serializeColumnAgentCanvasNodes(canvas, agentNodeIds, draft),
  };
}

async function agentEditFromLiveCanvas(args: {
  refreshCanvas: () => Promise<CanvasesCanvas | undefined>;
  agentNodeIds: string[];
  draft: PlanningReviewDraft;
}): Promise<{ versionId: string; canvasYaml: string }> {
  const canvas = await args.refreshCanvas();
  if (!canvas) {
    throw new Error("Agent canvas is not loaded");
  }
  return agentEditFromCanvas(canvas, args.agentNodeIds, args.draft);
}

function canvasDraftIsStale(summary: CanvasDraftSummary | undefined): boolean {
  return Boolean(summary?.hasStaging && summary.stale);
}

function isStaleStagingUpdateError(error: unknown): boolean {
  return getApiErrorMessage(error, "").includes(STALE_STAGING_UPDATE_MESSAGE);
}

async function stageColumnAgentDiscardingStaleDraft(args: {
  stageYaml: StageCanvasYaml;
  discardStaging: () => Promise<unknown>;
  versionId: string;
  canvasYaml: string;
  rebuildFromLiveCanvas: () => Promise<{ versionId: string; canvasYaml: string }>;
}): Promise<boolean> {
  try {
    await args.stageYaml({ versionId: args.versionId, canvasYaml: args.canvasYaml });
    return false;
  } catch (error) {
    if (!isStaleStagingUpdateError(error)) {
      throw error;
    }
  }

  await args.discardStaging();
  const refreshedEdit = await args.rebuildFromLiveCanvas();
  await args.stageYaml({ versionId: refreshedEdit.versionId, canvasYaml: refreshedEdit.canvasYaml });
  return true;
}
