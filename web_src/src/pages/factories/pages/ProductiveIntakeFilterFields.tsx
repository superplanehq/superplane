import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import { useIntegrationResources } from "@/hooks/useIntegrations";
import { cn } from "@/lib/utils";
import { useId, useMemo, useState, type Dispatch, type SetStateAction } from "react";

import type { IntakeSourceSettings } from "./intakeSourceSettingsModel";
import type { LineIntakeSourceId } from "./lineIntakeModel";
import { PRODUCTIVE_INTAKE_SETTINGS_COPY } from "./productiveIntakeSettingsCopy";

interface TaskListOption {
  id: string;
  name: string;
}

export function ProductiveIntakeFilterFields({
  sourceId,
  settings,
  onSettingsChange,
  organizationId,
  integrationId,
  projectId,
}: {
  sourceId: LineIntakeSourceId;
  settings: IntakeSourceSettings;
  onSettingsChange: Dispatch<SetStateAction<IntakeSourceSettings>>;
  organizationId?: string;
  integrationId?: string;
  projectId?: string;
}) {
  const idPrefix = useId();
  if (sourceId !== "productive-tasks") {
    return null;
  }

  return (
    <div className="flex flex-col gap-6">
      <fieldset className="min-w-0">
        <legend className="workspace-section-title">{PRODUCTIVE_INTAKE_SETTINGS_COPY.filtersLabel}</legend>
        <div className="mt-2 flex flex-col gap-2">
          <div className="flex flex-col gap-1.5">
            <FilterCheckbox
              id={`${idPrefix}-exclude-key-tasks`}
              title={PRODUCTIVE_INTAKE_SETTINGS_COPY.excludeKeyTasks}
              checked={settings.excludeKeyTasks}
              onChange={() =>
                onSettingsChange((current) => ({ ...current, excludeKeyTasks: !current.excludeKeyTasks }))
              }
            />
            <p className="px-1 text-[12px] leading-4 text-muted-foreground">
              {PRODUCTIVE_INTAKE_SETTINGS_COPY.excludeKeyTasksHelper}
            </p>
          </div>
          <TaskListFilter
            idPrefix={idPrefix}
            organizationId={organizationId}
            integrationId={integrationId}
            projectId={projectId}
            taskListIds={settings.taskListIds}
            onChange={(taskListIds) => onSettingsChange((current) => ({ ...current, taskListIds }))}
          />
        </div>
      </fieldset>
    </div>
  );
}

function TaskListFilter({
  idPrefix,
  organizationId,
  integrationId,
  projectId,
  taskListIds,
  onChange,
}: {
  idPrefix: string;
  organizationId?: string;
  integrationId?: string;
  projectId?: string;
  taskListIds: string[];
  onChange: (taskListIds: string[]) => void;
}) {
  const [filterByTaskList, setFilterByTaskList] = useState(taskListIds.length > 0);
  const enabled = Boolean(organizationId && integrationId && projectId);
  const taskListsQuery = useIntegrationResources(
    organizationId ?? "",
    integrationId ?? "",
    "task_list",
    enabled && projectId ? { project: projectId } : undefined,
    { enabled: enabled && filterByTaskList },
  );
  const options = useMemo(
    () => taskListOptions(taskListsQuery.data ?? [], taskListIds),
    [taskListsQuery.data, taskListIds],
  );

  function toggleFilter() {
    const next = !filterByTaskList;
    setFilterByTaskList(next);
    if (!next) {
      onChange([]);
    }
  }

  function toggleTaskList(id: string) {
    onChange(taskListIds.includes(id) ? taskListIds.filter((entry) => entry !== id) : [...taskListIds, id]);
  }

  return (
    <div className="flex flex-col gap-1.5">
      <FilterCheckbox
        id={`${idPrefix}-filter-by-task-list`}
        title={PRODUCTIVE_INTAKE_SETTINGS_COPY.filterByTaskList}
        checked={filterByTaskList}
        onChange={() => toggleFilter()}
      />
      <p className="px-1 text-[12px] leading-4 text-muted-foreground">
        {PRODUCTIVE_INTAKE_SETTINGS_COPY.filterByTaskListHelper}
      </p>
      {filterByTaskList ? (
        <div className="mb-1 ml-6 flex flex-col gap-2" data-testid="intake-task-list-options">
          <TaskListStatus
            loading={taskListsQuery.isLoading}
            error={taskListsQuery.isError}
            empty={options.length === 0}
            noneSelected={taskListIds.length === 0}
          />
          {options.length > 0 ? (
            <ul className="flex flex-wrap items-center gap-1.5">
              {options.map((option) => (
                <li key={option.id}>
                  <TaskListChip
                    option={option}
                    checked={taskListIds.includes(option.id)}
                    onToggle={() => toggleTaskList(option.id)}
                  />
                </li>
              ))}
            </ul>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}

function TaskListStatus({
  loading,
  error,
  empty,
  noneSelected,
}: {
  loading: boolean;
  error: boolean;
  empty: boolean;
  noneSelected: boolean;
}) {
  let message: string | null = null;
  if (loading) {
    message = PRODUCTIVE_INTAKE_SETTINGS_COPY.taskListsLoading;
  } else if (error) {
    message = PRODUCTIVE_INTAKE_SETTINGS_COPY.taskListsError;
  } else if (empty) {
    message = PRODUCTIVE_INTAKE_SETTINGS_COPY.taskListsEmpty;
  } else if (noneSelected) {
    message = PRODUCTIVE_INTAKE_SETTINGS_COPY.taskListsNoneSelected;
  }
  if (!message) {
    return null;
  }
  return <p className="text-[12px] text-muted-foreground">{message}</p>;
}

function TaskListChip({
  option,
  checked,
  onToggle,
}: {
  option: TaskListOption;
  checked: boolean;
  onToggle: () => void;
}) {
  return (
    <label
      className={cn(
        "inline-flex max-w-full cursor-pointer items-center gap-2 rounded-md border px-2 py-1 text-[13px]",
        checked
          ? "border-foreground/20 bg-accent/50 text-foreground"
          : "border-border bg-card text-muted-foreground hover:border-foreground/15",
      )}
    >
      <Checkbox checked={checked} onChange={onToggle} aria-label={option.name} />
      <span className="min-w-0 truncate">{option.name}</span>
    </label>
  );
}

function FilterCheckbox({
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

/**
 * Lists the project's task lists, then any selected id the project no longer
 * returns, so the user can still clear a stale selection.
 */
function taskListOptions(resources: { id?: string; name?: string }[], selectedIds: string[]): TaskListOption[] {
  const options: TaskListOption[] = [];
  for (const resource of resources) {
    const id = resource.id?.trim() ?? "";
    if (id.length === 0 || options.some((option) => option.id === id)) {
      continue;
    }
    options.push({ id, name: resource.name?.trim() || id });
  }
  for (const id of selectedIds) {
    if (!options.some((option) => option.id === id)) {
      options.push({ id, name: id });
    }
  }
  return options;
}
