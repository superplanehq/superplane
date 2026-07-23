import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { useOrganizationId } from "@/hooks/useOrganizationId";
import { cn } from "@/lib/utils";
import { IntegrationResourceFieldRenderer } from "@/ui/configurationFieldRenderer/IntegrationResourceFieldRenderer";
import { SecretPickerFieldRenderer } from "@/ui/configurationFieldRenderer/SecretPickerFieldRenderer";
import { Bug, FilePenLine, TestTube2, type LucideIcon } from "lucide-react";
import { useCallback, useMemo, useState } from "react";

import { getFactoryDefinition, type FactoryDefinition, type FactoryStartingTask } from "./factories";
import { IntegrationsSection, type IntegrationSelections } from "./InstallIntegrationsSection";
import { homeInstallPanelClassName } from "./homePageStyles";
import type { InstallParam } from "../install/types";

const STARTING_TASK_ICONS: Record<string, { icon: LucideIcon; iconClassName: string }> = {
  "unit-test": { icon: TestTube2, iconClassName: "text-amber-500" },
  "fix-bug": { icon: Bug, iconClassName: "text-red-500" },
  "improve-agents-md": { icon: FilePenLine, iconClassName: "text-violet-500" },
};

function normalizeResourceValue(val: string | string[] | undefined): string {
  if (typeof val === "string") return val;
  if (Array.isArray(val)) return val[0] ?? "";
  return "";
}

function connectHeading(integrations: string[]): string {
  if (integrations.length === 0) return "Configure your factory";
  if (integrations.length === 1) return `Connect your ${formatIntegrationLabel(integrations[0]!)}`;
  if (integrations.length === 2) {
    return `Connect your ${formatIntegrationLabel(integrations[0]!)} and ${formatIntegrationLabel(integrations[1]!)}`;
  }
  return "Connect your integrations";
}

const INTEGRATION_LABELS: Record<string, string> = {
  github: "GitHub",
  claude: "Claude",
  gitlab: "GitLab",
  slack: "Slack",
};

function formatIntegrationLabel(name: string): string {
  if (!name) return name;
  return INTEGRATION_LABELS[name] ?? name.charAt(0).toUpperCase() + name.slice(1);
}

export interface FactorySetupResult {
  integrations: IntegrationSelections;
  installParams: Record<string, string>;
  startingTaskPrompt: string;
}

interface FactorySetupPanelProps {
  factory?: FactoryDefinition;
  organizationId?: string;
  busy?: boolean;
  onCancel: () => void;
  onInstall: (result: FactorySetupResult) => void;
}

export function FactorySetupPanel({
  factory: factoryProp,
  organizationId: propOrgId,
  busy = false,
  onCancel,
  onInstall,
}: FactorySetupPanelProps) {
  const factory = factoryProp ?? getFactoryDefinition();
  const routeOrgId = useOrganizationId();
  const organizationId = propOrgId || routeOrgId || "";
  const [selections, setSelections] = useState<IntegrationSelections>({});
  const [paramValues, setParamValues] = useState<Record<string, string>>({});
  const [selectedTaskId, setSelectedTaskId] = useState<string | null>(null);

  const selectedTask = factory.startingTasks.find((task) => task.id === selectedTaskId) ?? null;
  // Allow Run without connections, params, or a starting task — template wires what is available.
  const canRun = !busy;

  const visibleParams = useMemo(
    () =>
      factory.installParams.filter((param) => {
        if (param.type !== "integration-resource" || !param.integration) return true;
        return Boolean(selections[param.integration]);
      }),
    [factory.installParams, selections],
  );

  const handleSelectionsChange = useCallback(
    (next: IntegrationSelections) => {
      setSelections((prev) => {
        for (const param of factory.installParams) {
          if (param.type !== "integration-resource" || !param.integration) continue;
          if (prev[param.integration]?.id !== next[param.integration]?.id) {
            setParamValues((values) => ({ ...values, [param.name]: "" }));
          }
        }
        return next;
      });
    },
    [factory.installParams],
  );

  return (
    <div className={homeInstallPanelClassName} role="region" aria-label={`${factory.title} setup`}>
      <div className="mb-5">
        <h3 className="text-base font-medium text-slate-900 dark:text-gray-100">
          {connectHeading(factory.integrations)}
        </h3>
        <p className="mt-1 text-sm text-slate-600 dark:text-gray-400">{factory.description}</p>
      </div>

      {factory.integrations.length > 0 && (
        <div className="mb-5">
          <IntegrationsSection
            integrations={factory.integrations}
            organizationId={organizationId}
            selections={selections}
            onSelectionsChange={handleSelectionsChange}
            variant="status"
          />
        </div>
      )}

      {visibleParams.length > 0 && (
        <FactoryParamsSection
          params={visibleParams}
          values={paramValues}
          onChange={setParamValues}
          organizationId={organizationId}
          integrationSelections={selections}
        />
      )}

      <StartingTaskSection
        busy={busy}
        tasks={factory.startingTasks}
        selectedTaskId={selectedTaskId}
        onSelectTask={setSelectedTaskId}
        prompt={selectedTask?.prompt ?? null}
      />

      <div className="flex flex-wrap items-center gap-2.5 pt-4">
        <Button
          type="button"
          disabled={!canRun}
          onClick={() => {
            onInstall({
              integrations: selections,
              installParams: paramValues,
              startingTaskPrompt: selectedTask?.prompt ?? "",
            });
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

function FactoryParamsSection({
  params,
  values,
  onChange,
  organizationId,
  integrationSelections,
}: {
  params: InstallParam[];
  values: Record<string, string>;
  onChange: React.Dispatch<React.SetStateAction<Record<string, string>>>;
  organizationId: string;
  integrationSelections: IntegrationSelections;
}) {
  return (
    <div className="mb-5 space-y-3 pt-4">
      {params.map((param) => (
        <div key={param.name} className="space-y-1">
          <Label
            htmlFor={`factory-param-${param.name}`}
            className="mb-2 block text-xs font-semibold text-slate-700 dark:text-gray-300"
          >
            {param.label}
            {param.required && <span className="text-red-500 ml-0.5">*</span>}
          </Label>
          {param.type === "integration-resource" && param.integration && param.resourceType ? (
            <IntegrationResourceFieldRenderer
              field={{
                name: param.name,
                label: param.label,
                type: "integration-resource",
                placeholder: param.placeholder,
                required: param.required,
                typeOptions: { resource: { type: param.resourceType, useNameAsValue: param.useNameAsValue } },
              }}
              value={values[param.name]}
              onChange={(val) => onChange((prev) => ({ ...prev, [param.name]: normalizeResourceValue(val) }))}
              organizationId={organizationId}
              integrationId={integrationSelections[param.integration]?.id}
            />
          ) : param.type === "secret_picker" ? (
            <SecretPickerFieldRenderer
              id={`factory-param-${param.name}`}
              placeholder={param.placeholder}
              required={param.required}
              value={values[param.name]}
              onChange={(val) => onChange((prev) => ({ ...prev, [param.name]: val }))}
              organizationId={organizationId}
            />
          ) : (
            <Input
              id={`factory-param-${param.name}`}
              value={values[param.name] ?? ""}
              placeholder={param.placeholder}
              className="h-8 text-xs"
              onChange={(e) => onChange((prev) => ({ ...prev, [param.name]: e.target.value }))}
            />
          )}
          {param.description && <p className="text-[10px] text-slate-400 dark:text-gray-500">{param.description}</p>}
        </div>
      ))}
    </div>
  );
}

function StartingTaskSection({
  busy,
  tasks,
  selectedTaskId,
  onSelectTask,
  prompt,
}: {
  busy: boolean;
  tasks: FactoryStartingTask[];
  selectedTaskId: string | null;
  onSelectTask: (id: string) => void;
  prompt: string | null;
}) {
  return (
    <div className="mb-5">
      <p className="mb-3 text-xs font-semibold text-slate-700 dark:text-gray-300">Choose starting task</p>
      <div className="flex flex-wrap gap-2">
        {tasks.map((task) => {
          const iconMeta = STARTING_TASK_ICONS[task.id];
          const Icon = iconMeta?.icon;
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
              {Icon && <Icon className={cn("size-3.5 shrink-0", iconMeta.iconClassName)} />}
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
