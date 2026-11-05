import type { Dispatch, SetStateAction } from "react";

import type { IntakeSourceSettings } from "./intakeSourceSettingsModel";
import { LinearIntakeScopeFields } from "./LinearIntakeScopeFields";
import { linearScopeFromSettings, linearSettingsFromScope, type LinearFilterChoices } from "./linearIntakeScope";
import type { LineIntakeSourceId } from "./lineIntakeModel";

export function LinearIntakeFilterFields({
  sourceId,
  settings,
  onSettingsChange,
  organizationId,
  integrationId,
  filterChoices,
  onFilterChoicesChange,
}: {
  sourceId: LineIntakeSourceId;
  settings: IntakeSourceSettings;
  onSettingsChange: Dispatch<SetStateAction<IntakeSourceSettings>>;
  organizationId?: string;
  integrationId?: string;
  filterChoices: LinearFilterChoices;
  onFilterChoicesChange: (choices: LinearFilterChoices) => void;
}) {
  if (sourceId !== "linear-issues") {
    return null;
  }

  return (
    <LinearIntakeScopeFields
      organizationId={organizationId ?? ""}
      integrationId={integrationId ?? ""}
      scope={linearScopeFromSettings(settings)}
      onChange={(scope) => onSettingsChange((current) => ({ ...current, ...linearSettingsFromScope(scope) }))}
      settingsFilters={{ choices: filterChoices, onChoicesChange: onFilterChoicesChange }}
    />
  );
}
