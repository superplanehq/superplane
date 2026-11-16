import { useEffect, useRef, useState } from "react";
import type { SuperplaneComponentsNode } from "@/api-client";
import {
  applyPlanningReviewComponentToNode,
  canvasNodeToPlanningReviewComponent,
  type CanvasSpecNode,
} from "../lib/columnCanvasAgent";
import { loadDefaultAgentPrompt } from "../lib/loadDefaultAgentPrompt";
import { implementDurationMinutes } from "./implementSettingsDraft";
import { MERGE_CONFIDENCE_CHECKS, type MergeConfidenceCheck } from "./mergeConfidenceChecks";
import { enabledMergeChecks, stepsWithMergeChecks } from "./mergeConfidenceStepSelection";
import { expandMergeConfidenceSteps } from "./mergeConfidenceSteps";
import type { NodeConfigurationSave, NodeConfigurationUpdate } from "./nodeConfigurationCanvas";
import type { PlanningReviewDraft, PlanningReviewStep } from "./planningReviewMockup";

const NO_PUBLICATION_NODES: SuperplaneComponentsNode[] = [];

type SettingsContext = { organizationId?: string; factoryId?: string; canvasId?: string };
type SettingsDraft = {
  agentBase: SuperplaneComponentsNode;
  publishingBase?: SuperplaneComponentsNode;
  publishResults: boolean;
  resultsBody: boolean;
  resultsComment: boolean;
  publishPRCheck: boolean;
  publicationNodes: SuperplaneComponentsNode[];
  agentDraft: PlanningReviewDraft;
  checks: MergeConfidenceCheck[];
  savedSnapshot: string;
  durationMinutes: string;
  durationInvalid: boolean;
};

function draftSteps(draft: PlanningReviewDraft): PlanningReviewStep[] {
  return (draft.components[0]?.configuration.steps as PlanningReviewStep[] | undefined) ?? [];
}

function nodeUpdate(draft: SettingsDraft): NodeConfigurationUpdate {
  const node = applyPlanningReviewComponentToNode(draft.agentBase as CanvasSpecNode, draft.agentDraft.components[0]);
  return {
    nodeId: node.id ?? "",
    name: node.name ?? "",
    configuration: node.configuration ?? {},
    integration: node.integration,
    concurrency: node.concurrency,
  };
}

function snapshot(draft: SettingsDraft): string {
  return JSON.stringify({
    update: nodeUpdate(draft),
    publishResults: draft.publishResults,
    resultsBody: draft.resultsBody,
    resultsComment: draft.resultsComment,
    publishPRCheck: draft.publishPRCheck,
    checks: draft.checks,
    durationMinutes: draft.durationMinutes,
    durationInvalid: draft.durationInvalid,
  });
}

function createDraft(
  agent: SuperplaneComponentsNode,
  publishing?: SuperplaneComponentsNode,
  publicationNodes: SuperplaneComponentsNode[] = NO_PUBLICATION_NODES,
): SettingsDraft {
  const agentBase = structuredClone(agent);
  const component = canvasNodeToPlanningReviewComponent(agentBase as CanvasSpecNode);
  const agentDraft = { title: component.title, components: [component] };
  const draft: SettingsDraft = {
    agentBase,
    publishingBase: publishing ? structuredClone(publishing) : undefined,
    publishResults:
      agent.configuration?.publishResults === undefined
        ? publishing?.configuration?.enabled !== false
        : agent.configuration.publishResults !== false,
    resultsBody: agent.configuration?.resultsBody !== false,
    resultsComment: agent.configuration?.resultsComment === true,
    publishPRCheck: agent.configuration?.publishPRCheck !== false,
    publicationNodes,
    agentDraft,
    checks: enabledMergeChecks(expandMergeConfidenceSteps(draftSteps(agentDraft))),
    savedSnapshot: "",
    durationMinutes: implementDurationMinutes(agent.configuration?.executionTimeoutSeconds),
    durationInvalid: false,
  };
  draft.savedSnapshot = snapshot(draft);
  return draft;
}

async function updateForSave(current: SettingsDraft, context: SettingsContext): Promise<NodeConfigurationUpdate> {
  const storedSteps = draftSteps(current.agentDraft);
  const expandedSteps = expandMergeConfidenceSteps(storedSteps);
  const storedChecks = enabledMergeChecks(expandedSteps);
  const update = nodeUpdate(current);
  const checksChanged = JSON.stringify(current.checks) !== JSON.stringify(storedChecks);
  const legacyPrompt = JSON.stringify(storedSteps) !== JSON.stringify(expandedSteps);
  if (!checksChanged && !legacyPrompt) return update;
  let defaults: PlanningReviewStep[] = [];
  if (checksChanged && current.checks.some((id) => !storedChecks.includes(id))) {
    const { organizationId, factoryId, canvasId } = context;
    if (!organizationId || !factoryId || !canvasId)
      throw new Error("Could not load default checks. Reload the settings and try again.");
    try {
      defaults = await loadDefaultAgentPrompt({
        organizationId,
        factoryId,
        automationId: canvasId,
        agentNodeId: "assess-risk",
      });
    } catch {
      throw new Error("Could not load default checks. Try saving again.");
    }
  }
  return {
    ...update,
    configuration: { ...update.configuration, steps: stepsWithMergeChecks(expandedSteps, current.checks, defaults) },
  };
}

export function useMergeConfidenceSettingsDraft(
  agent: SuperplaneComponentsNode | undefined,
  context: SettingsContext,
  publishing?: SuperplaneComponentsNode,
  publicationNodes: SuperplaneComponentsNode[] = NO_PUBLICATION_NODES,
) {
  const [draft, setDraft] = useState<SettingsDraft | null>(() =>
    agent ? createDraft(agent, publishing, publicationNodes) : null,
  );
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<string>();
  const draftRef = useRef(draft);
  const savingRef = useRef(false);

  useEffect(() => {
    if (
      !agent ||
      savingRef.current ||
      (draftRef.current && snapshot(draftRef.current) !== draftRef.current.savedSnapshot)
    )
      return;
    const incoming = createDraft(agent, publishing, publicationNodes);
    draftRef.current = incoming;
    setDraft(incoming);
  }, [agent, publishing, publicationNodes]);

  const patchDraft = (patch: Partial<SettingsDraft>) => {
    if (!draftRef.current) return;
    draftRef.current = { ...draftRef.current, ...patch };
    setDraft(draftRef.current);
  };

  const save = async (onSave: (update: NodeConfigurationSave) => Promise<void> | void, onClose: () => void) => {
    const current = draftRef.current;
    if (
      !current ||
      current.durationInvalid ||
      (current.publishResults && !current.resultsBody && !current.resultsComment) ||
      savingRef.current
    )
      return;
    if (snapshot(current) === current.savedSnapshot) {
      onClose();
      return;
    }
    savingRef.current = true;
    setSaving(true);
    setSaveError(undefined);
    try {
      const update = await updateForSave(current, context);
      const updates = publicationUpdates(current, update);
      try {
        await onSave(updates);
      } catch {
        throw new Error("Could not save Merge confidence settings. Try saving again.");
      }
      const component = current.agentDraft.components[0];
      const saved = {
        ...current,
        agentDraft: { ...current.agentDraft, components: [{ ...component, configuration: update.configuration }] },
      };
      if (draftRef.current && snapshot(draftRef.current) === snapshot(current)) {
        patchDraft({ ...saved, savedSnapshot: snapshot(saved) });
        onClose();
      }
    } catch (error) {
      setSaveError(
        error instanceof Error ? error.message : "Could not save Merge confidence settings. Try saving again.",
      );
    } finally {
      savingRef.current = false;
      setSaving(false);
    }
  };

  return {
    draft,
    saving,
    saveError,
    save,
    updateAgentDraft: (agentDraft: PlanningReviewDraft) => patchDraft({ agentDraft }),
    updatePublishResults: (publishResults: boolean) => patchDraft({ publishResults }),
    updateResultsBody: (resultsBody: boolean) => patchDraft({ resultsBody }),
    updateResultsComment: (resultsComment: boolean) => patchDraft({ resultsComment }),
    updatePublishPRCheck: (publishPRCheck: boolean) => patchDraft({ publishPRCheck }),
    updateChecks: (check: MergeConfidenceCheck, enabled: boolean) => {
      if (!draftRef.current) return;
      const checks = enabled
        ? [...draftRef.current.checks, check]
        : draftRef.current.checks.filter((id) => id !== check);
      patchDraft({ checks: MERGE_CONFIDENCE_CHECKS.filter((id) => checks.includes(id)) });
    },
    updateDuration: (state: { minutes: string; invalid: boolean }) =>
      patchDraft({ durationMinutes: state.minutes, durationInvalid: state.invalid }),
  };
}

function publicationUpdates(current: SettingsDraft, update: NodeConfigurationUpdate): NodeConfigurationSave {
  const body = current.publishingBase;
  if (!body) return update;
  const saved = current.agentBase.configuration;
  const github = body.component === "github.updatePullRequestBodySection";
  const destinations = publicationDestinations(current, body);
  update.configuration = {
    ...update.configuration,
    publishResults: current.publishResults,
    resultsBody: current.resultsBody,
    resultsComment: current.resultsComment,
    ...(github ? { publishPRCheck: current.publishPRCheck } : {}),
  };
  const updates = [update];
  for (const destination of destinations) {
    if (!destination.node) {
      if (destination.enabled && !destination.wasEnabled)
        throw new Error("Update this automation through admin reset before you enable this destination.");
      continue;
    }
    const generation =
      (Number(saved?.[destination.key]) || 1) + (destination.enabled && !destination.wasEnabled ? 1 : 0);
    update.configuration[destination.key] = generation;
    const node = destination.node;
    updates.push(
      publicationNodeUpdate(node, {
        ...node.configuration,
        enabled: destination.enabled,
        activationGeneration: generation,
        ...(node.id === "start-assessment-check" ? { activationNodeId: node.id } : {}),
      }),
    );
  }
  return [...updates, ...finalCheckUpdate(current, update.configuration)];
}

function publicationDestinations(current: SettingsDraft, body: SuperplaneComponentsNode) {
  const saved = current.agentBase.configuration;
  const wasPublished =
    saved?.publishResults === undefined ? body.configuration?.enabled !== false : saved.publishResults !== false;
  const github = body.component === "github.updatePullRequestBodySection";
  return [
    {
      node: body,
      enabled: current.publishResults && current.resultsBody,
      wasEnabled: wasPublished && saved?.resultsBody !== false,
      key: "resultsBodyActivationGeneration",
    },
    ...(github
      ? [
          {
            node: current.publicationNodes.find((node) => node.id === "comment-merge-confidence"),
            enabled: current.publishResults && current.resultsComment,
            wasEnabled: wasPublished && saved?.resultsComment === true,
            key: "resultsCommentActivationGeneration",
          },
          {
            node: current.publicationNodes.find((node) => node.id === "start-assessment-check"),
            enabled: current.publishPRCheck,
            wasEnabled: saved?.publishPRCheck !== false,
            key: "prCheckActivationGeneration",
          },
        ]
      : []),
  ];
}

function publicationNodeUpdate(
  node: SuperplaneComponentsNode,
  configuration: Record<string, unknown>,
): NodeConfigurationUpdate {
  return {
    nodeId: node.id ?? "",
    name: node.name ?? "",
    configuration,
    integration: node.integration,
    concurrency: node.concurrency,
  };
}

function finalCheckUpdate(current: SettingsDraft, configuration: Record<string, unknown>): NodeConfigurationUpdate[] {
  const start = current.publicationNodes.find((node) => node.id === "start-assessment-check");
  const finish = current.publicationNodes.find((node) => node.id === "finish-assessment-check");
  if (!start || !finish) return [];
  return [
    publicationNodeUpdate(finish, {
      ...finish.configuration,
      activationNodeId: start.id,
      activationGeneration: configuration.prCheckActivationGeneration,
    }),
  ];
}
