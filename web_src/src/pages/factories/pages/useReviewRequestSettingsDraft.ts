import { useState } from "react";
import type { SuperplaneComponentsNode } from "@/api-client";
import { getApiErrorMessage } from "@/lib/errors";
import { canvasNodeToPlanningReviewComponent } from "../lib/columnCanvasAgent";
import {
  readReviewFindings,
  reviewFindingsActivation,
  reviewFindingsMissingDestination,
  writeReviewFindings,
  type ReviewFindingsSettings,
} from "../lib/reviewRequestFindings";
import type { NodeConfigurationSave, NodeConfigurationUpdate } from "./nodeConfigurationCanvas";
import type { PlanningReviewDraft } from "./planningReviewMockup";
import type { IntakeAutomationGraph } from "./useIntakeAutomationCanvas";

export const REVIEW_REQUEST_TRIGGER_NODE_ID = "review-request-trigger";
export const REVIEW_REQUEST_AGENT_NODE_ID = "select-reviewers";
export const REVIEW_REQUEST_CHECK_NODE_ID = "pr-check-pending";

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
  const findings = readReviewFindings(draft?.components[0]?.configuration);
  const findingsMissingDestination = reviewFindingsMissingDestination(findings);

  const save = async () => {
    const component = draft?.components[0];
    if (saving || unavailable || findingsMissingDestination || !component || !agent || !draft) return;
    setSaving(true);
    setSaveError(undefined);
    try {
      await onSaveNode(settingsUpdates(graph, agent, draft));
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
    findings,
    findingsMissingDestination,
    updateFindings: (patch: Partial<ReviewFindingsSettings>) => {
      if (!draft) return;
      setEdited({
        ...draft,
        components: draft.components.map((component) => ({
          ...component,
          configuration: writeReviewFindings(component.configuration, { ...findings, ...patch }),
        })),
      });
    },
    publishPRCheck: draft?.components[0]?.configuration.publishPRCheck === true,
    updatePublishPRCheck: (publishPRCheck: boolean) => {
      if (!draft) return;
      setEdited({
        ...draft,
        components: draft.components.map((component) => ({
          ...component,
          configuration: { ...component.configuration, publishPRCheck },
        })),
      });
    },
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

// Publication settings stay on the agent node, which admin reset keeps. Each
// publication node receives its effective state and activation generation.
function settingsUpdates(
  graph: IntakeAutomationGraph,
  agent: SuperplaneComponentsNode,
  draft: PlanningReviewDraft,
): NodeConfigurationSave {
  const component = draft.components[0];
  const check = prCheckActivation(agent, component.configuration);
  const findings = reviewFindingsActivation(agent.configuration, component.configuration);
  const updates: NodeConfigurationUpdate[] = [
    {
      ...nodeUpdate(agent),
      configuration: {
        ...agent.configuration,
        ...component.configuration,
        fallbackReviewer: fallbackReviewer(draft),
        publishPRCheck: check.enabled,
        prCheckActivationGeneration: check.activationGeneration,
        ...findings.generations,
      },
    },
  ];
  const destinations = [
    { ...check, missingMessage: "Update this automation through admin reset before you publish PR checks." },
    ...findings.nodes.map((destination) => ({
      ...destination,
      missingMessage: "Update this automation through admin reset before you publish review findings.",
    })),
  ];
  for (const destination of destinations) {
    const node = graph.specNodes?.find((candidate) => candidate.id === destination.nodeId);
    if (!node && destination.enabled) throw new Error(destination.missingMessage);
    if (!node) continue;
    updates.push({
      ...nodeUpdate(node),
      configuration: {
        ...node.configuration,
        enabled: destination.enabled,
        activationGeneration: destination.activationGeneration,
      },
    });
  }
  return updates.length === 1 ? updates[0] : updates;
}

// An off-to-on save increases the generation, so old attempts cannot resume.
function prCheckActivation(agent: SuperplaneComponentsNode, configuration: Record<string, unknown>) {
  const enabled = configuration.publishPRCheck === true;
  const wasPublished = agent.configuration?.publishPRCheck === true;
  const savedGeneration = Number(agent.configuration?.prCheckActivationGeneration) || 1;
  return {
    nodeId: REVIEW_REQUEST_CHECK_NODE_ID,
    enabled,
    activationGeneration: savedGeneration + (enabled && !wasPublished ? 1 : 0),
  };
}
