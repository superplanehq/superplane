import type { CanvasesCanvas } from "@/api-client";
import { canvasKeys, useCanvas, useCommitCanvasStaging, useUpdateCanvasVersion } from "@/hooks/useCanvasData";
import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { getApiErrorMessage } from "@/lib/errors";
import { FEATURE_FACTORY_CREATE_WITH_AGENT } from "@/lib/experimentalFeatures";
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
const REFINEMENT_AGENT_NODE_ID = "refine-task";

interface ColumnCanvasAgentEditorOptions {
  showVisualEvidenceSetting?: boolean;
  synchronizeAgentNodes?: boolean;
}

function editableAgentNodeIds(
  canvas: CanvasesCanvas | undefined,
  primaryNodeId: string | undefined,
  synchronize: boolean,
): string[] {
  if (!synchronize) {
    return primaryNodeId ? [primaryNodeId] : [];
  }
  return findAgentNodes(canvas?.spec)
    .map((node) => node.id)
    .filter((id): id is string => Boolean(id));
}

export function useColumnCanvasAgentEditor(
  organizationId: string,
  appId: string | undefined,
  options: ColumnCanvasAgentEditorOptions = {},
) {
  const enabled = Boolean(appId);
  const canvasId = appId ?? "";
  const canvasQuery = useCanvas(organizationId, canvasId, { enabled });
  const updateVersion = useUpdateCanvasVersion(canvasId);
  const commitStaging = useCommitCanvasStaging(canvasId);
  const queryClient = useQueryClient();
  const [editorOpen, setEditorOpen] = useState(false);
  const features = useExperimentalFeature(organizationId);

  const canvas = canvasQuery.data;
  const preferredAgentNodeId = features.has(FEATURE_FACTORY_CREATE_WITH_AGENT) ? REFINEMENT_AGENT_NODE_ID : undefined;
  const agentNode = primaryAgentNode(canvas?.spec, preferredAgentNodeId);
  const agentNodeIds = editableAgentNodeIds(canvas, agentNode?.id, options.synchronizeAgentNodes === true);
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
    });
  };

  return {
    agentNode,
    agentNodeIds,
    isLoading: enabled && (canvasQuery.isPending || features.isLoading),
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
  stageYaml: (input: { versionId: string; canvasYaml: string }) => Promise<unknown>;
  commit: (message: string) => Promise<unknown>;
  invalidate: () => Promise<unknown> | unknown;
}) {
  const { canvas, agentNodeId, agentNodeIds, appId, draft, stageYaml, commit, invalidate } = args;
  const targetNodeIds = agentNodeIds?.length ? agentNodeIds : agentNodeId ? [agentNodeId] : [];
  if (!canvas || targetNodeIds.length === 0 || !appId) {
    throw new Error("Agent canvas is not loaded");
  }
  const liveVersionId = canvas.metadata?.liveVersionId;
  if (!liveVersionId) {
    throw new Error("Canvas has no live version");
  }

  try {
    await stageYaml({
      versionId: liveVersionId,
      canvasYaml: serializeColumnAgentCanvasNodes(canvas, targetNodeIds, draft),
    });
    await commit(UPDATE_AGENT_COMMIT_MESSAGE);
    await invalidate();
    showSuccessToast("Agent saved.");
  } catch (error) {
    showErrorToast(getApiErrorMessage(error, "Failed to save agent"));
    throw error;
  }
}
