import { MultiCombobox, MultiComboboxLabel } from "@/components/MultiCombobox/multi-combobox";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import { useId, type Dispatch, type SetStateAction } from "react";

import { IntakeSettingsRadioOption } from "./IntakeSettingsRadioOption";
import type { IntakeSourceSettings } from "./intakeSourceSettingsModel";
import type { LineIntakeSourceId } from "./lineIntakeModel";

/** The GitHub label SuperPlane uses as the primary intake label. */
export const GITHUB_INTAKE_MAIN_LABEL = "superplane";

export function GitHubIntakeFilterFields({
  sourceId,
  settings,
  onSettingsChange,
  labelOptions = [],
  labelOptionsLoading = false,
  disabled = false,
}: {
  sourceId: LineIntakeSourceId;
  settings: IntakeSourceSettings;
  onSettingsChange: Dispatch<SetStateAction<IntakeSourceSettings>>;
  /** Labels that exist in the connected repository. */
  labelOptions?: string[];
  labelOptionsLoading?: boolean;
  disabled?: boolean;
}) {
  const id = useId();
  if (sourceId !== "github-issues") return null;

  return (
    <fieldset className="min-w-0">
      <legend className="workspace-section-title">Which issues to include</legend>
      <div className="mt-4 flex flex-col gap-5">
        <GitHubIntakeLabels
          settings={settings}
          onSettingsChange={onSettingsChange}
          options={labelOptions}
          loading={labelOptionsLoading}
          disabled={disabled}
        />
        <div className="flex items-start gap-3">
          <Checkbox
            id={`${id}-authors`}
            className="mt-0.5"
            checked={settings.authorsWithAccess}
            aria-describedby={`${id}-authors-help`}
            onChange={(event) => {
              const authorsWithAccess = event.target.checked;
              onSettingsChange((current) => ({ ...current, authorsWithAccess }));
            }}
          />
          <div className="flex flex-col gap-1.5">
            <Label htmlFor={`${id}-authors`} className="text-sm font-normal leading-5">
              Only authors with repository access
            </Label>
            <p id={`${id}-authors-help`} className="text-xs text-muted-foreground">
              When off, issues from anyone can create tasks.
            </p>
          </div>
        </div>
        <div className="flex items-center gap-3">
          <Checkbox
            id={`${id}-reopened`}
            checked={settings.reopenedIssues}
            onChange={(event) => {
              const reopenedIssues = event.target.checked;
              onSettingsChange((current) => ({ ...current, reopenedIssues }));
            }}
          />
          <Label htmlFor={`${id}-reopened`} className="text-sm font-normal">
            Include reopened issues
          </Label>
        </div>
      </div>
    </fieldset>
  );
}

function GitHubIntakeLabels({
  settings,
  onSettingsChange,
  options,
  loading,
  disabled,
}: {
  settings: IntakeSourceSettings;
  onSettingsChange: Dispatch<SetStateAction<IntakeSourceSettings>>;
  options: string[];
  loading: boolean;
  disabled: boolean;
}) {
  const id = useId();
  const filterByLabel = settings.filterByLabel || settings.labels.length > 0;
  const choices = Array.from(new Set([GITHUB_INTAKE_MAIN_LABEL, ...options, ...settings.labels]));
  const selected = choices.filter((label) => settings.labels.includes(label));

  function setFilterByLabel(enabled: boolean) {
    onSettingsChange((current) => ({
      ...current,
      filterByLabel: enabled,
      labels: enabled ? current.labels : [],
      labelFilterMode: "include",
    }));
  }

  return (
    <fieldset className="min-w-0">
      <legend className="mb-3 text-sm font-medium">Labels</legend>
      <div className="flex flex-col gap-3">
        <IntakeSettingsRadioOption
          variant="plain"
          name={`${id}-labels`}
          value="all"
          title="All issues"
          checked={!filterByLabel}
          onChange={() => setFilterByLabel(false)}
        />
        <IntakeSettingsRadioOption
          variant="plain"
          name={`${id}-labels`}
          value="selected"
          title="Only issues with selected labels"
          checked={filterByLabel}
          onChange={() => setFilterByLabel(true)}
        />
        {filterByLabel ? (
          <div className="ml-7 flex min-w-0 flex-col gap-2">
            <MultiCombobox
              disabled={disabled}
              aria-label="Search or add a label"
              placeholder="Search or add a label"
              options={choices.map((label) => ({ id: label }))}
              value={selected.map((label) => ({ id: label }))}
              displayValue={(option) => option.id}
              filter={(option, query) => option.id.toLowerCase().includes(query.trim().toLowerCase())}
              allowCustomValues
              createCustomValue={(query) => ({ id: query.trim() })}
              validateInput={(query) =>
                query.trim().length > 0 && !choices.some((label) => label.toLowerCase() === query.trim().toLowerCase())
              }
              onChange={(options) =>
                onSettingsChange((current) => ({
                  ...current,
                  labels: options.map((option) => option.id),
                  filterByLabel: true,
                  labelFilterMode: "include",
                }))
              }
            >
              {(option, isSelected) => (
                <MultiComboboxLabel>
                  {isSelected || choices.includes(option.id) ? option.id : `Add label "${option.id}"`}
                </MultiComboboxLabel>
              )}
            </MultiCombobox>
            {loading ? (
              <p role="status" className="text-xs text-muted-foreground">
                Loading repository labels…
              </p>
            ) : null}
            {settings.labels.length === 0 ? (
              <p className="text-xs leading-5 text-muted-foreground">
                Select at least one label, or choose All issues.
              </p>
            ) : (
              <p className="text-xs leading-5 text-muted-foreground">
                Match any selected label. Adding a selected label to an open issue also creates a task.
              </p>
            )}
          </div>
        ) : null}
      </div>
    </fieldset>
  );
}
