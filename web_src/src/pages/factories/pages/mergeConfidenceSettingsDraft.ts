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
import type { NodeConfigurationSave, NodeConfigurationUpdate } from "./nodeConfigurationCanvas";
import type { PlanningReviewDraft, PlanningReviewStep } from "./planningReviewMockup";

type SettingsContext = { organizationId?: string; factoryId?: string; canvasId?: string };
type SettingsDraft = {
  agentBase: SuperplaneComponentsNode;
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
    checks: draft.checks,
    durationMinutes: draft.durationMinutes,
    durationInvalid: draft.durationInvalid,
  });
}

function createDraft(agent: SuperplaneComponentsNode): SettingsDraft {
  const agentBase = structuredClone(agent);
  const component = canvasNodeToPlanningReviewComponent(agentBase as CanvasSpecNode);
  const agentDraft = { title: component.title, components: [component] };
  const draft: SettingsDraft = {
    agentBase,
    agentDraft,
    checks: enabledMergeChecks(draftSteps(agentDraft)),
    savedSnapshot: "",
    durationMinutes: implementDurationMinutes(agent.configuration?.executionTimeoutSeconds),
    durationInvalid: false,
  };
  draft.savedSnapshot = snapshot(draft);
  return draft;
}

async function updateForSave(current: SettingsDraft, context: SettingsContext): Promise<NodeConfigurationUpdate> {
  const storedSteps = draftSteps(current.agentDraft);
  const storedChecks = enabledMergeChecks(storedSteps);
  const update = nodeUpdate(current);
  if (JSON.stringify(current.checks) === JSON.stringify(storedChecks)) return update;
  let defaults: PlanningReviewStep[] = [];
  if (current.checks.some((id) => !storedChecks.includes(id))) {
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
    configuration: { ...update.configuration, steps: stepsWithMergeChecks(storedSteps, current.checks, defaults) },
  };
}

export function useMergeConfidenceSettingsDraft(agent: SuperplaneComponentsNode | undefined, context: SettingsContext) {
  const [draft, setDraft] = useState<SettingsDraft | null>(() => (agent ? createDraft(agent) : null));
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
    const incoming = createDraft(agent);
    draftRef.current = incoming;
    setDraft(incoming);
  }, [agent]);

  const patchDraft = (patch: Partial<SettingsDraft>) => {
    if (!draftRef.current) return;
    draftRef.current = { ...draftRef.current, ...patch };
    setDraft(draftRef.current);
  };

  const save = async (onSave: (update: NodeConfigurationSave) => Promise<void> | void, onClose: () => void) => {
    const current = draftRef.current;
    if (!current || current.durationInvalid || savingRef.current) return;
    if (snapshot(current) === current.savedSnapshot) {
      onClose();
      return;
    }
    savingRef.current = true;
    setSaving(true);
    setSaveError(undefined);
    try {
      const update = await updateForSave(current, context);
      try {
        await onSave(update);
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
