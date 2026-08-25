import { Checkbox } from "@/components/ui/checkbox";
import { cn } from "@/lib/utils";
import type { Dispatch, SetStateAction } from "react";

import { IntakeSettingsRadioOption } from "./IntakeSettingsRadioOption";
import {
  GITHUB_INTAKE_LABEL_OPTIONS,
  INTAKE_SETTINGS_COPY,
  toggleIntakeLabel,
  type IntakeAssignmentFilter,
  type IntakeLabelFilterMode,
  type IntakeSourceSettings,
} from "./intakeSourceSettingsModel";
import type { LineIntakeSourceId } from "./lineIntakeModel";

export function GitHubIntakeFilterFields({
  sourceId,
  settings,
  onSettingsChange,
}: {
  sourceId: LineIntakeSourceId;
  settings: IntakeSourceSettings;
  onSettingsChange: Dispatch<SetStateAction<IntakeSourceSettings>>;
}) {
  if (sourceId !== "github-issues") {
    return null;
  }

  function update<K extends keyof IntakeSourceSettings>(key: K, value: IntakeSourceSettings[K]) {
    onSettingsChange((current) => ({ ...current, [key]: value }));
  }

  return (
    <section className="flex flex-col gap-6">
      <h3 className="workspace-section-title">{INTAKE_SETTINGS_COPY.filtersLabel}</h3>
      <fieldset className="min-w-0">
        <legend className="text-sm font-medium text-gray-800 dark:text-gray-100">
          {INTAKE_SETTINGS_COPY.labelsLabel}
        </legend>
        <p className="workspace-body-text mt-1 text-muted-foreground">{INTAKE_SETTINGS_COPY.labelsHelper}</p>
        <div className="mt-2 flex flex-col gap-2">
          <IntakeSettingsRadioOption
            name="intake-label-filter"
            value="include"
            checked={settings.labelFilterMode === "include"}
            title={INTAKE_SETTINGS_COPY.includeLabels}
            onChange={() => update("labelFilterMode", "include" satisfies IntakeLabelFilterMode)}
          />
          <IntakeSettingsRadioOption
            name="intake-label-filter"
            value="exclude"
            checked={settings.labelFilterMode === "exclude"}
            title={INTAKE_SETTINGS_COPY.excludeLabels}
            onChange={() => update("labelFilterMode", "exclude" satisfies IntakeLabelFilterMode)}
          />
        </div>
        <ul className="mt-3 flex flex-wrap gap-2" data-testid="intake-label-options">
          {GITHUB_INTAKE_LABEL_OPTIONS.map((label) => {
            const checked = settings.labels.includes(label);
            return (
              <li key={label}>
                <label
                  className={cn(
                    "inline-flex cursor-pointer items-center gap-2 rounded-md border px-2.5 py-1.5 text-[13px]",
                    checked
                      ? "border-foreground/20 bg-accent/50 text-foreground"
                      : "border-border bg-card text-muted-foreground hover:border-foreground/15",
                  )}
                >
                  <Checkbox
                    checked={checked}
                    onChange={() => update("labels", toggleIntakeLabel(settings.labels, label))}
                    aria-label={label}
                  />
                  {label}
                </label>
              </li>
            );
          })}
        </ul>
      </fieldset>
      <fieldset className="min-w-0">
        <legend className="text-sm font-medium text-gray-800 dark:text-gray-100">
          {INTAKE_SETTINGS_COPY.assignmentLabel}
        </legend>
        <div className="mt-2 flex flex-col gap-2">
          {(["any", "assigned", "unassigned"] as const).map((assignment) => (
            <IntakeSettingsRadioOption
              key={assignment}
              name="intake-assignment"
              value={assignment}
              checked={settings.assignment === assignment}
              title={assignmentLabel(assignment)}
              onChange={() => update("assignment", assignment)}
            />
          ))}
        </div>
      </fieldset>
    </section>
  );
}

function assignmentLabel(assignment: IntakeAssignmentFilter): string {
  if (assignment === "assigned") {
    return INTAKE_SETTINGS_COPY.assignmentAssigned;
  }
  if (assignment === "unassigned") {
    return INTAKE_SETTINGS_COPY.assignmentUnassigned;
  }
  return INTAKE_SETTINGS_COPY.assignmentAny;
}
