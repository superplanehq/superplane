import type { SuperplaneComponentsNode } from "@/api-client";
import { useEffect, useRef, useState } from "react";

import { canvasNodeToPlanningReviewComponent, type CanvasSpecNode } from "../lib/columnCanvasAgent";
import type { NodeConfigurationUpdate } from "./nodeConfigurationCanvas";
import { expandMergeConfidenceSteps } from "./mergeConfidenceSteps";
import { PlanningReviewForm } from "./PlanningReviewForm";
import type { PlanningReviewComponent, PlanningReviewDraft, PlanningReviewStep } from "./planningReviewMockup";

type SaveAgent = (update: NodeConfigurationUpdate) => Promise<void> | void;

type PendingSave = {
  draft: PlanningReviewDraft;
  node: SuperplaneComponentsNode;
};

type SaveState = {
  draft: PlanningReviewDraft;
  node: SuperplaneComponentsNode;
  onSave: SaveAgent;
  saving: boolean;
  pending: PendingSave | null;
  submitted: string;
};

/** Agent settings for one runner step. Choices save on change. Text saves when you leave the field. */
export function AgentRunnerSettings({
  node,
  organizationId,
  factoryId,
  factoryKey,
  onSave,
}: {
  node: SuperplaneComponentsNode;
  organizationId?: string;
  factoryId?: string;
  factoryKey?: string;
  onSave: SaveAgent;
}) {
  const initial = draftFromNode(node);
  const [draft, setDraft] = useState(initial);
  const saveState = useRef<SaveState>({
    draft: initial,
    node,
    onSave,
    saving: false,
    pending: null,
    submitted: snapshotOf(initial),
  });
  saveState.current.onSave = onSave;
  const nodeSignature = agentNodeSignature(node);

  useEffect(() => {
    acceptNodeUpdate(saveState.current, node, setDraft);
  }, [node, nodeSignature]);

  useEffect(() => {
    const state = saveState.current;
    return () => {
      void persistAgent(state.draft, state.node, state);
    };
  }, []);

  return (
    <div
      className="min-h-0 flex-1 overflow-y-auto px-10 py-6"
      onBlurCapture={(event) => {
        if (!isTextControl(event.target)) {
          return;
        }
        const state = saveState.current;
        void persistAgent(state.draft, state.node, state);
      }}
    >
      <PlanningReviewForm
        appearance="fields"
        showResources={false}
        showConcurrency={false}
        draft={draft}
        onChange={(next) => applyDraft(saveState.current, next, setDraft)}
        organizationId={organizationId}
        factoryId={factoryId}
        factoryKey={factoryKey}
      />
    </div>
  );
}

function acceptNodeUpdate(
  state: SaveState,
  node: SuperplaneComponentsNode,
  setDraft: (draft: PlanningReviewDraft) => void,
) {
  const incoming = draftFromNode(node);
  const current = state.draft;
  if ((state.node.id ?? "") !== (node.id ?? "")) {
    void persistAgent(current, state.node, state);
    replaceDraft(state, incoming, node, setDraft);
    return;
  }
  if (sameDraft(incoming, current) || keepsLocalText(incoming, current)) {
    state.node = node;
    return;
  }
  replaceDraft(state, incoming, node, setDraft);
}

function applyDraft(state: SaveState, next: PlanningReviewDraft, setDraft: (draft: PlanningReviewDraft) => void) {
  const previous = state.draft;
  state.draft = next;
  setDraft(next);
  if (savesOnChange(previous, next)) {
    void persistAgent(next, state.node, state);
  }
}

function replaceDraft(
  state: SaveState,
  incoming: PlanningReviewDraft,
  node: SuperplaneComponentsNode,
  setDraft: (draft: PlanningReviewDraft) => void,
) {
  state.draft = incoming;
  state.node = node;
  state.submitted = snapshotOf(incoming);
  setDraft(incoming);
}

async function persistAgent(draft: PlanningReviewDraft, node: SuperplaneComponentsNode, state: SaveState) {
  const component = draft.components[0];
  if (!component) {
    return;
  }
  const snapshot = JSON.stringify(agentSnapshot(component));
  if (snapshot === state.submitted || sameAgent(node, component)) {
    state.submitted = snapshot;
    return;
  }
  if (state.saving) {
    state.pending = { draft, node };
    return;
  }
  state.saving = true;
  try {
    await state.onSave(updateFromComponent(node, component));
    state.submitted = snapshot;
  } finally {
    state.saving = false;
    const pending = state.pending;
    state.pending = null;
    if (pending) {
      await persistAgent(pending.draft, pending.node, state);
    }
  }
}

function savesOnChange(previous: PlanningReviewDraft, next: PlanningReviewDraft): boolean {
  const before = previous.components[0];
  const after = next.components[0];
  if (!before || !after || before.id !== after.id) {
    return Boolean(before && after);
  }
  return structuralSnapshot(before) !== structuralSnapshot(after) || isReorder(before, after);
}

function keepsLocalText(incoming: PlanningReviewDraft, current: PlanningReviewDraft): boolean {
  const saved = incoming.components[0];
  const local = current.components[0];
  if (!saved || !local || saved.id !== local.id) {
    return false;
  }
  return !savesOnChange(incoming, current);
}

function sameDraft(left: PlanningReviewDraft, right: PlanningReviewDraft): boolean {
  const leftComponent = left.components[0];
  const rightComponent = right.components[0];
  if (!leftComponent || !rightComponent) {
    return false;
  }
  return JSON.stringify(agentSnapshot(leftComponent)) === JSON.stringify(agentSnapshot(rightComponent));
}

function structuralSnapshot(component: PlanningReviewComponent): string {
  const { steps: _steps, ...configuration } = component.configuration;
  return JSON.stringify({
    id: component.id,
    title: component.title,
    configuration,
    steps: stepsOf(component).map((step) => step.type),
  });
}

function isReorder(before: PlanningReviewComponent, after: PlanningReviewComponent): boolean {
  const previous = stepsOf(before).map(stepSignature);
  const next = stepsOf(after).map(stepSignature);
  if (previous.length !== next.length || previous.join("\n") === next.join("\n")) {
    return false;
  }
  return [...previous].sort().join("\n") === [...next].sort().join("\n");
}

function stepSignature(step: PlanningReviewStep): string {
  return JSON.stringify([step.type, step.name, step.command ?? "", step.prompt ?? "", step.workingDirectory ?? ""]);
}

function stepsOf(component: PlanningReviewComponent): PlanningReviewStep[] {
  const steps = component.configuration.steps;
  return Array.isArray(steps) ? (steps as PlanningReviewStep[]) : [];
}

function isTextControl(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) {
    return false;
  }
  return target.tagName === "INPUT" || target.tagName === "TEXTAREA" || target.isContentEditable;
}

function snapshotOf(draft: PlanningReviewDraft): string {
  const component = draft.components[0];
  return component ? JSON.stringify(agentSnapshot(component)) : "";
}

function draftFromNode(node: SuperplaneComponentsNode): PlanningReviewDraft {
  const component = canvasNodeToPlanningReviewComponent(node as CanvasSpecNode);
  const steps = component.configuration.steps;
  return {
    title: component.title,
    components: [
      {
        ...component,
        configuration: {
          ...component.configuration,
          steps: Array.isArray(steps) ? expandMergeConfidenceSteps(steps as PlanningReviewStep[]) : steps,
        },
      },
    ],
  };
}

function agentNodeSignature(node: SuperplaneComponentsNode) {
  return JSON.stringify({
    id: node.id,
    name: node.name,
    configuration: node.configuration,
    concurrency: node.concurrency,
  });
}

function sameAgent(node: SuperplaneComponentsNode, component: PlanningReviewComponent) {
  const current = canvasNodeToPlanningReviewComponent(node as CanvasSpecNode);
  return JSON.stringify(agentSnapshot(current)) === JSON.stringify(agentSnapshot(component));
}

function agentSnapshot(component: PlanningReviewComponent) {
  return {
    title: component.title,
    configuration: component.configuration,
    max: component.concurrency.max,
    key: component.concurrency.key,
  };
}

function updateFromComponent(
  node: SuperplaneComponentsNode,
  component: PlanningReviewComponent,
): NodeConfigurationUpdate {
  const next = nodeFromComponent(node, component);
  return {
    nodeId: node.id ?? "",
    name: component.title,
    configuration: next.configuration ?? {},
    concurrency: next.concurrency,
  };
}

function nodeFromComponent(
  node: SuperplaneComponentsNode,
  component: PlanningReviewComponent,
): SuperplaneComponentsNode {
  const parsedMax = Number.parseInt(component.concurrency.max, 10);
  const max = Number.isFinite(parsedMax) && parsedMax > 0 ? parsedMax : 1;
  const key = component.concurrency.key.trim();
  return {
    ...node,
    name: component.title,
    configuration: component.configuration,
    concurrency: key ? { max, key } : { max },
  };
}
