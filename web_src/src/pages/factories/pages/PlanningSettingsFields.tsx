import { Switch } from "@/ui/switch";

import { PLANNING_SETTINGS_COPY, type PlanningDraftSettings } from "./planningSettingsModel";

export function PlanningSettingsFields({
  draft,
  onUpdate,
}: {
  draft: PlanningDraftSettings;
  onUpdate: <K extends keyof PlanningDraftSettings>(key: K, value: PlanningDraftSettings[K]) => void;
}) {
  return (
    <div className="flex flex-col divide-y divide-border">
      <PlanningToggleRow
        title={PLANNING_SETTINGS_COPY.planningLabel}
        description={PLANNING_SETTINGS_COPY.planningHelper}
        checked={draft.enabled}
        onCheckedChange={(enabled) => onUpdate("enabled", enabled)}
        testId="planning-settings-enabled"
      />
      <PlanningToggleRow
        title={PLANNING_SETTINGS_COPY.clarityLabel}
        description={PLANNING_SETTINGS_COPY.clarityHelper}
        checked={draft.clarity}
        disabled={!draft.enabled}
        onCheckedChange={(clarity) => onUpdate("clarity", clarity)}
        testId="planning-settings-clarity"
      />
      <PlanningToggleRow
        title={PLANNING_SETTINGS_COPY.confidenceLabel}
        description={PLANNING_SETTINGS_COPY.confidenceHelper}
        checked={draft.confidence}
        disabled={!draft.enabled}
        onCheckedChange={(confidence) => onUpdate("confidence", confidence)}
        testId="planning-settings-confidence"
      />
    </div>
  );
}

export function PlanningHealthSection({ enabled }: { enabled: boolean }) {
  return (
    <section className="space-y-1" data-testid="planning-settings-health">
      <p className="text-[13px] font-medium text-foreground">
        {enabled ? PLANNING_SETTINGS_COPY.healthReady : PLANNING_SETTINGS_COPY.healthDisabled}
      </p>
      <p className="text-[12px] text-muted-foreground">
        {enabled ? PLANNING_SETTINGS_COPY.healthReadyHelper : PLANNING_SETTINGS_COPY.healthDisabledHelper}
      </p>
    </section>
  );
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
    <div className="flex items-start justify-between gap-6 py-3" data-testid={testId}>
      <div className="min-w-0 space-y-0.5">
        <p className="text-[13px] font-medium text-foreground">{title}</p>
        <p className="text-[12px] text-muted-foreground">{description}</p>
      </div>
      <Switch
        checked={checked}
        disabled={disabled}
        onCheckedChange={onCheckedChange}
        aria-label={title}
        className="mt-0.5"
      />
    </div>
  );
}
