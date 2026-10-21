import { useRef, useState } from "react";
import type { SuperplaneComponentsNode } from "@/api-client";

import {
  applyPlanningReviewComponentToNode,
  canvasNodeToPlanningReviewComponent,
  type CanvasSpecNode,
} from "../lib/columnCanvasAgent";
import { IMPLEMENT_AGENT_NODE_ID } from "../lib/columnAutomationHeadline";
import type { NodeConfigurationUpdate } from "./nodeConfigurationCanvas";
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

type SaveNode = (update: NodeConfigurationUpdate) => Promise<void> | void;

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

export function useImplementSettingsDraft(nodes: SuperplaneComponentsNode[] | undefined, enabled: boolean) {
  const agent = enabled ? nodes?.find((node) => node.id === IMPLEMENT_AGENT_NODE_ID) : undefined;
  const [draft, setDraft] = useState<ImplementSettingsDraft | null>(() =>
    agent && nodes ? createImplementSettingsDraft(agent, nodes) : null,
  );
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<string>();
  const draftRef = useRef<ImplementSettingsDraft | null>(draft);
  const savingRef = useRef(false);

  if (agent && nodes && !draftRef.current) {
    const initial = createImplementSettingsDraft(agent, nodes);
    draftRef.current = initial;
    setDraft(initial);
  }

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

    if (!agentChanged && !pullRequestChanged) {
      onClose();
      return;
    }

    savingRef.current = true;
    setSaving(true);
    setSaveError(undefined);
    try {
      if (agentChanged) {
        await onSaveNode(agentUpdate);
        patchDraft({ savedAgent: agentSnapshot });
      }
      if (pullRequestChanged && pullRequestUpdate) {
        await onSaveNode(pullRequestUpdate);
        patchDraft({ savedPullRequest: pullRequestSnapshot });
      }
      onClose();
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
