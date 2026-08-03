import type { FactoryApp, FactoriesFactoryLine } from "@/api-client";
import { Icon } from "@/components/Icon";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Text } from "@/components/Text/text";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { GitBranch } from "lucide-react";
import { homeListCardClassName } from "../home/homePageStyles";

interface FactoryLinesPanelProps {
  lines: FactoriesFactoryLine[];
  apps: FactoryApp[];
  isLoading: boolean;
  canUpdate: boolean;
  permissionsLoading: boolean;
  onCreateClick: () => void;
  onEditLine: (line: FactoriesFactoryLine) => void;
}

function appNameForStep(appId: string | undefined, apps: FactoryApp[]): string {
  if (!appId) {
    return "Unknown app";
  }
  return apps.find((app) => app.id === appId)?.name ?? appId;
}

export function FactoryLinesPanel({
  lines,
  apps,
  isLoading,
  canUpdate,
  permissionsLoading,
  onCreateClick,
  onEditLine,
}: FactoryLinesPanelProps) {
  if (isLoading) {
    return <Text className="text-sm text-gray-500">Loading lines…</Text>;
  }

  if (lines.length === 0) {
    return (
      <div className="flex min-h-64 flex-col items-center justify-center rounded-lg border border-dashed border-slate-300 bg-white px-6 py-12 text-center dark:border-gray-700 dark:bg-gray-900">
        <GitBranch className="h-10 w-10 text-slate-400 dark:text-gray-500" aria-hidden />
        <p className="mt-4 text-base font-medium text-slate-900 dark:text-gray-100">No lines yet</p>
        <p className="mt-2 max-w-md text-sm text-gray-500 dark:text-gray-400">
          Lines define how work orders flow through factory apps — each step runs an app entrypoint trigger.
        </p>
        <PermissionTooltip
          allowed={canUpdate || permissionsLoading}
          message="You don't have permission to update factory lines."
        >
          <Button type="button" className="mt-6" onClick={onCreateClick} disabled={!canUpdate}>
            <Icon name="plus" />
            Create line
          </Button>
        </PermissionTooltip>
      </div>
    );
  }

  return (
    <div>
      <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
        <Text className="text-sm text-gray-500 dark:text-gray-400">
          {lines.length} line{lines.length === 1 ? "" : "s"}
        </Text>
        <PermissionTooltip
          allowed={canUpdate || permissionsLoading}
          message="You don't have permission to update factory lines."
        >
          <Button type="button" onClick={onCreateClick} disabled={!canUpdate} data-testid="factory-lines-create-button">
            <Icon name="plus" />
            Create line
          </Button>
        </PermissionTooltip>
      </div>

      <ul className="space-y-3">
        {lines.map((line) => (
          <li key={line.id} className={cn(homeListCardClassName, "p-4")} data-testid={`factory-line-card-${line.name}`}>
            <div className="flex flex-wrap items-start justify-between gap-3">
              <div className="min-w-0">
                <p className="font-medium text-slate-900 dark:text-gray-100">{line.name}</p>
                <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                  {(line.steps?.length ?? 0) === 0
                    ? "No steps"
                    : `${line.steps?.length} step${line.steps?.length === 1 ? "" : "s"}`}
                </p>
              </div>
              {canUpdate ? (
                <Button type="button" variant="outline" size="sm" onClick={() => onEditLine(line)}>
                  Edit
                </Button>
              ) : null}
            </div>

            {line.steps && line.steps.length > 0 ? (
              <ol className="mt-4 space-y-2 border-t border-slate-200 pt-4 dark:border-gray-700">
                {line.steps.map((step, index) => (
                  <li key={`${line.id}-${step.name}-${index}`} className="text-sm">
                    <span className="font-medium text-slate-800 dark:text-gray-200">{step.name}</span>
                    <span className="text-gray-500 dark:text-gray-400">
                      {" "}
                      → {appNameForStep(step.app?.app, apps)}
                      {step.app?.entrypoint ? ` / ${step.app.entrypoint}` : ""}
                    </span>
                  </li>
                ))}
              </ol>
            ) : null}
          </li>
        ))}
      </ul>
    </div>
  );
}
