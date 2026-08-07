import type { FactoriesFactoryLine, FactoryApp } from "@/api-client";
import { Icon } from "@/components/Icon";
import { Link } from "@/components/Link/link";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Heading } from "@/components/Heading/heading";
import { Text } from "@/components/Text/text";
import { Button } from "@/components/ui/button";
import { usePermissions } from "@/contexts/usePermissions";
import { useFactoryApps } from "@/hooks/useFactoryData";
import { useCreateCanvas } from "@/hooks/useCanvasData";
import { factoryAppsKey } from "@/hooks/useFactoryData";
import { appPath } from "@/lib/appPaths";
import { cn } from "@/lib/utils";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { getUsageLimitToastMessage } from "@/lib/usageLimits";
import { useQueryClient } from "@tanstack/react-query";
import { ChevronRight, ExternalLink, MoreHorizontal, Pencil, Plus, Workflow } from "lucide-react";
import { useMemo, useState } from "react";
import { NavLink, useNavigate, useParams } from "react-router-dom";
import { CreateFactoryAppDialog } from "../CreateFactoryAppDialog";
import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import { createFactoryLinePath, editFactoryLinePath, factoryLineDetailPath } from "../lib/factoryPagePaths";
import { LineStepArrow, LineStepDisplayNode, LineStepFlow } from "../FactoryLineStepFlow";
import { factoryContentBodyClassName, factoryContentHeaderClassName } from "./factoryPageLayoutStyles";

export function AutomationsPage() {
  const { organizationId, factoryId, factory } = useFactoriesLayout();
  const { canAct, isLoading: permissionsLoading } = usePermissions();

  const { data: apps = [], isLoading: appsLoading } = useFactoryApps(organizationId, factoryId);
  const { lineId: routeLineId } = useParams<{ lineId: string }>();

  const canUpdate = canAct("factories", "update");
  const canCreateApp = canAct("canvases", "create");

  const lines = useMemo(() => factory?.lines ?? [], [factory?.lines]);
  const selectedLineId = routeLineId ?? lines[0]?.id ?? undefined;
  const selectedLine = useMemo(
    () => lines.find((line) => line.id === selectedLineId) ?? lines[0],
    [lines, selectedLineId],
  );

  return (
    <>
      <header className={factoryContentHeaderClassName}>
        <div>
          <Heading level={1} className="!text-xl text-gray-900 dark:text-gray-100">
            Automations
          </Heading>
          <Text className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Factory lines specialize how work moves through the workspace. Each phase runs an app.
          </Text>
        </div>
        <PermissionTooltip
          allowed={canUpdate || permissionsLoading}
          message="You don't have permission to create lines."
        >
          <Button type="button" asChild disabled={!canUpdate} data-testid="automations-create-line-button">
            <Link href={canUpdate ? createFactoryLinePath(organizationId, factoryId) : "#"}>
              <Icon name="plus" />
              New Line
            </Link>
          </Button>
        </PermissionTooltip>
      </header>

      <div className={factoryContentBodyClassName}>
        <div className="grid gap-6 lg:grid-cols-[minmax(0,220px)_minmax(0,1fr)]">
          <LinesList
            organizationId={organizationId}
            factoryId={factoryId}
            lines={lines}
            selectedLineId={selectedLine?.id}
            canUpdate={canUpdate}
          />
          <div className="space-y-6">
            {selectedLine ? (
              <SelectedLineCard
                organizationId={organizationId}
                factoryId={factoryId}
                line={selectedLine}
                canUpdate={canUpdate}
              />
            ) : (
              <EmptyLinesCard
                organizationId={organizationId}
                factoryId={factoryId}
                canUpdate={canUpdate || permissionsLoading}
              />
            )}
            <AppsCard
              organizationId={organizationId}
              factoryId={factoryId}
              apps={apps}
              isLoading={appsLoading}
              canCreate={canCreateApp}
              permissionsLoading={permissionsLoading}
            />
          </div>
        </div>
      </div>
    </>
  );
}

function LinesList({
  organizationId,
  factoryId,
  lines,
  selectedLineId,
  canUpdate,
}: {
  organizationId: string;
  factoryId: string;
  lines: FactoriesFactoryLine[];
  selectedLineId?: string;
  canUpdate: boolean;
}) {
  return (
    <aside
      className="overflow-hidden rounded-lg border border-slate-950/10 bg-white dark:border-gray-700/70 dark:bg-gray-900"
      data-testid="automations-lines-list"
    >
      <div className="flex items-center justify-between border-b border-slate-950/10 px-3 py-2 dark:border-gray-700/70">
        <p className="text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">Lines</p>
        {canUpdate ? (
          <Button
            type="button"
            variant="ghost"
            size="xs"
            asChild
            className="h-6 w-6 p-0 text-gray-500 dark:text-gray-400"
            aria-label="Create line"
            data-testid="automations-lines-list-create"
          >
            <Link href={createFactoryLinePath(organizationId, factoryId)}>
              <Plus className="h-3.5 w-3.5" aria-hidden />
            </Link>
          </Button>
        ) : null}
      </div>

      {lines.length === 0 ? (
        <p className="px-3 py-4 text-sm text-gray-500 dark:text-gray-400">No lines yet.</p>
      ) : (
        <ul>
          {lines.map((line) => {
            if (!line.id) {
              return null;
            }
            const isActive = line.id === selectedLineId;
            const stepsCount = line.steps?.length ?? 0;
            return (
              <li key={line.id}>
                <NavLink
                  to={factoryLineDetailPath(organizationId, factoryId, line.id)}
                  className={cn(
                    "flex items-center gap-2 border-b border-slate-950/5 px-3 py-2 text-sm text-gray-700 last:border-b-0 hover:bg-gray-50 dark:border-gray-700/50 dark:text-gray-300 dark:hover:bg-gray-800/40",
                    isActive && "bg-gray-50 font-medium text-gray-900 dark:bg-gray-800/60 dark:text-gray-100",
                  )}
                  data-testid={`automations-line-${line.id}`}
                >
                  <span className="min-w-0 flex-1 truncate">{line.name || "Unnamed line"}</span>
                  <span className="shrink-0 text-xs text-gray-500 dark:text-gray-400">
                    {stepsCount} {stepsCount === 1 ? "phase" : "phases"}
                  </span>
                </NavLink>
              </li>
            );
          })}
        </ul>
      )}
    </aside>
  );
}

function SelectedLineCard({
  organizationId,
  factoryId,
  line,
  canUpdate,
}: {
  organizationId: string;
  factoryId: string;
  line: FactoriesFactoryLine;
  canUpdate: boolean;
}) {
  const editHref = line.id ? editFactoryLinePath(organizationId, factoryId, line.id) : "#";
  const steps = line.steps ?? [];

  return (
    <section
      className="overflow-hidden rounded-lg border border-slate-950/10 bg-white dark:border-gray-700/70 dark:bg-gray-900"
      data-testid="automations-selected-line"
    >
      <div className="flex items-center justify-between border-b border-slate-950/10 px-4 py-3 dark:border-gray-700/70">
        <div className="flex items-center gap-2">
          <Workflow className="h-4 w-4 text-gray-500 dark:text-gray-400" aria-hidden />
          <h2 className="text-sm font-semibold text-gray-900 dark:text-gray-100">{line.name || "Unnamed line"}</h2>
        </div>
        {canUpdate ? (
          <Button
            type="button"
            variant="ghost"
            size="xs"
            asChild
            className="text-gray-600 dark:text-gray-300"
            data-testid="automations-line-edit"
          >
            <Link href={editHref}>
              <Pencil className="h-3.5 w-3.5" aria-hidden />
              Edit
            </Link>
          </Button>
        ) : null}
      </div>

      <div className="px-6 py-6">
        {steps.length === 0 ? (
          <p className="text-sm text-gray-500 dark:text-gray-400">
            No phases yet. Edit this line to add app-driven phases.
          </p>
        ) : (
          <LineStepFlow className="mx-auto max-w-sm gap-1">
            {steps.map((step, index) => (
              <div key={`${step.name}-${index}`} className="w-full">
                {index === 0 ? null : <LineStepArrow />}
                <LineStepDisplayNode
                  stepName={step.name ?? "step"}
                  appName={step.app?.app ?? "app"}
                  entrypoint={step.app?.entrypoint}
                />
              </div>
            ))}
          </LineStepFlow>
        )}
      </div>
    </section>
  );
}

function EmptyLinesCard({
  organizationId,
  factoryId,
  canUpdate,
}: {
  organizationId: string;
  factoryId: string;
  canUpdate: boolean;
}) {
  return (
    <div
      className="flex flex-col items-center rounded-lg border border-dashed border-slate-300 bg-white px-6 py-12 text-center dark:border-gray-700 dark:bg-gray-900"
      data-testid="automations-empty-state"
    >
      <Workflow className="h-8 w-8 text-slate-400 dark:text-gray-500" aria-hidden />
      <p className="mt-3 text-base font-medium text-slate-900 dark:text-gray-100">No lines yet</p>
      <p className="mt-1 max-w-md text-sm text-gray-500 dark:text-gray-400">
        Lines define how work orders flow through your factory apps.
      </p>
      <Button type="button" asChild className="mt-6" disabled={!canUpdate}>
        <Link href={canUpdate ? createFactoryLinePath(organizationId, factoryId) : "#"}>
          <Icon name="plus" />
          Create line
        </Link>
      </Button>
    </div>
  );
}

function AppsCard({
  organizationId,
  factoryId,
  apps,
  isLoading,
  canCreate,
  permissionsLoading,
}: {
  organizationId: string;
  factoryId: string;
  apps: FactoryApp[];
  isLoading: boolean;
  canCreate: boolean;
  permissionsLoading: boolean;
}) {
  const [createOpen, setCreateOpen] = useState(false);
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const createCanvas = useCreateCanvas(organizationId);

  const handleCreateApp = async (input: { name: string; description: string }) => {
    try {
      const result = await createCanvas.mutateAsync({
        name: input.name,
        description: input.description,
        factoryId,
        method: "ui",
      });
      const canvasId = result?.data?.canvas?.metadata?.id;
      setCreateOpen(false);
      void queryClient.invalidateQueries({ queryKey: factoryAppsKey(organizationId, factoryId) });
      if (canvasId) {
        showSuccessToast("Factory app created.");
        navigate(appPath(organizationId, canvasId, "?edit=1"));
      }
    } catch (error) {
      showErrorToast(getUsageLimitToastMessage(error, "Failed to create factory app"));
      throw error;
    }
  };

  return (
    <section
      className="overflow-hidden rounded-lg border border-slate-950/10 bg-white dark:border-gray-700/70 dark:bg-gray-900"
      data-testid="automations-apps-card"
    >
      <div className="flex items-center justify-between border-b border-slate-950/10 px-4 py-3 dark:border-gray-700/70">
        <div className="flex items-center gap-2">
          <MoreHorizontal className="h-4 w-4 text-gray-500 dark:text-gray-400" aria-hidden />
          <h2 className="text-sm font-semibold text-gray-900 dark:text-gray-100">Apps</h2>
          <span
            className="rounded-full bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-600 dark:bg-gray-800 dark:text-gray-300"
            data-testid="automations-apps-count"
          >
            {apps.length}
          </span>
        </div>
        <PermissionTooltip
          allowed={canCreate || permissionsLoading}
          message="You don't have permission to create apps."
        >
          <Button
            type="button"
            variant="ghost"
            size="xs"
            onClick={() => setCreateOpen(true)}
            disabled={!canCreate}
            className="text-gray-600 dark:text-gray-300"
            data-testid="automations-app-create-button"
          >
            <Plus className="h-3.5 w-3.5" aria-hidden />
            New App
          </Button>
        </PermissionTooltip>
      </div>

      {isLoading ? (
        <p className="px-4 py-4 text-sm text-gray-500 dark:text-gray-400">Loading apps…</p>
      ) : apps.length === 0 ? (
        <p className="px-4 py-4 text-sm text-gray-500 dark:text-gray-400">No apps yet. Apps power line phases.</p>
      ) : (
        <ul>
          {apps.map((app) => (
            <li key={app.id} className="border-b border-slate-950/5 last:border-b-0 dark:border-gray-700/50">
              <Link
                href={appPath(organizationId, app.id ?? "")}
                className="group flex items-center gap-3 px-4 py-3 text-sm text-gray-700 hover:bg-gray-50 dark:text-gray-300 dark:hover:bg-gray-800/40"
                data-testid={`automations-app-${app.id}`}
              >
                <p className="min-w-0 flex-1 truncate font-medium text-gray-900 dark:text-gray-100">{app.name}</p>
                <ExternalLink className="h-3.5 w-3.5 shrink-0 text-gray-400" aria-hidden />
                <ChevronRight
                  className="h-4 w-4 shrink-0 text-gray-400 transition group-hover:text-gray-700 dark:text-gray-500 dark:group-hover:text-gray-300"
                  aria-hidden
                />
              </Link>
            </li>
          ))}
        </ul>
      )}

      <CreateFactoryAppDialog
        open={createOpen}
        isSaving={createCanvas.isPending}
        onClose={() => setCreateOpen(false)}
        onCreate={handleCreateApp}
      />
    </section>
  );
}
