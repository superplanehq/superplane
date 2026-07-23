import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { useOrganizationId } from "@/hooks/useOrganizationId";
import { useIntegrationResources } from "@/hooks/useIntegrations";
import { cn } from "@/lib/utils";
import { useCallback, useMemo, useState } from "react";

import { IntegrationsSection, type IntegrationSelections } from "./InstallIntegrationsSection";
import { FACTORY_STARTING_TASKS, type FactoryStartingTaskId } from "./factoryStartingTasks";
import { homeInstallPanelClassName } from "./homePageStyles";

const FACTORY_INTEGRATIONS = ["github", "claude"] as const;

interface FactorySetupPanelProps {
  organizationId?: string;
  busy?: boolean;
  onCancel: () => void;
  onInstall: (selections: IntegrationSelections, repository: string, startingTaskPrompt: string) => void;
}

export function FactorySetupPanel({
  organizationId: propOrgId,
  busy = false,
  onCancel,
  onInstall,
}: FactorySetupPanelProps) {
  const routeOrgId = useOrganizationId();
  const organizationId = propOrgId || routeOrgId || "";
  const [selections, setSelections] = useState<IntegrationSelections>({});
  const [repository, setRepository] = useState("");
  const [selectedTaskId, setSelectedTaskId] = useState<FactoryStartingTaskId | null>(null);

  const githubConnected = Boolean(selections.github);
  const allConnected = FACTORY_INTEGRATIONS.every((name) => selections[name]);
  const selectedTask = FACTORY_STARTING_TASKS.find((task) => task.id === selectedTaskId) ?? null;
  const canRun = !busy && allConnected && repository.trim() !== "" && selectedTask !== null;

  const handleSelectionsChange = useCallback((next: IntegrationSelections) => {
    setSelections((prev) => {
      if (prev.github?.id !== next.github?.id) {
        setRepository("");
      }
      return next;
    });
  }, []);

  return (
    <div className={homeInstallPanelClassName} role="region" aria-label="Software Factory setup">
      <div className="mb-5">
        <h3 className="text-base font-medium text-slate-900 dark:text-gray-100">Connect your GitHub and Claude</h3>
        <p className="mt-1 text-sm text-slate-600 dark:text-gray-400">
          This will create software factory that automates your delivery from trigger to pull request.
        </p>
      </div>

      <div className="mb-5">
        <IntegrationsSection
          integrations={[...FACTORY_INTEGRATIONS]}
          organizationId={organizationId}
          selections={selections}
          onSelectionsChange={handleSelectionsChange}
          variant="status"
        />
      </div>

      {githubConnected && (
        <div className="mb-5 pt-4">
          <Label
            htmlFor="factory-repository"
            className="mb-2 block text-xs font-semibold text-slate-700 dark:text-gray-300"
          >
            Choose repository
          </Label>
          <FactoryRepositorySelect
            organizationId={organizationId}
            integrationId={selections.github?.id ?? ""}
            value={repository}
            onChange={setRepository}
          />
        </div>
      )}

      <StartingTaskSection
        busy={busy}
        selectedTaskId={selectedTaskId}
        onSelectTask={setSelectedTaskId}
        prompt={selectedTask?.prompt ?? null}
      />

      <div className="flex flex-wrap items-center gap-2.5 pt-4">
        <Button
          type="button"
          disabled={!canRun}
          onClick={() => {
            if (!selectedTask) return;
            onInstall(selections, repository.trim(), selectedTask.prompt);
          }}
        >
          Run
        </Button>
        <Button type="button" variant="outline" onClick={onCancel} disabled={busy}>
          Cancel
        </Button>
      </div>
    </div>
  );
}

function StartingTaskSection({
  busy,
  selectedTaskId,
  onSelectTask,
  prompt,
}: {
  busy: boolean;
  selectedTaskId: FactoryStartingTaskId | null;
  onSelectTask: (id: FactoryStartingTaskId) => void;
  prompt: string | null;
}) {
  return (
    <div className="mb-5">
      <p className="mb-3 text-xs font-semibold text-slate-700 dark:text-gray-300">Choose starting task</p>
      <div className="flex flex-wrap gap-2">
        {FACTORY_STARTING_TASKS.map((task) => {
          const Icon = task.icon;
          const selected = task.id === selectedTaskId;
          return (
            <Button
              key={task.id}
              type="button"
              variant="outline"
              size="sm"
              aria-pressed={selected}
              disabled={busy}
              onClick={() => onSelectTask(task.id)}
              className={cn(
                "h-auto rounded-md px-3 py-2 text-xs font-normal",
                selected &&
                  "border-primary/50 bg-primary/5 text-slate-900 dark:border-primary/40 dark:bg-primary/10 dark:text-gray-100",
              )}
            >
              <Icon className={cn("size-3.5 shrink-0", task.iconClassName)} />
              {task.label}
            </Button>
          );
        })}
      </div>

      {prompt && (
        <div className="mt-4">
          <Label
            htmlFor="factory-starting-task-prompt"
            className="mb-2 block text-xs font-semibold text-slate-700 dark:text-gray-300"
          >
            Prompt
          </Label>
          <Textarea
            id="factory-starting-task-prompt"
            readOnly
            value={prompt}
            rows={5}
            className="min-h-28 resize-none text-xs leading-relaxed text-slate-700 dark:text-gray-300"
          />
        </div>
      )}
    </div>
  );
}

function FactoryRepositorySelect({
  organizationId,
  integrationId,
  value,
  onChange,
}: {
  organizationId: string;
  integrationId: string;
  value: string;
  onChange: (value: string) => void;
}) {
  const {
    data: resources = [],
    isLoading,
    isError,
    refetch,
  } = useIntegrationResources(organizationId, integrationId, "repository");
  const options = useMemo(
    () =>
      resources
        .map((resource) => {
          const name = resource.name?.trim();
          if (!name) return null;
          return { value: name, label: name };
        })
        .filter((option): option is { value: string; label: string } => option !== null),
    [resources],
  );

  if (isError) {
    return (
      <div className="space-y-2">
        <p className="text-xs text-red-600 dark:text-red-400">Couldn't load repositories. Try again.</p>
        <Button type="button" variant="outline" size="sm" onClick={() => void refetch()}>
          Retry
        </Button>
      </div>
    );
  }

  if (!isLoading && options.length === 0) {
    return (
      <p className="text-xs text-slate-500 dark:text-gray-400">
        No repositories found for this GitHub connection. Check access, then try again.
      </p>
    );
  }

  return (
    <Select value={value || undefined} onValueChange={onChange} disabled={isLoading || options.length === 0}>
      <SelectTrigger id="factory-repository" className="w-full">
        <SelectValue placeholder={isLoading ? "Loading repositories…" : "Select a repository"} />
      </SelectTrigger>
      <SelectContent>
        {options.map((option) => (
          <SelectItem key={option.value} value={option.value}>
            {option.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
