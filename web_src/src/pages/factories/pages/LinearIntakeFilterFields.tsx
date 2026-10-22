import type { Dispatch, SetStateAction } from "react";

import type { IntakeSourceSettings } from "./intakeSourceSettingsModel";
import { LinearIntakeScopeFields } from "./LinearIntakeScopeFields";
import { linearScopeFromSettings, linearSettingsFromScope } from "./linearIntakeScope";
import type { LineIntakeSourceId } from "./lineIntakeModel";

export function LinearIntakeFilterFields({
  sourceId,
  settings,
  onSettingsChange,
  organizationId,
  integrationId,
}: {
  sourceId: LineIntakeSourceId;
  settings: IntakeSourceSettings;
  onSettingsChange: Dispatch<SetStateAction<IntakeSourceSettings>>;
  organizationId?: string;
  integrationId?: string;
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
    />
  );
}
