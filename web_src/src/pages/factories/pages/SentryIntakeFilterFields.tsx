import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import { cn } from "@/lib/utils";
import { useId, type Dispatch, type SetStateAction } from "react";

import { SENTRY_INTAKE_LEVELS, type IntakeSourceSettings } from "./intakeSourceSettingsModel";
import type { LineIntakeSourceId } from "./lineIntakeModel";

const SENTRY_INTAKE_SETTINGS_COPY = {
  intakeSection: "Create task when:",
  filtersLabel: "Filters",
  newIssues: "A new issue is created",
  regressedIssues: "An issue becomes unresolved",
  assignedIssues: "An issue is assigned",
  levelFatal: "Fatal",
  levelError: "Error",
  levelWarning: "Warning",
  levelInfo: "Info",
  levelDebug: "Debug",
} as const;

const SENTRY_LEVEL_COPY: Record<(typeof SENTRY_INTAKE_LEVELS)[number], string> = {
  fatal: SENTRY_INTAKE_SETTINGS_COPY.levelFatal,
  error: SENTRY_INTAKE_SETTINGS_COPY.levelError,
  warning: SENTRY_INTAKE_SETTINGS_COPY.levelWarning,
  info: SENTRY_INTAKE_SETTINGS_COPY.levelInfo,
  debug: SENTRY_INTAKE_SETTINGS_COPY.levelDebug,
};

export function SentryIntakeFilterFields({
  sourceId,
  settings,
  onSettingsChange,
}: {
  sourceId: LineIntakeSourceId;
  settings: IntakeSourceSettings;
  onSettingsChange: Dispatch<SetStateAction<IntakeSourceSettings>>;
}) {
  const idPrefix = useId();
  if (sourceId !== "sentry-exceptions") {
    return null;
  }

  function update<K extends keyof IntakeSourceSettings>(key: K, value: IntakeSourceSettings[K]) {
    onSettingsChange((current) => ({ ...current, [key]: value }));
  }

  function toggleLevel(level: (typeof SENTRY_INTAKE_LEVELS)[number]) {
    onSettingsChange((current) => {
      const selected = current.sentryLevels.includes(level)
        ? current.sentryLevels.filter((entry) => entry !== level)
        : [...current.sentryLevels, level];
      return { ...current, sentryLevels: selected };
    });
  }

  return (
    <div className="flex flex-col gap-6">
      <fieldset className="min-w-0">
        <legend className="workspace-section-title">{SENTRY_INTAKE_SETTINGS_COPY.intakeSection}</legend>
        <div className="mt-2 flex flex-col gap-2">
          <IntakeSettingsCheckbox
            id={`${idPrefix}-new-issues`}
            title={SENTRY_INTAKE_SETTINGS_COPY.newIssues}
            checked={settings.sentryNewIssues}
            onChange={() => update("sentryNewIssues", !settings.sentryNewIssues)}
          />
          <IntakeSettingsCheckbox
            id={`${idPrefix}-regressed-issues`}
            title={SENTRY_INTAKE_SETTINGS_COPY.regressedIssues}
            checked={settings.sentryRegressedIssues}
            onChange={() => update("sentryRegressedIssues", !settings.sentryRegressedIssues)}
          />
          <IntakeSettingsCheckbox
            id={`${idPrefix}-assigned-issues`}
            title={SENTRY_INTAKE_SETTINGS_COPY.assignedIssues}
            checked={settings.sentryAssignedIssues}
            onChange={() => update("sentryAssignedIssues", !settings.sentryAssignedIssues)}
          />
        </div>
      </fieldset>
      <fieldset className="min-w-0">
        <legend className="workspace-section-title">{SENTRY_INTAKE_SETTINGS_COPY.filtersLabel}</legend>
        <div className="mt-2 flex flex-col gap-2">
          {SENTRY_INTAKE_LEVELS.map((level) => (
            <IntakeSettingsCheckbox
              key={level}
              id={`${idPrefix}-level-${level}`}
              title={SENTRY_LEVEL_COPY[level]}
              checked={settings.sentryLevels.includes(level)}
              onChange={() => toggleLevel(level)}
            />
          ))}
        </div>
      </fieldset>
    </div>
  );
}

function IntakeSettingsCheckbox({
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
