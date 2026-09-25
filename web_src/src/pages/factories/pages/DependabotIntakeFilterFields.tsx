import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import { cn } from "@/lib/utils";
import { useId, type Dispatch, type SetStateAction } from "react";

import {
  DEPENDABOT_INTAKE_SEVERITIES,
  DEPENDABOT_INTAKE_SETTINGS_COPY,
  dependabotSeveritySelected,
  toggleDependabotSeverity,
  type IntakeSourceSettings,
} from "./intakeSourceSettingsModel";
import type { LineIntakeSourceId } from "./lineIntakeModel";

const SEVERITY_LABELS: Record<(typeof DEPENDABOT_INTAKE_SEVERITIES)[number], string> = {
  critical: DEPENDABOT_INTAKE_SETTINGS_COPY.critical,
  high: DEPENDABOT_INTAKE_SETTINGS_COPY.high,
  medium: DEPENDABOT_INTAKE_SETTINGS_COPY.medium,
  low: DEPENDABOT_INTAKE_SETTINGS_COPY.low,
};

export function DependabotIntakeFilterFields({
  sourceId,
  settings,
  onSettingsChange,
}: {
  sourceId: LineIntakeSourceId;
  settings: IntakeSourceSettings;
  onSettingsChange: Dispatch<SetStateAction<IntakeSourceSettings>>;
}) {
  const idPrefix = useId();
  if (sourceId !== "dependabot-alerts") {
    return null;
  }

  return (
    <fieldset className="min-w-0">
      <legend className="workspace-section-title">{DEPENDABOT_INTAKE_SETTINGS_COPY.severitySection}</legend>
      <div className="mt-2 flex flex-col gap-2" data-testid="dependabot-severity-options">
        {DEPENDABOT_INTAKE_SEVERITIES.map((severity) => (
          <SeverityCheckbox
            key={severity}
            id={`${idPrefix}-${severity}`}
            title={SEVERITY_LABELS[severity]}
            checked={dependabotSeveritySelected(settings.dependabotSeverities, severity)}
            onChange={() =>
              onSettingsChange((current) => ({
                ...current,
                dependabotSeverities: toggleDependabotSeverity(current.dependabotSeverities, severity),
              }))
            }
          />
        ))}
      </div>
      <p className="mt-2 text-[13px] leading-5 text-muted-foreground">{DEPENDABOT_INTAKE_SETTINGS_COPY.severityHelp}</p>
    </fieldset>
  );
}

function SeverityCheckbox({
  id,
  title,
  checked,
  onChange,
}: {
  id: string;
  title: string;
  checked: boolean;
  onChange: () => void;
}) {
  return (
    <div
      className={cn(
        "flex items-center gap-3 rounded-lg border px-3 py-2.5 transition-colors",
        checked ? "border-foreground/20 bg-accent/50" : "border-border bg-card hover:border-foreground/15",
      )}
    >
      <Checkbox id={id} checked={checked} onChange={onChange} />
      <Label htmlFor={id} className="min-w-0 cursor-pointer text-[13px] font-medium tracking-[-0.01em] text-foreground">
        {title}
      </Label>
    </div>
  );
}
