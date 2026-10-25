import { useEffect, useRef, useState } from "react";
import type { SuperplaneComponentsNode } from "@/api-client";

import {
  applyPlanningReviewComponentToNode,
  canvasNodeToPlanningReviewComponent,
  type CanvasSpecNode,
} from "../lib/columnCanvasAgent";
import { IMPLEMENT_AGENT_NODE_ID } from "../lib/columnAutomationHeadline";
import type { NodeConfigurationSave, NodeConfigurationUpdate } from "./nodeConfigurationCanvas";
import type { PlanningReviewDraft } from "./planningReviewMockup";

const IMPLEMENT_SETTINGS_SAVE_ERROR = "Could not save Implement settings.";
const DEFAULT_TIMEOUT_SECONDS = 3600;
const PULL_REQUEST_COMPONENTS = ["github.createPullRequest", "bitbucket.createPullRequest"];

type ImplementSettingsDraft = {
  agentBase: SuperplaneComponentsNode;
  agentDraft: PlanningReviewDraft;
  pullRequestNode?: SuperplaneComponentsNode;
  savedAgent: string;
  savedPullRequest?: string;
  durationMinutes: string;
  durationInvalid: boolean;
};

type SaveNode = (update: NodeConfigurationSave) => Promise<void> | void;

export function findImplementPullRequest(nodes: SuperplaneComponentsNode[]) {
  return nodes.find((node) => PULL_REQUEST_COMPONENTS.includes(node.component ?? ""));
}

export function implementDurationMinutes(seconds: unknown): string {
  const timeout = typeof seconds === "number" && seconds > 0 ? seconds : DEFAULT_TIMEOUT_SECONDS;
  return String(timeout / 60);
}

function implementNodeUpdate(node: SuperplaneComponentsNode): NodeConfigurationUpdate {
  return {
    nodeId: node.id ?? "",
    name: node.name ?? "",
    configuration: { ...(node.configuration ?? {}) },
    integration: node.integration,
    concurrency: node.concurrency,
  };
}

function implementNodeSnapshot(node: SuperplaneComponentsNode): string {
  return JSON.stringify(implementNodeUpdate(node));
}

function agentUpdateFromDraft(base: SuperplaneComponentsNode, draft: PlanningReviewDraft): NodeConfigurationUpdate {
  const component = draft.components[0];
  if (!component) return implementNodeUpdate(base);
  return implementNodeUpdate(applyPlanningReviewComponentToNode(base as CanvasSpecNode, component));
}

function createImplementSettingsDraft(
  agent: SuperplaneComponentsNode,
  nodes: SuperplaneComponentsNode[],
): ImplementSettingsDraft {
  const agentBase = structuredClone(agent);
  const component = canvasNodeToPlanningReviewComponent(agentBase as CanvasSpecNode);
  const agentDraft = { title: component.title, components: [component] };
  const pullRequest = findImplementPullRequest(nodes);
  return {
    agentBase,
    agentDraft,
    pullRequestNode: pullRequest ? structuredClone(pullRequest) : undefined,
    savedAgent: JSON.stringify(agentUpdateFromDraft(agentBase, agentDraft)),
    savedPullRequest: pullRequest ? implementNodeSnapshot(pullRequest) : undefined,
    durationMinutes: implementDurationMinutes(agent.configuration?.executionTimeoutSeconds),
    durationInvalid: false,
  };
}

function pendingNodeUpdates(
  agentChanged: boolean,
  agentUpdate: NodeConfigurationUpdate,
  pullRequestChanged: boolean,
  pullRequestUpdate: NodeConfigurationUpdate | undefined,
): NodeConfigurationUpdate[] {
  return [
    ...(agentChanged ? [agentUpdate] : []),
    ...(pullRequestChanged && pullRequestUpdate ? [pullRequestUpdate] : []),
  ];
}

function sameEditableSettings(left: ImplementSettingsDraft, right: ImplementSettingsDraft): boolean {
  return (
    left.durationMinutes === right.durationMinutes &&
    left.durationInvalid === right.durationInvalid &&
    JSON.stringify(left.agentDraft) === JSON.stringify(right.agentDraft) &&
    JSON.stringify(left.pullRequestNode ?? null) === JSON.stringify(right.pullRequestNode ?? null)
  );
}

function hasUnsavedSettings(draft: ImplementSettingsDraft): boolean {
  return (
    draft.durationInvalid ||
    JSON.stringify(agentUpdateFromDraft(draft.agentBase, draft.agentDraft)) !== draft.savedAgent ||
    (draft.pullRequestNode ? implementNodeSnapshot(draft.pullRequestNode) : undefined) !== draft.savedPullRequest
  );
}

export function useImplementSettingsDraft(nodes: SuperplaneComponentsNode[] | undefined) {
  const agent = nodes?.find((node) => node.id === IMPLEMENT_AGENT_NODE_ID);
  const pullRequest = findImplementPullRequest(nodes ?? []);
  const [draft, setDraft] = useState<ImplementSettingsDraft | null>(() =>
    agent && nodes ? createImplementSettingsDraft(agent, nodes) : null,
  );
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<string>();
  const draftRef = useRef<ImplementSettingsDraft | null>(draft);
  const savingRef = useRef(false);

  useEffect(() => {
    if (!agent || savingRef.current || (draftRef.current && hasUnsavedSettings(draftRef.current))) return;
    const incoming = createImplementSettingsDraft(agent, pullRequest ? [pullRequest] : []);
    draftRef.current = incoming;
    setDraft(incoming);
  }, [agent, pullRequest]);

  const patchDraft = (patch: Partial<ImplementSettingsDraft>) => {
    if (!draftRef.current) return;
    draftRef.current = { ...draftRef.current, ...patch };
    setDraft(draftRef.current);
  };

  const updateAgentDraft = (agentDraft: PlanningReviewDraft) => {
    patchDraft({ agentDraft });
  };

  const updatePullRequestDraft = (draftPullRequest: boolean) => {
    const pullRequest = draftRef.current?.pullRequestNode;
    if (!pullRequest) return;
    patchDraft({
      pullRequestNode: {
        ...pullRequest,
        configuration: { ...pullRequest.configuration, draft: draftPullRequest },
      },
    });
  };

  const updateDuration = (state: { minutes: string; invalid: boolean }) => {
    patchDraft({ durationMinutes: state.minutes, durationInvalid: state.invalid });
  };

  const save = async (onSaveNode: SaveNode, onClose: () => void) => {
    const current = draftRef.current;
    if (!current || current.durationInvalid || savingRef.current) return;

    const agentUpdate = agentUpdateFromDraft(current.agentBase, current.agentDraft);
    const agentSnapshot = JSON.stringify(agentUpdate);
    const agentChanged = agentSnapshot !== current.savedAgent;
    const pullRequestUpdate = current.pullRequestNode ? implementNodeUpdate(current.pullRequestNode) : undefined;
    const pullRequestSnapshot = pullRequestUpdate ? JSON.stringify(pullRequestUpdate) : undefined;
    const pullRequestChanged = Boolean(pullRequestSnapshot && pullRequestSnapshot !== current.savedPullRequest);
    const updates = pendingNodeUpdates(agentChanged, agentUpdate, pullRequestChanged, pullRequestUpdate);

    if (updates.length === 0) {
      onClose();
      return;
    }

    savingRef.current = true;
    setSaving(true);
    setSaveError(undefined);
    try {
      await onSaveNode(updates.length === 1 ? updates[0] : updates);
      const latest = draftRef.current;
      patchDraft({
        ...(agentChanged ? { savedAgent: agentSnapshot } : {}),
        ...(pullRequestChanged && pullRequestSnapshot ? { savedPullRequest: pullRequestSnapshot } : {}),
      });
      if (!latest || sameEditableSettings(current, latest)) {
        onClose();
      }
    } catch {
      setSaveError(IMPLEMENT_SETTINGS_SAVE_ERROR);
    } finally {
      savingRef.current = false;
      setSaving(false);
    }
  };

  return {
    draft,
    saving,
    saveError,
    updateAgentDraft,
    updatePullRequestDraft,
    updateDuration,
    save,
  };
}
