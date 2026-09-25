import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import { cn } from "@/lib/utils";
import { useId, type Dispatch, type SetStateAction } from "react";

import type { IntakeSourceSettings } from "./intakeSourceSettingsModel";
import type { LineIntakeSourceId } from "./lineIntakeModel";

const SENTRY_INTAKE_SETTINGS_COPY = {
  intakeSection: "Create task when:",
  newIssues: "A new issue is created",
} as const;

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
