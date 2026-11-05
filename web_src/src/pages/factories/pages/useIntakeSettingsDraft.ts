import { useCallback, useEffect, useRef, useState, type SetStateAction } from "react";

import { intakeSettingsContentKey, type IntakeSourceSettings } from "./intakeSourceSettingsModel";
import { linearFilterChoicesFromSettings, type LinearFilterChoices } from "./linearIntakeScope";

/**
 * Keeps unsaved edits while the parent passes equal settings in a new object.
 * `getDraft` returns the latest edit even before React re-renders.
 * The Linear filter choices live here too, so a tab change does not reset a
 * selected filter that has no items yet.
 */
export function useIntakeSettingsDraft(settings: IntakeSourceSettings) {
  const [draft, setDraft] = useState(settings);
  const [linearFilterChoices, setLinearFilterChoices] = useState<LinearFilterChoices>(() =>
    linearFilterChoicesFromSettings(settings),
  );
  const draftRef = useRef(settings);
  const settingsContentKey = intakeSettingsContentKey(settings);
  const syncedSettingsKey = useRef(settingsContentKey);

  const onDraftChange = useCallback((update: SetStateAction<IntakeSourceSettings>) => {
    setDraft((current) => {
      const next = typeof update === "function" ? update(current) : update;
      draftRef.current = next;
      return next;
    });
  }, []);
  const getDraft = useCallback(() => draftRef.current, []);

  useEffect(() => {
    if (syncedSettingsKey.current === settingsContentKey) {
      return;
    }
    syncedSettingsKey.current = settingsContentKey;
    setDraft(settings);
    draftRef.current = settings;
    setLinearFilterChoices(linearFilterChoicesFromSettings(settings));
  }, [settings, settingsContentKey]);

  return { draft, onDraftChange, getDraft, linearFilterChoices, setLinearFilterChoices };
}
