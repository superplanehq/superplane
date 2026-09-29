import { Badge } from "@/components/ui/badge";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { cn } from "@/lib/utils";

import {
  PLANNING_SETTINGS_COPY,
  planningAutoStartHelper,
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
  return (
    <>
      <PlanningToggleRow
        title={PLANNING_SETTINGS_COPY.planningLabel}
        description={PLANNING_SETTINGS_COPY.planningHelper}
        checked={draft.enabled}
        onCheckedChange={(enabled) => onUpdate("enabled", enabled)}
        testId="planning-settings-enabled"
      />
      <PlanningChecksSection draft={draft} onUpdate={onUpdate} />
      <AutoStartSection draft={draft} lines={lines} onUpdate={onUpdate} />
    </>
  );
}

function PlanningChecksSection({ draft, onUpdate }: { draft: PlanningDraftSettings; onUpdate: PlanningUpdate }) {
  const disabled = !draft.enabled;
  return (
    <section className="space-y-3" data-testid="planning-settings-checks" aria-disabled={disabled}>
      <div>
        <h3 className="text-sm font-medium text-gray-800 dark:text-gray-100">{PLANNING_SETTINGS_COPY.checksLabel}</h3>
        <p className="workspace-body-text mt-1 text-muted-foreground">
          {disabled ? PLANNING_SETTINGS_COPY.checksOffHelper : PLANNING_SETTINGS_COPY.checksHelper}
        </p>
      </div>
      <div
        className={cn(
          "flex flex-col divide-y divide-border rounded-lg border border-border bg-card px-3 transition-opacity",
          disabled && "opacity-60",
        )}
      >
        <PlanningToggleRow
          title={PLANNING_SETTINGS_COPY.confidenceLabel}
          description={PLANNING_SETTINGS_COPY.confidenceHelper}
          checked={draft.confidence}
          disabled={disabled}
          onCheckedChange={(confidence) => onUpdate("confidence", confidence)}
          testId="planning-settings-confidence"
        />
        <PlanningToggleRow
          title={PLANNING_SETTINGS_COPY.clarityLabel}
          description={PLANNING_SETTINGS_COPY.clarityHelper}
          checked={draft.clarity}
          disabled={disabled}
          onCheckedChange={(clarity) => onUpdate("clarity", clarity)}
          testId="planning-settings-clarity"
        />
      </div>
    </section>
  );
}

export function PlanningHealthSection({ enabled }: { enabled: boolean }) {
  return (
    <section data-testid="planning-settings-health">
      <div className="flex items-center justify-between gap-3">
        <h3 className="text-sm font-medium text-gray-800 dark:text-gray-100">{PLANNING_SETTINGS_COPY.healthLabel}</h3>
        <Badge variant="outline">
          {enabled ? PLANNING_SETTINGS_COPY.healthReady : PLANNING_SETTINGS_COPY.healthDisabled}
        </Badge>
      </div>
      <p className="workspace-body-text mt-1 text-muted-foreground">
        {enabled ? PLANNING_SETTINGS_COPY.healthReadyHelper : PLANNING_SETTINGS_COPY.healthDisabledHelper}
      </p>
    </section>
  );
}

function AutoStartSection({
  draft,
  lines,
  onUpdate,
}: {
  draft: PlanningDraftSettings;
  lines: PlanningAutoStartLine[];
  onUpdate: PlanningUpdate;
}) {
  const unavailable = !draft.enabled || !draft.confidence || lines.length === 0;
  const selected = lines.find((line) => line.id === draft.autoStartLineId);
  const checked = Boolean(selected);
  const named = selected ?? (lines.length === 1 ? lines[0] : undefined);
  const showLineSelect = lines.length > 1 && checked;

  return (
    <div data-testid="planning-settings-auto-start">
      <PlanningToggleRow
        title={PLANNING_SETTINGS_COPY.autoStartLabel}
        description={autoStartDescription(draft, lines, named)}
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
      {showLineSelect ? (
        <div className="max-w-xs pb-1">
          <Label htmlFor="planning-settings-auto-start-line" className="sr-only">
            {PLANNING_SETTINGS_COPY.autoStartStartOnLabel}
          </Label>
          <Select
            value={draft.autoStartLineId}
            disabled={unavailable}
            onValueChange={(lineId) => onUpdate("autoStartLineId", lineId)}
          >
            <SelectTrigger
              id="planning-settings-auto-start-line"
              className="h-8"
              aria-label={PLANNING_SETTINGS_COPY.autoStartStartOnLabel}
              data-testid="planning-settings-auto-start-line"
            >
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {lines.map((line) => (
                <SelectItem key={line.id} value={line.id}>
                  {line.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      ) : null}
    </div>
  );
}

function autoStartDescription(
  draft: PlanningDraftSettings,
  lines: PlanningAutoStartLine[],
  named: PlanningAutoStartLine | undefined,
): string {
  if (!draft.enabled) {
    return PLANNING_SETTINGS_COPY.autoStartPlanningOffHelper;
  }
  if (!draft.confidence) {
    return PLANNING_SETTINGS_COPY.autoStartConfidenceOffHelper;
  }
  if (lines.length === 0) {
    return PLANNING_SETTINGS_COPY.autoStartNoBoardHelper;
  }
  if (!named) {
    return PLANNING_SETTINGS_COPY.autoStartHelper;
  }
  return planningAutoStartHelper(named.name);
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
        <p className="text-[12px] leading-5 text-muted-foreground">{description}</p>
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
