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
  } = args;
  const targetNodeIds = agentNodeIds?.length ? agentNodeIds : agentNodeId ? [agentNodeId] : [];
  if (!canvas || targetNodeIds.length === 0 || !appId) {
    throw new Error("Agent canvas is not loaded");
  }
  const liveVersionId = canvas.metadata?.liveVersionId;
  if (!liveVersionId) {
    throw new Error("Canvas has no live version");
  }

  const canvasYaml = serializeColumnAgentCanvasNodes(canvas, targetNodeIds, draft);

  try {
    const summary = await readStagingSummary();
    let discardedEarlierEdits = false;
    if (canvasDraftIsStale(summary)) {
      await discardStaging();
      discardedEarlierEdits = true;
    }

    const discardedDuringStage = await stageColumnAgentDiscardingStaleDraft({
      stageYaml,
      discardStaging,
      versionId: liveVersionId,
      canvasYaml,
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
  await args.stageYaml({ versionId: args.versionId, canvasYaml: args.canvasYaml });
  return true;
}
