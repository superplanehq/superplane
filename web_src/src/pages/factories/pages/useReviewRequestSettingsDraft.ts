import { useState } from "react";
import type { SuperplaneComponentsNode } from "@/api-client";
import { getApiErrorMessage } from "@/lib/errors";
import { canvasNodeToPlanningReviewComponent } from "../lib/columnCanvasAgent";
import type { NodeConfigurationSave } from "./nodeConfigurationCanvas";
import type { PlanningReviewDraft } from "./planningReviewMockup";
import type { IntakeAutomationGraph } from "./useIntakeAutomationCanvas";

export const REVIEW_REQUEST_TRIGGER_NODE_ID = "review-request-trigger";
export const REVIEW_REQUEST_AGENT_NODE_ID = "select-reviewers";

export function useReviewRequestSettingsDraft({
  graph,
  loading,
  error,
  onSaveNode,
  onClose,
}: {
  graph: IntakeAutomationGraph;
  loading?: boolean;
  error?: boolean;
  onSaveNode: (update: NodeConfigurationSave) => Promise<void>;
  onClose: () => void;
}) {
  const agent = graph.specNodes?.find((node) => node.id === REVIEW_REQUEST_AGENT_NODE_ID);
  const trigger = graph.specNodes?.find((node) => node.id === REVIEW_REQUEST_TRIGGER_NODE_ID);
  const [edited, setEdited] = useState<PlanningReviewDraft>();
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<string>();
  const enabled = trigger?.configuration?.enabled !== false;
  const draft = edited ?? agentDraft(agent);
  const unavailable = Boolean(loading || error || !agent || !trigger || !draft);

  const save = async () => {
    const component = draft?.components[0];
    if (saving || unavailable || !component || !agent || !draft) return;
    setSaving(true);
    setSaveError(undefined);
    try {
      await onSaveNode({
        ...nodeUpdate(agent),
        configuration: { ...component.configuration, fallbackReviewer: fallbackReviewer(draft) },
      });
      onClose();
    } catch (failure) {
      setSaveError(getApiErrorMessage(failure, "Could not save Review request. Try again."));
    } finally {
      setSaving(false);
    }
  };
  const setEnabled = async (next: boolean) => {
    if (saving || !trigger) return;
    setSaving(true);
    setSaveError(undefined);
    try {
      // Pause changes admission immediately and leaves unsaved model edits alone.
      await onSaveNode({ ...nodeUpdate(trigger), configuration: { ...trigger.configuration, enabled: next } });
    } catch (failure) {
      setSaveError(getApiErrorMessage(failure, "Could not update Review request. Try again."));
    } finally {
      setSaving(false);
    }
  };
  const updateFallbackReviewer = (fallbackReviewer: string) => {
    if (!draft) return;
    setEdited({
      ...draft,
      components: draft.components.map((component) => ({
        ...component,
        configuration: { ...component.configuration, fallbackReviewer },
      })),
    });
  };

  return {
    draft,
    setEdited,
    saving,
    saveError,
    enabled,
    unavailable,
    save,
    setEnabled,
    updateFallbackReviewer,
    fallbackReviewer: draft ? fallbackReviewer(draft) : "",
  };
}

function agentDraft(agent: SuperplaneComponentsNode | undefined): PlanningReviewDraft | undefined {
  if (!agent) return undefined;
  return { title: "Review request", components: [canvasNodeToPlanningReviewComponent(agent)] };
}

function fallbackReviewer(draft: PlanningReviewDraft): string {
  const reviewer = draft.components[0]?.configuration.fallbackReviewer;
  return typeof reviewer === "string" ? reviewer : "";
}

function nodeUpdate(node: SuperplaneComponentsNode) {
  return {
    nodeId: node.id ?? "",
    name: node.name ?? "",
    configuration: node.configuration ?? {},
    integration: node.integration,
    concurrency: node.concurrency,
  };
}
