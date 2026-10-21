import { useCallback, useEffect, useRef, useState } from "react";
import { getApiErrorMessage } from "@/lib/errors";
import type { PlanningReviewAgentSlot } from "./PlanningReviewEditor";
import type { PlanningReviewDraft } from "./planningReviewMockup";
import { PLANNING_SETTINGS_COPY, type PlanningDraftSettings } from "./planningSettingsModel";

export function usePlanningSettingsDraft({
  settings,
  agent,
  savePending,
  onSave,
  onClose,
}: {
  settings: PlanningDraftSettings;
  agent?: PlanningReviewAgentSlot;
  savePending: boolean;
  onSave: (next: PlanningDraftSettings) => Promise<void> | void;
  onClose: () => void;
}) {
  const [draft, setDraft] = useState(settings);
  const previousSettings = useRef(settings);
  const [editedAgent, setEditedAgent] = useState<PlanningReviewDraft>();
  const [saving, setSaving] = useState(false);
  const [localSaveError, setLocalSaveError] = useState<string>();
  const agentDraft = editedAgent ?? agent?.draft;
  const agentUnavailable = Boolean(agent && (agent.isLoading || !agentDraft?.components.length));
  const busy = saving || savePending;
  const save = async () => {
    if (busy || agentUnavailable) return;
    setSaving(true);
    setLocalSaveError(undefined);
    try {
      if (editedAgent) {
        if (!agent?.onSave) throw new Error(PLANNING_SETTINGS_COPY.saveError);
        await agent.onSave(editedAgent);
      }
      await onSave(draft);
      onClose();
    } catch (error) {
      setLocalSaveError(getApiErrorMessage(error, PLANNING_SETTINGS_COPY.saveError));
    } finally {
      setSaving(false);
    }
  };
  const setPlanningEnabled = async (enabled: boolean) => {
    if (busy) return;
    setSaving(true);
    setLocalSaveError(undefined);
    try {
      // Pause is an immediate action. Do not publish unsaved form changes.
      await onSave({ ...settings, enabled });
      setDraft((current) => ({ ...current, enabled }));
    } catch (error) {
      setLocalSaveError(getApiErrorMessage(error, PLANNING_SETTINGS_COPY.saveError));
    } finally {
      setSaving(false);
    }
  };

  useEffect(() => {
    const previous = previousSettings.current;
    previousSettings.current = {
      enabled: settings.enabled,
      clarity: settings.clarity,
      confidence: settings.confidence,
      autoStartLineId: settings.autoStartLineId,
    };
    // Refresh saved fields without losing edits to other settings. Pause and
    // resume update the factory while this form can still have unsaved edits.
    const changed: Partial<PlanningDraftSettings> = {};
    if (previous.enabled !== settings.enabled) changed.enabled = settings.enabled;
    if (previous.clarity !== settings.clarity) changed.clarity = settings.clarity;
    if (previous.confidence !== settings.confidence) changed.confidence = settings.confidence;
    if (previous.autoStartLineId !== settings.autoStartLineId) changed.autoStartLineId = settings.autoStartLineId;
    setDraft((current) => ({ ...current, ...changed }));
  }, [settings.enabled, settings.clarity, settings.confidence, settings.autoStartLineId]);

  const update = useCallback(<K extends keyof PlanningDraftSettings>(key: K, value: PlanningDraftSettings[K]) => {
    setDraft((current) => ({ ...current, [key]: value }));
  }, []);

  return {
    draft,
    agentDraft,
    setEditedAgent,
    update,
    busy,
    saveDisabled: busy || agentUnavailable,
    save,
    saveError: localSaveError,
    setPlanningEnabled,
  };
}
