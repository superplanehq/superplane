import type { CanvasesCanvas } from "@/api-client";
import {
  canvasKeys,
  useCanvas,
  useCanvasStaging,
  useCommitCanvasStaging,
  useUpdateCanvasVersion,
} from "@/hooks/useCanvasData";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { fetchRepositorySpecFileContent } from "@/pages/app/lib/repository-spec-files";
import { dematerializeCanvasSpec } from "@/pages/app/lib/workflow-spec-files";
import { CANVAS_YAML_PATH } from "@/pages/app/lib/workflow-spec-paths";
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
const CURRENT_STAGING_KEPT_MESSAGE = "current staging cannot be discarded";
const STAGED_CANVAS_CHANGED_MESSAGE = "staged canvas changed";
const AGENT_MISSING_MESSAGE = "The agent is not on this canvas.";
const STAGED_CANVAS_CHANGED_NOTICE = "The canvas changed. Save the agent again.";
const REFINEMENT_AGENT_NODE_ID = "refine-task";

type CanvasDraftSummary = {
  hasStaging?: boolean;
  stale?: boolean;
};

type StageCanvasYaml = (input: {
  versionId: string;
  canvasYaml: string;
  replaceIfStale?: boolean;
  expectedCanvasYaml?: string;
}) => Promise<unknown>;

type StagedAgentCanvas = {
  canvas: CanvasesCanvas;
  canvasYaml: string;
};

type AgentCanvasEdit = {
  versionId: string;
  canvasYaml: string;
  expectedCanvasYaml?: string;
};

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
      refreshCanvas: async () => {
        const result = await canvasQuery.refetch();
        if (result.error) {
          throw result.error;
        }
        return result.data;
      },
      readStagedCanvas: () => readStagedAgentCanvas(canvasId, () => canvasQuery.refetch()),
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
  refreshCanvas: () => Promise<CanvasesCanvas | undefined>;
  readStagedCanvas: () => Promise<StagedAgentCanvas | undefined>;
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
    refreshCanvas,
    readStagedCanvas,
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
  const editFromStagedCanvas = () =>
    agentEditFromStagedCanvas({
      readStagedCanvas,
      agentNodeIds: targetNodeIds,
      draft,
    });

  try {
    const summary = await readStagingSummary();
    let discardedEarlierEdits = false;
    let stagedEdit = agentEditFromCanvas(canvas, targetNodeIds, draft);
    if (canvasDraftIsStale(summary)) {
      stagedEdit = await editFromLiveCanvas();
      discardedEarlierEdits = await stageReplacingStaleDraft({
        stageYaml,
        versionId: stagedEdit.versionId,
        canvasYaml: stagedEdit.canvasYaml,
        rebuildFromStagedCanvas: editFromStagedCanvas,
      });
    } else {
      discardedEarlierEdits = await stageColumnAgentDiscardingStaleDraft({
        stageYaml,
        versionId: stagedEdit.versionId,
        canvasYaml: stagedEdit.canvasYaml,
        rebuildFromLiveCanvas: editFromLiveCanvas,
        rebuildFromStagedCanvas: editFromStagedCanvas,
      });
    }
    await commit(UPDATE_AGENT_COMMIT_MESSAGE);
    await invalidate();
    showSuccessToast(discardedEarlierEdits ? AGENT_SAVED_AFTER_DISCARD_NOTICE : AGENT_SAVED_NOTICE);
  } catch (error) {
    showErrorToast(getApiErrorMessage(error, "Failed to save agent"));
    throw error;
  }
}

function agentEditFromCanvas(
  canvas: CanvasesCanvas,
  agentNodeIds: string[],
  draft: PlanningReviewDraft,
): AgentCanvasEdit {
  const versionId = canvas.metadata?.liveVersionId;
  if (!versionId) {
    throw new Error("Canvas has no live version");
  }
  if (!canvasHasAgentNode(canvas, agentNodeIds)) {
    throw new Error(AGENT_MISSING_MESSAGE);
  }
  return {
    versionId,
    canvasYaml: serializeColumnAgentCanvasNodes(canvas, agentNodeIds, draft),
  };
}

function canvasHasAgentNode(canvas: CanvasesCanvas, agentNodeIds: string[]): boolean {
  const nodeIds = new Set((canvas.spec?.nodes ?? []).map((node) => node.id));
  return agentNodeIds.some((id) => nodeIds.has(id));
}

async function agentEditFromLiveCanvas(args: {
  refreshCanvas: () => Promise<CanvasesCanvas | undefined>;
  agentNodeIds: string[];
  draft: PlanningReviewDraft;
}): Promise<AgentCanvasEdit> {
  const canvas = await args.refreshCanvas();
  if (!canvas) {
    throw new Error("Agent canvas is not loaded");
  }
  return agentEditFromCanvas(canvas, args.agentNodeIds, args.draft);
}

async function agentEditFromStagedCanvas(args: {
  readStagedCanvas: () => Promise<StagedAgentCanvas | undefined>;
  agentNodeIds: string[];
  draft: PlanningReviewDraft;
}): Promise<AgentCanvasEdit> {
  const staged = await args.readStagedCanvas();
  if (!staged?.canvas || !staged.canvasYaml) {
    throw new Error("Agent canvas is not loaded");
  }
  return {
    ...agentEditFromCanvas(staged.canvas, args.agentNodeIds, args.draft),
    expectedCanvasYaml: staged.canvasYaml,
  };
}

function canvasDraftIsStale(summary: CanvasDraftSummary | undefined): boolean {
  return Boolean(summary?.hasStaging && summary.stale);
}

function isStaleStagingUpdateError(error: unknown): boolean {
  return getApiErrorMessage(error, "").includes(STALE_STAGING_UPDATE_MESSAGE);
}

function isCurrentStagingKeptError(error: unknown): boolean {
  return getApiErrorMessage(error, "").includes(CURRENT_STAGING_KEPT_MESSAGE);
}

function isStagedCanvasChangedError(error: unknown): boolean {
  return getApiErrorMessage(error, "").includes(STAGED_CANVAS_CHANGED_MESSAGE);
}

async function stageColumnAgentDiscardingStaleDraft(args: {
  stageYaml: StageCanvasYaml;
  versionId: string;
  canvasYaml: string;
  rebuildFromLiveCanvas: () => Promise<AgentCanvasEdit>;
  rebuildFromStagedCanvas: () => Promise<AgentCanvasEdit>;
}): Promise<boolean> {
  try {
    await args.stageYaml({ versionId: args.versionId, canvasYaml: args.canvasYaml });
    return false;
  } catch (error) {
    if (!isStaleStagingUpdateError(error)) {
      throw error;
    }
  }

  const refreshedEdit = await args.rebuildFromLiveCanvas();
  return stageReplacingStaleDraft({
    stageYaml: args.stageYaml,
    versionId: refreshedEdit.versionId,
    canvasYaml: refreshedEdit.canvasYaml,
    rebuildFromStagedCanvas: args.rebuildFromStagedCanvas,
  });
}

async function stageReplacingStaleDraft(args: {
  stageYaml: StageCanvasYaml;
  versionId: string;
  canvasYaml: string;
  rebuildFromStagedCanvas: () => Promise<AgentCanvasEdit>;
}): Promise<boolean> {
  try {
    await args.stageYaml({
      versionId: args.versionId,
      canvasYaml: args.canvasYaml,
      replaceIfStale: true,
    });
    return true;
  } catch (error) {
    if (!isCurrentStagingKeptError(error)) {
      throw error;
    }
  }

  await stageAgentEditOnCurrentDraft(args.stageYaml, args.rebuildFromStagedCanvas);
  return false;
}

async function stageAgentEditOnCurrentDraft(
  stageYaml: StageCanvasYaml,
  rebuildFromStagedCanvas: () => Promise<AgentCanvasEdit>,
): Promise<void> {
  const stagedEdit = await rebuildFromStagedCanvas();
  try {
    await stageCurrentDraft(stageYaml, stagedEdit);
    return;
  } catch (error) {
    if (!isStagedCanvasChangedError(error)) {
      throw error;
    }
  }

  try {
    await stageCurrentDraft(stageYaml, await rebuildFromStagedCanvas());
  } catch (error) {
    if (isStagedCanvasChangedError(error)) {
      throw new Error(STAGED_CANVAS_CHANGED_NOTICE, { cause: error });
    }
    throw error;
  }
}

function stageCurrentDraft(stageYaml: StageCanvasYaml, edit: AgentCanvasEdit): Promise<unknown> {
  return stageYaml({
    versionId: edit.versionId,
    canvasYaml: edit.canvasYaml,
    expectedCanvasYaml: edit.expectedCanvasYaml,
  });
}

async function readStagedAgentCanvas(
  canvasId: string,
  refetchCanvas: () => Promise<{ data?: CanvasesCanvas; error?: unknown }>,
): Promise<StagedAgentCanvas> {
  const result = await refetchCanvas();
  if (result.error) {
    throw result.error;
  }
  const live = result.data;
  const liveVersionId = live?.metadata?.liveVersionId;
  if (!live || !liveVersionId) {
    throw new Error("Agent canvas is not loaded");
  }
  const canvasYaml = await fetchRepositorySpecFileContent(canvasId, CANVAS_YAML_PATH, undefined, true);
  const spec = dematerializeCanvasSpec(canvasYaml);
  if (!spec || !canvasYaml) {
    throw new Error("Agent canvas is not loaded");
  }
  return {
    canvas: {
      ...live,
      spec,
    },
    canvasYaml,
  };
}
