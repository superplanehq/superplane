import { Checkbox } from "@/components/ui/checkbox";
import { useIntegrationResources } from "@/hooks/useIntegrations";
import { cn } from "@/lib/utils";
import { useMemo, type Dispatch, type SetStateAction } from "react";

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
  if (sourceId !== "productive-tasks") {
    return null;
  }

  return (
    <div className="flex flex-col gap-6">
      <fieldset className="min-w-0">
        <legend className="workspace-section-title">{PRODUCTIVE_INTAKE_SETTINGS_COPY.filtersLabel}</legend>
        <div className="mt-2 flex flex-col gap-2">
          <div className="flex flex-col gap-1.5">
            <IntakeSettingsSectionTitle title={PRODUCTIVE_INTAKE_SETTINGS_COPY.filterByTaskList} />
            <TaskListField
              organizationId={organizationId}
              integrationId={integrationId}
              projectId={projectId}
              taskListIds={settings.taskListIds}
              onChange={(taskListIds) => onSettingsChange((current) => ({ ...current, taskListIds }))}
            />
          </div>
        </div>
      </fieldset>
    </div>
  );
}

function IntakeSettingsSectionTitle({ title }: { title: string }) {
  return (
    <div className="rounded-lg border border-border bg-card px-3 py-2.5">
      <span className="min-w-0 text-[13px] font-medium tracking-[-0.01em] text-foreground">{title}</span>
    </div>
  );
}

function TaskListField({
  organizationId,
  integrationId,
  projectId,
  taskListIds,
  onChange,
}: {
  organizationId?: string;
  integrationId?: string;
  projectId?: string;
  taskListIds: string[];
  onChange: (taskListIds: string[]) => void;
}) {
  const enabled = Boolean(organizationId && integrationId && projectId);
  const taskListsQuery = useIntegrationResources(
    organizationId ?? "",
    integrationId ?? "",
    "task_list",
    enabled && projectId ? { project: projectId } : undefined,
    { enabled },
  );
  const options = useMemo(
    () => taskListOptions(taskListsQuery.data ?? [], taskListIds),
    [taskListsQuery.data, taskListIds],
  );
  function toggleTaskList(id: string) {
    onChange(taskListIds.includes(id) ? taskListIds.filter((entry) => entry !== id) : [...taskListIds, id]);
  }

  return (
    <div className="mb-1 ml-6 flex flex-col gap-2" data-testid="intake-task-list-options">
      <TaskListStatus loading={taskListsQuery.isLoading} error={taskListsQuery.isError} empty={options.length === 0} />
      {options.length > 0 && taskListIds.length === 0 ? (
        <p className="text-[12px] text-muted-foreground">{PRODUCTIVE_INTAKE_SETTINGS_COPY.taskListsNoneSelected}</p>
      ) : null}
      {options.length > 0 ? (
        <ul className="flex flex-wrap items-center gap-1.5">
          {options.map((option) => {
            const checked = taskListIds.includes(option.id);
            return (
              <li key={option.id}>
                <label
                  className={cn(
                    "inline-flex max-w-full cursor-pointer items-center gap-2 rounded-md border px-2 py-1 text-[13px]",
                    checked
                      ? "border-foreground/20 bg-accent/50 text-foreground"
                      : "border-border bg-card text-muted-foreground hover:border-foreground/15",
                  )}
                >
                  <Checkbox checked={checked} onChange={() => toggleTaskList(option.id)} aria-label={option.name} />
                  <span className="min-w-0 truncate">{option.name}</span>
                </label>
              </li>
            );
          })}
        </ul>
      ) : null}
    </div>
  );
}

function TaskListStatus({ loading, error, empty }: { loading: boolean; error: boolean; empty: boolean }) {
  let message: string | null = null;
  if (loading) {
    message = PRODUCTIVE_INTAKE_SETTINGS_COPY.taskListsLoading;
  } else if (error) {
    message = PRODUCTIVE_INTAKE_SETTINGS_COPY.taskListsError;
  } else if (empty) {
    message = PRODUCTIVE_INTAKE_SETTINGS_COPY.taskListsEmpty;
  }
  if (!message) {
    return null;
  }
  return <p className="text-[12px] text-muted-foreground">{message}</p>;
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
