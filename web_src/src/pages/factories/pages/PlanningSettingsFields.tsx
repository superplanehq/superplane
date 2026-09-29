import { Badge } from "@/components/ui/badge";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { cn } from "@/lib/utils";

import { PLANNING_SETTINGS_COPY, type PlanningDraftSettings } from "./planningSettingsModel";

type PlanningUpdate = <K extends keyof PlanningDraftSettings>(key: K, value: PlanningDraftSettings[K]) => void;

export type PlanningAutoStartLine = {
  name: string;
};

export function PlanningSettingsFields({
  draft,
  lines = [],
  onUpdate,
}: {
  draft: PlanningDraftSettings;
  lines?: PlanningAutoStartLine[];
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
      <PlanningAutoStartSection draft={draft} lines={lines} onUpdate={onUpdate} />
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

function PlanningAutoStartSection({
  draft,
  lines,
  onUpdate,
}: {
  draft: PlanningDraftSettings;
  lines: PlanningAutoStartLine[];
  onUpdate: PlanningUpdate;
}) {
  const disabled = !draft.enabled || !draft.confidence;
  const lineNames = lines.map((line) => line.name).filter((name) => name !== "");
  const selectedLine = draft.autoStartLine.trim();
  const options = selectedLine !== "" && !lineNames.includes(selectedLine) ? [selectedLine, ...lineNames] : lineNames;

  return (
    <section className="space-y-3" data-testid="planning-settings-auto-start" aria-disabled={disabled}>
      <div>
        <h3 className="text-sm font-medium text-gray-800 dark:text-gray-100">
          {PLANNING_SETTINGS_COPY.autoStartLabel}
        </h3>
        <p className="workspace-body-text mt-1 text-muted-foreground">{PLANNING_SETTINGS_COPY.autoStartHelper}</p>
      </div>
      <div
        className={cn(
          "flex flex-col divide-y divide-border rounded-lg border border-border bg-card px-3 transition-opacity",
          disabled && "opacity-60",
        )}
      >
        <PlanningToggleRow
          title={PLANNING_SETTINGS_COPY.autoStartSwitchLabel}
          checked={draft.autoStart}
          disabled={disabled}
          onCheckedChange={(autoStart) => onUpdate("autoStart", autoStart)}
          testId="planning-settings-auto-start-switch"
        />
        <div className="space-y-1.5 py-3">
          <Label htmlFor="planning-auto-start-line" className="text-[13px] font-medium text-foreground">
            {PLANNING_SETTINGS_COPY.autoStartLineLabel}
          </Label>
          <Select
            value={selectedLine || undefined}
            onValueChange={(line) => onUpdate("autoStartLine", line)}
            disabled={disabled}
          >
            <SelectTrigger
              id="planning-auto-start-line"
              className="w-full"
              disabled={disabled}
              data-testid="planning-settings-auto-start-line"
            >
              <SelectValue placeholder={PLANNING_SETTINGS_COPY.autoStartLinePlaceholder} />
            </SelectTrigger>
            <SelectContent position="popper">
              {options.map((name) => (
                <SelectItem key={name} value={name}>
                  {name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
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

function PlanningToggleRow({
  title,
  description,
  checked,
  disabled = false,
  onCheckedChange,
  testId,
}: {
  title: string;
  description?: string;
  checked: boolean;
  disabled?: boolean;
  onCheckedChange: (checked: boolean) => void;
  testId: string;
}) {
  return (
    <div className="flex items-start justify-between gap-6 py-3" data-testid={testId}>
      <div className="min-w-0 space-y-0.5">
        <p className="text-[13px] font-medium text-foreground">{title}</p>
        {description ? <p className="text-[12px] leading-5 text-muted-foreground">{description}</p> : null}
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
