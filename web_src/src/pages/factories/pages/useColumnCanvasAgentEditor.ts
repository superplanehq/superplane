import type { CanvasesCanvas } from "@/api-client";
import { canvasKeys, useCanvas, useCommitCanvasStaging, useUpdateCanvasVersion } from "@/hooks/useCanvasData";
import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { getApiErrorMessage } from "@/lib/errors";
import { FEATURE_FACTORY_CREATE_WITH_AGENT } from "@/lib/experimentalFeatures";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";

import { planningReviewDraftFromCanvas, primaryAgentNode, serializeColumnAgentCanvas } from "../lib/columnCanvasAgent";
import type { PlanningReviewDraft } from "./planningReviewMockup";

const UPDATE_AGENT_COMMIT_MESSAGE = "Update agent";
const REFINEMENT_AGENT_NODE_ID = "refine-task";

export function useColumnCanvasAgentEditor(organizationId: string, appId: string | undefined) {
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
  const draft = canvas && agentNode?.id ? planningReviewDraftFromCanvas(canvas, agentNode.id) : null;

  const save = async (nextDraft: PlanningReviewDraft) => {
    await persistColumnAgent({
      appId: canvasId,
      canvas,
      agentNodeId: agentNode?.id,
      draft: nextDraft,
      stageYaml: (input) => updateVersion.mutateAsync(input),
      commit: (message) => commitStaging.mutateAsync(message),
      invalidate: () => queryClient.invalidateQueries({ queryKey: canvasKeys.detail(organizationId, canvasId) }),
    });
  };

  return {
    agentNode,
    isLoading: enabled && (canvasQuery.isPending || features.isLoading),
    draft,
    editorOpen,
    openEditor: agentNode ? () => setEditorOpen(true) : undefined,
    closeEditor: () => setEditorOpen(false),
    save,
  };
}

export async function persistColumnAgent(args: {
  appId: string;
  canvas: CanvasesCanvas | undefined;
  agentNodeId: string | undefined;
  draft: PlanningReviewDraft;
  stageYaml: (input: { versionId: string; canvasYaml: string }) => Promise<unknown>;
  commit: (message: string) => Promise<unknown>;
  invalidate: () => Promise<unknown> | unknown;
}) {
  const { canvas, agentNodeId, appId, draft, stageYaml, commit, invalidate } = args;
  if (!canvas || !agentNodeId || !appId) {
    throw new Error("Agent canvas is not loaded");
  }
  const liveVersionId = canvas.metadata?.liveVersionId;
  if (!liveVersionId) {
    throw new Error("Canvas has no live version");
  }

  try {
    await stageYaml({
      versionId: liveVersionId,
      canvasYaml: serializeColumnAgentCanvas(canvas, agentNodeId, draft),
    });
    await commit(UPDATE_AGENT_COMMIT_MESSAGE);
    await invalidate();
    showSuccessToast("Agent saved.");
  } catch (error) {
    showErrorToast(getApiErrorMessage(error, "Failed to save agent"));
    throw error;
  }
}
