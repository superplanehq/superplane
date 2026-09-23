import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import { cn } from "@/lib/utils";
import { useId, type Dispatch, type SetStateAction } from "react";

import { intakeSettingsSectionDomId, type IntakeSourceSettings } from "./intakeSourceSettingsModel";
import type { LineIntakeSourceId } from "./lineIntakeModel";

const PRODUCTIVE_INTAKE_SETTINGS_COPY = {
  filtersLabel: "Filters",
  excludeKeyTasks: "Ignore key tasks",
  excludeKeyTasksHelper: "SuperPlane skips Productive.io key tasks (milestones).",
} as const;

export function ProductiveIntakeFilterFields({
  sourceId,
  settings,
  onSettingsChange,
}: {
  sourceId: LineIntakeSourceId;
  settings: IntakeSourceSettings;
  onSettingsChange: Dispatch<SetStateAction<IntakeSourceSettings>>;
}) {
  const idPrefix = useId();
  if (sourceId !== "productive-tasks") {
    return null;
  }

  return (
    <div className="flex flex-col gap-6">
      <fieldset id={intakeSettingsSectionDomId("filters")} className="scroll-mt-6 min-w-0">
        <legend className="workspace-section-title">{PRODUCTIVE_INTAKE_SETTINGS_COPY.filtersLabel}</legend>
        <div className="mt-2 flex flex-col gap-1.5">
          <div
            className={cn(
              "flex items-center gap-3 rounded-lg border px-3 py-2.5 transition-colors",
              settings.excludeKeyTasks
                ? "border-foreground/20 bg-accent/50"
                : "border-border bg-card hover:border-foreground/15",
            )}
          >
            <Checkbox
              id={`${idPrefix}-exclude-key-tasks`}
              checked={settings.excludeKeyTasks}
              onChange={() =>
                onSettingsChange((current) => ({ ...current, excludeKeyTasks: !current.excludeKeyTasks }))
              }
            />
            <Label
              htmlFor={`${idPrefix}-exclude-key-tasks`}
              className="min-w-0 cursor-pointer text-[13px] font-medium tracking-[-0.01em] text-foreground"
            >
              {PRODUCTIVE_INTAKE_SETTINGS_COPY.excludeKeyTasks}
            </Label>
          </div>
          <p className="px-1 text-[12px] leading-4 text-muted-foreground">
            {PRODUCTIVE_INTAKE_SETTINGS_COPY.excludeKeyTasksHelper}
          </p>
        </div>
      </fieldset>
    </div>
  );
}
