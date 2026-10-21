import { ImplementSettingRow } from "./ImplementSettingRow";
import { Switch } from "@/components/ui/switch";

import {
  PLANNING_SETTINGS_COPY,
  type PlanningAutoStartLine,
  type PlanningDraftSettings,
} from "./planningSettingsModel";

type PlanningUpdate = <K extends keyof PlanningDraftSettings>(key: K, value: PlanningDraftSettings[K]) => void;

export function PlanningSettingsFields({
  draft,
  lines,
  onUpdate,
}: {
  draft: PlanningDraftSettings;
  lines: PlanningAutoStartLine[];
  onUpdate: PlanningUpdate;
}) {
  const unavailable = !draft.enabled || lines.length === 0;
  const selected = lines.find((line) => line.id === draft.autoStartLineId);
  const checked = Boolean(selected);

  return (
    <div data-testid="planning-settings-auto-start">
      <PlanningToggleRow
        title={PLANNING_SETTINGS_COPY.autoStartLabel}
        description={autoStartDescription(draft, lines)}
        checked={checked}
        disabled={unavailable}
        onCheckedChange={(next) => {
          if (!next) {
            onUpdate("autoStartLineId", "");
            return;
          }
          onUpdate("autoStartLineId", selected?.id ?? lines[0]?.id ?? "");
        }}
        testId="planning-settings-auto-start-toggle"
      />
    </div>
  );
}

function autoStartDescription(draft: PlanningDraftSettings, lines: PlanningAutoStartLine[]): string {
  if (!draft.enabled) {
    return PLANNING_SETTINGS_COPY.autoStartPlanningOffHelper;
  }
  if (lines.length === 0) {
    return PLANNING_SETTINGS_COPY.autoStartNoBoardHelper;
  }
  return PLANNING_SETTINGS_COPY.autoStartReviewHelper;
}

function PlanningToggleRow({
  title,
  description,
  checked,
  disabled = false,
  onCheckedChange,
  testId,
}: {
  title: string;
  description: string;
  checked: boolean;
  disabled?: boolean;
  onCheckedChange: (checked: boolean) => void;
  testId: string;
}) {
  return (
    <div data-testid={testId}>
      <ImplementSettingRow label={title} description={description} htmlFor={`${testId}-switch`}>
        <Switch
          id={`${testId}-switch`}
          checked={checked}
          disabled={disabled}
          onCheckedChange={onCheckedChange}
          aria-describedby={`${testId}-switch-help`}
        />
      </ImplementSettingRow>
    </div>
  );
}
